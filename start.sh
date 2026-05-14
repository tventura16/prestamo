#!/usr/bin/env bash
# ───────────────────────────────────────────────
# Levanta todo el stack del sistema de préstamos
# ───────────────────────────────────────────────
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# ─── Colores ───
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log()   { echo -e "${BLUE}[$(date +%H:%M:%S)]${NC} $*"; }
ok()    { echo -e "${GREEN}[OK]${NC} $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*" >&2; }

show_status() {
    echo
    log "Estado de los contenedores:"
    docker compose ps --format "table {{.Name}}\t{{.Status}}\t{{.Ports}}"
}

show_urls() {
    cat <<EOF

${GREEN}═══════════════════════════════════════════════${NC}
${GREEN}  URLs disponibles${NC}
${GREEN}═══════════════════════════════════════════════${NC}

  Frontend Angular         ─▶  http://localhost
  API Gateway (Kong)       ─▶  http://localhost:8000
  Kong Admin API           ─▶  http://localhost:8001
  Keycloak (Auth)          ─▶  http://localhost:8080
  Consul UI                ─▶  http://localhost:8500
  RabbitMQ Management      ─▶  http://localhost:15672
  Prometheus               ─▶  http://localhost:9090
  Grafana                  ─▶  http://localhost:3000

  Microservicios (directo, normalmente vía gateway):
    client-service           http://localhost:8081
    loan-service             http://localhost:8082
    payment-service          http://localhost:8083
    report-service           http://localhost:8084
    document-service         http://localhost:8085

${GREEN}═══════════════════════════════════════════════${NC}
EOF
}

# ─── 1. Verificar dependencias ───
log "Verificando dependencias..."
command -v docker >/dev/null 2>&1 || { error "Docker no está instalado"; exit 1; }
docker compose version >/dev/null 2>&1 || { error "Docker Compose v2 no está disponible"; exit 1; }

if ! docker info >/dev/null 2>&1; then
    error "El daemon de Docker no está corriendo"
    exit 1
fi
ok "Docker y Docker Compose disponibles"

# ─── 2. Preparar .env ───
if [[ ! -f .env ]]; then
    if [[ -f .env.example ]]; then
        warn ".env no existe — copiando desde .env.example"
        cp .env.example .env
        warn "Edita .env con credenciales reales antes de usar en producción"
    else
        error "No se encontró .env ni .env.example"
        exit 1
    fi
fi
ok ".env listo"

# ─── 3. Modo de operación ───
COMMAND="${1:-up}"

case "$COMMAND" in

    up)
        log "Construyendo imágenes (puede tardar la primera vez)..."
        docker compose build --parallel
        ok "Imágenes construidas"

        log "Levantando infraestructura base (postgres, redis, kafka, rabbitmq, consul)..."
        docker compose up -d postgres redis kafka rabbitmq consul

        log "Esperando a que postgres esté listo..."
        until docker compose exec -T postgres pg_isready -U "${DB_USER:-prestamos}" >/dev/null 2>&1; do
            sleep 2
            echo -n "."
        done
        echo
        ok "PostgreSQL listo"

        log "Levantando auth-service (Keycloak)..."
        docker compose up -d auth-service

        log "Levantando microservicios Go..."
        docker compose up -d \
            client-service \
            loan-service \
            payment-service \
            report-service \
            document-service

        log "Levantando API Gateway (Kong)..."
        docker compose up -d api-gateway

        log "Levantando observabilidad (Prometheus, Grafana)..."
        docker compose up -d prometheus grafana

        log "Levantando frontend..."
        docker compose up -d frontend

        echo
        ok "Todos los servicios levantados"
        echo
        show_status
        show_urls
        ;;

    down)
        log "Deteniendo todos los servicios..."
        docker compose down
        ok "Servicios detenidos"
        ;;

    restart)
        log "Reiniciando stack..."
        docker compose down
        exec "$0" up
        ;;

    clean)
        warn "Esto BORRARÁ todos los datos (volúmenes incluidos)"
        read -r -p "¿Continuar? [s/N]: " confirm
        if [[ "$confirm" =~ ^[sS]$ ]]; then
            docker compose down -v
            ok "Stack detenido y volúmenes eliminados"
        else
            log "Cancelado"
        fi
        ;;

    logs)
        SERVICE="${2:-}"
        if [[ -n "$SERVICE" ]]; then
            docker compose logs -f "$SERVICE"
        else
            docker compose logs -f
        fi
        ;;

    status|ps)
        show_status
        show_urls
        ;;

    rebuild)
        SERVICE="${2:-}"
        if [[ -z "$SERVICE" ]]; then
            log "Reconstruyendo todas las imágenes..."
            docker compose build --no-cache --parallel
            docker compose up -d
        else
            log "Reconstruyendo $SERVICE..."
            docker compose build --no-cache "$SERVICE"
            docker compose up -d --no-deps "$SERVICE"
        fi
        ok "Reconstrucción completa"
        ;;

    *)
        cat <<EOF
Uso: $0 [comando]

Comandos:
  up               Levanta todo el stack (default)
  down             Detiene todos los servicios
  restart          Reinicia el stack
  clean            Detiene y borra TODOS los volúmenes (¡destructivo!)
  logs [servicio]  Muestra logs (todos o de un servicio)
  status | ps      Muestra estado de contenedores y URLs
  rebuild [svc]    Reconstruye imágenes (todas o una)

Ejemplos:
  $0 up
  $0 logs loan-service
  $0 rebuild payment-service
EOF
        exit 0
        ;;
esac
