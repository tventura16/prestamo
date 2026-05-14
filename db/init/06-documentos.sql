-- ───────────────────────────────────────────────
-- Esquema: documentos (document-service)
-- ───────────────────────────────────────────────
-- Metadatos de PDFs y documentos generados. Los binarios residen
-- en el volumen Docker "documentstore" (/var/documents).
\connect documentos

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TYPE documento_tipo AS ENUM (
    'contrato',
    'plan_pagos',
    'recibo',
    'estado_cuenta',
    'carta_mora',
    'otro'
);

CREATE TYPE documento_estado AS ENUM (
    'pendiente',
    'generado',
    'enviado',
    'error',
    'archivado'
);

-- ───── Plantillas ─────
CREATE TABLE plantillas (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tipo            documento_tipo NOT NULL,
    nombre          VARCHAR(150)   NOT NULL,
    version         INTEGER        NOT NULL DEFAULT 1,
    contenido       TEXT           NOT NULL,        -- HTML/template
    activo          BOOLEAN        NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    UNIQUE (tipo, version)
);

CREATE INDEX idx_plantillas_tipo ON plantillas (tipo) WHERE activo = TRUE;

-- ───── Documentos generados ─────
CREATE TABLE documentos_generados (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tipo            documento_tipo NOT NULL,
    cliente_id      UUID,                            -- ref. lógica
    prestamo_id     UUID,                            -- ref. lógica
    pago_id         UUID,                            -- ref. lógica (para recibos)
    plantilla_id   UUID REFERENCES plantillas(id),
    nombre_archivo  VARCHAR(255)   NOT NULL,
    ruta            TEXT           NOT NULL,         -- /var/documents/...
    hash_sha256     CHAR(64),
    tamanio_kb      INTEGER,
    estado          documento_estado NOT NULL DEFAULT 'pendiente',
    error_mensaje   TEXT,
    generado_por    UUID,
    generado_at     TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    enviado_at      TIMESTAMPTZ
);

CREATE INDEX idx_docs_tipo       ON documentos_generados (tipo);
CREATE INDEX idx_docs_cliente    ON documentos_generados (cliente_id);
CREATE INDEX idx_docs_prestamo   ON documentos_generados (prestamo_id);
CREATE INDEX idx_docs_pago       ON documentos_generados (pago_id);
CREATE INDEX idx_docs_estado     ON documentos_generados (estado);
CREATE INDEX idx_docs_fecha      ON documentos_generados (generado_at DESC);

-- ───── Trigger para updated_at en plantillas ─────
CREATE OR REPLACE FUNCTION trg_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER plantillas_updated_at
    BEFORE UPDATE ON plantillas
    FOR EACH ROW EXECUTE FUNCTION trg_set_updated_at();

-- ───── Seeds: plantillas iniciales (placeholder) ─────
INSERT INTO plantillas (tipo, nombre, contenido) VALUES
    ('contrato',      'Contrato de Préstamo v1',  '<html><!-- TODO: plantilla --></html>'),
    ('plan_pagos',    'Plan de Pagos v1',         '<html><!-- TODO: plantilla --></html>'),
    ('recibo',        'Recibo de Pago v1',        '<html><!-- TODO: plantilla --></html>'),
    ('estado_cuenta', 'Estado de Cuenta v1',      '<html><!-- TODO: plantilla --></html>');
