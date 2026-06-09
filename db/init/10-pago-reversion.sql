-- ───────────────────────────────────────────────
-- Reversión de pagos (anulación) — payment-service
-- ───────────────────────────────────────────────
-- Soporte de idempotencia para revertir la aplicación de un pago a su cuota.
-- Las columnas de auditoría de la anulación (anulado/anulado_at/anulado_por)
-- ya existen en pagos.pagos (04-pagos.sql).
--
-- Idempotente (IF NOT EXISTS): puede aplicarse sobre un volumen ya
-- inicializado con `psql`.

-- ── DB pagos: motivo de la anulación (auditoría) ──
\connect pagos

ALTER TABLE pagos
    ADD COLUMN IF NOT EXISTS motivo_anulacion TEXT;

COMMENT ON COLUMN pagos.motivo_anulacion IS
    'Motivo registrado al anular el pago (obligatorio en la anulación).';

-- ── DB prestamos: guard de idempotencia de la reversión ──
\connect prestamos

-- Marca de reversión sobre el ledger de aplicaciones. NULL = aplicación
-- vigente; con valor = el pago fue revertido y no debe revertirse de nuevo
-- (guard de idempotencia, simétrico al guard de aplicación por pago_id).
ALTER TABLE pago_aplicaciones
    ADD COLUMN IF NOT EXISTS reverted_at TIMESTAMPTZ;

COMMENT ON COLUMN pago_aplicaciones.reverted_at IS
    'Fecha de reversión del pago aplicado (anulación). NULL = aplicación vigente.';
