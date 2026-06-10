package models

import (
	"time"

	"github.com/google/uuid"
)

type MetodoPago string

const (
	MetodoEfectivo      MetodoPago = "efectivo"
	MetodoTransferencia MetodoPago = "transferencia"
	MetodoCheque        MetodoPago = "cheque"
	MetodoTarjeta       MetodoPago = "tarjeta"
	MetodoQR            MetodoPago = "qr"
)

type TipoPago string

const (
	TipoTotal   TipoPago = "total"
	TipoParcial TipoPago = "parcial"
)

type Pago struct {
	ID              uuid.UUID  `json:"id"`
	ClienteID       uuid.UUID  `json:"cliente_id"`
	PrestamoID      uuid.UUID  `json:"prestamo_id"`
	CuotaID         *uuid.UUID `json:"cuota_id,omitempty"`
	FechaPago       time.Time  `json:"fecha_pago"`
	MontoPagado     float64    `json:"monto_pagado"`
	CapitalPagado   float64    `json:"capital_pagado"`
	InteresPagado   float64    `json:"interes_pagado"`
	MoraPagada      float64    `json:"mora_pagada"`
	Tipo            TipoPago   `json:"tipo"`
	MetodoPago      MetodoPago `json:"metodo_pago"`
	UsuarioID       uuid.UUID  `json:"usuario_id"`
	NumeroRecibo    *string    `json:"numero_recibo,omitempty"`
	Observaciones   *string    `json:"observaciones,omitempty"`
	Anulado         bool       `json:"anulado"`
	AnuladoAt       *time.Time `json:"anulado_at,omitempty"`
	AnuladoPor      *uuid.UUID `json:"anulado_por,omitempty"`
	MotivoAnulacion *string    `json:"motivo_anulacion,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

type Movimiento struct {
	ID        uuid.UUID `json:"id"`
	PagoID    uuid.UUID `json:"pago_id"`
	Concepto  string    `json:"concepto"`
	Monto     float64   `json:"monto"`
	CreatedAt time.Time `json:"created_at"`
}

// CreatePagoInput aplica un pago a una cuota específica. v1 solo soporta
// un pago = una cuota. Pagos que cubren múltiples cuotas requieren llamadas
// separadas.
type CreatePagoInput struct {
	CuotaID       uuid.UUID  `json:"cuota_id"      binding:"required"`
	MontoPagado   float64    `json:"monto_pagado"  binding:"required,gt=0"`
	MetodoPago    MetodoPago `json:"metodo_pago"   binding:"required,oneof=efectivo transferencia cheque tarjeta qr"`
	UsuarioID     uuid.UUID  `json:"usuario_id"    binding:"required"`
	Observaciones *string    `json:"observaciones"`
}

// AnularPagoInput es el cuerpo de la anulación. El motivo es obligatorio para
// dejar traza de auditoría; el operador se toma del token (no del body).
type AnularPagoInput struct {
	Motivo string `json:"motivo" binding:"required"`
}

type ListResult struct {
	Items []Pago `json:"items"`
	Total int64  `json:"total"`
	Page  int    `json:"page"`
	Limit int    `json:"limit"`
}

// PagoResult es lo que devuelve la creación: el pago + el estado actual
// de la cuota tras la aplicación.
type PagoResult struct {
	Pago        Pago         `json:"pago"`
	Movimientos []Movimiento `json:"movimientos"`
	Cuota       CuotaInfo    `json:"cuota"`
	Prestamo    PrestamoInfo `json:"prestamo"`
}

type CuotaInfo struct {
	ID             uuid.UUID `json:"id"`
	PrestamoID     uuid.UUID `json:"prestamo_id"`
	Numero         int       `json:"numero"`
	SaldoPendiente float64   `json:"saldo_pendiente"`
	MoraAcumulada  float64   `json:"mora_acumulada"`
	Estado         string    `json:"estado"`
}

type PrestamoInfo struct {
	ID     uuid.UUID `json:"id"`
	Estado string    `json:"estado"`
}
