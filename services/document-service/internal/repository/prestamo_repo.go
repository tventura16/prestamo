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

var ErrPrestamoNotFound = errors.New("préstamo no encontrado")

type PrestamoRepository struct {
	pool *pgxpool.Pool
}

func NewPrestamoRepository(pool *pgxpool.Pool) *PrestamoRepository {
	return &PrestamoRepository{pool: pool}
}

func (r *PrestamoRepository) GetByID(ctx context.Context, id uuid.UUID) (models.Prestamo, error) {
	var p models.Prestamo
	var monto *float64
	err := r.pool.QueryRow(ctx,
		`SELECT id, cliente_id, monto_aprobado, tasa_interes, num_cuotas,
		        frecuencia::text, estado::text, fecha_desembolso, fecha_solicitud
		 FROM prestamos WHERE id = $1`, id,
	).Scan(
		&p.ID, &p.ClienteID, &monto, &p.TasaInteres, &p.NumCuotas,
		&p.Frecuencia, &p.Estado, &p.FechaDesembolso, &p.FechaSolicitud,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Prestamo{}, ErrPrestamoNotFound
		}
		return models.Prestamo{}, fmt.Errorf("get prestamo: %w", err)
	}
	if monto != nil {
		p.MontoAprobado = *monto
	}
	return p, nil
}

func (r *PrestamoRepository) ListCuotas(ctx context.Context, prestamoID uuid.UUID) ([]models.Cuota, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT numero, fecha_vencimiento, capital, interes, total, saldo_pendiente,
		        estado::text, fecha_pago
		 FROM cuotas
		 WHERE prestamo_id = $1
		 ORDER BY numero`, prestamoID,
	)
	if err != nil {
		return nil, fmt.Errorf("query cuotas: %w", err)
	}
	defer rows.Close()

	out := make([]models.Cuota, 0)
	for rows.Next() {
		var c models.Cuota
		if err := rows.Scan(
			&c.Numero, &c.FechaVencimiento, &c.Capital, &c.Interes,
			&c.Total, &c.SaldoPendiente, &c.Estado, &c.FechaPago,
		); err != nil {
			return nil, fmt.Errorf("scan cuota: %w", err)
		}
		out = append(out, c)
	}
	return out, nil
}

// ListPrestamosPorCliente devuelve todos los préstamos del cliente para
// el estado de cuenta.
func (r *PrestamoRepository) ListPrestamosPorCliente(ctx context.Context, clienteID uuid.UUID) ([]models.Prestamo, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, cliente_id, COALESCE(monto_aprobado, 0), tasa_interes, num_cuotas,
		        frecuencia::text, estado::text, fecha_desembolso, fecha_solicitud
		 FROM prestamos WHERE cliente_id = $1
		 ORDER BY fecha_solicitud DESC`, clienteID,
	)
	if err != nil {
		return nil, fmt.Errorf("query prestamos: %w", err)
	}
	defer rows.Close()

	out := make([]models.Prestamo, 0)
	for rows.Next() {
		var p models.Prestamo
		if err := rows.Scan(
			&p.ID, &p.ClienteID, &p.MontoAprobado, &p.TasaInteres, &p.NumCuotas,
			&p.Frecuencia, &p.Estado, &p.FechaDesembolso, &p.FechaSolicitud,
		); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		out = append(out, p)
	}
	return out, nil
}

// CuotaNumero devuelve el número de una cuota (para incluir en recibos).
func (r *PrestamoRepository) CuotaNumero(ctx context.Context, cuotaID uuid.UUID) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx,
		`SELECT numero FROM cuotas WHERE id = $1`, cuotaID,
	).Scan(&n)
	if err != nil {
		return 0, err
	}
	return n, nil
}
