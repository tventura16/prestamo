-- ───────────────────────────────────────────────
-- Esquema: prestamos (loan-service) — control de devengo de mora
-- ───────────────────────────────────────────────
-- Soporte para el job diario de mora del loan-service. Idempotente: puede
-- aplicarse sobre un volumen ya inicializado con `psql`.
\connect prestamos

-- Fecha hasta la cual el job de mora ya devengó interés moratorio sobre la
-- cuota. Permite que el devengo sea incremental e idempotente: cada corrida
-- suma solo los días aún no aplicados. NULL = nunca devengada.
ALTER TABLE cuotas
    ADD COLUMN IF NOT EXISTS mora_aplicada_hasta DATE;

COMMENT ON COLUMN cuotas.mora_aplicada_hasta IS
    'Fecha hasta la cual el job de mora devengó interés moratorio. NULL = pendiente.';

-- El job localiza cuotas vivas con saldo por las que aún puede correr mora.
CREATE INDEX IF NOT EXISTS idx_cuotas_mora_devengo
    ON cuotas (fecha_vencimiento)
    WHERE estado IN ('pendiente', 'parcial', 'vencida');
