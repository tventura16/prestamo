//go:build integration

package repository

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestReversePagoFromCuota_Integration valida la reversión real contra una BD
// prestamos: devolución de saldo/mora, recálculo de estado, reactivación del
// préstamo finalizado e idempotencia (segunda reversión = Skipped, sin doble
// devolución). Requiere TEST_DATABASE_URL.
func TestReversePagoFromCuota_Integration(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("sin TEST_DATABASE_URL")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("conectar: %v", err)
	}
	defer pool.Close()

	r := NewLoanRepository(pool)

	// Seed: préstamo finalizado por un pago total de la única cuota
	// (capital 1000 + interés 20). La cuota venció hace 5 días.
	prestamoID, cuotaID, pagoID := uuid.New(), uuid.New(), uuid.New()
	mustExec(t, ctx, pool,
		`INSERT INTO prestamos (id, cliente_id, monto_solicitado, tasa_interes, num_cuotas, frecuencia, estado, fecha_desembolso)
		 VALUES ($1, gen_random_uuid(), 1000, 0.02, 1, 'mensual', 'finalizado', CURRENT_DATE-30)`, prestamoID)
	mustExec(t, ctx, pool,
		`INSERT INTO cuotas (id, prestamo_id, numero, fecha_vencimiento, capital, interes, total, saldo_pendiente, mora_acumulada, estado, fecha_pago)
		 VALUES ($1, $2, 1, CURRENT_DATE-5, 1000, 20, 1020, 0, 0, 'pagada', NOW())`, cuotaID, prestamoID)
	mustExec(t, ctx, pool,
		`INSERT INTO pago_aplicaciones (pago_id, cuota_id, capital, interes, mora)
		 VALUES ($1, $2, 1000, 20, 0)`, pagoID, cuotaID)

	// ── Primera reversión ──
	tx, _ := pool.Begin(ctx)
	res, amounts, err := r.ReversePagoFromCuota(ctx, tx, pagoID)
	if err != nil {
		t.Fatalf("reverse: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if res.Skipped {
		t.Error("primera reversión no debería ser Skipped")
	}
	if res.Cuota.SaldoPendiente != 1020 {
		t.Errorf("saldo devuelto: got %.2f want 1020.00", res.Cuota.SaldoPendiente)
	}
	if res.Cuota.Estado != "vencida" {
		t.Errorf("estado cuota: got %q want \"vencida\" (venció hace 5d)", res.Cuota.Estado)
	}
	if res.Prestamo.Estado != "activo" {
		t.Errorf("estado préstamo: got %q want \"activo\" (reactivado)", res.Prestamo.Estado)
	}
	if amounts.Capital != 1000 || amounts.Interes != 20 {
		t.Errorf("montos revertidos: got cap=%.2f int=%.2f want 1000/20", amounts.Capital, amounts.Interes)
	}

	// ── Segunda reversión: idempotente ──
	tx2, _ := pool.Begin(ctx)
	res2, _, err := r.ReversePagoFromCuota(ctx, tx2, pagoID)
	if err != nil {
		t.Fatalf("reverse idempotente: %v", err)
	}
	if err := tx2.Commit(ctx); err != nil {
		t.Fatalf("commit2: %v", err)
	}
	if !res2.Skipped {
		t.Error("segunda reversión debería ser Skipped")
	}
	if res2.Cuota.SaldoPendiente != 1020 {
		t.Errorf("idempotencia: saldo no debe duplicarse, got %.2f want 1020.00", res2.Cuota.SaldoPendiente)
	}
}

func mustExec(t *testing.T, ctx context.Context, pool *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(ctx, sql, args...); err != nil {
		t.Fatalf("seed exec: %v", err)
	}
}
