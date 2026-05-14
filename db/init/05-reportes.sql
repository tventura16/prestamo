-- ───────────────────────────────────────────────
-- Esquema: reportes (report-service)
-- ───────────────────────────────────────────────
-- Almacena snapshots agregados y auditoría. Los datos crudos viven
-- en cada microservicio; aquí se persisten consultas costosas y eventos.
\connect reportes

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ───── Auditoría de acciones de usuario ─────
CREATE TABLE audit_log (
    id              BIGSERIAL PRIMARY KEY,
    timestamp       TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    usuario_id      UUID,
    usuario_email   VARCHAR(150),
    servicio        VARCHAR(50)     NOT NULL,
    accion          VARCHAR(100)    NOT NULL,
    recurso_tipo    VARCHAR(50),
    recurso_id      UUID,
    ip_address      INET,
    user_agent      TEXT,
    payload         JSONB,
    request_id      VARCHAR(100)
);

CREATE INDEX idx_audit_timestamp     ON audit_log (timestamp DESC);
CREATE INDEX idx_audit_usuario       ON audit_log (usuario_id);
CREATE INDEX idx_audit_servicio      ON audit_log (servicio);
CREATE INDEX idx_audit_recurso       ON audit_log (recurso_tipo, recurso_id);
CREATE INDEX idx_audit_request_id    ON audit_log (request_id);

-- ───── Snapshots diarios (rollup) ─────
CREATE TABLE reporte_diario (
    fecha               DATE PRIMARY KEY,
    ingresos            NUMERIC(14,2) NOT NULL DEFAULT 0,
    pagos_recibidos     INTEGER       NOT NULL DEFAULT 0,
    mora_cobrada        NUMERIC(14,2) NOT NULL DEFAULT 0,
    prestamos_nuevos    INTEGER       NOT NULL DEFAULT 0,
    prestamos_aprobados INTEGER       NOT NULL DEFAULT 0,
    cuotas_vencidas     INTEGER       NOT NULL DEFAULT 0,
    clientes_nuevos     INTEGER       NOT NULL DEFAULT 0,
    actualizado_at      TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

-- ───── Snapshots mensuales (rollup) ─────
CREATE TABLE reporte_mensual (
    anio                INTEGER NOT NULL,
    mes                 INTEGER NOT NULL CHECK (mes BETWEEN 1 AND 12),
    ingresos            NUMERIC(14,2) NOT NULL DEFAULT 0,
    intereses_generados NUMERIC(14,2) NOT NULL DEFAULT 0,
    mora_cobrada        NUMERIC(14,2) NOT NULL DEFAULT 0,
    prestamos_otorgados INTEGER       NOT NULL DEFAULT 0,
    clientes_nuevos     INTEGER       NOT NULL DEFAULT 0,
    actualizado_at      TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    PRIMARY KEY (anio, mes)
);

-- ───── Cache de reportes generados ─────
CREATE TABLE reportes_generados (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tipo            VARCHAR(50)   NOT NULL,
    parametros      JSONB,
    formato         VARCHAR(10)   NOT NULL,    -- 'pdf' | 'xlsx' | 'csv'
    url             TEXT,
    generado_por    UUID,
    generado_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    expira_at       TIMESTAMPTZ
);

CREATE INDEX idx_reportes_tipo ON reportes_generados (tipo, generado_at DESC);
CREATE INDEX idx_reportes_user ON reportes_generados (generado_por);
