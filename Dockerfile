# =============================================================================
# Waygates - Combined Backend + Caddy Container
# =============================================================================
# This Dockerfile creates a single container running both the Waygates backend
# and Caddy server. The backend manages Caddy configuration via Caddyfiles.
# =============================================================================

# =============================================================================
# Stage 1: Build UI (only once on build platform - output is static files)
# =============================================================================
FROM --platform=$BUILDPLATFORM node:22-alpine AS ui-builder

WORKDIR /app

# Install pnpm
RUN corepack enable && corepack prepare pnpm@latest --activate

# Copy UI package files
COPY ui/package.json ui/pnpm-lock.yaml* ./

# Install dependencies
RUN pnpm install --frozen-lockfile || pnpm install

# Copy UI source code
COPY ui/ .

# Build the UI
RUN pnpm run build

# =============================================================================
# Stage 2: Build Caddy with Cloudflare DNS plugin
# =============================================================================
FROM --platform=$BUILDPLATFORM caddy:2.10.2-builder AS caddy-builder

ARG TARGETOS
ARG TARGETARCH

RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} xcaddy build \
    --with github.com/caddy-dns/cloudflare

# =============================================================================
# Stage 3: Build Backend
# =============================================================================
FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS backend-builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files (at project root level)
COPY go.mod go.sum ./

# Download dependencies (allow toolchain auto-download for newer Go versions)
ENV GOTOOLCHAIN=auto
RUN go mod download

# Copy backend source code
COPY backend/ ./backend/

# Build the application (cross-compile for target platform)
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} GOTOOLCHAIN=auto go build -o /app/server ./backend/cmd/server

# =============================================================================
# Stage 4: Runtime (Combined Backend + Caddy)
# =============================================================================
FROM alpine:3.19

WORKDIR /app

# OCI Image Labels
LABEL org.opencontainers.image.title="Waygates"
LABEL org.opencontainers.image.description="A modern reverse proxy manager with React UI and Go backend"
LABEL org.opencontainers.image.vendor="aloks98"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.source="https://github.com/aloks98/waygates"

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata tini curl

# Copy Caddy binary
COPY --from=caddy-builder /usr/bin/caddy /usr/bin/caddy

# Copy backend binary
COPY --from=backend-builder /app/server /app/server

# Copy migrations, RBAC config, and templates
COPY backend/migrations /app/backend/migrations
COPY backend/rbac.yaml /app/backend/rbac.yaml
COPY backend/templates /app/templates

# Copy UI dist from ui builder
COPY --from=ui-builder /app/dist /app/ui

# Copy entrypoint script
COPY docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Copy default Caddyfile template and security snippets to /app/defaults
# (NOT /etc/caddy which is a volume mount that gets overwritten)
COPY docker/Caddyfile.default /app/defaults/Caddyfile.default
COPY conf/snippets /app/defaults/snippets

# Create required directories
RUN mkdir -p /etc/caddy/sites /etc/caddy/backup /data /config

# Expose ports
# 80   - HTTP (redirect to HTTPS)
# 443  - HTTPS (proxy traffic)
# 8080 - Backend API
EXPOSE 80 443 8080

# Environment variables (defaults for Docker)
ENV UI_ENABLED=true
ENV UI_PATH=/app/ui

# Volume for persistent data
VOLUME ["/data", "/config", "/etc/caddy"]

# Use tini as init system for proper signal handling
ENTRYPOINT ["/sbin/tini", "--"]
CMD ["/entrypoint.sh"]
