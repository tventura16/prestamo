// Package outbox contiene el relay que publica los eventos del outbox
// transaccional (DB pagos) hacia Kafka, con garantía at-least-once.
package outbox

import (
	"context"
	"log/slog"
	"time"

	"github.com/prestamos/payment-service/internal/messaging"
	"github.com/prestamos/payment-service/internal/repository"
)

// Relay hace polling del outbox y publica los eventos no publicados.
type Relay struct {
	repo      *repository.OutboxRepository
	pub       *messaging.Publisher
	topic     string
	interval  time.Duration
	batchSize int
	logger    *slog.Logger
}

func NewRelay(repo *repository.OutboxRepository, pub *messaging.Publisher,
	topic string, interval time.Duration, logger *slog.Logger,
) *Relay {
	return &Relay{
		repo:      repo,
		pub:       pub,
		topic:     topic,
		interval:  interval,
		batchSize: 100,
		logger:    logger,
	}
}

// Run bloquea hasta que ctx se cancela. Pensado para ejecutarse en una goroutine.
func (r *Relay) Run(ctx context.Context) {
	t := time.NewTicker(r.interval)
	defer t.Stop()
	r.logger.Info("outbox relay started", "topic", r.topic, "interval", r.interval.String())
	for {
		select {
		case <-ctx.Done():
			r.logger.Info("outbox relay stopped")
			return
		case <-t.C:
			r.drain(ctx)
		}
	}
}

// drain publica todos los eventos pendientes en lotes.
func (r *Relay) drain(ctx context.Context) {
	for {
		batch, err := r.repo.FetchUnpublished(ctx, r.batchSize)
		if err != nil {
			r.logger.Error("outbox fetch failed", "err", err)
			return
		}
		if len(batch) == 0 {
			return
		}
		for _, e := range batch {
			if ctx.Err() != nil {
				return
			}
			// La key es el aggregate_id (pago_id); el evento pago.registrado
			// lleva el cuota_id en el payload para ordenar la aplicación.
			key := []byte(e.AggregateID.String())
			if err := r.pub.Publish(ctx, r.topic, key, e.Payload); err != nil {
				r.logger.Error("outbox publish failed", "event_id", e.ID, "err", err)
				_ = r.repo.MarkFailed(ctx, e.ID, err.Error())
				continue
			}
			if err := r.repo.MarkPublished(ctx, e.ID); err != nil {
				// Publicado pero no marcado: se republicará. El consumer es
				// idempotente, así que el duplicado es inofensivo.
				r.logger.Error("outbox mark published failed", "event_id", e.ID, "err", err)
				continue
			}
			r.logger.Debug("outbox event published", "event_id", e.ID, "type", e.EventType)
		}
		if len(batch) < r.batchSize {
			return
		}
	}
}
