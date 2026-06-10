package models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type SubtipoGarantia string

const (
	SubtipoVehiculo SubtipoGarantia = "vehiculo"
	SubtipoInmueble SubtipoGarantia = "inmueble"
	SubtipoGarante  SubtipoGarantia = "garante"
	SubtipoMueble   SubtipoGarantia = "mueble"
)

// Garantia es un respaldo del préstamo. Los campos propios de cada subtipo
// viven en Datos (JSONB) validados por ValidarDatosGarantia.
type Garantia struct {
	ID               uuid.UUID        `json:"id"`
	PrestamoID       uuid.UUID        `json:"prestamo_id"`
	Subtipo          SubtipoGarantia  `json:"subtipo"`
	Descripcion      *string          `json:"descripcion,omitempty"`
	ValorEstimado    *float64         `json:"valor_estimado,omitempty"`
	Moneda           string           `json:"moneda"`
	ClienteGaranteID *uuid.UUID       `json:"cliente_garante_id,omitempty"`
	Datos            json.RawMessage  `json:"datos"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
	Imagenes         []GarantiaImagen `json:"imagenes,omitempty"`
}

// GarantiaImagen es un adjunto de una garantía. Ruta es interna (no se expone).
type GarantiaImagen struct {
	ID            uuid.UUID  `json:"id"`
	GarantiaID    uuid.UUID  `json:"garantia_id"`
	NombreArchivo string     `json:"nombre_archivo"`
	Mime          string     `json:"mime"`
	TamanioBytes  int64      `json:"tamanio_bytes"`
	Descripcion   *string    `json:"descripcion,omitempty"`
	SubidoPor     *uuid.UUID `json:"subido_por,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	Ruta          string     `json:"-"`
}

// CreateGarantiaInput crea una garantía para un préstamo.
type CreateGarantiaInput struct {
	Subtipo          SubtipoGarantia `json:"subtipo"        binding:"required,oneof=vehiculo inmueble garante mueble"`
	Descripcion      *string         `json:"descripcion"`
	ValorEstimado    *float64        `json:"valor_estimado" binding:"omitempty,gte=0"`
	Moneda           string          `json:"moneda"`
	ClienteGaranteID *uuid.UUID      `json:"cliente_garante_id"`
	Datos            json.RawMessage `json:"datos"`
}

// ─── Esquemas de datos por subtipo ───

type DatosVehiculo struct {
	Placa     string `json:"placa"`
	Marca     string `json:"marca"`
	Modelo    string `json:"modelo,omitempty"`
	Anio      int    `json:"anio,omitempty"`
	Color     string `json:"color,omitempty"`
	NroMotor  string `json:"nro_motor,omitempty"`
	NroChasis string `json:"nro_chasis,omitempty"`
}

type DatosInmueble struct {
	TipoInmueble   string  `json:"tipo_inmueble"`
	Direccion      string  `json:"direccion"`
	MatriculaFolio string  `json:"matricula_folio,omitempty"`
	SuperficieM2   float64 `json:"superficie_m2,omitempty"`
	Gravamenes     string  `json:"gravamenes,omitempty"`
}

type DatosGarante struct {
	Nombres   string `json:"nombres"`
	Apellidos string `json:"apellidos,omitempty"`
	CI        string `json:"ci"`
	Telefono  string `json:"telefono,omitempty"`
	Direccion string `json:"direccion,omitempty"`
	Actividad string `json:"actividad,omitempty"`
}

type DatosMueble struct {
	Descripcion string `json:"descripcion"`
	Ubicacion   string `json:"ubicacion,omitempty"`
	Marca       string `json:"marca,omitempty"`
	Cantidad    int    `json:"cantidad,omitempty"`
}

// ValidarDatosGarantia valida los campos requeridos de `datos` según el subtipo
// y devuelve el JSON normalizado (solo campos del esquema). Mantiene la
// validación en el dominio, no en el esquema de la BD (JSONB tipado).
func ValidarDatosGarantia(subtipo SubtipoGarantia, datos json.RawMessage) (json.RawMessage, error) {
	if len(datos) == 0 {
		return nil, fmt.Errorf("se requieren los datos de la garantía")
	}
	switch subtipo {
	case SubtipoVehiculo:
		var d DatosVehiculo
		if err := json.Unmarshal(datos, &d); err != nil {
			return nil, fmt.Errorf("datos de vehículo inválidos: %w", err)
		}
		if d.Placa == "" || d.Marca == "" {
			return nil, fmt.Errorf("vehículo: placa y marca son obligatorios")
		}
		return json.Marshal(d)
	case SubtipoInmueble:
		var d DatosInmueble
		if err := json.Unmarshal(datos, &d); err != nil {
			return nil, fmt.Errorf("datos de inmueble inválidos: %w", err)
		}
		if d.Direccion == "" {
			return nil, fmt.Errorf("inmueble: la dirección es obligatoria")
		}
		switch d.TipoInmueble {
		case "casa", "terreno", "local":
		default:
			return nil, fmt.Errorf("inmueble: tipo_inmueble debe ser casa, terreno o local")
		}
		return json.Marshal(d)
	case SubtipoGarante:
		var d DatosGarante
		if err := json.Unmarshal(datos, &d); err != nil {
			return nil, fmt.Errorf("datos de garante inválidos: %w", err)
		}
		if d.Nombres == "" || d.CI == "" {
			return nil, fmt.Errorf("garante: nombres y CI son obligatorios")
		}
		return json.Marshal(d)
	case SubtipoMueble:
		var d DatosMueble
		if err := json.Unmarshal(datos, &d); err != nil {
			return nil, fmt.Errorf("datos de bien mueble inválidos: %w", err)
		}
		if d.Descripcion == "" {
			return nil, fmt.Errorf("bien mueble: la descripción es obligatoria")
		}
		return json.Marshal(d)
	default:
		return nil, fmt.Errorf("subtipo de garantía no soportado: %s", subtipo)
	}
}
