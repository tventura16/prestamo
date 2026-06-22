package handler

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/prestamos/loan-service/internal/auth"
	"github.com/prestamos/loan-service/internal/models"
	"github.com/prestamos/loan-service/internal/repository"
)

const maxGarantiaBytes = 5 << 20 // 5 MiB

var allowedMimes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

type GarantiaHandler struct {
	repo      *repository.GarantiaRepository
	storePath string
	verifier  *auth.Verifier
}

func NewGarantiaHandler(repo *repository.GarantiaRepository, storePath string, verifier *auth.Verifier) *GarantiaHandler {
	return &GarantiaHandler{repo: repo, storePath: storePath, verifier: verifier}
}

// Register monta las rutas de garantías (entidad + imágenes) bajo /loans/:id.
// Crear/editar es operación de registro (cajero/admin); consultar/descargar,
// abierto a todo rol autenticado.
func (h *GarantiaHandler) Register(rg *gin.RouterGroup) {
	rg.POST("/:id/garantias", h.verifier.GuardRole("admin", "cajero"), h.Create)
	rg.GET("/:id/garantias", h.verifier.GuardRole("admin", "supervisor", "cajero"), h.List)
	rg.GET("/:id/garantias/:gid", h.verifier.GuardRole("admin", "supervisor", "cajero"), h.Get)
	rg.DELETE("/:id/garantias/:gid", h.verifier.GuardRole("admin", "cajero"), h.Delete)
	rg.POST("/:id/garantias/:gid/imagenes", h.verifier.GuardRole("admin", "cajero"), h.UploadImagen)
	rg.GET("/:id/garantias/:gid/imagenes/:iid/download", h.verifier.GuardRole("admin", "supervisor", "cajero"), h.DownloadImagen)
	rg.DELETE("/:id/garantias/:gid/imagenes/:iid", h.verifier.GuardRole("admin", "cajero"), h.DeleteImagen)
}

func (h *GarantiaHandler) Create(c *gin.Context) {
	prestamoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de préstamo inválido"})
		return
	}
	exists, err := h.repo.PrestamoExists(ctxOf(c), prestamoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "préstamo no encontrado"})
		return
	}

	var in models.CreateGarantiaInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	datos, err := models.ValidarDatosGarantia(in.Subtipo, in.Datos)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	moneda := in.Moneda
	if moneda == "" {
		moneda = "BOB"
	}

	g, err := h.repo.Insert(ctxOf(c), models.Garantia{
		PrestamoID:       prestamoID,
		Subtipo:          in.Subtipo,
		Descripcion:      in.Descripcion,
		ValorEstimado:    in.ValorEstimado,
		Moneda:           moneda,
		ClienteGaranteID: in.ClienteGaranteID,
		Datos:            datos,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	g.Imagenes = []models.GarantiaImagen{}
	c.JSON(http.StatusCreated, g)
}

func (h *GarantiaHandler) List(c *gin.Context) {
	prestamoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de préstamo inválido"})
		return
	}
	gs, err := h.repo.ListByPrestamo(ctxOf(c), prestamoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	for i := range gs {
		imgs, err := h.repo.ListImagenesByGarantia(ctxOf(c), gs[i].ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		gs[i].Imagenes = imgs
	}
	c.JSON(http.StatusOK, gin.H{"total": len(gs), "items": gs})
}

func (h *GarantiaHandler) Get(c *gin.Context) {
	prestamoID, gid, ok := h.parseIDs(c)
	if !ok {
		return
	}
	g, valid := h.loadGarantia(c, gid, prestamoID)
	if !valid {
		return
	}
	imgs, e := h.repo.ListImagenesByGarantia(ctxOf(c), gid)
	if e != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": e.Error()})
		return
	}
	g.Imagenes = imgs
	c.JSON(http.StatusOK, g)
}

func (h *GarantiaHandler) Delete(c *gin.Context) {
	prestamoID, gid, ok := h.parseIDs(c)
	if !ok {
		return
	}
	if _, valid := h.loadGarantia(c, gid, prestamoID); !valid {
		return
	}
	// Borra archivos antes de eliminar el registro (cascade elimina filas).
	imgs, _ := h.repo.ListImagenesByGarantia(ctxOf(c), gid)
	if err := h.repo.Delete(ctxOf(c), gid); err != nil {
		h.respondErr(c, err)
		return
	}
	for _, m := range imgs {
		_ = os.Remove(m.Ruta)
	}
	c.JSON(http.StatusOK, gin.H{"deleted": gid})
}

func (h *GarantiaHandler) UploadImagen(c *gin.Context) {
	prestamoID, gid, ok := h.parseIDs(c)
	if !ok {
		return
	}
	if _, valid := h.loadGarantia(c, gid, prestamoID); !valid {
		return
	}

	fileHeader, err := c.FormFile("imagen")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "se requiere el archivo 'imagen'"})
		return
	}
	if fileHeader.Size > maxGarantiaBytes {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "la imagen excede el máximo de 5 MB"})
		return
	}
	src, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo leer el archivo"})
		return
	}
	defer src.Close()

	head := make([]byte, 512)
	n, _ := io.ReadFull(src, head)
	mime := http.DetectContentType(head[:n])
	ext, allowed := allowedMimes[mime]
	if !allowed {
		c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": "tipo no permitido; solo JPG, PNG o WEBP"})
		return
	}
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo procesar el archivo"})
		return
	}

	dir := filepath.Join(h.storePath, gid.String())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo preparar el almacenamiento"})
		return
	}
	ruta := filepath.Join(dir, uuid.NewString()+ext)
	dst, err := os.Create(ruta)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo guardar la imagen"})
		return
	}
	// Acota la escritura al máximo permitido: fileHeader.Size es el tamaño
	// declarado por el cliente y puede mentir; LimitReader corta de verdad.
	written, copyErr := io.Copy(dst, io.LimitReader(src, maxGarantiaBytes+1))
	closeErr := dst.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(ruta)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo escribir la imagen"})
		return
	}
	if written > maxGarantiaBytes {
		_ = os.Remove(ruta)
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "la imagen excede el máximo de 5 MB"})
		return
	}

	var descripcion *string
	if d := c.PostForm("descripcion"); d != "" {
		descripcion = &d
	}
	var subidoPor *uuid.UUID
	if claims, ok := auth.FromContext(c); ok {
		subidoPor = &claims.UserID
	}

	m, err := h.repo.InsertImagen(ctxOf(c), models.GarantiaImagen{
		GarantiaID:    gid,
		NombreArchivo: fileHeader.Filename,
		Ruta:          ruta,
		Mime:          mime,
		TamanioBytes:  written,
		Descripcion:   descripcion,
		SubidoPor:     subidoPor,
	})
	if err != nil {
		_ = os.Remove(ruta)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, m)
}

func (h *GarantiaHandler) DownloadImagen(c *gin.Context) {
	_, gid, ok := h.parseIDs(c)
	if !ok {
		return
	}
	iid, err := uuid.Parse(c.Param("iid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de imagen inválido"})
		return
	}
	m, err := h.repo.GetImagen(ctxOf(c), iid)
	if err != nil {
		h.respondErr(c, err)
		return
	}
	if m.GarantiaID != gid {
		c.JSON(http.StatusNotFound, gin.H{"error": "imagen no encontrada"})
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, m.NombreArchivo))
	c.File(m.Ruta)
}

func (h *GarantiaHandler) DeleteImagen(c *gin.Context) {
	_, gid, ok := h.parseIDs(c)
	if !ok {
		return
	}
	iid, err := uuid.Parse(c.Param("iid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de imagen inválido"})
		return
	}
	m, err := h.repo.GetImagen(ctxOf(c), iid)
	if err != nil {
		h.respondErr(c, err)
		return
	}
	if m.GarantiaID != gid {
		c.JSON(http.StatusNotFound, gin.H{"error": "imagen no encontrada"})
		return
	}
	if _, err := h.repo.DeleteImagen(ctxOf(c), iid); err != nil {
		h.respondErr(c, err)
		return
	}
	_ = os.Remove(m.Ruta)
	c.JSON(http.StatusOK, gin.H{"deleted": iid})
}

// ─── helpers ───

// parseIDs lee :id (préstamo) y :gid (garantía); responde 400 y retorna false
// si alguno es inválido.
func (h *GarantiaHandler) parseIDs(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	prestamoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de préstamo inválido"})
		return uuid.Nil, uuid.Nil, false
	}
	gid, err := uuid.Parse(c.Param("gid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de garantía inválido"})
		return uuid.Nil, uuid.Nil, false
	}
	return prestamoID, gid, true
}

// loadGarantia obtiene la garantía y verifica que pertenezca al préstamo.
// Responde el error apropiado y retorna ok=false si no procede.
func (h *GarantiaHandler) loadGarantia(c *gin.Context, gid, prestamoID uuid.UUID) (models.Garantia, bool) {
	g, err := h.repo.Get(ctxOf(c), gid)
	if err != nil {
		h.respondErr(c, err)
		return models.Garantia{}, false
	}
	if g.PrestamoID != prestamoID {
		c.JSON(http.StatusNotFound, gin.H{"error": "garantía no encontrada"})
		return models.Garantia{}, false
	}
	return g, true
}

func (h *GarantiaHandler) respondErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, repository.ErrGarantiaNotFound), errors.Is(err, repository.ErrImagenNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
