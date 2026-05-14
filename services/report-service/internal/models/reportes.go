package models

import (
	"time"

	"github.com/google/uuid"
)

type Dashboard struct {
	PrestamosActivos    int     `json:"prestamos_activos"`
	PrestamosEnMora     int     `json:"prestamos_en_mora"`
	ClientesActivos     int     `json:"clientes_activos"`
	CuotasVencidas      int     `json:"cuotas_vencidas"`
	IngresosMes         float64 `json:"ingresos_mes"`
	IngresosHoy         float64 `json:"ingresos_hoy"`
	CarteraOutstanding  float64 `json:"cartera_outstanding"`
}

type ReporteDiario struct {
	Fecha            string  `json:"fecha"`
	Ingresos         float64 `json:"ingresos"`
	PagosRecibidos   int     `json:"pagos_recibidos"`
	MoraCobrada      float64 `json:"mora_cobrada"`
	PrestamosNuevos  int     `json:"prestamos_nuevos"`
	ClientesNuevos   int     `json:"clientes_nuevos"`
}

type ReporteMensual struct {
	Anio              int     `json:"anio"`
	Mes               int     `json:"mes"`
	Ingresos          float64 `json:"ingresos"`
	InteresesPagados  float64 `json:"intereses_pagados"`
	MoraCobrada       float64 `json:"mora_cobrada"`
	PrestamosNuevos   int     `json:"prestamos_nuevos"`
	ClientesNuevos    int     `json:"clientes_nuevos"`
	PagosRecibidos    int     `json:"pagos_recibidos"`
}

type CuotaVencida struct {
	CuotaID          uuid.UUID `json:"cuota_id"`
	PrestamoID       uuid.UUID `json:"prestamo_id"`
	ClienteID        uuid.UUID `json:"cliente_id"`
	Numero           int       `json:"numero"`
	FechaVencimiento time.Time `json:"fecha_vencimiento"`
	DiasVencidos     int       `json:"dias_vencidos"`
	Total            float64   `json:"total"`
	SaldoPendiente   float64   `json:"saldo_pendiente"`
	MoraAcumulada    float64   `json:"mora_acumulada"`
	Estado           string    `json:"estado"`
}

type ReporteCliente struct {
	ClienteID          uuid.UUID         `json:"cliente_id"`
	NumPrestamos       int               `json:"num_prestamos"`
	PrestamosActivos   int               `json:"prestamos_activos"`
	TotalPrestado      float64           `json:"total_prestado"`
	TotalPagado        float64           `json:"total_pagado"`
	SaldoTotal         float64           `json:"saldo_total"`
	MoraTotal          float64           `json:"mora_total"`
	CuotasVencidas     int               `json:"cuotas_vencidas"`
	Prestamos          []PrestamoResumen `json:"prestamos"`
}

type PrestamoResumen struct {
	ID               uuid.UUID `json:"id"`
	MontoAprobado    *float64  `json:"monto_aprobado,omitempty"`
	Estado           string    `json:"estado"`
	NumCuotas        int       `json:"num_cuotas"`
	CuotasPagadas    int       `json:"cuotas_pagadas"`
	CuotasVencidas   int       `json:"cuotas_vencidas"`
	SaldoPendiente   float64   `json:"saldo_pendiente"`
	TotalPagado      float64   `json:"total_pagado"`
	FechaSolicitud   time.Time `json:"fecha_solicitud"`
}
