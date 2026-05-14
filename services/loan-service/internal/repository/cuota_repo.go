package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prestamos/loan-service/internal/models"
)

type CuotaRepository struct {
	pool *pgxpool.Pool
}

func NewCuotaRepository(pool *pgxpool.Pool) *CuotaRepository {
	return &CuotaRepository{pool: pool}
}

func (r *CuotaRepository) ListByPrestamo(ctx context.Context, prestamoID uuid.UUID) ([]models.Cuota, error) {
	rows, err := r.pool.Query(ctx, `SELECT
			id, prestamo_id, numero, fecha_vencimiento, capital, interes,
			total, saldo_pendiente, mora_acumulada, estado, fecha_pago,
			created_at, updated_at
		FROM cuotas
		WHERE prestamo_id = $1
		ORDER BY numero`, prestamoID)
	if err != nil {
		return nil, fmt.Errorf("query cuotas: %w", err)
	}
	defer rows.Close()

	cuotas := make([]models.Cuota, 0)
	for rows.Next() {
		var c models.Cuota
		if err := rows.Scan(
			&c.ID, &c.PrestamoID, &c.Numero, &c.FechaVencimiento,
			&c.Capital, &c.Interes, &c.Total, &c.SaldoPendiente,
			&c.MoraAcumulada, &c.Estado, &c.FechaPago,
			&c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan cuota: %w", err)
		}
		cuotas = append(cuotas, c)
	}
	return cuotas, nil
}
