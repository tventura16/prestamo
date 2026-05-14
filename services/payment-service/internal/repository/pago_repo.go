package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prestamos/payment-service/internal/models"
)

var ErrPagoNotFound = errors.New("pago no encontrado")

type PagoRepository struct {
	pool *pgxpool.Pool
}

func NewPagoRepository(pool *pgxpool.Pool) *PagoRepository {
	return &PagoRepository{pool: pool}
}

const pagoColumns = `id, cliente_id, prestamo_id, cuota_id, fecha_pago, monto_pagado,
		capital_pagado, interes_pagado, mora_pagada, tipo, metodo_pago,
		usuario_id, numero_recibo, observaciones, anulado, created_at`

func scanPago(row pgx.Row) (models.Pago, error) {
	var p models.Pago
	err := row.Scan(
		&p.ID, &p.ClienteID, &p.PrestamoID, &p.CuotaID, &p.FechaPago, &p.MontoPagado,
		&p.CapitalPagado, &p.InteresPagado, &p.MoraPagada, &p.Tipo, &p.MetodoPago,
		&p.UsuarioID, &p.NumeroRecibo, &p.Observaciones, &p.Anulado, &p.CreatedAt,
	)
	return p, err
}

// PreviousPaidByCuota devuelve cuánto interés y capital se han pagado
// previamente para una cuota (sumando pagos no anulados).
func (r *PagoRepository) PreviousPaidByCuota(ctx context.Context, cuotaID uuid.UUID) (interes, capital float64, err error) {
	err = r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(interes_pagado), 0),
		       COALESCE(SUM(capital_pagado), 0)
		FROM pagos
		WHERE cuota_id = $1 AND NOT anulado`, cuotaID,
	).Scan(&interes, &capital)
	return
}

// InsertPago crea el pago + movimientos en transacción atómica dentro
// de la DB pagos.
func (r *PagoRepository) InsertPago(ctx context.Context,
	p models.Pago, movimientos []models.Movimiento,
) (models.Pago, []models.Movimiento, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return models.Pago{}, nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var numero string
	if err := tx.QueryRow(ctx, `SELECT 'R-' || LPAD(nextval('seq_recibo')::TEXT, 8, '0')`).Scan(&numero); err != nil {
		return models.Pago{}, nil, fmt.Errorf("next recibo: %w", err)
	}
	p.NumeroRecibo = &numero

	pagoRow := tx.QueryRow(ctx, `INSERT INTO pagos
		(cliente_id, prestamo_id, cuota_id, fecha_pago, monto_pagado,
		 capital_pagado, interes_pagado, mora_pagada, tipo, metodo_pago,
		 usuario_id, numero_recibo, observaciones)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING `+pagoColumns,
		p.ClienteID, p.PrestamoID, p.CuotaID, p.FechaPago, p.MontoPagado,
		p.CapitalPagado, p.InteresPagado, p.MoraPagada, p.Tipo, p.MetodoPago,
		p.UsuarioID, p.NumeroRecibo, p.Observaciones,
	)
	pagoInsertado, err := scanPago(pagoRow)
	if err != nil {
		return models.Pago{}, nil, fmt.Errorf("insert pago: %w", err)
	}

	insertedMovs := make([]models.Movimiento, 0, len(movimientos))
	for _, m := range movimientos {
		if m.Monto == 0 {
			continue
		}
		var inserted models.Movimiento
		err := tx.QueryRow(ctx, `INSERT INTO movimientos_pago
			(pago_id, concepto, monto)
			VALUES ($1, $2, $3)
			RETURNING id, pago_id, concepto, monto, created_at`,
			pagoInsertado.ID, m.Concepto, m.Monto,
		).Scan(&inserted.ID, &inserted.PagoID, &inserted.Concepto, &inserted.Monto, &inserted.CreatedAt)
		if err != nil {
			return models.Pago{}, nil, fmt.Errorf("insert movimiento %s: %w", m.Concepto, err)
		}
		insertedMovs = append(insertedMovs, inserted)
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Pago{}, nil, fmt.Errorf("commit: %w", err)
	}
	return pagoInsertado, insertedMovs, nil
}

func (r *PagoRepository) GetByID(ctx context.Context, id uuid.UUID) (models.Pago, []models.Movimiento, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+pagoColumns+` FROM pagos WHERE id = $1`, id)
	p, err := scanPago(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Pago{}, nil, ErrPagoNotFound
		}
		return models.Pago{}, nil, fmt.Errorf("get pago: %w", err)
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, pago_id, concepto, monto, created_at
		 FROM movimientos_pago WHERE pago_id = $1 ORDER BY concepto`, id)
	if err != nil {
		return models.Pago{}, nil, fmt.Errorf("query movimientos: %w", err)
	}
	defer rows.Close()

	movs := make([]models.Movimiento, 0)
	for rows.Next() {
		var m models.Movimiento
		if err := rows.Scan(&m.ID, &m.PagoID, &m.Concepto, &m.Monto, &m.CreatedAt); err != nil {
			return models.Pago{}, nil, fmt.Errorf("scan movimiento: %w", err)
		}
		movs = append(movs, m)
	}
	return p, movs, nil
}

func (r *PagoRepository) List(ctx context.Context, page, limit int,
	prestamoID, cuotaID, clienteID *uuid.UUID,
) (models.ListResult, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 20
	}
	offset := (page - 1) * limit

	args := []any{}
	where := ""
	addFilter := func(col string, val any) {
		args = append(args, val)
		if where == "" {
			where = fmt.Sprintf(" WHERE %s = $%d", col, len(args))
		} else {
			where += fmt.Sprintf(" AND %s = $%d", col, len(args))
		}
	}
	if prestamoID != nil {
		addFilter("prestamo_id", *prestamoID)
	}
	if cuotaID != nil {
		addFilter("cuota_id", *cuotaID)
	}
	if clienteID != nil {
		addFilter("cliente_id", *clienteID)
	}

	var total int64
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM pagos`+where, args...).Scan(&total); err != nil {
		return models.ListResult{}, fmt.Errorf("count: %w", err)
	}

	q := `SELECT ` + pagoColumns + ` FROM pagos` + where +
		fmt.Sprintf(` ORDER BY fecha_pago DESC LIMIT $%d OFFSET $%d`, len(args)+1, len(args)+2)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return models.ListResult{}, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	items := make([]models.Pago, 0, limit)
	for rows.Next() {
		p, err := scanPago(rows)
		if err != nil {
			return models.ListResult{}, fmt.Errorf("scan: %w", err)
		}
		items = append(items, p)
	}
	return models.ListResult{Items: items, Total: total, Page: page, Limit: limit}, nil
}
