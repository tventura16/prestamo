package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prestamos/report-service/internal/models"
)

type ClientesRepository struct {
	pool *pgxpool.Pool
}

func NewClientesRepository(pool *pgxpool.Pool) *ClientesRepository {
	return &ClientesRepository{pool: pool}
}

func (r *ClientesRepository) CountActivos(ctx context.Context) (int, error) {
	var n int
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM clientes WHERE estado = 'activo'`,
	).Scan(&n); err != nil {
		return 0, fmt.Errorf("count clientes: %w", err)
	}
	return n, nil
}

// GetCliente devuelve los datos identificatorios del cliente; (nil, nil) si no
// existe (el reporte se entrega igual, sin la sección de datos).
func (r *ClientesRepository) GetCliente(ctx context.Context, id uuid.UUID) (*models.ClienteInfo, error) {
	var c models.ClienteInfo
	var tel, email *string
	err := r.pool.QueryRow(ctx,
		`SELECT nombres, apellidos, ci, telefono, email, estado FROM clientes WHERE id = $1`, id,
	).Scan(&c.Nombres, &c.Apellidos, &c.CI, &tel, &email, &c.Estado)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get cliente: %w", err)
	}
	if tel != nil {
		c.Telefono = *tel
	}
	if email != nil {
		c.Email = *email
	}
	return &c, nil
}

func (r *ClientesRepository) CountNuevosEnRango(ctx context.Context, desde, hasta time.Time) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM clientes
		 WHERE created_at >= $1 AND created_at < $2`,
		desde, hasta,
	).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count nuevos: %w", err)
	}
	return n, nil
}
