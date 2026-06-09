package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/prestamos/report-service/internal/auth"
	"github.com/prestamos/report-service/internal/service"
)

type ReportHandler struct {
	svc      *service.ReportService
	verifier *auth.Verifier
}

func NewReportHandler(svc *service.ReportService, verifier *auth.Verifier) *ReportHandler {
	return &ReportHandler{svc: svc, verifier: verifier}
}

// Register monta las rutas de reportes con autorización por rol (alcance §4):
// la reportería financiera y operativa es competencia de supervisor/admin.
func (h *ReportHandler) Register(rg *gin.RouterGroup) {
	reportes := h.verifier.GuardRole("admin", "supervisor")
	rg.GET("/dashboard", reportes, h.Dashboard)
	rg.GET("/daily", reportes, h.Daily)
	rg.GET("/monthly", reportes, h.Monthly)
	rg.GET("/overdue", reportes, h.Overdue)
	rg.GET("/clients/:id", reportes, h.PorCliente)
}

func (h *ReportHandler) Dashboard(c *gin.Context) {
	res, err := h.svc.Dashboard(ctxOf(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *ReportHandler) Daily(c *gin.Context) {
	fechaStr := c.DefaultQuery("date", time.Now().Format("2006-01-02"))
	fecha, err := time.Parse("2006-01-02", fechaStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date debe ser YYYY-MM-DD"})
		return
	}
	res, err := h.svc.Diario(ctxOf(c), fecha)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *ReportHandler) Monthly(c *gin.Context) {
	now := time.Now()
	year, _ := strconv.Atoi(c.DefaultQuery("year", strconv.Itoa(now.Year())))
	month, _ := strconv.Atoi(c.DefaultQuery("month", strconv.Itoa(int(now.Month()))))

	res, err := h.svc.Mensual(ctxOf(c), year, month)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *ReportHandler) Overdue(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	cuotas, err := h.svc.Vencidas(ctxOf(c), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"total": len(cuotas), "items": cuotas})
}

func (h *ReportHandler) PorCliente(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}
	res, err := h.svc.PorCliente(ctxOf(c), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func ctxOf(c *gin.Context) context.Context {
	return c.Request.Context()
}
