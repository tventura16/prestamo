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

var ErrPagoNotFound = errors.New("pago no encontrado")

type PagoRepository struct {
	pool *pgxpool.Pool
}

func NewPagoRepository(pool *pgxpool.Pool) *PagoRepository {
	return &PagoRepository{pool: pool}
}

func (r *PagoRepository) GetByID(ctx context.Context, id uuid.UUID) (models.Pago, error) {
	var p models.Pago
	err := r.pool.QueryRow(ctx,
		`SELECT id, numero_recibo, fecha_pago, cliente_id, prestamo_id, cuota_id,
		        monto_pagado, capital_pagado, interes_pagado, mora_pagada,
		        metodo_pago::text, tipo::text, anulado
		 FROM pagos WHERE id = $1`, id,
	).Scan(
		&p.ID, &p.NumeroRecibo, &p.FechaPago, &p.ClienteID, &p.PrestamoID, &p.CuotaID,
		&p.MontoPagado, &p.CapitalPagado, &p.InteresPagado, &p.MoraPagada,
		&p.MetodoPago, &p.Tipo, &p.Anulado,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Pago{}, ErrPagoNotFound
		}
		return models.Pago{}, fmt.Errorf("get pago: %w", err)
	}
	return p, nil
}

func (r *PagoRepository) ListByCliente(ctx context.Context, clienteID uuid.UUID) ([]models.Pago, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, numero_recibo, fecha_pago, cliente_id, prestamo_id, cuota_id,
		        monto_pagado, capital_pagado, interes_pagado, mora_pagada,
		        metodo_pago::text, tipo::text, anulado
		 FROM pagos WHERE cliente_id = $1 AND NOT anulado
		 ORDER BY fecha_pago DESC`, clienteID,
	)
	if err != nil {
		return nil, fmt.Errorf("list pagos: %w", err)
	}
	defer rows.Close()

	out := make([]models.Pago, 0)
	for rows.Next() {
		var p models.Pago
		if err := rows.Scan(
			&p.ID, &p.NumeroRecibo, &p.FechaPago, &p.ClienteID, &p.PrestamoID, &p.CuotaID,
			&p.MontoPagado, &p.CapitalPagado, &p.InteresPagado, &p.MoraPagada,
			&p.MetodoPago, &p.Tipo, &p.Anulado,
		); err != nil {
			return nil, fmt.Errorf("scan pago: %w", err)
		}
		out = append(out, p)
	}
	return out, nil
}
