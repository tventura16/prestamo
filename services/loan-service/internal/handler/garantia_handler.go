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

// allowedMimes mapea el content-type detectado por contenido a su extensión.
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

// Register monta las rutas de garantías colgando de /loans/:id. Subir/eliminar
// es operación de registro (cajero/admin); consultar/descargar, abierto a todo
// rol autenticado.
func (h *GarantiaHandler) Register(rg *gin.RouterGroup) {
	rg.POST("/:id/garantias", h.verifier.GuardRole("admin", "cajero"), h.Upload)
	rg.GET("/:id/garantias", h.verifier.GuardRole("admin", "supervisor", "cajero"), h.List)
	rg.GET("/:id/garantias/:gid/download", h.verifier.GuardRole("admin", "supervisor", "cajero"), h.Download)
	rg.DELETE("/:id/garantias/:gid", h.verifier.GuardRole("admin", "cajero"), h.Delete)
}

func (h *GarantiaHandler) Upload(c *gin.Context) {
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

	// Detecta el tipo por CONTENIDO (no por el header del cliente, manipulable).
	head := make([]byte, 512)
	n, _ := io.ReadFull(src, head)
	mime := http.DetectContentType(head[:n])
	ext, ok := allowedMimes[mime]
	if !ok {
		c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": "tipo no permitido; solo JPG, PNG o WEBP"})
		return
	}
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo procesar el archivo"})
		return
	}

	// Guarda en <store>/<prestamo_id>/<uuid>.<ext>.
	dir := filepath.Join(h.storePath, prestamoID.String())
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
	written, copyErr := io.Copy(dst, src)
	closeErr := dst.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(ruta)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo escribir la imagen"})
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

	g, err := h.repo.Insert(ctxOf(c), models.Garantia{
		PrestamoID:    prestamoID,
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
	c.JSON(http.StatusOK, gin.H{"total": len(gs), "items": gs})
}

func (h *GarantiaHandler) Download(c *gin.Context) {
	prestamoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de préstamo inválido"})
		return
	}
	gid, err := uuid.Parse(c.Param("gid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de garantía inválido"})
		return
	}
	g, err := h.repo.Get(ctxOf(c), gid)
	if err != nil {
		h.respondGarantiaError(c, err)
		return
	}
	if g.PrestamoID != prestamoID {
		c.JSON(http.StatusNotFound, gin.H{"error": "garantía no encontrada"})
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, g.NombreArchivo))
	c.File(g.Ruta)
}

func (h *GarantiaHandler) Delete(c *gin.Context) {
	prestamoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de préstamo inválido"})
		return
	}
	gid, err := uuid.Parse(c.Param("gid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de garantía inválido"})
		return
	}
	// Verifica pertenencia antes de borrar.
	g, err := h.repo.Get(ctxOf(c), gid)
	if err != nil {
		h.respondGarantiaError(c, err)
		return
	}
	if g.PrestamoID != prestamoID {
		c.JSON(http.StatusNotFound, gin.H{"error": "garantía no encontrada"})
		return
	}
	if _, err := h.repo.Delete(ctxOf(c), gid); err != nil {
		h.respondGarantiaError(c, err)
		return
	}
	_ = os.Remove(g.Ruta) // best-effort: el registro ya no existe
	c.JSON(http.StatusOK, gin.H{"deleted": gid})
}

func (h *GarantiaHandler) respondGarantiaError(c *gin.Context, err error) {
	if errors.Is(err, repository.ErrGarantiaNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}
