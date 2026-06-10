package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/prestamos/report-service/internal/models"
	"github.com/prestamos/report-service/internal/repository"
)

// ReportService orquesta consultas a las 3 DBs para producir agregaciones.
type ReportService struct {
	pagos     *repository.PagosRepository
	prestamos *repository.PrestamosRepository
	clientes  *repository.ClientesRepository
	tz        *time.Location
}

func NewReportService(p *repository.PagosRepository, pr *repository.PrestamosRepository, c *repository.ClientesRepository) *ReportService {
	tz, err := time.LoadLocation("America/La_Paz")
	if err != nil {
		tz = time.UTC
	}
	return &ReportService{pagos: p, prestamos: pr, clientes: c, tz: tz}
}

// Dashboard devuelve métricas para la pantalla principal.
func (s *ReportService) Dashboard(ctx context.Context) (models.Dashboard, error) {
	var d models.Dashboard
	var err error

	if d.PrestamosActivos, err = s.prestamos.CountByEstado(ctx, "activo"); err != nil {
		return d, err
	}
	if d.PrestamosEnMora, err = s.prestamos.CountByEstado(ctx, "mora"); err != nil {
		return d, err
	}
	if d.ClientesActivos, err = s.clientes.CountActivos(ctx); err != nil {
		return d, err
	}
	if d.CuotasVencidas, err = s.prestamos.CountCuotasVencidas(ctx); err != nil {
		return d, err
	}
	if d.CarteraOutstanding, err = s.prestamos.CarteraOutstanding(ctx); err != nil {
		return d, err
	}

	now := time.Now().In(s.tz)
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, s.tz)
	startOfNextMonth := startOfMonth.AddDate(0, 1, 0)
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, s.tz)
	startOfNextDay := startOfDay.AddDate(0, 0, 1)

	mes, err := s.pagos.IngresosEnRango(ctx, startOfMonth, startOfNextMonth)
	if err != nil {
		return d, err
	}
	d.IngresosMes = mes.Ingresos

	hoy, err := s.pagos.IngresosEnRango(ctx, startOfDay, startOfNextDay)
	if err != nil {
		return d, err
	}
	d.IngresosHoy = hoy.Ingresos

	return d, nil
}

// Diario devuelve el reporte de un día específico (YYYY-MM-DD en TZ La Paz).
func (s *ReportService) Diario(ctx context.Context, fecha time.Time) (models.ReporteDiario, error) {
	startOfDay := time.Date(fecha.Year(), fecha.Month(), fecha.Day(), 0, 0, 0, 0, s.tz)
	endOfDay := startOfDay.AddDate(0, 0, 1)

	rep := models.ReporteDiario{Fecha: startOfDay.Format("2006-01-02")}

	ingresos, err := s.pagos.IngresosEnRango(ctx, startOfDay, endOfDay)
	if err != nil {
		return rep, err
	}
	rep.Ingresos = ingresos.Ingresos
	rep.PagosRecibidos = ingresos.NumPagos
	rep.MoraCobrada = ingresos.Mora

	if rep.PrestamosNuevos, err = s.prestamos.CountNuevosEnRango(ctx, startOfDay, endOfDay); err != nil {
		return rep, err
	}
	if rep.ClientesNuevos, err = s.clientes.CountNuevosEnRango(ctx, startOfDay, endOfDay); err != nil {
		return rep, err
	}
	return rep, nil
}

// Mensual devuelve el reporte de un mes (year+month).
func (s *ReportService) Mensual(ctx context.Context, year, month int) (models.ReporteMensual, error) {
	if month < 1 || month > 12 {
		return models.ReporteMensual{}, fmt.Errorf("mes inválido: %d", month)
	}
	startOfMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, s.tz)
	endOfMonth := startOfMonth.AddDate(0, 1, 0)

	rep := models.ReporteMensual{Anio: year, Mes: month}

	ingresos, err := s.pagos.IngresosEnRango(ctx, startOfMonth, endOfMonth)
	if err != nil {
		return rep, err
	}
	rep.Ingresos = ingresos.Ingresos
	rep.InteresesPagados = ingresos.Intereses
	rep.MoraCobrada = ingresos.Mora
	rep.PagosRecibidos = ingresos.NumPagos

	if rep.PrestamosNuevos, err = s.prestamos.CountNuevosEnRango(ctx, startOfMonth, endOfMonth); err != nil {
		return rep, err
	}
	if rep.ClientesNuevos, err = s.clientes.CountNuevosEnRango(ctx, startOfMonth, endOfMonth); err != nil {
		return rep, err
	}
	return rep, nil
}

// Vencidas lista las cuotas vencidas sin pagar (con días de atraso).
func (s *ReportService) Vencidas(ctx context.Context, limit int) ([]models.CuotaVencida, error) {
	return s.prestamos.CuotasVencidas(ctx, limit)
}

// PorCliente devuelve el reporte completo de un cliente.
func (s *ReportService) PorCliente(ctx context.Context, clienteID uuid.UUID) (models.ReporteCliente, error) {
	rep := models.ReporteCliente{ClienteID: clienteID}

	prestamos, err := s.prestamos.ResumenPorCliente(ctx, clienteID)
	if err != nil {
		return rep, err
	}
	rep.NumPrestamos = len(prestamos)
	for _, p := range prestamos {
		if p.Estado == "activo" || p.Estado == "mora" {
			rep.PrestamosActivos++
		}
		rep.CuotasVencidas += p.CuotasVencidas
	}

	if rep.TotalPrestado, err = s.prestamos.TotalPrestadoPorCliente(ctx, clienteID); err != nil {
		return rep, err
	}
	if rep.TotalPagado, err = s.pagos.TotalPagadoPorCliente(ctx, clienteID); err != nil {
		return rep, err
	}
	if rep.SaldoTotal, rep.MoraTotal, err = s.prestamos.SaldoYMoraCliente(ctx, clienteID); err != nil {
		return rep, err
	}

	// Asociar total_pagado por préstamo individual
	for i, p := range prestamos {
		paid, err := s.pagos.TotalPagadoPorPrestamo(ctx, p.ID)
		if err != nil {
			return rep, err
		}
		prestamos[i].TotalPagado = paid
	}
	rep.Prestamos = prestamos

	// Datos del cliente (no fatal si falta).
	if cli, err := s.clientes.GetCliente(ctx, clienteID); err != nil {
		return rep, err
	} else {
		rep.Cliente = cli
	}

	// Elegibilidad para un nuevo préstamo según el historial (regla §7).
	aprobarSiMora, maxActivos, err := s.prestamos.ParametrosElegibilidad(ctx)
	if err != nil {
		return rep, err
	}
	rep.Elegible, rep.MotivoInelegible = evaluarElegibilidad(rep, aprobarSiMora, maxActivos)

	return rep, nil
}

// evaluarElegibilidad aplica las reglas de negocio: sin mora activa (salvo
// parámetro) y por debajo del máximo de préstamos activos. "Mora activa"
// significa tener cuotas vencidas impagas —igual que la regla que aplica el
// loan-service al aprobar—, aunque el interés moratorio aún no se haya
// devengado por el período de gracia.
func evaluarElegibilidad(rep models.ReporteCliente, aprobarSiMora bool, maxActivos int) (bool, string) {
	if (rep.CuotasVencidas > 0 || rep.MoraTotal > 0) && !aprobarSiMora {
		return false, "el cliente tiene mora activa"
	}
	if maxActivos > 0 && rep.PrestamosActivos >= maxActivos {
		return false, fmt.Sprintf("alcanzó el máximo de préstamos activos (%d)", maxActivos)
	}
	return true, ""
}
