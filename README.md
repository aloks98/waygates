<div align="center">

<img src="ui/src/assets/waygates-mark.svg" alt="Waygates" width="96" height="96" />

<h1>Waygates</h1>

<p>A modern reverse proxy manager with a React UI and Go backend.<br />
Manage your Caddy reverse proxy configurations through a clean web interface.</p>

</div>

## Features

**Proxying**
- **HTTP reverse proxies** — upstream routing, redirects, static file serving, load balancing
- **L4 (TCP/UDP) proxies** — protocol-aware routing with TLS/SNI, SSH, PostgreSQL, HTTP, RDP, SOCKS5 matchers; IP ACL; Proxy Protocol v1/v2 support

**Access control**
- **ACL groups** — basic-auth, OAuth/SSO (OIDC), and forward-auth modes; per-group login branding; attach groups to any proxy for instant protection

**Observability**
- **Traffic metrics** dashboard — per-proxy request counts, error rates, and latency; opt-in Prometheus `/metrics` endpoint for external scraping
- **Caddy logs viewer** — live SSE stream of runtime and access logs from within the UI
- **Caddy config preview** — view the generated JSON config (global and per-proxy) without leaving the UI
- **Audit log / activity** viewer — full record of every configuration change

**Management**
- **Settings** — default/404 page, login branding, metrics publishing toggle
- **Periodic sync** — background service reconciles the database with Caddy every 60 seconds; manual trigger via API
- **RBAC** — fine-grained role-based access control with JWT authentication
- **Single Docker image** — backend + Caddy (with L4 plugin) + React UI in one container

## Quick Start

### Docker Compose (recommended)

```bash
# 1. Copy and configure environment
cp .env.example .env
# Edit .env — set JWT_SECRET, DB_PASSWORD, and any TLS vars

# 2. Start everything
make up
```

The UI is available at `http://localhost:8080`. Default credentials can be set via `DEFAULT_USER_*` env vars (see below); if not set, the first-run admin is created automatically.

For a complete deployment walkthrough (TLS, DNS providers, reverse-proxy-behind-tunnel), see **[docs/DEPLOYMENT.md](docs/DEPLOYMENT.md)**.

### Quickstart with prebuilt image

```bash
docker pull ghcr.io/aloks98/waygates:latest
# then run with docker compose — see docs/DEPLOYMENT.md
```

## Architecture

Waygates runs as a **single container** combining:

- **Go Backend** (port 8080): REST API + Caddyfile/JSON generation + sync service + UI static files
- **Caddy** (ports 80, 443): Reverse proxy with automatic HTTPS and L4 plugin
- **React UI**: Management interface served by the backend

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

Waygates delegates HTTPS to Caddy. Set `CADDY_ACME_PROVIDER` to choose how certificates are obtained:

| Provider | Additional variables required |
|----------|-------------------------------|
| `off` | None (HTTPS disabled — use for local dev) |
| `http` | None (HTTP-01 challenge; ports 80/443 must be reachable) |
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

Example:

```env
CADDY_ACME_PROVIDER=cloudflare
CADDY_EMAIL=admin@example.com
CLOUDFLARE_API_TOKEN=your-token
```

## Configuration

Copy `.env.example` to `.env` and edit. Key variables:

### Required

| Variable | Description |
|----------|-------------|
| `JWT_SECRET` | JWT signing key — minimum 32 characters |
| `DB_HOST` | PostgreSQL host |
| `DB_PASSWORD` | PostgreSQL password |

### Common optional variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_HOST` | `0.0.0.0` | Bind address for the backend |
| `SERVER_PORT` | `8080` | Backend API + UI port |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `waygates` | PostgreSQL user |
| `DB_NAME` | `waygates` | Database name |
| `JWT_ACCESS_EXPIRY` | `15m` | Access token lifetime |
| `JWT_REFRESH_EXPIRY` | `168h` | Refresh token lifetime |
| `BCRYPT_COST` | `12` | Password hashing cost |
| `LOG_LEVEL` | `info` | Log verbosity (`debug`/`info`/`warn`/`error`) |
| `LOG_FORMAT` | `json` | Log format (`json` or `console`) |
| `CORS_ORIGINS` | `http://localhost:8080` | Allowed CORS origins |
| `UI_ENABLED` | `true` | Serve the React UI from the backend |
| `DEFAULT_USER_NAME` | — | Display name for the bootstrap admin user |
| `DEFAULT_USER_USERNAME` | — | Username for the bootstrap admin user |
| `DEFAULT_USER_EMAIL` | — | Email for the bootstrap admin user |
| `DEFAULT_USER_PASSWORD` | — | Password for the bootstrap admin user |
| `CADDY_ACME_PROVIDER` | `off` | ACME provider (see TLS table above) |
| `CADDY_EMAIL` | — | Email for ACME certificate notifications |
| `CADDY_TRUSTED_PROXIES` | — | Comma-separated CIDRs of upstream connectors to trust for real-IP forwarding (e.g. behind Cloudflare Tunnel or Pangolin) |
| `CADDY_CLIENT_IP_HEADERS` | — | Comma-separated headers carrying the real client IP (e.g. `Cf-Connecting-Ip`) |
| `CADDY_LOG_PATH` | — | Path for Caddy access log file inside the container |

For the full variable reference and deployment scenarios see **[docs/DEPLOYMENT.md](docs/DEPLOYMENT.md)**.

## Local Development

### Prerequisites

- Go 1.25+
- Node.js 22+ with pnpm
- PostgreSQL 14+ (or Docker)

### Running locally

```bash
# Start PostgreSQL only
docker compose up -d postgres

# Run the backend (reads .env automatically)
make backend-run

# In another terminal — run the UI dev server (from repo root)
pnpm --dir ui dev
```

The UI dev server proxies API requests to the backend. Hot reload works for both frontend and backend independently.

### Useful make targets

| Command | Description |
|---------|-------------|
| `make backend-run` | Run Go backend locally |
| `make backend-build` | Compile backend binary to `bin/waygates` |
| `make backend-test` | Run all backend tests |
| `make backend-test-coverage` | Run tests with coverage report |
| `pnpm --dir ui dev` | Start UI dev server (port 8008) |
| `pnpm --dir ui build` | Build UI for production |
| `make lint` | Lint backend + UI |
| `make format` | Format backend + UI |
| `make check` | Lint + test everything |
| `make build` | Build the Docker image |
| `make up` / `make down` | Start / stop Docker Compose stack |
| `make rebuild` | Clean, rebuild image, and restart |
| `make migrate-create NAME=x` | Scaffold a new migration pair |

## Documentation

- **[API Reference](docs/API.md)** — Full REST API documentation
- **[Deployment Guide](docs/DEPLOYMENT.md)** — Deployment scenarios, configuration, and TLS setup
- **[Contributor Guide](.claude/rules/development-guidelines.md)** — Architecture, conventions, and development workflow

## License

MIT License
