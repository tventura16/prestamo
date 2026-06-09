package models

import (
	"time"

	"github.com/google/uuid"
)

// Garantia es un adjunto (imagen) de la garantía de un préstamo. El binario se
// guarda en el volumen del servicio; `Ruta` es interna y no se expone en la API.
type Garantia struct {
	ID            uuid.UUID  `json:"id"`
	PrestamoID    uuid.UUID  `json:"prestamo_id"`
	NombreArchivo string     `json:"nombre_archivo"`
	Mime          string     `json:"mime"`
	TamanioBytes  int64      `json:"tamanio_bytes"`
	Descripcion   *string    `json:"descripcion,omitempty"`
	SubidoPor     *uuid.UUID `json:"subido_por,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	Ruta          string     `json:"-"` // ruta en el filesystem (uso interno)
}
