-- ───────────────────────────────────────────────
-- Garantías del préstamo (loan-service)
-- ───────────────────────────────────────────────
-- Tipo de garantía del préstamo + adjuntos de imágenes (escaneos/fotos del
-- bien o documento de respaldo). Idempotente: aplicable con `psql` sobre un
-- volumen ya inicializado.
\connect prestamos

-- Tipo de garantía. NULL = préstamo sin garantía.
ALTER TABLE prestamos
    ADD COLUMN IF NOT EXISTS tipo_garantia VARCHAR(20)
        CHECK (tipo_garantia IS NULL OR tipo_garantia IN ('garante', 'prendaria', 'hipotecaria'));

COMMENT ON COLUMN prestamos.tipo_garantia IS
    'Tipo de garantía: garante | prendaria | hipotecaria. NULL = sin garantía.';

-- Imágenes/adjuntos de la garantía. El binario vive en el volumen del
-- loan-service; aquí solo los metadatos.
CREATE TABLE IF NOT EXISTS prestamo_garantias (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    prestamo_id    UUID         NOT NULL REFERENCES prestamos(id) ON DELETE CASCADE,
    nombre_archivo VARCHAR(255) NOT NULL,
    ruta           TEXT         NOT NULL,
    mime           VARCHAR(100) NOT NULL,
    tamanio_bytes  BIGINT       NOT NULL CHECK (tamanio_bytes > 0),
    descripcion    TEXT,
    subido_por     UUID,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_prestamo_garantias_prestamo
    ON prestamo_garantias (prestamo_id);
