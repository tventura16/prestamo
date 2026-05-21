package handler

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/prestamos/document-service/internal/models"
	"github.com/prestamos/document-service/internal/repository"
	"github.com/prestamos/document-service/internal/service"
)

type DocumentoHandler struct {
	svc  *service.DocumentService
	repo *repository.DocumentoRepository
}

func NewDocumentoHandler(svc *service.DocumentService, repo *repository.DocumentoRepository) *DocumentoHandler {
	return &DocumentoHandler{svc: svc, repo: repo}
}

func (h *DocumentoHandler) Register(rg *gin.RouterGroup) {
	rg.GET("", h.List)
	rg.GET("/:id", h.Get)
	rg.GET("/:id/download", h.Download)
	rg.POST("/contract", h.Contract)
	rg.POST("/plan", h.Plan)
	rg.POST("/receipt", h.Receipt)
	rg.POST("/statement", h.Statement)
}

func (h *DocumentoHandler) Contract(c *gin.Context) {
	var in models.ContratoInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	d, err := h.svc.GenerarContrato(ctxOf(c), in)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, d)
}

func (h *DocumentoHandler) Plan(c *gin.Context) {
	var in models.PlanPagosInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	d, err := h.svc.GenerarPlanPagos(ctxOf(c), in)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, d)
}

func (h *DocumentoHandler) Receipt(c *gin.Context) {
	var in models.ReciboInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	d, err := h.svc.GenerarRecibo(ctxOf(c), in)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, d)
}

func (h *DocumentoHandler) Statement(c *gin.Context) {
	var in models.EstadoCuentaInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	d, err := h.svc.GenerarEstadoCuenta(ctxOf(c), in)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, d)
}

func (h *DocumentoHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	tipo := c.Query("tipo")

	res, err := h.repo.List(ctxOf(c), page, limit, tipo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *DocumentoHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}
	d, err := h.repo.GetByID(ctxOf(c), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, d)
}

func (h *DocumentoHandler) Download(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}
	d, err := h.repo.GetByID(ctxOf(c), id)
	if err != nil {
		respondError(c, err)
		return
	}
	if _, err := os.Stat(d.Ruta); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "archivo no encontrado en disco", "ruta": d.Ruta})
		return
	}
	c.Header("Content-Disposition", `inline; filename="`+d.NombreArchivo+`"`)
	c.Header("Content-Type", "application/pdf")
	c.File(d.Ruta)
}

func respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, repository.ErrClienteNotFound),
		errors.Is(err, repository.ErrPrestamoNotFound),
		errors.Is(err, repository.ErrPagoNotFound),
		errors.Is(err, repository.ErrDocumentoNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

func ctxOf(c *gin.Context) context.Context { return c.Request.Context() }
