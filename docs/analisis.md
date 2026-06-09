# Sistema de Préstamos Empresarial
# Documento de Requerimientos y Arquitectura

---

# 1. Descripción General

El sistema de préstamos permitirá administrar clientes, préstamos, planes de pago, pagos, generación de documentos y reportes financieros.

El objetivo principal es controlar todo el ciclo del préstamo desde el registro del cliente hasta el cierre total de la deuda.

---

# 2. Objetivos del Sistema

- Registrar y administrar clientes.
- Registrar solicitudes de préstamos.
- Aprobar o rechazar préstamos.
- Generar automáticamente planes de pago.
- Registrar pagos de clientes.
- Generar contratos y documentos PDF.
- Obtener reportes financieros y operativos.
- Controlar préstamos en mora.
- Administrar usuarios y roles.

---

# 3. Módulos del Sistema

---

## 3.1 Módulo de Clientes

### Funcionalidades

- Registrar clientes.
- Editar clientes.
- Buscar clientes.
- Consultar historial crediticio.
- Adjuntar documentos.

### Datos del Cliente

- nombres
- apellidos
- CI / documento identidad
- fecha nacimiento
- teléfono
- dirección
- correo electrónico
- referencias personales
- estado del cliente
- fotografía
- documentos adjuntos

---

## 3.2 Módulo de Préstamos

### Funcionalidades

- Registrar solicitudes.
- Aprobar préstamos.
- Rechazar préstamos.
- Configurar monto, interés y plazo.
- Generar plan de pagos.
- Consultar estado del préstamo.

### Datos del Préstamo

- cliente
- monto solicitado
- tasa interés
- tipo interés
- fecha desembolso
- número cuotas
- frecuencia pagos:
  - diaria
  - semanal
  - quincenal
  - mensual

### Estados

- pendiente
- aprobado
- rechazado
- activo
- finalizado
- mora

---

## 3.3 Módulo de Plan de Pagos

### Funcionalidades

- Generar cuotas automáticamente.
- Calcular capital e interés.
- Mostrar saldo pendiente.
- Calcular mora automática.
- Controlar vencimientos.

### Datos de Cuotas

- número cuota
- fecha vencimiento
- capital
- interés
- total cuota
- saldo pendiente
- estado cuota
- fecha pago

---

## 3.4 Módulo de Pagos

### Funcionalidades

- Registrar pagos.
- Registrar pagos parciales.
- Registrar pagos totales.
- Aplicar pagos a cuotas.
- Calcular mora.
- Generar recibos.

### Datos del Pago

- cliente
- préstamo
- cuota
- fecha pago
- monto pagado
- mora pagada
- método pago
- usuario operador

---

## 3.5 Módulo de Documentos

### Funcionalidades

- Generar contratos.
- Generar recibos.
- Generar estado cuenta.
- Exportar PDF.
- Imprimir documentos.

### Documentos

- contrato préstamo
- plan pagos
- recibos
- estado cuenta

---

## 3.6 Módulo de Reportes

### Reportes Financieros

#### Diario

- ingresos diarios
- pagos recibidos
- mora cobrada

#### Mensual

- ingresos mensuales
- intereses generados
- préstamos otorgados
- clientes nuevos

### Reportes Operativos

- pagos por cliente
- préstamos activos
- clientes morosos
- cuotas vencidas
- préstamos finalizados

### Exportación

- PDF
- Excel

---

# 4. Roles del Sistema

## Administrador

### Permisos

- acceso total
- gestión usuarios
- configuración sistema
- acceso reportes

---

## Supervisor

### Permisos

- aprobar préstamos
- revisar reportes
- monitorear mora

---

## Cajero / Operador

### Permisos

- registrar clientes
- registrar pagos
- consultar préstamos

---

# 5. Requerimientos Funcionales

| Código | Requerimiento |
|---|---|
| RF-001 | Registrar clientes |
| RF-002 | Editar clientes |
| RF-003 | Buscar clientes |
| RF-004 | Registrar préstamos |
| RF-005 | Aprobar préstamos |
| RF-006 | Generar planes de pago |
| RF-007 | Calcular intereses |
| RF-008 | Calcular mora automática |
| RF-009 | Registrar pagos |
| RF-010 | Registrar pagos parciales |
| RF-011 | Generar contratos PDF |
| RF-012 | Generar recibos |
| RF-013 | Generar reportes diarios |
| RF-014 | Generar reportes mensuales |
| RF-015 | Generar reportes por cliente |
| RF-016 | Exportar reportes |
| RF-017 | Administrar usuarios |
| RF-018 | Gestionar roles y permisos |

---

# 6. Requerimientos No Funcionales

## Seguridad

- autenticación JWT
- control roles
- cifrado contraseñas
- auditoría acciones usuarios

---

## Rendimiento

- soporte múltiples usuarios
- consultas optimizadas

---

## Disponibilidad

- respaldo base datos
- recuperación fallos

---

## Escalabilidad

- arquitectura escalable
- soporte crecimiento futuro

---

## Usabilidad

- interfaz **responsive (mobile-first)** construida con **Tailwind CSS v4**
- navegación adaptable: menú colapsable (hamburguesa) en móvil y barra horizontal en escritorio
- tablas con desplazamiento horizontal en pantallas pequeñas; formularios y grids que se apilan en móvil
- estados de carga, error y vacío en todas las pantallas

---

# 7. Reglas de Negocio

- un cliente puede tener múltiples préstamos
- no se puede aprobar préstamo con mora activa
- los pagos primero cubren mora e interés
- cuotas vencidas deben marcarse automáticamente
- préstamo se cierra cuando todas las cuotas estén pagadas

---

# 8. Arquitectura Tecnológica Empresarial

| Capa | Tecnología |
|---|---|
| Frontend | Angular 20 + Tailwind CSS v4 (UI responsive) |
| Arquitectura Backend | Microservicios |
| Backend | Go 1.26 + Gin |
| Seguridad | Keycloak 26 |
| API Gateway | Kong 3.10 |
| Service Discovery | Consul 1.21 |
| Base de Datos | PostgreSQL 18 |
| Cache | Redis 8 |
| Documentación API | Swagger/OpenAPI 3.1 |
| Infraestructura | Docker 28 |
| Mensajería | Kafka 4.0 / RabbitMQ 4.1 |
| Monitoring | Grafana 12 + Prometheus 3 |

---

# 9. Arquitectura General del Sistema

```plaintext
                    ┌──────────────────┐
                    │     Frontend     │
                    │    Angular 20    │
                    └────────┬─────────┘
                             │
                             ▼
                  ┌────────────────────┐
                  │     API Gateway    │
                  │        Kong        │
                  └────────┬───────────┘
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
        ▼                  ▼                  ▼

┌────────────────┐ ┌────────────────┐ ┌────────────────┐
│ Auth Service   │ │ Client Service │ │ Loan Service   │
│ Keycloak       │ │ Clientes       │ │ Préstamos      │
└────────────────┘ └────────────────┘ └────────────────┘

        ▼                  ▼                  ▼

┌────────────────┐ ┌────────────────┐ ┌────────────────┐
│ Payment Service│ │ Report Service │ │ Document Service│
│ Pagos          │ │ Reportes       │ │ PDFs/Contratos │
└────────────────┘ └────────────────┘ └────────────────┘

                           │
                           ▼

                 ┌──────────────────┐
                 │      Consul      │
                 │ Service Discovery│
                 └──────────────────┘
```

---

## 9.1 Microservicios y Responsabilidades

### Auth Service
- Autenticación y autorización de usuarios.
- Gestión de tokens JWT mediante Keycloak.
- Validación de roles y permisos.
- Single Sign-On (SSO).
- **Base de datos:** PostgreSQL (esquema dedicado de Keycloak).

### Client Service
- Registro, edición y búsqueda de clientes.
- Gestión de documentos adjuntos y fotografías.
- Consulta de historial crediticio.
- Validación de datos de identificación (CI).
- **Base de datos:** PostgreSQL (esquema `clientes`).

### Loan Service
- Registro de solicitudes de préstamos.
- Aprobación y rechazo de préstamos.
- Cálculo de intereses y plazos.
- Generación automática del plan de pagos.
- Control de estados del préstamo.
- **Base de datos:** PostgreSQL (esquema `prestamos`).

### Payment Service
- Registro de pagos totales y parciales.
- Aplicación de pagos a cuotas según reglas de negocio.
- Cálculo automático de mora.
- Generación de recibos electrónicos.
- Actualización de saldos pendientes.
- **Base de datos:** PostgreSQL (esquema `pagos`).

### Report Service
- Generación de reportes financieros (diarios, mensuales).
- Reportes operativos (mora, cuotas vencidas, préstamos activos).
- Exportación a PDF y Excel.
- Consultas optimizadas con cache en Redis.
- **Base de datos:** PostgreSQL (réplica de lectura) + Redis.

### Document Service
- Generación de contratos en PDF.
- Generación de estados de cuenta.
- Plantillas de documentos parametrizables.
- Almacenamiento de documentos generados.
- **Base de datos:** PostgreSQL (metadatos) + almacenamiento de archivos.

---

## 9.2 Patrones de Comunicación

### Comunicación Síncrona (REST/HTTP)
- Frontend → API Gateway (Kong) → Microservicios.
- Comunicación entre servicios para consultas inmediatas.
- Documentada con Swagger/OpenAPI 3.1.

### Comunicación Asíncrona (Kafka / RabbitMQ)
- **Eventos principales:**
  - `loan.approved` → notifica a Document Service para generar contrato.
  - `payment.registered` → notifica a Loan Service para actualizar cuotas.
  - `payment.registered` → notifica a Report Service para actualizar métricas.
  - `loan.overdue` → notifica a clientes y supervisores.
  - `client.created` → notifica auditoría y reportería.
- Garantiza desacoplamiento y resiliencia entre servicios.

### Diagrama de Eventos Asíncronos

```plaintext
   ┌──────────────────┐                              ┌──────────────────┐
   │  Client Service  │                              │   Loan Service   │
   └────────┬─────────┘                              └────────┬─────────┘
            │                                                 │
            │ client.created                  loan.approved   │
            │                                  loan.overdue   │
            ▼                                                 ▼
   ╔════════════════════════════════════════════════════════════════════╗
   ║                                                                    ║
   ║              EVENT BUS  (Kafka 4.0 / RabbitMQ 4.1)                 ║
   ║                                                                    ║
   ║   Topics:                                                          ║
   ║     • client.created                                               ║
   ║     • loan.approved                                                ║
   ║     • loan.overdue                                                 ║
   ║     • payment.registered                                           ║
   ║     • document.generated                                           ║
   ║                                                                    ║
   ╚═══╤══════════════╤═════════════╤═════════════╤═════════════╤══════╝
       │              │             │             │             │
       │              │             │             │             │
       ▼              ▼             ▼             ▼             ▼
  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐
  │ Document │  │  Report  │  │  Loan    │  │  Audit   │  │  Notif.  │
  │ Service  │  │ Service  │  │ Service  │  │ (logs)   │  │ (email)  │
  └──────────┘  └──────────┘  └──────────┘  └──────────┘  └──────────┘
       ▲
       │ payment.registered
       │
  ┌────┴───────────┐
  │ Payment Service│
  └────────────────┘


  Flujos principales:

  [1] client.created
      Client Service  ──▶  Event Bus  ──▶  Report Service, Audit

  [2] loan.approved
      Loan Service    ──▶  Event Bus  ──▶  Document Service (genera contrato)
                                       ──▶  Notification Service (avisa cliente)
                                       ──▶  Report Service

  [3] payment.registered
      Payment Service ──▶  Event Bus  ──▶  Loan Service (actualiza cuotas/saldo)
                                       ──▶  Report Service (métricas diarias)
                                       ──▶  Document Service (genera recibo)

  [4] loan.overdue   (job programado en Loan Service)
      Loan Service    ──▶  Event Bus  ──▶  Notification Service (cliente)
                                       ──▶  Report Service (mora)
                                       ──▶  Audit
```

---

## 9.3 Persistencia y Cache

- **PostgreSQL 18:** una base de datos por microservicio (Database per Service).
- **Redis 8:** cache de consultas frecuentes, sesiones y locks distribuidos.
- **Almacenamiento de archivos:** documentos PDF y adjuntos.

---

## 9.4 Infraestructura Transversal

| Componente | Función |
|---|---|
| Kong | API Gateway: enrutamiento, rate limiting, autenticación |
| Consul | Service Discovery y health checks |
| Keycloak | Identity Provider centralizado |
| Kafka / RabbitMQ | Bus de eventos asíncronos |
| Prometheus | Recolección de métricas |
| Grafana | Visualización y dashboards |
| Docker | Containerización de cada microservicio |

---

## 9.5 Principios de Diseño

- **Database per Service:** cada microservicio gestiona su propia persistencia.
- **API First:** contratos definidos antes de la implementación.
- **Stateless Services:** los servicios no mantienen estado local.
- **Circuit Breaker:** tolerancia a fallos en llamadas entre servicios.
- **Observability:** logs estructurados, métricas y trazas distribuidas.
- **CI/CD independiente:** cada servicio se despliega de forma autónoma.

---

# 10. Arquitectura de Despliegue con Docker

Cada microservicio se empaqueta como una imagen Docker independiente y se orquesta mediante Docker Compose tanto en entornos de desarrollo como en producción.

## 10.1 Diagrama de Despliegue

```plaintext
┌─────────────────────────────────────────────────────────────────────────┐
│                         HOST / CLUSTER  (Docker 28)                     │
│                                                                         │
│  ┌───────────────────────────────────────────────────────────────────┐  │
│  │                       Red:  frontend-net                          │  │
│  │  ┌──────────────────────────┐    ┌──────────────────────────┐     │  │
│  │  │  container: frontend     │    │  container: api-gateway  │     │  │
│  │  │  image: angular-20:nginx │───▶│  image: kong:3.10        │     │  │
│  │  │  port: 80/443            │    │  port: 8000/8443         │     │  │
│  │  └──────────────────────────┘    └────────────┬─────────────┘     │  │
│  └───────────────────────────────────────────────┼──────────────────┘   │
│                                                  │                      │
│  ┌───────────────────────────────────────────────┼──────────────────┐   │
│  │                       Red:  backend-net       ▼                  │   │
│  │                                                                  │   │
│  │  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐│   │
│  │  │ auth-service     │  │ client-service   │  │ loan-service     ││   │
│  │  │ image: keycloak  │  │ image: go-1.26   │  │ image: go-1.26   ││   │
│  │  │       :26        │  │ port: 8081       │  │ port: 8082       ││   │
│  │  │ port: 8080       │  │                  │  │                  ││   │
│  │  └────────┬─────────┘  └────────┬─────────┘  └────────┬─────────┘│   │
│  │           │                     │                     │          │   │
│  │  ┌────────┴─────────┐  ┌────────┴─────────┐  ┌────────┴─────────┐│   │
│  │  │ payment-service  │  │ report-service   │  │ document-service ││   │
│  │  │ image: go-1.26   │  │ image: go-1.26   │  │ image: go-1.26   ││   │
│  │  │ port: 8083       │  │ port: 8084       │  │ port: 8085       ││   │
│  │  └──────────────────┘  └──────────────────┘  └──────────────────┘│   │
│  │                                                                  │   │
│  │  ┌──────────────────┐  ┌──────────────────┐                      │   │
│  │  │ consul           │  │ kafka            │                      │   │
│  │  │ image: 1.21      │  │ image: kafka:4.0 │                      │   │
│  │  │ port: 8500       │  │ port: 9092       │                      │   │
│  │  └──────────────────┘  └──────────────────┘                      │   │
│  └──────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │                       Red:  data-net                             │   │
│  │                                                                  │   │
│  │  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐│   │
│  │  │ postgres         │  │ redis            │  │ rabbitmq         ││   │
│  │  │ image: pg:18     │  │ image: redis:8   │  │ image: rmq:4.1   ││   │
│  │  │ port: 5432       │  │ port: 6379       │  │ port: 5672/15672 ││   │
│  │  │ vol: pgdata      │  │ vol: redisdata   │  │ vol: rmqdata     ││   │
│  │  └──────────────────┘  └──────────────────┘  └──────────────────┘│   │
│  └──────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │                       Red:  observability-net                    │   │
│  │                                                                  │   │
│  │  ┌──────────────────┐  ┌──────────────────┐                      │   │
│  │  │ prometheus       │  │ grafana          │                      │   │
│  │  │ image: prom:3    │  │ image: grafana:12│                      │   │
│  │  │ port: 9090       │  │ port: 3000       │                      │   │
│  │  └──────────────────┘  └──────────────────┘                      │   │
│  └──────────────────────────────────────────────────────────────────┘   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

## 10.2 Redes Docker

| Red | Contenedores | Función |
|---|---|---|
| `frontend-net` | frontend, api-gateway | Tráfico expuesto al exterior |
| `backend-net` | microservicios, consul, kafka | Comunicación interna entre servicios |
| `data-net` | postgres, redis, rabbitmq | Acceso restringido a persistencia |
| `observability-net` | prometheus, grafana | Métricas y monitoreo |

## 10.3 Volúmenes Persistentes

| Volumen | Contenedor | Contenido |
|---|---|---|
| `pgdata` | postgres | Datos de PostgreSQL |
| `redisdata` | redis | Snapshots de Redis |
| `rmqdata` | rabbitmq | Cola de mensajes |
| `kafkadata` | kafka | Logs de topics |
| `keycloakdata` | auth-service | Configuración de realms |
| `grafanadata` | grafana | Dashboards y configuración |
| `documentstore` | document-service | PDFs generados |

## 10.4 Variables de Entorno por Servicio

Cada microservicio recibe configuración vía variables de entorno o secretos:

- `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD`
- `REDIS_HOST`, `REDIS_PORT`
- `KAFKA_BROKERS` / `RABBITMQ_URL`
- `KEYCLOAK_URL`, `KEYCLOAK_REALM`, `KEYCLOAK_CLIENT_ID`
- `CONSUL_HOST`, `CONSUL_PORT`
- `LOG_LEVEL`, `SERVICE_NAME`, `SERVICE_PORT`

## 10.5 Estrategia de Despliegue

- **Desarrollo y Producción:** Docker Compose orquesta todo el stack (`docker compose up -d`).
- **Health Checks:** cada contenedor expone `/health` y `/ready`, monitoreados por Docker.
- **Reinicio automático:** `restart: unless-stopped` en todos los servicios.
- **Logs centralizados:** stdout de cada contenedor agregado vía driver de logging.
- **Escalado horizontal:** réplicas por microservicio con `docker compose up --scale <servicio>=N`.
- **Actualizaciones:** despliegue por servicio con `docker compose up -d --no-deps <servicio>`.
- **Backups:** volúmenes respaldados periódicamente (PostgreSQL, documentos generados).
- **Migración futura:** la arquitectura queda preparada para migrar a Kubernetes si el crecimiento lo requiere.

---

# 11. Estado de Implementación

Capacidades implementadas y verificadas sobre la arquitectura descrita, con su
ubicación técnica principal.

## 11.1 Autorización por rol (RBAC)

- Validación del JWT de Keycloak en el borde (Kong, plugin `jwt`) **y** control
  de rol **por endpoint** en cada microservicio (`admin` / `supervisor` / `cajero`).
- Middleware nil-safe `Verifier.GuardRole(...)`: actúa como passthrough cuando
  `AUTH_ENABLED=false` (desarrollo), sin romper el flujo local.
- Reparto (§4): el cajero registra clientes y pagos; el supervisor aprueba/rechaza
  préstamos y consulta reportes; el admin tiene acceso total.

## 11.2 Control de mora automático (loan-service)

- Job programado (`internal/mora`) que en cada intervalo (`MORA_JOB_INTERVAL`,
  por defecto 24h): devenga interés moratorio sobre cuotas vencidas (parámetros
  `tasa_mora_diaria` y `dias_gracia_mora`), marca las cuotas como `vencida` y
  transiciona los préstamos `activo` ↔ `mora` (incluida la regularización).
- Devengo **idempotente** (columna `mora_aplicada_hasta`) y seguro ante réplicas
  (`pg_try_advisory_xact_lock`).
- Migración: `db/init/09-mora-control.sql`.

## 11.3 Anulación de pagos (payment-service)

- `POST /payments/{id}/void` (supervisor/admin): anula el pago **sin borrarlo**
  (auditoría inmutable: `anulado_at`, `anulado_por`, `motivo_anulacion`) y
  **revierte de forma compensatoria** su aplicación a la cuota.
- Mismo patrón que el registro: evento `pago.anulado` por outbox + reconciliación
  por el consumer; reversión **idempotente** vía `pago_aplicaciones.reverted_at`.
- El pipeline de eventos enruta por `event_type` (header Kafka).
- Migración: `db/init/10-pago-reversion.sql`.

## 11.4 Exportación de reportes (report-service)

- Todos los endpoints de reportes aceptan `?format=json|csv|xlsx|pdf` (RF-016).
- CSV (stdlib), **XLSX** (`excelize`) y **PDF** (`gofpdf`); descarga con
  cabecera `Content-Disposition`.
- Frontend: botones de descarga (CSV/Excel/PDF) por sección en la pantalla de Reportes.

## 11.5 Historial crediticio del cliente

- `GET /reports/clients/{id}` consolida datos del cliente + resumen (préstamos,
  total prestado/pagado, saldo, mora, cuotas vencidas) + **veredicto de
  elegibilidad** para un nuevo préstamo.
- Regla de negocio §7 aplicada: el loan-service **rechaza la aprobación** si el
  cliente tiene mora activa (cuotas vencidas impagas), salvo que el parámetro
  `aprobar_si_mora_activa` lo permita.
- Frontend: pantalla `/clientes/:id` con perfil, listado de préstamos e historial de pagos.

## 11.6 Documentación de APIs — OpenAPI 3.1

- Cada microservicio publica su contrato **OpenAPI 3.1** en `/openapi.yaml`
  (embebido con `go:embed`, fuente de verdad del contrato) y una UI **Swagger**
  en `/docs`, ambas públicas dentro de la red.

## 11.7 Frontend

- UI migrada a **Tailwind CSS v4** y **responsive** (mobile-first): nav colapsable
  en móvil, tablas con scroll horizontal, formularios/grids apilables y estados
  de carga/error/vacío (ver §6 Usabilidad).

## 11.8 Decisiones de arquitectura (ADR)

- Las decisiones se registran en `docs/adr/`. **ADR-0001: Kong Gateway OSS**
  (Apache 2.0) como API Gateway — no requiere licencia en producción.
