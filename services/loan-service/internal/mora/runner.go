// Package mora implementa el devengo automático de mora y las transiciones de
// estado por vencimiento que el alcance describe como "control de mora".
//
// El loan-service es el dueño del ciclo de vida del préstamo, por lo que es
// aquí —y no en el report-service, que solo consulta— donde se materializan
// las reglas de negocio: las cuotas se marcan como vencidas, se devenga el
// interés moratorio sobre el saldo impago y el préstamo transiciona a "mora"
// (o vuelve a "activo" tras regularizarse).
package mora

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// advisoryLockKey identifica el job para pg_try_advisory_xact_lock, de modo que
// solo una instancia lo ejecute aunque el servicio corra con varias réplicas.
const advisoryLockKey int64 = 0x4D4F5241 // 'MORA' en ASCII

// Valores por defecto si parametros_sistema no estuviera disponible.
const (
	defaultTasaMoraDiaria = 0.05 // % diario sobre el saldo vencido
	defaultDiasGracia     = 1
)

// Result resume el efecto de una corrida del job (para logging/observabilidad).
type Result struct {
	MoraDevengada          int64 // cuotas a las que se sumó mora
	CuotasVencidas         int64 // cuotas que pasaron a estado 'vencida'
	PrestamosEnMora        int64 // préstamos activo -> mora
	PrestamosRegularizados int64 // préstamos mora -> activo
	Skipped                bool  // otra instancia tenía el lock
}

// Runner ejecuta el job de mora sobre la base de datos del loan-service.
type Runner struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

func NewRunner(pool *pgxpool.Pool, logger *slog.Logger) *Runner {
	return &Runner{pool: pool, logger: logger}
}

// Run ejecuta una pasada completa del job en una sola transacción:
//  1. devenga mora diaria sobre cuotas vivas pasadas del período de gracia;
//  2. marca como 'vencida' las cuotas impagas con vencimiento anterior a hoy;
//  3. transiciona a 'mora' los préstamos activos con cuotas vencidas;
//  4. devuelve a 'activo' los préstamos en mora ya regularizados.
//
// El devengo es idempotente: mora_aplicada_hasta acota los días ya aplicados,
// así dos corridas el mismo día no duplican la mora.
func (r *Runner) Run(ctx context.Context) (Result, error) {
	var res Result

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return res, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Lock cooperativo a nivel de transacción: se libera solo en commit/rollback.
	var locked bool
	if err := tx.QueryRow(ctx, `SELECT pg_try_advisory_xact_lock($1)`, advisoryLockKey).Scan(&locked); err != nil {
		return res, fmt.Errorf("advisory lock: %w", err)
	}
	if !locked {
		res.Skipped = true
		return res, nil
	}

	tasa, gracia := r.loadParams(ctx, tx)

	// 1) Devengo de mora. GREATEST(mora_aplicada_hasta, venc+gracia) ignora NULL
	//    en Postgres, por lo que en la primera corrida arranca en venc+gracia.
	tag, err := tx.Exec(ctx, `
		UPDATE cuotas
		SET mora_acumulada = ROUND(
				mora_acumulada
				+ saldo_pendiente * ($1::numeric / 100.0)
				  * (CURRENT_DATE - GREATEST(mora_aplicada_hasta, fecha_vencimiento + $2::int))
			, 2),
			mora_aplicada_hasta = CURRENT_DATE
		WHERE estado IN ('pendiente', 'parcial', 'vencida')
		  AND saldo_pendiente > 0
		  AND (fecha_vencimiento + $2::int) < CURRENT_DATE
		  AND COALESCE(mora_aplicada_hasta, fecha_vencimiento + $2::int) < CURRENT_DATE`,
		tasa, gracia)
	if err != nil {
		return res, fmt.Errorf("devengo mora: %w", err)
	}
	res.MoraDevengada = tag.RowsAffected()

	// 2) Marcar cuotas vencidas (sin gracia: una cuota está vencida al día
	//    siguiente de su fecha; la gracia aplica solo al cobro monetario).
	tag, err = tx.Exec(ctx, `
		UPDATE cuotas
		SET estado = 'vencida'
		WHERE estado IN ('pendiente', 'parcial')
		  AND saldo_pendiente > 0
		  AND fecha_vencimiento < CURRENT_DATE`)
	if err != nil {
		return res, fmt.Errorf("marcar vencidas: %w", err)
	}
	res.CuotasVencidas = tag.RowsAffected()

	// 3) Préstamos activos con al menos una cuota vencida impaga -> mora.
	tag, err = tx.Exec(ctx, `
		UPDATE prestamos p
		SET estado = 'mora'
		WHERE p.estado = 'activo'
		  AND EXISTS (
		      SELECT 1 FROM cuotas c
		      WHERE c.prestamo_id = p.id
		        AND c.estado = 'vencida'
		        AND c.saldo_pendiente > 0
		  )`)
	if err != nil {
		return res, fmt.Errorf("prestamos a mora: %w", err)
	}
	res.PrestamosEnMora = tag.RowsAffected()

	// 4) Préstamos en mora ya sin cuotas vencidas impagas -> activo
	//    (regularización tras pago registrado por el payment-service).
	tag, err = tx.Exec(ctx, `
		UPDATE prestamos p
		SET estado = 'activo'
		WHERE p.estado = 'mora'
		  AND NOT EXISTS (
		      SELECT 1 FROM cuotas c
		      WHERE c.prestamo_id = p.id
		        AND c.estado = 'vencida'
		        AND c.saldo_pendiente > 0
		  )`)
	if err != nil {
		return res, fmt.Errorf("prestamos regularizados: %w", err)
	}
	res.PrestamosRegularizados = tag.RowsAffected()

	if err := tx.Commit(ctx); err != nil {
		return res, fmt.Errorf("commit: %w", err)
	}
	return res, nil
}

// loadParams lee la tasa de mora diaria y los días de gracia de
// parametros_sistema, con fallback a los valores por defecto del alcance.
func (r *Runner) loadParams(ctx context.Context, tx pgx.Tx) (tasa float64, gracia int) {
	tasa, gracia = defaultTasaMoraDiaria, defaultDiasGracia

	rows, err := tx.Query(ctx,
		`SELECT clave, valor FROM parametros_sistema
		 WHERE clave IN ('tasa_mora_diaria', 'dias_gracia_mora')`)
	if err != nil {
		r.logger.Warn("no se pudieron leer parámetros de mora, usando defaults", "err", err)
		return tasa, gracia
	}
	defer rows.Close()

	for rows.Next() {
		var clave, valor string
		if err := rows.Scan(&clave, &valor); err != nil {
			continue
		}
		switch clave {
		case "tasa_mora_diaria":
			if f, err := strconv.ParseFloat(valor, 64); err == nil {
				tasa = f
			}
		case "dias_gracia_mora":
			if n, err := strconv.Atoi(valor); err == nil {
				gracia = n
			}
		}
	}
	return tasa, gracia
}

// Schedule corre el job de inmediato y luego en cada tick de `interval`, hasta
// que el contexto se cancele. Pensado para ejecutarse en una goroutine.
func (r *Runner) Schedule(ctx context.Context, interval time.Duration) {
	r.runOnce(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			r.logger.Info("scheduler de mora detenido")
			return
		case <-ticker.C:
			r.runOnce(ctx)
		}
	}
}

func (r *Runner) runOnce(ctx context.Context) {
	start := time.Now()
	res, err := r.Run(ctx)
	if err != nil {
		r.logger.Error("job de mora falló", "err", err)
		return
	}
	if res.Skipped {
		r.logger.Info("job de mora omitido (otra instancia en ejecución)")
		return
	}
	r.logger.Info("job de mora ejecutado",
		"mora_devengada", res.MoraDevengada,
		"cuotas_vencidas", res.CuotasVencidas,
		"prestamos_en_mora", res.PrestamosEnMora,
		"prestamos_regularizados", res.PrestamosRegularizados,
		"duration_ms", time.Since(start).Milliseconds(),
	)
}
