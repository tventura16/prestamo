-- ───────────────────────────────────────────────
-- Esquema: clientes (client-service)
-- ───────────────────────────────────────────────
\connect clientes

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Estados del cliente
CREATE TYPE cliente_estado AS ENUM (
    'activo',
    'inactivo',
    'bloqueado'
);

-- ───── Tabla principal de clientes ─────
CREATE TABLE clientes (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nombres             VARCHAR(100)  NOT NULL,
    apellidos           VARCHAR(100)  NOT NULL,
    ci                  VARCHAR(20)   NOT NULL UNIQUE,
    fecha_nacimiento    DATE          NOT NULL,
    telefono            VARCHAR(20),
    direccion           TEXT,
    email               VARCHAR(150),
    estado              cliente_estado NOT NULL DEFAULT 'activo',
    foto_url            TEXT,
    created_at          TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_clientes_ci         ON clientes (ci);
CREATE INDEX idx_clientes_apellidos  ON clientes (apellidos);
CREATE INDEX idx_clientes_estado     ON clientes (estado);

-- ───── Referencias personales ─────
CREATE TABLE referencias_personales (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cliente_id   UUID         NOT NULL REFERENCES clientes(id) ON DELETE CASCADE,
    nombre       VARCHAR(200) NOT NULL,
    telefono     VARCHAR(20),
    parentesco   VARCHAR(50),
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_referencias_cliente ON referencias_personales (cliente_id);

-- ───── Documentos adjuntos ─────
CREATE TABLE documentos_cliente (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cliente_id   UUID         NOT NULL REFERENCES clientes(id) ON DELETE CASCADE,
    tipo         VARCHAR(50)  NOT NULL,
    nombre       VARCHAR(255) NOT NULL,
    url          TEXT         NOT NULL,
    mime_type    VARCHAR(100),
    tamanio_kb   INTEGER,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_docs_cliente ON documentos_cliente (cliente_id);

-- ───── Trigger para mantener updated_at ─────
CREATE OR REPLACE FUNCTION trg_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER clientes_updated_at
    BEFORE UPDATE ON clientes
    FOR EACH ROW EXECUTE FUNCTION trg_set_updated_at();
