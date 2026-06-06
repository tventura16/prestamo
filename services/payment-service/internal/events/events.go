// Package events define los contratos de eventos de dominio que el
// payment-service publica vía outbox → Kafka.
package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Topics y tipos de evento.
const (
	TopicPagos    = "pagos.eventos"
	TopicPagosDLQ = "pagos.eventos.dlq"

	TypePagoRegistrado = "pago.registrado"

	AggregatePago = "pago"
)

// PagoRegistrado es el payload del evento que dispara la aplicación del pago
// a la cuota/préstamo en la DB prestamos. Lleva los montos ya distribuidos
// (mora → interés → capital) calculados en el momento del registro.
//
// El consumer re-valida contra el saldo vivo de la cuota antes de aplicar,
// por lo que estos montos son la intención de distribución, no un hecho
// inmutable: si un pago concurrente ya consumió parte del saldo, se clampean.
type PagoRegistrado struct {
	PagoID     uuid.UUID `json:"pago_id"`
	CuotaID    uuid.UUID `json:"cuota_id"`
	PrestamoID uuid.UUID `json:"prestamo_id"`
	ClienteID  uuid.UUID `json:"cliente_id"`
	Capital    float64   `json:"capital"`
	Interes    float64   `json:"interes"`
	Mora       float64   `json:"mora"`
	OcurridoEn time.Time `json:"ocurrido_en"`
}

// Marshal serializa el payload para almacenarlo en el outbox.
func (p PagoRegistrado) Marshal() ([]byte, error) {
	return json.Marshal(p)
}

// UnmarshalPagoRegistrado deserializa el payload desde Kafka/outbox.
func UnmarshalPagoRegistrado(b []byte) (PagoRegistrado, error) {
	var p PagoRegistrado
	err := json.Unmarshal(b, &p)
	return p, err
}
