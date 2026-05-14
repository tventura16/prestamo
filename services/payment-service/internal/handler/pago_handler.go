package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/prestamos/payment-service/internal/models"
	"github.com/prestamos/payment-service/internal/repository"
	"github.com/prestamos/payment-service/internal/service"
)

type PagoHandler struct {
	svc  *service.PaymentService
	repo *repository.PagoRepository
}

func NewPagoHandler(svc *service.PaymentService, repo *repository.PagoRepository) *PagoHandler {
	return &PagoHandler{svc: svc, repo: repo}
}

func (h *PagoHandler) Register(rg *gin.RouterGroup) {
	rg.GET("", h.List)
	rg.POST("", h.Create)
	rg.GET("/:id", h.Get)
	rg.GET("/:id/receipt", h.Receipt)
}

func (h *PagoHandler) Create(c *gin.Context) {
	var in models.CreatePagoInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	res, err := h.svc.Register(ctxOf(c), in)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, res)
}

func (h *PagoHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	var prestamoID, cuotaID, clienteID *uuid.UUID
	if s := c.Query("prestamo_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "prestamo_id inválido"})
			return
		}
		prestamoID = &id
	}
	if s := c.Query("cuota_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cuota_id inválido"})
			return
		}
		cuotaID = &id
	}
	if s := c.Query("cliente_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cliente_id inválido"})
			return
		}
		clienteID = &id
	}

	res, err := h.repo.List(ctxOf(c), page, limit, prestamoID, cuotaID, clienteID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *PagoHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}
	p, movs, err := h.repo.GetByID(ctxOf(c), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"pago": p, "movimientos": movs})
}

func (h *PagoHandler) Receipt(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}
	p, movs, err := h.repo.GetByID(ctxOf(c), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"recibo_numero":  p.NumeroRecibo,
		"fecha_pago":     p.FechaPago,
		"cliente_id":     p.ClienteID,
		"prestamo_id":    p.PrestamoID,
		"cuota_id":       p.CuotaID,
		"monto_pagado":   p.MontoPagado,
		"capital":        p.CapitalPagado,
		"interes":        p.InteresPagado,
		"mora":           p.MoraPagada,
		"metodo_pago":    p.MetodoPago,
		"movimientos":    movs,
		"operador":       p.UsuarioID,
		"anulado":        p.Anulado,
	})
}

func respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, repository.ErrCuotaNotFound), errors.Is(err, repository.ErrPagoNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, repository.ErrCuotaPagada):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, repository.ErrOverpayment):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

func ctxOf(c *gin.Context) context.Context {
	return c.Request.Context()
}
