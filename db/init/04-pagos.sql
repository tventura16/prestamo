-- ───────────────────────────────────────────────
-- Esquema: pagos (payment-service)
-- ───────────────────────────────────────────────
\connect pagos

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TYPE pago_metodo AS ENUM (
    'efectivo',
    'transferencia',
    'cheque',
    'tarjeta',
    'qr'
);

CREATE TYPE pago_tipo AS ENUM (
    'total',
    'parcial'
);

-- ───── Pagos ─────
CREATE TABLE pagos (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cliente_id      UUID            NOT NULL,             -- ref. lógica a clientes
    prestamo_id     UUID            NOT NULL,             -- ref. lógica a prestamos
    cuota_id        UUID,                                 -- ref. lógica a cuotas
    fecha_pago      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    monto_pagado    NUMERIC(14,2)   NOT NULL CHECK (monto_pagado > 0),
    capital_pagado  NUMERIC(14,2)   NOT NULL DEFAULT 0,
    interes_pagado  NUMERIC(14,2)   NOT NULL DEFAULT 0,
    mora_pagada     NUMERIC(14,2)   NOT NULL DEFAULT 0,
    tipo            pago_tipo       NOT NULL DEFAULT 'total',
    metodo_pago     pago_metodo     NOT NULL,
    usuario_id      UUID            NOT NULL,             -- operador que registra
    numero_recibo   VARCHAR(50)     UNIQUE,
    observaciones   TEXT,
    anulado         BOOLEAN         NOT NULL DEFAULT FALSE,
    anulado_at      TIMESTAMPTZ,
    anulado_por     UUID,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_pagos_cliente       ON pagos (cliente_id);
CREATE INDEX idx_pagos_prestamo      ON pagos (prestamo_id);
CREATE INDEX idx_pagos_cuota         ON pagos (cuota_id);
CREATE INDEX idx_pagos_fecha         ON pagos (fecha_pago);
CREATE INDEX idx_pagos_usuario       ON pagos (usuario_id);
CREATE INDEX idx_pagos_recibo        ON pagos (numero_recibo);

-- ───── Secuencia para numeración de recibos ─────
CREATE SEQUENCE seq_recibo START 1000;

-- ───── Movimientos / desglose contable ─────
CREATE TABLE movimientos_pago (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pago_id         UUID            NOT NULL REFERENCES pagos(id) ON DELETE CASCADE,
    concepto        VARCHAR(50)     NOT NULL,    -- 'capital' | 'interes' | 'mora'
    monto           NUMERIC(14,2)   NOT NULL,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_mov_pago ON movimientos_pago (pago_id);
