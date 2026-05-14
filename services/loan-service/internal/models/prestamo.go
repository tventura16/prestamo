package models

import (
	"time"

	"github.com/google/uuid"
)

type Estado string

const (
	EstadoPendiente  Estado = "pendiente"
	EstadoAprobado   Estado = "aprobado"
	EstadoRechazado  Estado = "rechazado"
	EstadoActivo     Estado = "activo"
	EstadoFinalizado Estado = "finalizado"
	EstadoMora       Estado = "mora"
)

type TipoInteres string

const (
	TipoInteresFijo     TipoInteres = "fijo"
	TipoInteresVariable TipoInteres = "variable"
)

type Frecuencia string

const (
	FrecuenciaDiaria    Frecuencia = "diaria"
	FrecuenciaSemanal   Frecuencia = "semanal"
	FrecuenciaQuincenal Frecuencia = "quincenal"
	FrecuenciaMensual   Frecuencia = "mensual"
)

type Prestamo struct {
	ID              uuid.UUID   `json:"id"`
	ClienteID       uuid.UUID   `json:"cliente_id"`
	MontoSolicitado float64     `json:"monto_solicitado"`
	MontoAprobado   *float64    `json:"monto_aprobado,omitempty"`
	TasaInteres     float64     `json:"tasa_interes"`
	TipoInteres     TipoInteres `json:"tipo_interes"`
	FechaSolicitud  time.Time   `json:"fecha_solicitud"`
	FechaDesembolso *time.Time  `json:"fecha_desembolso,omitempty"`
	NumCuotas       int         `json:"num_cuotas"`
	Frecuencia      Frecuencia  `json:"frecuencia"`
	Estado          Estado      `json:"estado"`
	AprobadoPor     *uuid.UUID  `json:"aprobado_por,omitempty"`
	Observaciones   *string     `json:"observaciones,omitempty"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
}

// CreatePrestamoInput crea una solicitud en estado "pendiente".
type CreatePrestamoInput struct {
	ClienteID       uuid.UUID    `json:"cliente_id"       binding:"required"`
	MontoSolicitado float64      `json:"monto_solicitado" binding:"required,gt=0"`
	TasaInteres     float64      `json:"tasa_interes"     binding:"required,gte=0"`
	TipoInteres     *TipoInteres `json:"tipo_interes"     binding:"omitempty,oneof=fijo variable"`
	NumCuotas       int          `json:"num_cuotas"       binding:"required,gt=0,lte=600"`
	Frecuencia      Frecuencia   `json:"frecuencia"       binding:"required,oneof=diaria semanal quincenal mensual"`
	Observaciones   *string      `json:"observaciones"`
}

// ApprovePrestamoInput aprueba y desembolsa; genera el plan de pagos.
type ApprovePrestamoInput struct {
	MontoAprobado   *float64   `json:"monto_aprobado"   binding:"omitempty,gt=0"`
	FechaDesembolso *string    `json:"fecha_desembolso" binding:"omitempty,datetime=2006-01-02"`
	AprobadoPor     uuid.UUID  `json:"aprobado_por"     binding:"required"`
	Observaciones   *string    `json:"observaciones"`
}

type RejectPrestamoInput struct {
	AprobadoPor   uuid.UUID `json:"aprobado_por"  binding:"required"`
	Observaciones string    `json:"observaciones" binding:"required,min=1"`
}

type ListResult struct {
	Items []Prestamo `json:"items"`
	Total int64      `json:"total"`
	Page  int        `json:"page"`
	Limit int        `json:"limit"`
}
