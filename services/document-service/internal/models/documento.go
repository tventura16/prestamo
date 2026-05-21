package models

import (
	"time"

	"github.com/google/uuid"
)

type TipoDocumento string

const (
	TipoContrato     TipoDocumento = "contrato"
	TipoPlanPagos    TipoDocumento = "plan_pagos"
	TipoRecibo       TipoDocumento = "recibo"
	TipoEstadoCuenta TipoDocumento = "estado_cuenta"
	TipoCartaMora    TipoDocumento = "carta_mora"
)

type EstadoDocumento string

const (
	EstadoPendiente EstadoDocumento = "pendiente"
	EstadoGenerado  EstadoDocumento = "generado"
	EstadoEnviado   EstadoDocumento = "enviado"
	EstadoError     EstadoDocumento = "error"
)

type Documento struct {
	ID            uuid.UUID       `json:"id"`
	Tipo          TipoDocumento   `json:"tipo"`
	ClienteID     *uuid.UUID      `json:"cliente_id,omitempty"`
	PrestamoID    *uuid.UUID      `json:"prestamo_id,omitempty"`
	PagoID        *uuid.UUID      `json:"pago_id,omitempty"`
	NombreArchivo string          `json:"nombre_archivo"`
	Ruta          string          `json:"ruta"`
	HashSHA256    *string         `json:"hash_sha256,omitempty"`
	TamanioKB     *int            `json:"tamanio_kb,omitempty"`
	Estado        EstadoDocumento `json:"estado"`
	ErrorMensaje  *string         `json:"error_mensaje,omitempty"`
	GeneradoPor   *uuid.UUID      `json:"generado_por,omitempty"`
	GeneradoAt    time.Time       `json:"generado_at"`
}

// Inputs

type ContratoInput struct {
	PrestamoID  uuid.UUID  `json:"prestamo_id"  binding:"required"`
	GeneradoPor *uuid.UUID `json:"generado_por"`
}

type ReciboInput struct {
	PagoID      uuid.UUID  `json:"pago_id"      binding:"required"`
	GeneradoPor *uuid.UUID `json:"generado_por"`
}

type EstadoCuentaInput struct {
	ClienteID   uuid.UUID  `json:"cliente_id"   binding:"required"`
	GeneradoPor *uuid.UUID `json:"generado_por"`
}

type PlanPagosInput struct {
	PrestamoID  uuid.UUID  `json:"prestamo_id"  binding:"required"`
	GeneradoPor *uuid.UUID `json:"generado_por"`
}

type ListResult struct {
	Items []Documento `json:"items"`
	Total int64       `json:"total"`
	Page  int         `json:"page"`
	Limit int         `json:"limit"`
}

// Tipos de datos cross-DB (los repos los pueblan):

type Cliente struct {
	ID              uuid.UUID
	Nombres         string
	Apellidos       string
	CI              string
	FechaNacimiento time.Time
	Telefono        *string
	Direccion       *string
	Email           *string
}

type Prestamo struct {
	ID              uuid.UUID
	ClienteID       uuid.UUID
	MontoAprobado   float64
	TasaInteres     float64
	NumCuotas       int
	Frecuencia      string
	Estado          string
	FechaDesembolso *time.Time
	FechaSolicitud  time.Time
}

type Cuota struct {
	Numero           int
	FechaVencimiento time.Time
	Capital          float64
	Interes          float64
	Total            float64
	SaldoPendiente   float64
	Estado           string
	FechaPago        *time.Time
}

type Pago struct {
	ID            uuid.UUID
	NumeroRecibo  *string
	FechaPago     time.Time
	ClienteID     uuid.UUID
	PrestamoID    uuid.UUID
	CuotaID       *uuid.UUID
	CuotaNumero   *int
	MontoPagado   float64
	CapitalPagado float64
	InteresPagado float64
	MoraPagada    float64
	MetodoPago    string
	Tipo          string
	Anulado       bool
}
