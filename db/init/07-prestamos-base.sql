-- ───────────────────────────────────────────────
-- Base de datos "prestamos" (POSTGRES_DB por defecto)
-- ───────────────────────────────────────────────
-- Usada como base operativa compartida para tablas transversales:
-- catálogos generales, parámetros de sistema, etc.
\connect prestamos

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ───── Parámetros del sistema ─────
CREATE TABLE parametros_sistema (
    clave       VARCHAR(100) PRIMARY KEY,
    valor       TEXT          NOT NULL,
    descripcion TEXT,
    updated_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_by  UUID
);

INSERT INTO parametros_sistema (clave, valor, descripcion) VALUES
    ('moneda',                   'BOB',  'Moneda base del sistema'),
    ('tasa_mora_diaria',         '0.05', 'Porcentaje diario de mora sobre saldo vencido'),
    ('dias_gracia_mora',         '1',    'Días de gracia antes de aplicar mora'),
    ('max_prestamos_activos',    '3',    'Máximo de préstamos activos por cliente'),
    ('aprobar_si_mora_activa',   'false','Permite aprobar préstamo si cliente tiene mora');

-- ───── Catálogo de métodos de pago habilitados ─────
CREATE TABLE metodos_pago (
    codigo      VARCHAR(20) PRIMARY KEY,
    nombre      VARCHAR(100) NOT NULL,
    activo      BOOLEAN      NOT NULL DEFAULT TRUE,
    orden       INTEGER      NOT NULL DEFAULT 0
);

INSERT INTO metodos_pago (codigo, nombre, orden) VALUES
    ('efectivo',       'Efectivo',         1),
    ('transferencia',  'Transferencia',    2),
    ('cheque',         'Cheque',           3),
    ('tarjeta',        'Tarjeta',          4),
    ('qr',             'Pago QR',          5);
