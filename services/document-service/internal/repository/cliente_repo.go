package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prestamos/document-service/internal/models"
)

var ErrClienteNotFound = errors.New("cliente no encontrado")

type ClienteRepository struct {
	pool *pgxpool.Pool
}

func NewClienteRepository(pool *pgxpool.Pool) *ClienteRepository {
	return &ClienteRepository{pool: pool}
}

func (r *ClienteRepository) GetByID(ctx context.Context, id uuid.UUID) (models.Cliente, error) {
	var c models.Cliente
	err := r.pool.QueryRow(ctx,
		`SELECT id, nombres, apellidos, ci, fecha_nacimiento, telefono, direccion, email
		 FROM clientes WHERE id = $1`, id,
	).Scan(
		&c.ID, &c.Nombres, &c.Apellidos, &c.CI, &c.FechaNacimiento,
		&c.Telefono, &c.Direccion, &c.Email,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Cliente{}, ErrClienteNotFound
		}
		return models.Cliente{}, fmt.Errorf("get cliente: %w", err)
	}
	return c, nil
}
