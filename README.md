# Waygates

A modern reverse proxy manager with a React UI and Go backend. Manage your Caddy reverse proxy configurations through a clean web interface.

[![codecov](https://codecov.io/gh/aloks98/waygates/graph/badge.svg?token=J4V92R1NN1)](https://codecov.io/gh/aloks98/waygates)
## Features

- Web UI for managing proxy configurations
- REST API for automation
- JWT-based authentication with RBAC
- Automatic TLS with 10+ DNS providers supported
- **Layer 7 (HTTP)**: Reverse proxy, redirect, and static file serving
- **Layer 4 (TCP/UDP)**: Raw TCP/UDP proxying with protocol-aware routing
- PostgreSQL for persistent storage
- Single Docker image (backend + Caddy + UI)

### L4 Proxy Features

- TCP and UDP protocol support
- Protocol matchers: TLS/SNI, SSH, PostgreSQL, HTTP, RDP, SOCKS5
- IP-based access control
- Load balancing (round-robin, least-conn, random, ip-hash)
- TLS passthrough or termination
- Proxy Protocol v1/v2 support

## Quick Start

```bash
# Pull the image
docker pull ghcr.io/aloks98/waygates:latest

# Run with Docker Compose (recommended)
# See docs/DEPLOYMENT.md for full setup
```

**[Full Deployment Guide](docs/DEPLOYMENT.md)** - Comprehensive instructions for all deployment scenarios.

## Prerequisites

- Docker with Docker Compose V2
- PostgreSQL 14+

## Architecture

Waygates runs as a **single container** combining:

- **Go Backend**: REST API + Caddyfile generation + sync service
- **Caddy**: Reverse proxy with automatic HTTPS
- **React UI**: Management interface (served by backend)

```
┌─────────────────────────────────────────────────┐
│              Waygates Container                 │
│  ┌──────────────────────────────────────────┐  │
│  │  Go Backend (Port 8080)                  │  │
│  │  - REST API                              │  │
│  │  - UI static files                       │  │
│  │  - JSON config generation                │  │
│  │  - Sync service (DB → Caddy)             │  │
│  └────────────────┬─────────────────────────┘  │
│                   │ reload                      │
│  ┌────────────────▼─────────────────────────┐  │
│  │  Caddy + L4 Plugin                       │  │
│  │  - HTTP proxy (Ports 80, 443)            │  │
│  │  - L4 proxy (Custom TCP/UDP ports)       │  │
│  │  - Automatic HTTPS                       │  │
│  │  - DNS challenge support                 │  │
│  └──────────────────────────────────────────┘  │
└─────────────────────────────────────────────────┘
                     │
        ┌────────────┴────────────┐
        ▼                         ▼
┌───────────────┐         ┌───────────────┐
│  PostgreSQL   │         │  Your Apps    │
│  (Database)   │         │  (Upstreams)  │
└───────────────┘         └───────────────┘
```

## TLS Configuration

Waygates supports multiple ACME providers for automatic TLS certificates:

| Provider | Environment Variables Required |
|----------|-------------------------------|
| `off` | None (HTTPS disabled) |
| `http` | None (HTTP challenge, ports 80/443 must be open) |
| `cloudflare` | `CLOUDFLARE_API_TOKEN` |
| `route53` | `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` |
| `digitalocean` | `DO_AUTH_TOKEN` |
| `duckdns` | `DUCKDNS_API_TOKEN` |
| `hetzner` | `HETZNER_API_TOKEN` |
| `porkbun` | `PORKBUN_API_KEY`, `PORKBUN_API_SECRET_KEY` |
| `vultr` | `VULTR_API_KEY` |
| `namecheap` | `NAMECHEAP_API_USER`, `NAMECHEAP_API_KEY` |
| `ovh` | `OVH_ENDPOINT`, `OVH_APPLICATION_KEY`, `OVH_APPLICATION_SECRET`, `OVH_CONSUMER_KEY` |
| `azure` | `AZURE_TENANT_ID`, `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET`, `AZURE_SUBSCRIPTION_ID`, `AZURE_RESOURCE_GROUP` |

Configure via environment variables:

```env
CADDY_ACME_PROVIDER=cloudflare
CADDY_EMAIL=admin@example.com
CLOUDFLARE_API_TOKEN=your-token
```

## Project Structure

```
waygates/
├── backend/                   # Go API server
│   ├── cmd/server/            # Main entry point
│   ├── internal/              # Internal packages
│   │   ├── api/               # HTTP handlers and routes
│   │   ├── caddy/             # Caddyfile generation
│   │   ├── config/            # Configuration
│   │   ├── models/            # Data models
│   │   ├── repository/        # Database layer
│   │   └── service/           # Business logic + sync
│   ├── migrations/            # Database migrations
│   └── rbac.yaml              # Role-based access control config
├── ui/                        # React frontend
│   └── src/                   # Source code
├── conf/
│   └── snippets/              # Security snippets for Caddy
├── docker/
│   └── entrypoint.sh          # Container entrypoint
├── docs/                      # Documentation
├── docker-compose.yml         # Docker services
├── Dockerfile                 # Combined image (backend + Caddy + UI)
├── Makefile                   # Build commands
└── .env.example               # Environment template
```

## Environment Variables

### Required

| Variable | Description |
|----------|-------------|
| `JWT_SECRET` | JWT signing key (min 32 characters) |
| `DB_HOST` | PostgreSQL host |
| `DB_PASSWORD` | PostgreSQL password |

### TLS Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `CADDY_ACME_PROVIDER` | `off` | ACME provider (see table above) |
| `CADDY_EMAIL` | - | Email for ACME certificates |

### Optional

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `waygates` | PostgreSQL user |
| `DB_NAME` | `waygates` | Database name |
| `SERVER_PORT` | `8080` | Backend API port |
| `JWT_ACCESS_EXPIRY` | `15m` | Access token expiry |
| `JWT_REFRESH_EXPIRY` | `168h` | Refresh token expiry |
| `BCRYPT_COST` | `12` | Password hashing cost |
| `LOG_LEVEL` | `info` | Log level (debug/info/warn/error) |

## Documentation

- **[Deployment Guide](docs/DEPLOYMENT.md)** - Full deployment instructions
- **[API Authentication](docs/API_AUTH.md)** - Authentication endpoints
- **[API Proxy](docs/API_PROXY.md)** - HTTP proxy management endpoints
- **[API L4 Proxy](docs/API_L4_PROXY.md)** - TCP/UDP proxy management endpoints
- **[Access Control Lists](docs/ACL.md)** - ACL configuration
- **[OpenAPI Spec](docs/openapi.yaml)** - API specification
- **[TLS Skip Verify](docs/TLS_SKIP_VERIFY.md)** - Upstream TLS configuration

## Development

### Prerequisites

- Go 1.24+
- Node.js 22+ with pnpm
- PostgreSQL 14+

### Running Locally

```bash
# Start PostgreSQL
docker compose up -d postgres

# Build and run backend
go run backend/cmd/server/main.go

# In another terminal, build UI
cd ui && pnpm install && pnpm dev
```

### Building Docker Image

```bash
docker build -t waygates:local .
```

## License

MIT License
