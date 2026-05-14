package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PagosRepository struct {
	pool *pgxpool.Pool
}

func NewPagosRepository(pool *pgxpool.Pool) *PagosRepository {
	return &PagosRepository{pool: pool}
}

type IngresosAgg struct {
	Ingresos       float64
	Intereses      float64
	Capital        float64
	Mora           float64
	NumPagos       int
}

// IngresosEnRango suma pagos no anulados entre dos fechas inclusive (por
// fecha_pago).
func (r *PagosRepository) IngresosEnRango(ctx context.Context, desde, hasta time.Time) (IngresosAgg, error) {
	var agg IngresosAgg
	err := r.pool.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(monto_pagado),  0),
			COALESCE(SUM(interes_pagado), 0),
			COALESCE(SUM(capital_pagado), 0),
			COALESCE(SUM(mora_pagada),    0),
			COUNT(*)
		FROM pagos
		WHERE NOT anulado
		  AND fecha_pago >= $1 AND fecha_pago < $2`,
		desde, hasta,
	).Scan(&agg.Ingresos, &agg.Intereses, &agg.Capital, &agg.Mora, &agg.NumPagos)
	if err != nil {
		return IngresosAgg{}, fmt.Errorf("ingresos rango: %w", err)
	}
	return agg, nil
}

// TotalPagadoPorPrestamo devuelve el total y desglose pagado por préstamo.
func (r *PagosRepository) TotalPagadoPorPrestamo(ctx context.Context, prestamoID uuid.UUID) (float64, error) {
	var total float64
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(monto_pagado), 0) FROM pagos
		 WHERE prestamo_id = $1 AND NOT anulado`,
		prestamoID,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("total pagado: %w", err)
	}
	return total, nil
}

// TotalPagadoPorCliente suma todos los pagos no anulados del cliente.
func (r *PagosRepository) TotalPagadoPorCliente(ctx context.Context, clienteID uuid.UUID) (float64, error) {
	var total float64
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(monto_pagado), 0) FROM pagos
		 WHERE cliente_id = $1 AND NOT anulado`,
		clienteID,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("total pagado cliente: %w", err)
	}
	return total, nil
}
