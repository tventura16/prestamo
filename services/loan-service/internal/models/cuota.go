package models

import (
	"time"

	"github.com/google/uuid"
)

type EstadoCuota string

const (
	CuotaPendiente EstadoCuota = "pendiente"
	CuotaPagada    EstadoCuota = "pagada"
	CuotaParcial   EstadoCuota = "parcial"
	CuotaVencida   EstadoCuota = "vencida"
)

type Cuota struct {
	ID               uuid.UUID   `json:"id"`
	PrestamoID       uuid.UUID   `json:"prestamo_id"`
	Numero           int         `json:"numero"`
	FechaVencimiento time.Time   `json:"fecha_vencimiento"`
	Capital          float64     `json:"capital"`
	Interes          float64     `json:"interes"`
	Total            float64     `json:"total"`
	SaldoPendiente   float64     `json:"saldo_pendiente"`
	MoraAcumulada    float64     `json:"mora_acumulada"`
	Estado           EstadoCuota `json:"estado"`
	FechaPago        *time.Time  `json:"fecha_pago,omitempty"`
	CreatedAt        time.Time   `json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`
}
