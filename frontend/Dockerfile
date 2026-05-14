
# ───────────────────────────────────────────────
# Stage 1: Build Angular
# ───────────────────────────────────────────────
FROM node:22-alpine AS builder

WORKDIR /app

# Cache de dependencias
COPY package.json package-lock.json* ./
RUN if [ -f package-lock.json ]; then npm ci; else npm install; fi

# Código fuente
COPY . .

# Build de producción (Angular 20)
# El output queda en dist/<project-name>/browser
ARG BUILD_CONFIG=production
RUN npm run build -- --configuration=${BUILD_CONFIG}

# ───────────────────────────────────────────────
# Stage 2: Runtime - Nginx
# ───────────────────────────────────────────────
FROM nginx:1.27-alpine

# Quitar config default
RUN rm /etc/nginx/conf.d/default.conf

# Config personalizada para SPA + proxy al API Gateway
COPY nginx.conf /etc/nginx/conf.d/prestamos.conf

# Copiar build de Angular
# Ajustar el nombre del proyecto si difiere (por defecto: dist/prestamos/browser)
COPY --from=builder /app/dist/*/browser /usr/share/nginx/html

# Health check endpoint
RUN echo "ok" > /usr/share/nginx/html/health

EXPOSE 80 443

# nginx ya viene como entrypoint en la imagen base
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost/health || exit 1
