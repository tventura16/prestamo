package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/prestamos/loan-service/internal/auth"
	"github.com/prestamos/loan-service/internal/models"
	"github.com/prestamos/loan-service/internal/repository"
)

type PrestamoHandler struct {
	repo     *repository.PrestamoRepository
	cuota    *repository.CuotaRepository
	verifier *auth.Verifier
}

func NewPrestamoHandler(p *repository.PrestamoRepository, c *repository.CuotaRepository, verifier *auth.Verifier) *PrestamoHandler {
	return &PrestamoHandler{repo: p, cuota: c, verifier: verifier}
}

// Register monta las rutas de préstamos con autorización por rol (alcance §4):
// consulta abierta a cualquier rol; alta de solicitudes para cajero; la
// aprobación y el rechazo quedan reservados a supervisor/admin.
func (h *PrestamoHandler) Register(rg *gin.RouterGroup) {
	rg.GET("", h.verifier.GuardRole("admin", "supervisor", "cajero"), h.List)
	rg.POST("", h.verifier.GuardRole("admin", "cajero"), h.Create)
	rg.GET("/:id", h.verifier.GuardRole("admin", "supervisor", "cajero"), h.Get)
	rg.POST("/:id/approve", h.verifier.GuardRole("admin", "supervisor"), h.Approve)
	rg.POST("/:id/reject", h.verifier.GuardRole("admin", "supervisor"), h.Reject)
	rg.GET("/:id/schedule", h.verifier.GuardRole("admin", "supervisor", "cajero"), h.Schedule)
}

func (h *PrestamoHandler) Create(c *gin.Context) {
	var in models.CreatePrestamoInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p, err := h.repo.Create(ctxOf(c), in)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, p)
}

func (h *PrestamoHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	var clienteID *uuid.UUID
	if s := c.Query("cliente_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cliente_id inválido"})
			return
		}
		clienteID = &id
	}

	var estado *models.Estado
	if s := c.Query("estado"); s != "" {
		e := models.Estado(s)
		estado = &e
	}

	res, err := h.repo.List(ctxOf(c), page, limit, clienteID, estado)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *PrestamoHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}
	p, err := h.repo.GetByID(ctxOf(c), id)
	if err != nil {
		respondRepoError(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *PrestamoHandler) Approve(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}
	var in models.ApprovePrestamoInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p, cuotas, err := h.repo.Approve(ctxOf(c), id, in)
	if err != nil {
		respondRepoError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"prestamo": p,
		"cuotas":   cuotas,
	})
}

func (h *PrestamoHandler) Reject(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}
	var in models.RejectPrestamoInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p, err := h.repo.Reject(ctxOf(c), id, in)
	if err != nil {
		respondRepoError(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *PrestamoHandler) Schedule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}
	if _, err := h.repo.GetByID(ctxOf(c), id); err != nil {
		respondRepoError(c, err)
		return
	}
	cuotas, err := h.cuota.ListByPrestamo(ctxOf(c), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"cuotas": cuotas})
}

func respondRepoError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, repository.ErrAlreadyDecided), errors.Is(err, repository.ErrInvalidState),
		errors.Is(err, repository.ErrClienteConMora):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

func ctxOf(c *gin.Context) context.Context {
	return c.Request.Context()
}
