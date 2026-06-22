#!/usr/bin/env bash
# Genera un certificado TLS autofirmado para el frontend (nginx) en ./certs.
#
# ⚠️  Autofirmado = solo para desarrollo / staging interno. En producción real
#     usar certificados de una CA (Let's Encrypt / corporativa) y montarlos en
#     el mismo path (certs/server.crt, certs/server.key).
set -euo pipefail

cd "$(dirname "$0")/.."

CN="${1:-localhost}"
mkdir -p certs

if [[ -f certs/server.crt && -f certs/server.key ]]; then
  echo "✗ Ya existen certs/server.crt y certs/server.key — no se sobrescriben." >&2
  exit 1
fi

openssl req -x509 -nodes -newkey rsa:2048 -days 365 \
  -keyout certs/server.key \
  -out    certs/server.crt \
  -subj   "/CN=${CN}" \
  -addext "subjectAltName=DNS:${CN},DNS:localhost,IP:127.0.0.1"

chmod 600 certs/server.key
echo "✓ Certificado autofirmado generado en ./certs (CN=${CN}, 365 días)."
