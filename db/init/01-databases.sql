-- ───────────────────────────────────────────────
-- Creación de bases de datos (Database per Service)
-- ───────────────────────────────────────────────
-- Cada microservicio tiene su propia DB. No hay FKs entre DBs:
-- las referencias entre servicios se hacen por ID lógico.
-- La base "prestamos" ya existe (POSTGRES_DB).

CREATE DATABASE keycloak;
CREATE DATABASE clientes;
CREATE DATABASE pagos;
CREATE DATABASE reportes;
CREATE DATABASE documentos;

\echo 'Bases de datos creadas.'
