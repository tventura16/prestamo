package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
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
