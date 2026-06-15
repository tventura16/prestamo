#!/usr/bin/env bash
# Genera un .env con secretos fuertes a partir de .env.example.
# No sobrescribe un .env existente (para no rotar credenciales por accidente).
set -euo pipefail

cd "$(dirname "$0")/.."

if [[ -f .env ]]; then
  echo "✗ Ya existe .env — no se sobrescribe. Bórralo a mano si quieres rotar." >&2
  exit 1
fi

# Secreto urlsafe (sin / + = que rompan URLs como la de RabbitMQ).
secret() { openssl rand -base64 24 | tr -d '/+=' | cut -c1-24; }

cat > .env <<EOF
# Generado por scripts/gen-secrets.sh — NO versionar.
DB_USER=prestamos
DB_PASSWORD=$(secret)

KEYCLOAK_ADMIN=admin
KEYCLOAK_ADMIN_PASSWORD=$(secret)

RABBITMQ_USER=prestamos
RABBITMQ_PASSWORD=$(secret)

GRAFANA_USER=admin
GRAFANA_PASSWORD=$(secret)
EOF

chmod 600 .env
echo "✓ .env generado con secretos fuertes (chmod 600)."
echo "  Revísalo con: cat .env"
