# 0001. Kong Gateway OSS (Apache 2.0) como API Gateway

- **Estado:** Aceptado
- **Fecha:** 2026-06-09
- **Decisores:** Arquitectura / Plataforma

## Contexto

El sistema de préstamos expone seis microservicios (client, loan, payment,
report, document + Keycloak) detrás de un único punto de entrada. El borde
(edge) debe resolver responsabilidades transversales **sin** mezclarlas con la
lógica de negocio:

- **Autenticación de borde:** validar el JWT emitido por Keycloak (firma RS256,
  expiración) antes de que la petición llegue a los servicios.
- **Protección:** rate limiting, CORS, cabeceras de seguridad.
- **Trazabilidad/observabilidad:** correlation-id por petición y métricas
  Prometheus, requisitos no negociables en un dominio financiero (auditoría).

Restricciones relevantes:

- **Producción sobre Docker Compose** (ver entorno actual), no Kubernetes.
- **Coste:** preocupa incurrir en licencias al pasar a producción.
- Stack del equipo: Go en backend, Angular en frontend.
- El flujo OIDC completo (login, refresh, PKCE) ya lo realiza el **frontend**
  con `keycloak-js`; el gateway solo necesita **validar** el token, no
  orquestar el flujo de autorización.

La pregunta concreta que motiva este ADR: *¿el uso de Kong obliga a pagar una
licencia en producción?*

## Opciones consideradas

### Opción A — Kong Gateway OSS (`image: kong`, Apache 2.0), DB-less
- Pros:
  - **Gratis, también en producción** (licencia Apache 2.0); sin límite de uso.
  - Funcionalidad de borde **probada en producción** sin escribir código:
    plugins `jwt`, `rate-limiting`, `cors`, `response-transformer`,
    `correlation-id`, `prometheus`, `file-log` — **todos del core OSS**.
  - Configuración **declarativa** (`gateway/kong.yml`), versionable, sin base
    de datos (modo DB-less) → menos piezas que operar.
  - Reduce superficie de riesgo: la authn de borde la hace un componente
    maduro, no código propio.
- Contras:
  - Stack distinto al del equipo (OpenResty/Nginx/Lua); customizaciones
    profundas requieren Lua.
  - Una pieza más de infraestructura y un hop de red adicional (latencia mínima).

### Opción B — Kong Gateway Enterprise (`kong/kong-gateway`) / Kong Konnect (SaaS)
- Pros:
  - Kong Manager (GUI), RBAC del panel, plugin OIDC, mTLS avanzado, OPA,
    secrets manager, soporte con SLA.
- Contras:
  - **De pago** (licencia/suscripción).
  - **Ninguna de esas features es necesaria** hoy: el OIDC lo cubre el frontend
    y la administración declarativa por `kong.yml` no requiere GUI.

### Opción C — Gateway propio escrito en Go
- Pros:
  - Mismo stack que el equipo; control total; binario mínimo.
- Contras:
  - Reinventar y **mantener** rate limiting, circuit breaker, validación JWT,
    CORS, hot-reload, métricas.
  - Un gateway casero es **código crítico de seguridad**: un bug propio en la
    authz de borde es más peligroso que en la lógica de negocio.

### Opción D — Gateway OSS en Go (KrakenD CE / Tyk OSS / Traefik)
- Pros:
  - Gratis y open source; homogeneiza el stack en Go; sin vendor lock-in de Kong.
- Contras:
  - Coste de migración sin beneficio inmediato (Kong OSS ya cubre lo requerido).
  - Re-aprender configuración y re-validar seguridad/observabilidad.

## Decisión

Se adopta **Kong Gateway OSS** (imagen `kong:3.9`, Apache 2.0) en modo
**DB-less** con configuración declarativa en `gateway/kong.yml`.

Motivos:

1. **No requiere licencia en producción.** La imagen `kong` es la edición OSS;
   la edición de pago es `kong/kong-gateway`. La configuración actual **no usa
   ningún plugin Enterprise** ni Kong Manager/Konnect.
2. **Cubre todas las fuerzas del contexto** (JWT, rate limiting, CORS, headers,
   correlation-id, métricas) con plugins del core OSS, sin código propio.
3. El flujo OIDC vive en el frontend; Kong solo **valida la firma del JWT** con
   el plugin `jwt` (OSS), por lo que **no se necesita el plugin OIDC Enterprise**.

## Consecuencias

- **Positivas:**
  - Coste de gateway = 0 en producción.
  - Edge endurecido (authn, rate limiting, headers) con componente battle-tested.
  - Config declarativa y versionada; despliegue reproducible en Docker Compose.
- **Negativas / deuda asumida:**
  - Dependencia de un stack (OpenResty/Lua) ajeno al del equipo para
    customizaciones avanzadas.
  - Sin GUI de administración ni soporte con SLA (aceptable para el alcance).
- **Impacto:**
  - *Seguridad:* validación de JWT en el borde antes de los servicios; los
    servicios además aplican RBAC por rol (defensa en profundidad).
  - *Observabilidad:* métricas Prometheus y correlation-id desde el gateway.
  - *Operación/CI-CD:* fijar imagen OSS por tag/digest; mantener modo DB-less.

## Seguimiento

Acciones:
- Fijar la imagen como `kong:3.9` (idealmente con digest); **nunca**
  `kong/kong-gateway`.
- Mantener el modo **DB-less** (`kong.yml`), sin base de datos de Kong.
- En revisiones, verificar que no se introduzcan plugins marcados Enterprise
  (Kong falla al cargarlos sin licencia, lo que serviría de alerta temprana).

Condiciones que dispararían reconsiderar esta decisión (evaluar Enterprise o la
Opción D):
- Necesidad real de **Kong Manager**, RBAC del panel, **plugin OIDC**, mTLS
  avanzado/OPA o **soporte con SLA**.
- Migración a Kubernetes con requerimientos de gateway/API management más
  amplios.
- Decisión de **homogeneizar el stack en Go** o eliminar la dependencia de Kong
  → evaluar **KrakenD CE / Tyk OSS / Traefik**.

Métrica de éxito: el gateway opera en producción cubriendo authn de borde, rate
limiting y observabilidad **sin coste de licencia** y sin incidentes de
seguridad atribuibles al edge.
