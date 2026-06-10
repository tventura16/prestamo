package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prestamos/loan-service/internal/models"
	"github.com/prestamos/loan-service/internal/service"
)

var (
	ErrNotFound       = errors.New("préstamo no encontrado")
	ErrInvalidState   = errors.New("transición de estado inválida")
	ErrAlreadyDecided = errors.New("el préstamo ya fue aprobado o rechazado")
	ErrClienteConMora = errors.New("el cliente tiene mora activa; no puede aprobarse un nuevo préstamo")
)

type PrestamoRepository struct {
	pool *pgxpool.Pool
}

func NewPrestamoRepository(pool *pgxpool.Pool) *PrestamoRepository {
	return &PrestamoRepository{pool: pool}
}

const prestamoColumns = `id, cliente_id, monto_solicitado, monto_aprobado,
		tasa_interes, tipo_interes, fecha_solicitud, fecha_desembolso,
		num_cuotas, frecuencia, estado, aprobado_por, observaciones,
		created_at, updated_at`

func scanPrestamo(row pgx.Row) (models.Prestamo, error) {
	var p models.Prestamo
	err := row.Scan(
		&p.ID, &p.ClienteID, &p.MontoSolicitado, &p.MontoAprobado,
		&p.TasaInteres, &p.TipoInteres, &p.FechaSolicitud, &p.FechaDesembolso,
		&p.NumCuotas, &p.Frecuencia, &p.Estado, &p.AprobadoPor, &p.Observaciones,
		&p.CreatedAt, &p.UpdatedAt,
	)
	return p, err
}

func (r *PrestamoRepository) Create(ctx context.Context, in models.CreatePrestamoInput) (models.Prestamo, error) {
	tipo := models.TipoInteresFijo
	if in.TipoInteres != nil {
		tipo = *in.TipoInteres
	}

	query := `INSERT INTO prestamos
		(cliente_id, monto_solicitado, tasa_interes, tipo_interes, num_cuotas, frecuencia, observaciones)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING ` + prestamoColumns

	row := r.pool.QueryRow(ctx, query,
		in.ClienteID, in.MontoSolicitado, in.TasaInteres, tipo,
		in.NumCuotas, in.Frecuencia, in.Observaciones,
	)
	p, err := scanPrestamo(row)
	if err != nil {
		return models.Prestamo{}, fmt.Errorf("insert prestamo: %w", err)
	}
	return p, nil
}

func (r *PrestamoRepository) GetByID(ctx context.Context, id uuid.UUID) (models.Prestamo, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+prestamoColumns+` FROM prestamos WHERE id = $1`, id)
	p, err := scanPrestamo(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Prestamo{}, ErrNotFound
		}
		return models.Prestamo{}, fmt.Errorf("get prestamo: %w", err)
	}
	return p, nil
}

func (r *PrestamoRepository) List(ctx context.Context, page, limit int, clienteID *uuid.UUID, estado *models.Estado) (models.ListResult, error) {
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
	if clienteID != nil {
		addFilter("cliente_id", *clienteID)
	}
	if estado != nil {
		addFilter("estado", *estado)
	}

	var total int64
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM prestamos`+where, args...).Scan(&total); err != nil {
		return models.ListResult{}, fmt.Errorf("count prestamos: %w", err)
	}

	listQuery := `SELECT ` + prestamoColumns + ` FROM prestamos` + where +
		fmt.Sprintf(` ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, len(args)+1, len(args)+2)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, listQuery, args...)
	if err != nil {
		return models.ListResult{}, fmt.Errorf("list prestamos: %w", err)
	}
	defer rows.Close()

	items := make([]models.Prestamo, 0, limit)
	for rows.Next() {
		p, err := scanPrestamo(rows)
		if err != nil {
			return models.ListResult{}, fmt.Errorf("scan: %w", err)
		}
		items = append(items, p)
	}
	return models.ListResult{Items: items, Total: total, Page: page, Limit: limit}, nil
}

// Approve aprueba y desembolsa el préstamo, generando el plan de pagos en
// una transacción atómica. El préstamo pasa a estado "activo".
func (r *PrestamoRepository) Approve(ctx context.Context, id uuid.UUID, in models.ApprovePrestamoInput) (models.Prestamo, []models.Cuota, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return models.Prestamo{}, nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Lock + lectura del préstamo
	row := tx.QueryRow(ctx,
		`SELECT `+prestamoColumns+` FROM prestamos WHERE id = $1 FOR UPDATE`, id)
	p, err := scanPrestamo(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Prestamo{}, nil, ErrNotFound
		}
		return models.Prestamo{}, nil, fmt.Errorf("lock prestamo: %w", err)
	}

	if p.Estado != models.EstadoPendiente {
		return models.Prestamo{}, nil, ErrAlreadyDecided
	}

	// Regla de negocio §7: no se aprueba un préstamo si el cliente tiene mora
	// activa, salvo que el parámetro aprobar_si_mora_activa lo permita.
	aprobarSiMora, err := leerAprobarSiMora(ctx, tx)
	if err != nil {
		return models.Prestamo{}, nil, err
	}
	if !aprobarSiMora {
		conMora, err := clienteTieneMoraActiva(ctx, tx, p.ClienteID)
		if err != nil {
			return models.Prestamo{}, nil, err
		}
		if conMora {
			return models.Prestamo{}, nil, ErrClienteConMora
		}
	}

	monto := p.MontoSolicitado
	if in.MontoAprobado != nil {
		monto = *in.MontoAprobado
	}

	fechaDesembolso := time.Now()
	if in.FechaDesembolso != nil && *in.FechaDesembolso != "" {
		fd, err := time.Parse("2006-01-02", *in.FechaDesembolso)
		if err != nil {
			return models.Prestamo{}, nil, fmt.Errorf("parse fecha_desembolso: %w", err)
		}
		fechaDesembolso = fd
	}

	obs := p.Observaciones
	if in.Observaciones != nil {
		obs = in.Observaciones
	}

	// Update préstamo a estado activo
	updRow := tx.QueryRow(ctx, `UPDATE prestamos SET
			estado = 'activo',
			monto_aprobado = $1,
			fecha_desembolso = $2,
			aprobado_por = $3,
			observaciones = $4
		WHERE id = $5
		RETURNING `+prestamoColumns,
		monto, fechaDesembolso, in.AprobadoPor, obs, id,
	)
	pUpdated, err := scanPrestamo(updRow)
	if err != nil {
		return models.Prestamo{}, nil, fmt.Errorf("update prestamo: %w", err)
	}

	// Generar plan de pagos
	plan := service.GenerarPlanFrances(monto, pUpdated.TasaInteres, pUpdated.NumCuotas, pUpdated.Frecuencia, fechaDesembolso)

	cuotas := make([]models.Cuota, 0, len(plan))
	for _, c := range plan {
		var inserted models.Cuota
		err := tx.QueryRow(ctx,
			`INSERT INTO cuotas
				(prestamo_id, numero, fecha_vencimiento, capital, interes, total, saldo_pendiente)
				VALUES ($1, $2, $3, $4, $5, $6, $7)
				RETURNING id, prestamo_id, numero, fecha_vencimiento, capital, interes,
					total, saldo_pendiente, mora_acumulada, estado, fecha_pago, created_at, updated_at`,
			id, c.Numero, c.FechaVencimiento, c.Capital, c.Interes, c.Total, c.SaldoPendiente,
		).Scan(
			&inserted.ID, &inserted.PrestamoID, &inserted.Numero, &inserted.FechaVencimiento,
			&inserted.Capital, &inserted.Interes, &inserted.Total, &inserted.SaldoPendiente,
			&inserted.MoraAcumulada, &inserted.Estado, &inserted.FechaPago,
			&inserted.CreatedAt, &inserted.UpdatedAt,
		)
		if err != nil {
			return models.Prestamo{}, nil, fmt.Errorf("insert cuota %d: %w", c.Numero, err)
		}
		cuotas = append(cuotas, inserted)
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Prestamo{}, nil, fmt.Errorf("commit: %w", err)
	}
	return pUpdated, cuotas, nil
}

// clienteTieneMoraActiva indica si el cliente tiene alguna cuota vencida con
// saldo o mora en sus préstamos activos/en mora. Se evalúa dentro de la misma
// transacción que la aprobación para ver un estado consistente.
func clienteTieneMoraActiva(ctx context.Context, tx pgx.Tx, clienteID uuid.UUID) (bool, error) {
	var exists bool
	err := tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM cuotas c
			JOIN prestamos p ON p.id = c.prestamo_id
			WHERE p.cliente_id = $1
			  AND p.estado IN ('activo', 'mora')
			  AND c.estado = 'vencida'
			  AND (c.saldo_pendiente > 0 OR c.mora_acumulada > 0)
		)`, clienteID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("verificar mora del cliente: %w", err)
	}
	return exists, nil
}

// leerAprobarSiMora lee el parámetro de sistema; default false (no aprobar con
// mora) si la fila no existe.
func leerAprobarSiMora(ctx context.Context, tx pgx.Tx) (bool, error) {
	var valor string
	err := tx.QueryRow(ctx,
		`SELECT valor FROM parametros_sistema WHERE clave = 'aprobar_si_mora_activa'`).Scan(&valor)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("leer parámetro de mora: %w", err)
	}
	return valor == "true", nil
}

func (r *PrestamoRepository) Reject(ctx context.Context, id uuid.UUID, in models.RejectPrestamoInput) (models.Prestamo, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return models.Prestamo{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx,
		`SELECT `+prestamoColumns+` FROM prestamos WHERE id = $1 FOR UPDATE`, id)
	p, err := scanPrestamo(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Prestamo{}, ErrNotFound
		}
		return models.Prestamo{}, fmt.Errorf("lock: %w", err)
	}
	if p.Estado != models.EstadoPendiente {
		return models.Prestamo{}, ErrAlreadyDecided
	}

	updRow := tx.QueryRow(ctx, `UPDATE prestamos SET
			estado = 'rechazado',
			aprobado_por = $1,
			observaciones = $2
		WHERE id = $3
		RETURNING `+prestamoColumns,
		in.AprobadoPor, in.Observaciones, id,
	)
	pUpdated, err := scanPrestamo(updRow)
	if err != nil {
		return models.Prestamo{}, fmt.Errorf("update: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return models.Prestamo{}, fmt.Errorf("commit: %w", err)
	}
	return pUpdated, nil
}
