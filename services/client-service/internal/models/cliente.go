package models

import (
	"time"

	"github.com/google/uuid"
)

type Estado string

const (
	EstadoActivo    Estado = "activo"
	EstadoInactivo  Estado = "inactivo"
	EstadoBloqueado Estado = "bloqueado"
)

func (e Estado) Valid() bool {
	switch e {
	case EstadoActivo, EstadoInactivo, EstadoBloqueado:
		return true
	}
	return false
}

type Cliente struct {
	ID              uuid.UUID `json:"id"`
	Nombres         string    `json:"nombres"`
	Apellidos       string    `json:"apellidos"`
	CI              string    `json:"ci"`
	FechaNacimiento time.Time `json:"fecha_nacimiento"`
	Telefono        *string   `json:"telefono,omitempty"`
	Direccion       *string   `json:"direccion,omitempty"`
	Email           *string   `json:"email,omitempty"`
	Estado          Estado    `json:"estado"`
	FotoURL         *string   `json:"foto_url,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// CreateClienteInput es el payload de creación. Las fechas vienen como string
// (formato YYYY-MM-DD) para evitar fricciones con JSON.
type CreateClienteInput struct {
	Nombres         string  `json:"nombres"          binding:"required,min=1,max=100"`
	Apellidos       string  `json:"apellidos"        binding:"required,min=1,max=100"`
	CI              string  `json:"ci"               binding:"required,min=4,max=20"`
	FechaNacimiento string  `json:"fecha_nacimiento" binding:"required,datetime=2006-01-02"`
	Telefono        *string `json:"telefono"         binding:"omitempty,max=20"`
	Direccion       *string `json:"direccion"`
	Email           *string `json:"email"            binding:"omitempty,email,max=150"`
	Estado          *Estado `json:"estado"           binding:"omitempty,oneof=activo inactivo bloqueado"`
	FotoURL         *string `json:"foto_url"`
}

type UpdateClienteInput struct {
	Nombres         *string `json:"nombres"          binding:"omitempty,min=1,max=100"`
	Apellidos       *string `json:"apellidos"        binding:"omitempty,min=1,max=100"`
	CI              *string `json:"ci"               binding:"omitempty,min=4,max=20"`
	FechaNacimiento *string `json:"fecha_nacimiento" binding:"omitempty,datetime=2006-01-02"`
	Telefono        *string `json:"telefono"         binding:"omitempty,max=20"`
	Direccion       *string `json:"direccion"`
	Email           *string `json:"email"            binding:"omitempty,email,max=150"`
	Estado          *Estado `json:"estado"           binding:"omitempty,oneof=activo inactivo bloqueado"`
	FotoURL         *string `json:"foto_url"`
}

type ListResult struct {
	Items []Cliente `json:"items"`
	Total int64     `json:"total"`
	Page  int       `json:"page"`
	Limit int       `json:"limit"`
}
