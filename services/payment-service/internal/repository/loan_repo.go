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

var (
	ErrCuotaNotFound = errors.New("cuota no encontrada")
	ErrCuotaPagada   = errors.New("la cuota ya está pagada")
	ErrOverpayment   = errors.New("el monto excede lo adeudado en la cuota")
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
	ID               uuid.UUID
	PrestamoID       uuid.UUID
	Numero           int
	Capital          float64
	Interes          float64
	Total            float64
	SaldoPendiente   float64
	MoraAcumulada    float64
	Estado           string
	ClienteID        uuid.UUID
	EstadoPrestamo   string
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

// UpdateCuotaAfterPayment actualiza saldo, mora, estado y fecha_pago en una cuota.
func (r *LoanRepository) UpdateCuotaAfterPayment(ctx context.Context, tx pgx.Tx,
	cuotaID uuid.UUID, newSaldo, newMora float64, estado string, marcarPagoAhora bool,
) (models.CuotaInfo, error) {
	var info models.CuotaInfo
	var query string
	if marcarPagoAhora {
		query = `UPDATE cuotas SET
			saldo_pendiente = $1, mora_acumulada = $2,
			estado = $3, fecha_pago = NOW()
			WHERE id = $4
			RETURNING id, prestamo_id, numero, saldo_pendiente, mora_acumulada, estado`
	} else {
		query = `UPDATE cuotas SET
			saldo_pendiente = $1, mora_acumulada = $2, estado = $3
			WHERE id = $4
			RETURNING id, prestamo_id, numero, saldo_pendiente, mora_acumulada, estado`
	}
	err := tx.QueryRow(ctx, query, newSaldo, newMora, estado, cuotaID).Scan(
		&info.ID, &info.PrestamoID, &info.Numero,
		&info.SaldoPendiente, &info.MoraAcumulada, &info.Estado,
	)
	if err != nil {
		return models.CuotaInfo{}, fmt.Errorf("update cuota: %w", err)
	}
	return info, nil
}

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
