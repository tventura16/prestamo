package repository

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prestamos/payment-service/internal/models"
)

var (
	ErrCuotaNotFound        = errors.New("cuota no encontrada")
	ErrCuotaPagada          = errors.New("la cuota ya está pagada")
	ErrOverpayment          = errors.New("el monto excede lo adeudado en la cuota")
	ErrAplicacionNoConcilia = errors.New("el pago aún no fue aplicado a la cuota; reintente la anulación")
)

// LoanRepository accede a la DB "prestamos" (cuotas + prestamos) desde el
// payment-service. En arquitecturas más maduras esto sería una llamada
// HTTP al loan-service o un evento; aquí es acceso directo por simplicidad.
type LoanRepository struct {
	pool *pgxpool.Pool
}

func NewLoanRepository(pool *pgxpool.Pool) *LoanRepository {
	return &LoanRepository{pool: pool}
}

// CuotaSnapshot es el estado de una cuota leído bajo lock.
type CuotaSnapshot struct {
	ID             uuid.UUID
	PrestamoID     uuid.UUID
	Numero         int
	Capital        float64
	Interes        float64
	Total          float64
	SaldoPendiente float64
	MoraAcumulada  float64
	Estado         string
	ClienteID      uuid.UUID
	EstadoPrestamo string
}

// LockCuotaWithPrestamo lee la cuota + cliente + estado del préstamo bajo
// SELECT FOR UPDATE. Debe llamarse dentro de una transacción.
func (r *LoanRepository) LockCuotaWithPrestamo(ctx context.Context, tx pgx.Tx, cuotaID uuid.UUID) (CuotaSnapshot, error) {
	var s CuotaSnapshot
	err := tx.QueryRow(ctx, `
		SELECT c.id, c.prestamo_id, c.numero, c.capital, c.interes, c.total,
		       c.saldo_pendiente, c.mora_acumulada, c.estado,
		       p.cliente_id, p.estado
		FROM cuotas c
		JOIN prestamos p ON p.id = c.prestamo_id
		WHERE c.id = $1
		FOR UPDATE OF c, p`, cuotaID,
	).Scan(
		&s.ID, &s.PrestamoID, &s.Numero, &s.Capital, &s.Interes, &s.Total,
		&s.SaldoPendiente, &s.MoraAcumulada, &s.Estado,
		&s.ClienteID, &s.EstadoPrestamo,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CuotaSnapshot{}, ErrCuotaNotFound
		}
		return CuotaSnapshot{}, fmt.Errorf("lock cuota: %w", err)
	}
	return s, nil
}

// PreviousPaid es lo ya cobrado en interes y capital de una cuota
// (sumando pagos no anulados). Necesario para distribuir correctamente
// pagos parciales subsiguientes.
type PreviousPaid struct {
	Interes float64
	Capital float64
}

// GetPreviousPaid consulta los pagos previos para esta cuota desde la DB pagos.
// Se llama desde el repo de pagos, no desde acá — pero el modelo lo dejamos
// definido aquí para mantenerlo cerca de la lógica de negocio.

// PagoAplicacion es la intención de aplicar un pago concreto a una cuota.
// Los montos provienen del cálculo de distribución hecho al registrar el
// pago; ApplyPagoToCuota los clampa contra el saldo vivo.
type PagoAplicacion struct {
	PagoID  uuid.UUID
	CuotaID uuid.UUID
	Capital float64
	Interes float64
	Mora    float64
}

// ApplyResult informa el resultado de aplicar un pago a una cuota.
type ApplyResult struct {
	Cuota    models.CuotaInfo
	Prestamo models.PrestamoInfo
	Skipped  bool // true si el pago ya estaba aplicado (idempotencia)
}

// ApplyPagoToCuota aplica un pago a su cuota de forma IDEMPOTENTE dentro de
// la transacción dada (DB prestamos). Es el único punto de escritura del
// saldo de la cuota, usado tanto por el fast-path inline como por el consumer
// del outbox.
//
//  1. Guard: si pago_aplicaciones ya tiene el pago_id → no reaplica (Skipped).
//  2. SELECT FOR UPDATE de la cuota: serializa pagos concurrentes.
//  3. Clampa los montos al saldo/mora vivos (un pago concurrente pudo
//     reducir el saldo desde que se calculó la distribución).
//  4. Actualiza la cuota; si queda saldada y todas las del préstamo también,
//     marca el préstamo como finalizado.
//  5. Inserta el guard de idempotencia.
func (r *LoanRepository) ApplyPagoToCuota(ctx context.Context, tx pgx.Tx, app PagoAplicacion) (ApplyResult, error) {
	// 1. Guard de idempotencia.
	var exists bool
	if err := tx.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM pago_aplicaciones WHERE pago_id = $1)`, app.PagoID,
	).Scan(&exists); err != nil {
		return ApplyResult{}, fmt.Errorf("check guard: %w", err)
	}
	if exists {
		cuota, prestamo, err := r.readCuotaPrestamo(ctx, tx, app.CuotaID)
		if err != nil {
			return ApplyResult{}, err
		}
		return ApplyResult{Cuota: cuota, Prestamo: prestamo, Skipped: true}, nil
	}

	// 2. Lock de la cuota + estado del préstamo.
	var saldo, mora float64
	var prestamoID uuid.UUID
	var numero int
	var estadoPrestamo string
	err := tx.QueryRow(ctx, `
		SELECT c.prestamo_id, c.numero, c.saldo_pendiente, c.mora_acumulada, p.estado
		FROM cuotas c JOIN prestamos p ON p.id = c.prestamo_id
		WHERE c.id = $1
		FOR UPDATE OF c, p`, app.CuotaID,
	).Scan(&prestamoID, &numero, &saldo, &mora, &estadoPrestamo)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ApplyResult{}, ErrCuotaNotFound
		}
		return ApplyResult{}, fmt.Errorf("lock cuota: %w", err)
	}

	// 3. Clamp contra saldo vivo. El interés se cubre antes que el capital.
	moraAplic := round2(math.Min(app.Mora, mora))
	saldoAplic := round2(math.Min(round2(app.Interes+app.Capital), saldo))
	interesAplic := round2(math.Min(app.Interes, saldoAplic))
	capitalAplic := round2(saldoAplic - interesAplic)
	newSaldo := round2(saldo - saldoAplic)
	if newSaldo < 0 {
		newSaldo = 0
	}
	newMora := round2(mora - moraAplic)
	if newMora < 0 {
		newMora = 0
	}

	estado := "parcial"
	marcarFecha := false
	if newSaldo == 0 && newMora == 0 {
		estado = "pagada"
		marcarFecha = true
	}

	// 4. Update cuota.
	var info models.CuotaInfo
	query := `UPDATE cuotas SET saldo_pendiente = $1, mora_acumulada = $2, estado = $3`
	if marcarFecha {
		query += `, fecha_pago = NOW()`
	}
	query += ` WHERE id = $4
		RETURNING id, prestamo_id, numero, saldo_pendiente, mora_acumulada, estado`
	if err := tx.QueryRow(ctx, query, newSaldo, newMora, estado, app.CuotaID).Scan(
		&info.ID, &info.PrestamoID, &info.Numero,
		&info.SaldoPendiente, &info.MoraAcumulada, &info.Estado,
	); err != nil {
		return ApplyResult{}, fmt.Errorf("update cuota: %w", err)
	}

	// Préstamo: finalizar si todas las cuotas quedaron saldadas.
	prestamoInfo := models.PrestamoInfo{ID: prestamoID, Estado: estadoPrestamo}
	if estado == "pagada" {
		allPaid, err := r.AllCuotasPagadas(ctx, tx, prestamoID)
		if err != nil {
			return ApplyResult{}, err
		}
		if allPaid {
			prestamoInfo, err = r.MarkPrestamoFinalizado(ctx, tx, prestamoID)
			if err != nil {
				return ApplyResult{}, err
			}
		}
	}

	// 5. Guard de idempotencia con los montos efectivamente aplicados.
	if _, err := tx.Exec(ctx, `
		INSERT INTO pago_aplicaciones (pago_id, cuota_id, capital, interes, mora)
		VALUES ($1, $2, $3, $4, $5)`,
		app.PagoID, app.CuotaID, capitalAplic, interesAplic, moraAplic,
	); err != nil {
		return ApplyResult{}, fmt.Errorf("insert guard: %w", err)
	}

	return ApplyResult{Cuota: info, Prestamo: prestamoInfo}, nil
}

// ReversedAmounts son los montos que efectivamente se devolvieron a la cuota
// al revertir un pago (leídos del ledger pago_aplicaciones). El service los
// necesita para construir el evento pago.anulado.
type ReversedAmounts struct {
	Capital float64
	Interes float64
	Mora    float64
}

// ReversePagoFromCuota deshace la aplicación de un pago a su cuota de forma
// IDEMPOTENTE dentro de la transacción dada (DB prestamos). Es el simétrico de
// ApplyPagoToCuota y el único punto que revierte el saldo de una cuota, usado
// tanto por el fast-path inline de la anulación como por el consumer del outbox.
//
//  1. Lee del ledger los montos realmente aplicados (con lock). Si no hay
//     registro, el pago no llegó a aplicarse → ErrAplicacionNoConcilia.
//  2. Guard de idempotencia: si reverted_at ya tiene valor → no revierte de
//     nuevo (Skipped).
//  3. Devuelve capital+interés al saldo y la mora cobrada a mora_acumulada.
//  4. Recalcula el estado de la cuota (deja de estar 'pagada'); si la fecha ya
//     venció queda 'vencida' (el job de mora retomará el devengo).
//  5. Reactiva el préstamo si estaba 'finalizado'.
//  6. Sella reverted_at.
func (r *LoanRepository) ReversePagoFromCuota(ctx context.Context, tx pgx.Tx, pagoID uuid.UUID) (ApplyResult, ReversedAmounts, error) {
	// 1. Montos aplicados desde el ledger, bajo lock.
	var cuotaID uuid.UUID
	var aplCapital, aplInteres, aplMora float64
	var revertedAt *time.Time
	err := tx.QueryRow(ctx, `
		SELECT cuota_id, capital, interes, mora, reverted_at
		FROM pago_aplicaciones WHERE pago_id = $1
		FOR UPDATE`, pagoID,
	).Scan(&cuotaID, &aplCapital, &aplInteres, &aplMora, &revertedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ApplyResult{}, ReversedAmounts{}, ErrAplicacionNoConcilia
		}
		return ApplyResult{}, ReversedAmounts{}, fmt.Errorf("lock aplicacion: %w", err)
	}
	amounts := ReversedAmounts{Capital: aplCapital, Interes: aplInteres, Mora: aplMora}

	// 2. Ya revertido → idempotente.
	if revertedAt != nil {
		cuota, prestamo, err := r.readCuotaPrestamo(ctx, tx, cuotaID)
		if err != nil {
			return ApplyResult{}, ReversedAmounts{}, err
		}
		return ApplyResult{Cuota: cuota, Prestamo: prestamo, Skipped: true}, amounts, nil
	}

	// 3. Lock de la cuota + datos para recalcular estado.
	var saldo, mora, total float64
	var prestamoID uuid.UUID
	var numero int
	var estadoPrestamo string
	var vencida bool
	err = tx.QueryRow(ctx, `
		SELECT c.prestamo_id, c.numero, c.saldo_pendiente, c.mora_acumulada,
		       c.total, (c.fecha_vencimiento < CURRENT_DATE), p.estado
		FROM cuotas c JOIN prestamos p ON p.id = c.prestamo_id
		WHERE c.id = $1
		FOR UPDATE OF c, p`, cuotaID,
	).Scan(&prestamoID, &numero, &saldo, &mora, &total, &vencida, &estadoPrestamo)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ApplyResult{}, ReversedAmounts{}, ErrCuotaNotFound
		}
		return ApplyResult{}, ReversedAmounts{}, fmt.Errorf("lock cuota: %w", err)
	}

	newSaldo := round2(saldo + aplCapital + aplInteres)
	newMora := round2(mora + aplMora)

	// 4. Recalcular estado.
	estado := "pendiente"
	switch {
	case newSaldo <= 0.005 && newMora <= 0.005:
		estado = "pagada"
	case vencida:
		estado = "vencida"
	case newSaldo+0.005 < total:
		estado = "parcial"
	}

	var info models.CuotaInfo
	query := `UPDATE cuotas SET saldo_pendiente = $1, mora_acumulada = $2, estado = $3`
	if estado == "pagada" {
		query += ` WHERE id = $4`
	} else {
		query += `, fecha_pago = NULL WHERE id = $4`
	}
	query += ` RETURNING id, prestamo_id, numero, saldo_pendiente, mora_acumulada, estado`
	if err := tx.QueryRow(ctx, query, newSaldo, newMora, estado, cuotaID).Scan(
		&info.ID, &info.PrestamoID, &info.Numero,
		&info.SaldoPendiente, &info.MoraAcumulada, &info.Estado,
	); err != nil {
		return ApplyResult{}, ReversedAmounts{}, fmt.Errorf("update cuota: %w", err)
	}

	// 5. Reactivar préstamo finalizado (el job de mora lo pasará a mora si aplica).
	prestamoInfo := models.PrestamoInfo{ID: prestamoID, Estado: estadoPrestamo}
	if estadoPrestamo == "finalizado" {
		if err := tx.QueryRow(ctx,
			`UPDATE prestamos SET estado = 'activo' WHERE id = $1 RETURNING id, estado`,
			prestamoID,
		).Scan(&prestamoInfo.ID, &prestamoInfo.Estado); err != nil {
			return ApplyResult{}, ReversedAmounts{}, fmt.Errorf("reactivar prestamo: %w", err)
		}
	}

	// 6. Sellar la reversión (guard de idempotencia).
	if _, err := tx.Exec(ctx,
		`UPDATE pago_aplicaciones SET reverted_at = NOW() WHERE pago_id = $1`, pagoID,
	); err != nil {
		return ApplyResult{}, ReversedAmounts{}, fmt.Errorf("sellar reversion: %w", err)
	}

	return ApplyResult{Cuota: info, Prestamo: prestamoInfo}, amounts, nil
}

// readCuotaPrestamo lee el estado actual (sin lock) para respuestas idempotentes.
func (r *LoanRepository) readCuotaPrestamo(ctx context.Context, tx pgx.Tx, cuotaID uuid.UUID) (models.CuotaInfo, models.PrestamoInfo, error) {
	var c models.CuotaInfo
	var p models.PrestamoInfo
	err := tx.QueryRow(ctx, `
		SELECT c.id, c.prestamo_id, c.numero, c.saldo_pendiente, c.mora_acumulada, c.estado,
		       p.id, p.estado
		FROM cuotas c JOIN prestamos p ON p.id = c.prestamo_id
		WHERE c.id = $1`, cuotaID,
	).Scan(&c.ID, &c.PrestamoID, &c.Numero, &c.SaldoPendiente, &c.MoraAcumulada, &c.Estado,
		&p.ID, &p.Estado)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.CuotaInfo{}, models.PrestamoInfo{}, ErrCuotaNotFound
		}
		return models.CuotaInfo{}, models.PrestamoInfo{}, fmt.Errorf("read cuota: %w", err)
	}
	return c, p, nil
}

func round2(x float64) float64 { return math.Round(x*100) / 100 }

// AllCuotasPagadas devuelve true si no quedan cuotas con saldo > 0.
func (r *LoanRepository) AllCuotasPagadas(ctx context.Context, tx pgx.Tx, prestamoID uuid.UUID) (bool, error) {
	var pendientes int
	err := tx.QueryRow(ctx,
		`SELECT COUNT(*) FROM cuotas
		 WHERE prestamo_id = $1
		   AND (saldo_pendiente > 0 OR mora_acumulada > 0)`,
		prestamoID,
	).Scan(&pendientes)
	if err != nil {
		return false, fmt.Errorf("count cuotas: %w", err)
	}
	return pendientes == 0, nil
}

// MarkPrestamoFinalizado cambia el estado a finalizado.
func (r *LoanRepository) MarkPrestamoFinalizado(ctx context.Context, tx pgx.Tx, prestamoID uuid.UUID) (models.PrestamoInfo, error) {
	var info models.PrestamoInfo
	err := tx.QueryRow(ctx,
		`UPDATE prestamos SET estado = 'finalizado'
		 WHERE id = $1
		 RETURNING id, estado`,
		prestamoID,
	).Scan(&info.ID, &info.Estado)
	if err != nil {
		return models.PrestamoInfo{}, fmt.Errorf("update prestamo: %w", err)
	}
	return info, nil
}

// GetPrestamoEstado lee el estado actual del préstamo (sin lock).
func (r *LoanRepository) GetPrestamoEstado(ctx context.Context, prestamoID uuid.UUID) (models.PrestamoInfo, error) {
	var info models.PrestamoInfo
	err := r.pool.QueryRow(ctx,
		`SELECT id, estado FROM prestamos WHERE id = $1`, prestamoID,
	).Scan(&info.ID, &info.Estado)
	if err != nil {
		return models.PrestamoInfo{}, err
	}
	return info, nil
}

// Pool expone el pool para que el service inicie transacciones.
func (r *LoanRepository) Pool() *pgxpool.Pool { return r.pool }
