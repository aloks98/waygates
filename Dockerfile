# =============================================================================
# Waygates - Combined Backend + Caddy Container
# =============================================================================
# This Dockerfile creates a single container running both the Waygates backend
# and Caddy server. The backend manages Caddy configuration via JSON API.
#
# CUSTOMIZATION:
#
# 1. TLS/HTTPS Configuration:
#    Set CADDY_ACME_PROVIDER environment variable. Options:
#      off          - No HTTPS (development)
#      http         - HTTP challenge (requires ports 80/443 public)
#      cloudflare   - Cloudflare DNS challenge
#      route53      - AWS Route53 DNS challenge
#      digitalocean - DigitalOcean DNS challenge
#      duckdns      - DuckDNS challenge
#      hetzner      - Hetzner DNS challenge
#      porkbun      - Porkbun DNS challenge
#      azure        - Azure DNS challenge
#      vultr        - Vultr DNS challenge
#      namecheap    - Namecheap DNS challenge
#      ovh          - OVH DNS challenge
#    See .env.example for required environment variables per provider.
#
# 2. Caddy Plugins:
#    Modify the "xcaddy build" command in Stage 2 below to add/remove plugins.
#
#    Included plugins:
#      --with github.com/mholt/caddy-l4 (TCP/UDP Layer 4 proxy support)
#
#    Common DNS plugins for ACME:
#      --with github.com/caddy-dns/cloudflare
#      --with github.com/caddy-dns/route53
#      --with github.com/caddy-dns/digitalocean
#      --with github.com/caddy-dns/duckdns
#      --with github.com/caddy-dns/godaddy
#    See all plugins: https://caddyserver.com/docs/modules/
#
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
# Stage 2: Build Caddy with L4 plugin and DNS challenge plugins
# =============================================================================
# Pre-built with:
#   - caddy-l4: TCP/UDP Layer 4 proxy support
#   - DNS providers: For ACME challenges (configure via CADDY_ACME_PROVIDER)
# =============================================================================
# NOTE: caddy-l4 v0.1.1 requires caddy/v2 >= v2.11.3, so the builder image must
# match. Keep this base image and the pinned caddy-l4 version below in sync.
FROM --platform=$BUILDPLATFORM caddy:2.11.3-builder AS caddy-builder

ARG TARGETOS
ARG TARGETARCH

# Build Caddy with L4 plugin and all supported DNS challenge providers.
# caddy-l4 is pinned so an upstream release cannot silently break the build by
# requiring a newer Caddy than the builder image above.
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} xcaddy build \
    --with github.com/mholt/caddy-l4@v0.1.1 \
    --with github.com/caddy-dns/cloudflare \
    --with github.com/caddy-dns/route53 \
    --with github.com/caddy-dns/duckdns \
    --with github.com/caddy-dns/digitalocean \
    --with github.com/caddy-dns/hetzner \
    --with github.com/caddy-dns/porkbun \
    --with github.com/caddy-dns/azure \
    --with github.com/caddy-dns/vultr \
    --with github.com/caddy-dns/namecheap \
    --with github.com/caddy-dns/ovh

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

# Note: JSON configuration (caddy.json) is generated dynamically by the backend

# Create required directories
RUN mkdir -p /etc/caddy/backup /data /config

# Expose ports
# 80   - HTTP (redirect to HTTPS)
# 443  - HTTPS (proxy traffic)
# 8080 - Backend API
# Note: L4 (TCP/UDP) proxies use custom ports configured per proxy.
#       Expose additional ports in docker-compose.yml as needed.
EXPOSE 80 443 8080

# Environment variables (defaults for Docker)
ENV UI_ENABLED=true
ENV UI_PATH=/app/ui

# Volume for persistent data
VOLUME ["/data", "/config", "/etc/caddy"]

# Use tini as init system for proper signal handling
ENTRYPOINT ["/sbin/tini", "--"]
CMD ["/entrypoint.sh"]
