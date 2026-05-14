
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
    -o /out/document-service ./cmd/document-service

# ───────────────────────────────────────────────
# Stage 2: Runtime
# ───────────────────────────────────────────────
# document-service usa fuentes para generar PDFs; usamos imagen mínima
# con soporte de fuentes y zona horaria.
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata font-noto font-noto-extra \
    && addgroup -S app && adduser -S app -G app \
    && mkdir -p /var/documents \
    && chown -R app:app /var/documents

COPY --from=builder /out/document-service /app/document-service

ENV TZ=America/La_Paz \
    SERVICE_NAME=document-service \
    SERVICE_PORT=8085 \
    DOCUMENT_STORE_PATH=/var/documents

EXPOSE 8085

USER app:app

VOLUME ["/var/documents"]

ENTRYPOINT ["/app/document-service"]
