
# ───────────────────────────────────────────────
# Stage 1: Build
# ───────────────────────────────────────────────
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /src

COPY go.mod ./
COPY . .
RUN go mod tidy

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" \
    -o /out/loan-service ./cmd/loan-service

# Directorio de almacenamiento de garantías; se copia con owner nonroot para
# que el volumen montado encima herede ese propietario (distroless no tiene shell).
RUN mkdir -p /var/garantias

# ───────────────────────────────────────────────
# Stage 2: Runtime
# ───────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /out/loan-service /app/loan-service
COPY --from=builder --chown=nonroot:nonroot /var/garantias /var/garantias

ENV TZ=America/La_Paz \
    SERVICE_NAME=loan-service \
    SERVICE_PORT=8082

EXPOSE 8082

USER nonroot:nonroot

ENTRYPOINT ["/app/loan-service"]
