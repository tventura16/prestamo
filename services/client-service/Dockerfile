
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
    -o /out/client-service ./cmd/client-service

# ───────────────────────────────────────────────
# Stage 2: Runtime
# ───────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /out/client-service /app/client-service

ENV TZ=America/La_Paz \
    SERVICE_NAME=client-service \
    SERVICE_PORT=8081

EXPOSE 8081

USER nonroot:nonroot

ENTRYPOINT ["/app/client-service"]
