// Package consumer aplica los eventos pago.registrado a la DB prestamos de
// forma confiable e idempotente. Es la red de seguridad cuando el fast-path
// inline no pudo commitear la actualización de la cuota.
package consumer

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/prestamos/payment-service/internal/events"
	"github.com/prestamos/payment-service/internal/messaging"
	"github.com/prestamos/payment-service/internal/repository"
)

type PagoConsumer struct {
	reader     *kafka.Reader
	pub        *messaging.Publisher
	loanRepo   *repository.LoanRepository
	dlqTopic   string
	maxRetries int
	logger     *slog.Logger
}

func NewPagoConsumer(reader *kafka.Reader, pub *messaging.Publisher,
	loanRepo *repository.LoanRepository, dlqTopic string, maxRetries int, logger *slog.Logger,
) *PagoConsumer {
	return &PagoConsumer{
		reader:     reader,
		pub:        pub,
		loanRepo:   loanRepo,
		dlqTopic:   dlqTopic,
		maxRetries: maxRetries,
		logger:     logger,
	}
}

// Run bloquea hasta que ctx se cancela. Pensado para una goroutine.
func (c *PagoConsumer) Run(ctx context.Context) {
	c.logger.Info("pago consumer started", "topic", c.reader.Config().Topic)
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, context.Canceled) {
				c.logger.Info("pago consumer stopped")
				return
			}
			c.logger.Error("fetch message failed", "err", err)
			continue
		}

		if err := c.handle(ctx, msg); err != nil {
			// Agotados los reintentos: a la DLQ y avanzar el offset para no
			// bloquear la partición.
			c.logger.Error("evento enviado a DLQ", "err", err, "key", string(msg.Key))
			if derr := c.pub.Publish(ctx, c.dlqTopic, msg.Key, msg.Value); derr != nil {
				c.logger.Error("publish DLQ failed", "err", derr)
			}
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			c.logger.Error("commit offset failed", "err", err)
		}
	}
}

// handle procesa un mensaje con reintentos in-process y backoff.
func (c *PagoConsumer) handle(ctx context.Context, msg kafka.Message) error {
	evt, err := events.UnmarshalPagoRegistrado(msg.Value)
	if err != nil {
		// Mensaje irrecuperable (payload corrupto) → directo a DLQ.
		return err
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt) * 200 * time.Millisecond):
			}
		}
		if err := c.apply(ctx, evt); err != nil {
			lastErr = err
			c.logger.Warn("apply event failed, retrying", "attempt", attempt, "pago_id", evt.PagoID, "err", err)
			continue
		}
		c.logger.Debug("event applied", "pago_id", evt.PagoID, "cuota_id", evt.CuotaID)
		return nil
	}
	return lastErr
}

// apply ejecuta la aplicación idempotente dentro de una transacción.
func (c *PagoConsumer) apply(ctx context.Context, evt events.PagoRegistrado) error {
	tx, err := c.loanRepo.Pool().Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	res, err := c.loanRepo.ApplyPagoToCuota(ctx, tx, repository.PagoAplicacion{
		PagoID:  evt.PagoID,
		CuotaID: evt.CuotaID,
		Capital: evt.Capital,
		Interes: evt.Interes,
		Mora:    evt.Mora,
	})
	if err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	if res.Skipped {
		c.logger.Debug("event already applied (idempotent skip)", "pago_id", evt.PagoID)
	}
	return nil
}
