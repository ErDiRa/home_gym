# Production image: the Go API serving the built frontend. Build from the repo root:
#   docker build -t homegym .

# --- Frontend: vue-tsc type-check + Vite build into /src/dist ---
FROM node:24-alpine AS frontend
WORKDIR /src
# Dependencies first, so this layer stays cached while only the code changes
COPY frontend/app/package.json frontend/app/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY frontend/app/ ./
RUN npm run build

# --- Backend: static binary ---
FROM golang:1.26-alpine AS backend
# Same sqlc version as Dockerfile.dev; its own layer, so it is only built once
RUN go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.29.0
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
# Regenerate the repos/ packages from queries.sql and the schema, and internal/api from the
# OpenAPI spec, so the binary never builds against stale generated code
RUN sqlc generate
RUN go generate ./internal/api
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# --- Runtime ---
# Alpine rather than distroless so busybox wget can run the healthcheck
FROM alpine:3.22
# Runs as an unprivileged user with a fixed UID/GID (10001), so it is stable across rebuilds and
# can be matched by host permissions; the binary and web files stay root-owned and read-only.
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S -g 10001 app && adduser -S -D -H -u 10001 -G app -s /sbin/nologin app
WORKDIR /app
COPY --from=backend /out/api /app/api
COPY --from=frontend /src/dist /app/web
ENV STATIC_DIR=/app/web \
    PORT=8080 \
    COOKIE_SECURE=true \
    LOG_FORMAT=json \
    LOG_LEVEL=info
USER 10001:10001
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -q -O /dev/null http://127.0.0.1:8080/healthz || exit 1
ENTRYPOINT ["/app/api"]
