# Waygates Backend

Go-based backend API for managing reverse proxy configurations with Caddy.

## Architecture

Waygates uses a **file-based Caddyfile approach** for configuration management:

- **Modular Configuration**: Each proxy gets its own config file in `sites/` directory
- **Main Caddyfile**: Imports all proxy configs with `import sites/*.conf`
- **Enable/Disable**: Proxies are disabled by renaming to `.conf.disabled`
- **Atomic Writes**: All file operations use temp-file + rename pattern
- **Combined Container**: Backend + Caddy run in a single container

## Features

- User authentication (JWT with refresh tokens)
- RBAC permissions via goauth library
- Proxy management (reverse proxy, redirect, static file server)
- Automatic Caddyfile generation
- Periodic sync between database and Caddy (1 minute interval)
- Configurable 404 behavior (default page or redirect)
- Dynamic TLS configuration (10+ ACME providers supported)

## Available Endpoints

### Public Endpoints

| Method | Endpoint             | Description                               |
|--------|----------------------|-------------------------------------------|
| GET    | `/api/health`        | Health check                              |
| GET    | `/api/status`        | Get application and Caddy status          |
| POST   | `/api/auth/register` | Register a new user                       |
| POST   | `/api/auth/login`    | Log in and receive JWT tokens             |
| POST   | `/api/auth/refresh`  | Refresh access token using refresh token  |

### Protected Endpoints (Auth Required)

| Method | Endpoint                    | Description                     | Permission      |
|--------|-----------------------------|---------------------------------|-----------------|
| GET    | `/api/auth/me`              | Get current user info & perms   | authenticated   |
| POST   | `/api/auth/logout`          | Revoke tokens and log out       | authenticated   |
| GET    | `/api/proxies`              | List all proxies                | proxies:read    |
| GET    | `/api/proxies/:id`          | Get a single proxy              | proxies:read    |
| GET    | `/api/proxies/stats`        | Get proxy statistics            | proxies:read    |
| POST   | `/api/proxies`              | Create a new proxy              | proxies:create  |
| PUT    | `/api/proxies/:id`          | Update a proxy                  | proxies:update  |
| DELETE | `/api/proxies/:id`          | Delete a proxy                  | proxies:delete  |
| POST   | `/api/proxies/:id/enable`   | Enable a proxy                  | proxies:update  |
| POST   | `/api/proxies/:id/disable`  | Disable a proxy                 | proxies:update  |
| GET    | `/api/settings`             | Get all settings                | settings:read   |
| GET    | `/api/settings/404`         | Get 404 settings                | settings:read   |
| PUT    | `/api/settings/404`         | Update 404 settings             | settings:write  |
| GET    | `/api/sync/status`          | Get sync status                 | sync:read       |
| POST   | `/api/sync/trigger`         | Trigger manual sync             | sync:trigger    |

## Configuration Reference

All configuration is loaded from environment variables:

```bash
# Server
SERVER_HOST=0.0.0.0
SERVER_PORT=8080

# Database (PostgreSQL)
DB_HOST=postgres
DB_PORT=5432
DB_USER=waygates
DB_PASSWORD=waygates
DB_NAME=waygates

# TLS/ACME Configuration
CADDY_ACME_PROVIDER=off    # off, http, cloudflare, route53, digitalocean, etc.
CADDY_EMAIL=               # Email for ACME certificates

# DNS Provider Credentials (set based on CADDY_ACME_PROVIDER)
# Cloudflare: CLOUDFLARE_API_TOKEN
# Route53: AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY
# DigitalOcean: DO_AUTH_TOKEN
# See .env.example for full list

# JWT
JWT_SECRET=required-secret-key-min-32-chars
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=168h

# Security
BCRYPT_COST=12
CORS_ORIGINS=http://localhost:8080

# Logging
LOG_LEVEL=info          # debug, info, warn, error
LOG_FORMAT=console      # json or console

# Default User (optional, for initial setup)
DEFAULT_USER_NAME=Admin
DEFAULT_USER_USERNAME=admin
DEFAULT_USER_EMAIL=admin@example.com
DEFAULT_USER_PASSWORD=changeme123

# UI Static File Serving
UI_ENABLED=true         # Enable/disable UI serving
UI_PATH=/app/ui         # Path to the UI dist folder
```

## File Structure

```
/etc/caddy/
├── Caddyfile           # Main config (generated based on CADDY_ACME_PROVIDER)
├── catchall.conf       # Default 404 handler
├── sites/              # Individual proxy configs
│   ├── 1_myapp.conf
│   ├── 2_api.conf
│   └── 3_blog.conf.disabled  # Disabled proxy
├── snippets/           # Security snippets
│   └── security.caddy
└── backup/             # Timestamped backups
```

## Generated Caddyfile Examples

### With ACME off (development):
```caddyfile
# Managed by Waygates - DO NOT EDIT MANUALLY
# ACME Provider: off

{
    auto_https off
    admin localhost:2019
}

import sites/*.conf
import catchall.conf
```

### With Cloudflare DNS challenge:
```caddyfile
# Managed by Waygates - DO NOT EDIT MANUALLY
# ACME Provider: cloudflare

{
    email admin@example.com
    acme_dns cloudflare {$CLOUDFLARE_API_TOKEN}
    admin localhost:2019
}

import sites/*.conf
import catchall.conf
```

## Development

### Prerequisites

- Go 1.24+
- PostgreSQL 14+
- Caddy 2.11+ (required by the caddy-l4 plugin)

### Running Locally

```bash
# From project root
cd backend

# Install dependencies
go mod tidy

# Run with development config
go run cmd/server/main.go
```

### Running Tests

```bash
# Unit tests only
go test ./... -short

# All tests including integration (requires Docker)
go test ./...
```

### Building

```bash
# Build binary
go build -o waygates ./cmd/server

# Build Docker image (from project root)
docker build -t waygates:latest .
```

## Sync Service

The sync service runs automatically and:

1. **Periodic Sync**: Every 60 seconds, regenerates all config files from database
2. **Initial Sync**: On startup, generates Caddyfile and syncs all proxies
3. **Orphan Cleanup**: Removes config files that don't match database records
4. **Caddy Reload**: Validates config then reloads Caddy after changes

### Manual Sync

Trigger a manual sync via API:

```bash
curl -X POST http://localhost:8080/api/sync/trigger \
  -H "Authorization: Bearer $TOKEN"
```

## Troubleshooting

**"JWT_SECRET is required"**
- Set `JWT_SECRET` environment variable with at least 32 characters

**"Cannot connect to database"**
- Verify PostgreSQL is running and connection details are correct

**"invalid CADDY_ACME_PROVIDER"**
- Check spelling. Valid options: off, http, cloudflare, route53, duckdns, digitalocean, hetzner, porkbun, azure, vultr, namecheap, ovh

**"CADDY_ACME_PROVIDER 'cloudflare' requires CLOUDFLARE_API_TOKEN"**
- Set the required environment variable for your chosen provider

**"Caddy validation failed"**
- Check generated Caddyfile syntax: `caddy validate --config /etc/caddy/Caddyfile`
- Review logs for specific validation errors

## License

See project root LICENSE file.
