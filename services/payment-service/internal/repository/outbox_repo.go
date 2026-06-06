package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OutboxEvent es una fila pendiente de publicar del outbox transaccional
// (DB pagos).
type OutboxEvent struct {
	ID            uuid.UUID
	AggregateType string
	AggregateID   uuid.UUID
	EventType     string
	Payload       []byte
}

// OutboxRepository accede a la tabla outbox_events en la DB pagos.
type OutboxRepository struct {
	pool *pgxpool.Pool
}

func NewOutboxRepository(pool *pgxpool.Pool) *OutboxRepository {
	return &OutboxRepository{pool: pool}
}

// InsertTx inserta un evento en el outbox dentro de la transacción dada.
// DEBE ejecutarse en la misma TX que persiste el pago, para que ambos
// commiteen atomicamente.
func InsertOutboxTx(ctx context.Context, tx pgx.Tx, e OutboxEvent) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO outbox_events (aggregate_type, aggregate_id, event_type, payload)
		VALUES ($1, $2, $3, $4)`,
		e.AggregateType, e.AggregateID, e.EventType, e.Payload,
	)
	if err != nil {
		return fmt.Errorf("insert outbox: %w", err)
	}
	return nil
}

// FetchUnpublished devuelve hasta `limit` eventos pendientes de publicar,
// más antiguos primero.
func (r *OutboxRepository) FetchUnpublished(ctx context.Context, limit int) ([]OutboxEvent, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, aggregate_type, aggregate_id, event_type, payload
		FROM outbox_events
		WHERE published_at IS NULL
		ORDER BY created_at
		LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("fetch unpublished: %w", err)
	}
	defer rows.Close()

	var out []OutboxEvent
	for rows.Next() {
		var e OutboxEvent
		if err := rows.Scan(&e.ID, &e.AggregateType, &e.AggregateID, &e.EventType, &e.Payload); err != nil {
			return nil, fmt.Errorf("scan outbox: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// MarkPublished marca un evento como publicado.
func (r *OutboxRepository) MarkPublished(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE outbox_events SET published_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("mark published: %w", err)
	}
	return nil
}

// MarkFailed incrementa el contador de intentos y guarda el último error,
// sin marcar como publicado (el relay reintentará).
func (r *OutboxRepository) MarkFailed(ctx context.Context, id uuid.UUID, cause string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE outbox_events SET attempts = attempts + 1, last_error = $2 WHERE id = $1`,
		id, cause)
	if err != nil {
		return fmt.Errorf("mark failed: %w", err)
	}
	return nil
}
