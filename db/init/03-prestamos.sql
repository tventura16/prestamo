-- ───────────────────────────────────────────────
-- Esquema: prestamos (loan-service)
-- ───────────────────────────────────────────────
\connect prestamos

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Enumeraciones
CREATE TYPE prestamo_estado AS ENUM (
    'pendiente',
    'aprobado',
    'rechazado',
    'activo',
    'finalizado',
    'mora'
);

CREATE TYPE prestamo_tipo_interes AS ENUM (
    'fijo',
    'variable'
);

CREATE TYPE prestamo_frecuencia AS ENUM (
    'diaria',
    'semanal',
    'quincenal',
    'mensual'
);

CREATE TYPE cuota_estado AS ENUM (
    'pendiente',
    'pagada',
    'parcial',
    'vencida'
);

-- ───── Préstamos ─────
CREATE TABLE prestamos (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cliente_id          UUID            NOT NULL,           -- ref. lógica a clientes
    monto_solicitado    NUMERIC(14,2)   NOT NULL CHECK (monto_solicitado > 0),
    monto_aprobado      NUMERIC(14,2),
    tasa_interes        NUMERIC(6,4)    NOT NULL CHECK (tasa_interes >= 0),
    tipo_interes        prestamo_tipo_interes NOT NULL DEFAULT 'fijo',
    fecha_solicitud     DATE            NOT NULL DEFAULT CURRENT_DATE,
    fecha_desembolso    DATE,
    num_cuotas          INTEGER         NOT NULL CHECK (num_cuotas > 0),
    frecuencia          prestamo_frecuencia NOT NULL,
    estado              prestamo_estado NOT NULL DEFAULT 'pendiente',
    aprobado_por        UUID,
    observaciones       TEXT,
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_prestamos_cliente   ON prestamos (cliente_id);
CREATE INDEX idx_prestamos_estado    ON prestamos (estado);
CREATE INDEX idx_prestamos_fecha     ON prestamos (fecha_desembolso);

-- ───── Cuotas (plan de pagos) ─────
CREATE TABLE cuotas (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    prestamo_id         UUID            NOT NULL REFERENCES prestamos(id) ON DELETE CASCADE,
    numero              INTEGER         NOT NULL,
    fecha_vencimiento   DATE            NOT NULL,
    capital             NUMERIC(14,2)   NOT NULL CHECK (capital >= 0),
    interes             NUMERIC(14,2)   NOT NULL CHECK (interes >= 0),
    total               NUMERIC(14,2)   NOT NULL,
    saldo_pendiente     NUMERIC(14,2)   NOT NULL,
    mora_acumulada      NUMERIC(14,2)   NOT NULL DEFAULT 0,
    estado              cuota_estado    NOT NULL DEFAULT 'pendiente',
    fecha_pago          TIMESTAMPTZ,
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    UNIQUE (prestamo_id, numero)
);

CREATE INDEX idx_cuotas_prestamo     ON cuotas (prestamo_id);
CREATE INDEX idx_cuotas_estado       ON cuotas (estado);
CREATE INDEX idx_cuotas_vencimiento  ON cuotas (fecha_vencimiento);
CREATE INDEX idx_cuotas_vencidas     ON cuotas (estado, fecha_vencimiento)
    WHERE estado IN ('pendiente', 'parcial');

-- ───── Trigger para updated_at ─────
CREATE OR REPLACE FUNCTION trg_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER prestamos_updated_at
    BEFORE UPDATE ON prestamos
    FOR EACH ROW EXECUTE FUNCTION trg_set_updated_at();

CREATE TRIGGER cuotas_updated_at
    BEFORE UPDATE ON cuotas
    FOR EACH ROW EXECUTE FUNCTION trg_set_updated_at();
