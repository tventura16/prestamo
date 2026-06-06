-- ───────────────────────────────────────────────
-- P0: Outbox + Idempotencia (payment-service)
--
-- Objetivo: el dinero recibido nunca se pierde ni se duplica, y el estado
-- de la cuota converge de forma confiable aun cuando la actualización en
-- la DB prestamos falle tras registrar el pago.
--
-- Idempotente (IF NOT EXISTS): db/init/ sólo corre en volumen nuevo, pero
-- este archivo puede ejecutarse manualmente sobre una BD existente con psql.
-- ───────────────────────────────────────────────

-- ═══════════════════════════════════════════════
-- DB pagos: idempotencia + outbox
-- ═══════════════════════════════════════════════
\connect pagos

-- Idempotencia de POST /payments. La clave la provee el cliente (header
-- Idempotency-Key). Guardamos la respuesta serializada para replicarla
-- exactamente en reintentos, sin reprocesar el pago.
CREATE TABLE IF NOT EXISTS idempotency_keys (
    key         TEXT        PRIMARY KEY,
    pago_id     UUID        NOT NULL,
    response    JSONB       NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Outbox transaccional: se inserta en la MISMA transacción que el pago,
-- garantizando que "pago registrado" y "evento por publicar" commitean juntos.
CREATE TABLE IF NOT EXISTS outbox_events (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_type TEXT        NOT NULL,             -- 'pago'
    aggregate_id   UUID        NOT NULL,             -- pago_id
    event_type     TEXT        NOT NULL,             -- 'pago.registrado'
    payload        JSONB       NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at   TIMESTAMPTZ,                      -- NULL = pendiente de publicar
    attempts       INTEGER     NOT NULL DEFAULT 0,
    last_error     TEXT
);

-- Índice parcial: el relay sólo escanea lo no publicado.
CREATE INDEX IF NOT EXISTS idx_outbox_unpublished
    ON outbox_events (created_at)
    WHERE published_at IS NULL;

-- ═══════════════════════════════════════════════
-- DB prestamos: guard de idempotencia de aplicación a cuota
-- ═══════════════════════════════════════════════
\connect prestamos

-- Ledger de pagos ya aplicados a una cuota. Hace que aplicar el mismo pago
-- dos veces (fast-path inline + consumer del outbox) sea idempotente: la
-- segunda aplicación detecta el pago_id y hace skip.
CREATE TABLE IF NOT EXISTS pago_aplicaciones (
    pago_id     UUID        PRIMARY KEY,             -- ref. lógica a pagos.pagos
    cuota_id    UUID        NOT NULL,                -- ref. lógica a cuotas
    capital     NUMERIC(14,2) NOT NULL DEFAULT 0,
    interes     NUMERIC(14,2) NOT NULL DEFAULT 0,
    mora        NUMERIC(14,2) NOT NULL DEFAULT 0,
    applied_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pago_aplicaciones_cuota
    ON pago_aplicaciones (cuota_id);
