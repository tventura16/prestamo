package repository

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prestamos/report-service/internal/models"
)

type PrestamosRepository struct {
	pool *pgxpool.Pool
}

func NewPrestamosRepository(pool *pgxpool.Pool) *PrestamosRepository {
	return &PrestamosRepository{pool: pool}
}

// ParametrosElegibilidad lee de parametros_sistema (BD prestamos) los criterios
// para evaluar si un cliente puede tomar un nuevo préstamo. Devuelve defaults
// razonables si la tabla o las claves no existieran.
func (r *PrestamosRepository) ParametrosElegibilidad(ctx context.Context) (aprobarSiMora bool, maxActivos int, err error) {
	aprobarSiMora, maxActivos = false, 3
	rows, err := r.pool.Query(ctx,
		`SELECT clave, valor FROM parametros_sistema
		 WHERE clave IN ('aprobar_si_mora_activa', 'max_prestamos_activos')`)
	if err != nil {
		return aprobarSiMora, maxActivos, nil // tolerante: usa defaults
	}
	defer rows.Close()
	for rows.Next() {
		var clave, valor string
		if err := rows.Scan(&clave, &valor); err != nil {
			continue
		}
		switch clave {
		case "aprobar_si_mora_activa":
			aprobarSiMora = valor == "true"
		case "max_prestamos_activos":
			if n, e := strconv.Atoi(valor); e == nil {
				maxActivos = n
			}
		}
	}
	return aprobarSiMora, maxActivos, nil
}

// CountByEstado cuenta préstamos por estado.
func (r *PrestamosRepository) CountByEstado(ctx context.Context, estados ...string) (int, error) {
	if len(estados) == 0 {
		return 0, nil
	}
	args := make([]any, len(estados))
	for i, e := range estados {
		args[i] = e
	}
	q := `SELECT COUNT(*) FROM prestamos WHERE estado = ANY($1)`
	var n int
	if err := r.pool.QueryRow(ctx, q, estados).Scan(&n); err != nil {
		return 0, fmt.Errorf("count prestamos: %w", err)
	}
	return n, nil
}

// CountNuevosEnRango cuenta préstamos creados en el rango.
func (r *PrestamosRepository) CountNuevosEnRango(ctx context.Context, desde, hasta time.Time) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM prestamos
		 WHERE created_at >= $1 AND created_at < $2`,
		desde, hasta,
	).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count nuevos: %w", err)
	}
	return n, nil
}

// CuotasVencidas lista todas las cuotas con vencimiento pasado y saldo > 0.
func (r *PrestamosRepository) CuotasVencidas(ctx context.Context, limit int) ([]models.CuotaVencida, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx, `
		SELECT
			c.id, c.prestamo_id, p.cliente_id, c.numero, c.fecha_vencimiento,
			(CURRENT_DATE - c.fecha_vencimiento)::int AS dias_vencidos,
			c.total, c.saldo_pendiente, c.mora_acumulada, c.estado
		FROM cuotas c
		JOIN prestamos p ON p.id = c.prestamo_id
		WHERE c.fecha_vencimiento < CURRENT_DATE
		  AND c.estado IN ('pendiente', 'parcial', 'vencida')
		  AND c.saldo_pendiente > 0
		ORDER BY c.fecha_vencimiento ASC, c.numero ASC
		LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("query vencidas: %w", err)
	}
	defer rows.Close()

	out := make([]models.CuotaVencida, 0)
	for rows.Next() {
		var c models.CuotaVencida
		if err := rows.Scan(
			&c.CuotaID, &c.PrestamoID, &c.ClienteID, &c.Numero, &c.FechaVencimiento,
			&c.DiasVencidos, &c.Total, &c.SaldoPendiente, &c.MoraAcumulada, &c.Estado,
		); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		out = append(out, c)
	}
	return out, nil
}

// CountCuotasVencidas devuelve solo el total.
func (r *PrestamosRepository) CountCuotasVencidas(ctx context.Context) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM cuotas
		WHERE fecha_vencimiento < CURRENT_DATE
		  AND estado IN ('pendiente', 'parcial', 'vencida')
		  AND saldo_pendiente > 0`,
	).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count vencidas: %w", err)
	}
	return n, nil
}

// CarteraOutstanding suma todos los saldos pendientes + mora de préstamos no
// finalizados/rechazados — el dinero que aún hay que cobrar.
func (r *PrestamosRepository) CarteraOutstanding(ctx context.Context) (float64, error) {
	var total float64
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(c.saldo_pendiente + c.mora_acumulada), 0)
		FROM cuotas c
		JOIN prestamos p ON p.id = c.prestamo_id
		WHERE p.estado IN ('activo', 'mora', 'aprobado')`,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("cartera: %w", err)
	}
	return total, nil
}

// ResumenPorCliente devuelve resumen de cada préstamo del cliente.
func (r *PrestamosRepository) ResumenPorCliente(ctx context.Context, clienteID uuid.UUID) ([]models.PrestamoResumen, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			p.id, p.monto_aprobado, p.estado, p.num_cuotas, p.fecha_solicitud,
			COUNT(c.*) FILTER (WHERE c.estado = 'pagada') AS cuotas_pagadas,
			COUNT(c.*) FILTER (WHERE c.fecha_vencimiento < CURRENT_DATE
			                   AND c.saldo_pendiente > 0) AS cuotas_vencidas,
			COALESCE(SUM(c.saldo_pendiente + c.mora_acumulada), 0) AS saldo_pendiente
		FROM prestamos p
		LEFT JOIN cuotas c ON c.prestamo_id = p.id
		WHERE p.cliente_id = $1
		GROUP BY p.id
		ORDER BY p.fecha_solicitud DESC`, clienteID)
	if err != nil {
		return nil, fmt.Errorf("query resumen: %w", err)
	}
	defer rows.Close()

	out := make([]models.PrestamoResumen, 0)
	for rows.Next() {
		var p models.PrestamoResumen
		if err := rows.Scan(
			&p.ID, &p.MontoAprobado, &p.Estado, &p.NumCuotas, &p.FechaSolicitud,
			&p.CuotasPagadas, &p.CuotasVencidas, &p.SaldoPendiente,
		); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		out = append(out, p)
	}
	return out, nil
}

// TotalPrestadoPorCliente suma todos los monto_aprobado del cliente.
func (r *PrestamosRepository) TotalPrestadoPorCliente(ctx context.Context, clienteID uuid.UUID) (float64, error) {
	var total float64
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(monto_aprobado), 0) FROM prestamos
		 WHERE cliente_id = $1 AND monto_aprobado IS NOT NULL`,
		clienteID,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("total prestado: %w", err)
	}
	return total, nil
}

// SaldoYMoraCliente suma saldo + mora de las cuotas no pagadas del cliente.
func (r *PrestamosRepository) SaldoYMoraCliente(ctx context.Context, clienteID uuid.UUID) (saldo, mora float64, err error) {
	err = r.pool.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(c.saldo_pendiente), 0),
			COALESCE(SUM(c.mora_acumulada), 0)
		FROM cuotas c
		JOIN prestamos p ON p.id = c.prestamo_id
		WHERE p.cliente_id = $1
		  AND p.estado IN ('activo', 'mora', 'aprobado')`,
		clienteID,
	).Scan(&saldo, &mora)
	return
}
