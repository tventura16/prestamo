package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/prestamos/report-service/internal/auth"
	"github.com/prestamos/report-service/internal/export"
	"github.com/prestamos/report-service/internal/models"
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
// Todos los endpoints aceptan ?format=json|csv|xlsx|pdf (json por defecto).
func (h *ReportHandler) Register(rg *gin.RouterGroup) {
	reportes := h.verifier.GuardRole("admin", "supervisor")
	rg.GET("/dashboard", reportes, h.Dashboard)
	rg.GET("/daily", reportes, h.Daily)
	rg.GET("/monthly", reportes, h.Monthly)
	rg.GET("/overdue", reportes, h.Overdue)
	rg.GET("/clients/:id", reportes, h.PorCliente)
}

// respond entrega el reporte en JSON (default) o construye el export al formato
// pedido. El builder solo se invoca si efectivamente se exporta.
func (h *ReportHandler) respond(c *gin.Context, jsonObj any, build func() export.Report) {
	f, ok := export.ParseFormat(c.Query("format"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "format inválido: usa json|csv|xlsx|pdf"})
		return
	}
	if f == export.FormatJSON {
		c.JSON(http.StatusOK, jsonObj)
		return
	}
	export.Respond(c, f, build())
}

func (h *ReportHandler) Dashboard(c *gin.Context) {
	res, err := h.svc.Dashboard(ctxOf(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.respond(c, res, func() export.Report { return dashboardReport(res) })
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
	h.respond(c, res, func() export.Report { return diarioReport(res) })
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
	h.respond(c, res, func() export.Report { return mensualReport(res) })
}

func (h *ReportHandler) Overdue(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	cuotas, err := h.svc.Vencidas(ctxOf(c), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.respond(c, gin.H{"total": len(cuotas), "items": cuotas}, func() export.Report { return vencidasReport(cuotas) })
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
	h.respond(c, res, func() export.Report { return clienteReport(res) })
}

func ctxOf(c *gin.Context) context.Context {
	return c.Request.Context()
}

// ───── Builders: modelo → reporte tabular exportable ─────

func f2(x float64) string { return strconv.FormatFloat(x, 'f', 2, 64) }

func dashboardReport(d models.Dashboard) export.Report {
	return export.Report{
		Filename: "dashboard-" + time.Now().Format("2006-01-02"),
		Title:    "Dashboard general",
		Sheets: []export.Sheet{{
			Title:   "Dashboard",
			Headers: []string{"Métrica", "Valor"},
			Rows: [][]string{
				{"Préstamos activos", strconv.Itoa(d.PrestamosActivos)},
				{"Préstamos en mora", strconv.Itoa(d.PrestamosEnMora)},
				{"Clientes activos", strconv.Itoa(d.ClientesActivos)},
				{"Cuotas vencidas", strconv.Itoa(d.CuotasVencidas)},
				{"Ingresos del mes", f2(d.IngresosMes)},
				{"Ingresos de hoy", f2(d.IngresosHoy)},
				{"Cartera vigente", f2(d.CarteraOutstanding)},
			},
		}},
	}
}

func diarioReport(d models.ReporteDiario) export.Report {
	return export.Report{
		Filename: "reporte-diario-" + d.Fecha,
		Title:    "Reporte diario · " + d.Fecha,
		Sheets: []export.Sheet{{
			Title:   "Diario",
			Headers: []string{"Métrica", "Valor"},
			Rows: [][]string{
				{"Fecha", d.Fecha},
				{"Ingresos", f2(d.Ingresos)},
				{"Pagos recibidos", strconv.Itoa(d.PagosRecibidos)},
				{"Mora cobrada", f2(d.MoraCobrada)},
				{"Préstamos nuevos", strconv.Itoa(d.PrestamosNuevos)},
				{"Clientes nuevos", strconv.Itoa(d.ClientesNuevos)},
			},
		}},
	}
}

func mensualReport(m models.ReporteMensual) export.Report {
	periodo := fmt.Sprintf("%04d-%02d", m.Anio, m.Mes)
	return export.Report{
		Filename: "reporte-mensual-" + periodo,
		Title:    "Reporte mensual · " + periodo,
		Sheets: []export.Sheet{{
			Title:   "Mensual",
			Headers: []string{"Métrica", "Valor"},
			Rows: [][]string{
				{"Período", periodo},
				{"Ingresos", f2(m.Ingresos)},
				{"Intereses pagados", f2(m.InteresesPagados)},
				{"Mora cobrada", f2(m.MoraCobrada)},
				{"Pagos recibidos", strconv.Itoa(m.PagosRecibidos)},
				{"Préstamos nuevos", strconv.Itoa(m.PrestamosNuevos)},
				{"Clientes nuevos", strconv.Itoa(m.ClientesNuevos)},
			},
		}},
	}
}

func vencidasReport(cuotas []models.CuotaVencida) export.Report {
	rows := make([][]string, 0, len(cuotas))
	for _, c := range cuotas {
		rows = append(rows, []string{
			c.PrestamoID.String(),
			c.ClienteID.String(),
			strconv.Itoa(c.Numero),
			c.FechaVencimiento.Format("2006-01-02"),
			strconv.Itoa(c.DiasVencidos),
			f2(c.Total),
			f2(c.SaldoPendiente),
			f2(c.MoraAcumulada),
			c.Estado,
		})
	}
	return export.Report{
		Filename: "cuotas-vencidas-" + time.Now().Format("2006-01-02"),
		Title:    "Cuotas vencidas",
		Sheets: []export.Sheet{{
			Title:   "Vencidas",
			Headers: []string{"Préstamo", "Cliente", "Cuota #", "Vencimiento", "Días venc.", "Total", "Saldo", "Mora", "Estado"},
			Rows:    rows,
		}},
	}
}

func clienteReport(r models.ReporteCliente) export.Report {
	resumen := export.Sheet{
		Title:   "Resumen",
		Headers: []string{"Métrica", "Valor"},
		Rows: [][]string{
			{"Cliente", r.ClienteID.String()},
			{"Préstamos", strconv.Itoa(r.NumPrestamos)},
			{"Préstamos activos", strconv.Itoa(r.PrestamosActivos)},
			{"Total prestado", f2(r.TotalPrestado)},
			{"Total pagado", f2(r.TotalPagado)},
			{"Saldo total", f2(r.SaldoTotal)},
			{"Mora total", f2(r.MoraTotal)},
			{"Cuotas vencidas", strconv.Itoa(r.CuotasVencidas)},
		},
	}

	prows := make([][]string, 0, len(r.Prestamos))
	for _, p := range r.Prestamos {
		monto := ""
		if p.MontoAprobado != nil {
			monto = f2(*p.MontoAprobado)
		}
		prows = append(prows, []string{
			p.ID.String(),
			p.Estado,
			monto,
			strconv.Itoa(p.NumCuotas),
			strconv.Itoa(p.CuotasPagadas),
			strconv.Itoa(p.CuotasVencidas),
			f2(p.SaldoPendiente),
			f2(p.TotalPagado),
			p.FechaSolicitud.Format("2006-01-02"),
		})
	}
	prestamos := export.Sheet{
		Title:   "Préstamos",
		Headers: []string{"ID", "Estado", "Monto aprobado", "Cuotas", "Pagadas", "Vencidas", "Saldo", "Pagado", "Solicitud"},
		Rows:    prows,
	}

	return export.Report{
		Filename: "reporte-cliente-" + r.ClienteID.String()[:8],
		Title:    "Reporte por cliente",
		Sheets:   []export.Sheet{resumen, prestamos},
	}
}
