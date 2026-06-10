-- ───────────────────────────────────────────────
-- Garantías clasificadas con datos por subtipo (loan-service)
-- ───────────────────────────────────────────────
-- Reemplaza el modelo simple anterior (prestamos.tipo_garantia +
-- prestamo_garantias) por una entidad `garantias` (1..N por préstamo) con
-- datos específicos por subtipo en JSONB, y las imágenes colgando de la
-- garantía. Idempotente: aplicable con `psql` sobre un volumen existente.
\connect prestamos

-- ── Limpieza del modelo anterior (datos de prueba; ver ADR/historial) ──
DROP TABLE IF EXISTS prestamo_garantias;
ALTER TABLE prestamos DROP COLUMN IF EXISTS tipo_garantia;

-- ── Entidad garantía ──
CREATE TABLE IF NOT EXISTS garantias (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    prestamo_id        UUID         NOT NULL REFERENCES prestamos(id) ON DELETE CASCADE,
    subtipo            VARCHAR(20)  NOT NULL
                         CHECK (subtipo IN ('vehiculo', 'inmueble', 'garante', 'mueble')),
    descripcion        TEXT,
    valor_estimado     NUMERIC(14,2) CHECK (valor_estimado IS NULL OR valor_estimado >= 0),
    moneda             VARCHAR(3)   NOT NULL DEFAULT 'BOB',
    cliente_garante_id UUID,                                  -- garante vinculado a un cliente (opcional)
    datos              JSONB        NOT NULL DEFAULT '{}',     -- campos propios del subtipo
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_garantias_prestamo ON garantias (prestamo_id);

-- ── Imágenes de la garantía ──
CREATE TABLE IF NOT EXISTS garantia_imagenes (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    garantia_id    UUID         NOT NULL REFERENCES garantias(id) ON DELETE CASCADE,
    nombre_archivo VARCHAR(255) NOT NULL,
    ruta           TEXT         NOT NULL,
    mime           VARCHAR(100) NOT NULL,
    tamanio_bytes  BIGINT       NOT NULL CHECK (tamanio_bytes > 0),
    descripcion    TEXT,
    subido_por     UUID,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_garantia_imagenes_garantia ON garantia_imagenes (garantia_id);

-- Trigger de updated_at (reutiliza la función trg_set_updated_at de 03-prestamos.sql).
DROP TRIGGER IF EXISTS garantias_updated_at ON garantias;
CREATE TRIGGER garantias_updated_at
    BEFORE UPDATE ON garantias
    FOR EACH ROW EXECUTE FUNCTION trg_set_updated_at();
