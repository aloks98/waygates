# Waygates Backend

Go-based backend API for managing reverse proxy configurations with Caddy.

## Architecture

Waygates uses a **file-based Caddyfile approach** for configuration management:

- **Modular Configuration**: Each proxy gets its own config file in `sites/` directory
- **Main Caddyfile**: Imports all proxy configs and catchall with `import sites/*.conf`
- **Enable/Disable**: Proxies are disabled by renaming to `.conf.disabled`
- **Atomic Writes**: All file operations use temp-file + rename pattern
- **Backup/Restore**: Automatic backups before sync with rollback on failure
- **Combined Container**: Backend + Caddy run in a single container for production

## Features

- User authentication (JWT with refresh tokens)
- RBAC permissions via goauth library
- Proxy management (reverse proxy, redirect, static file server)
- Automatic Caddyfile generation
- Periodic sync between database and Caddy (1 minute interval)
- Configurable 404 behavior (default page or redirect)
- Health checks and status endpoints

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

# Database (PostgreSQL recommended for production)
DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=waygates

# Caddy Configuration (File-based)
CADDY_BASE_PATH=/etc/caddy          # Base directory for Caddy configs
CADDY_CADDYFILE_PATH=/etc/caddy/Caddyfile  # Main Caddyfile path
CADDY_BINARY=/usr/bin/caddy         # Caddy binary location
CADDY_EMAIL=admin@example.com       # Email for ACME certificates

# Cloudflare (for wildcard SSL)
CLOUDFLARE_EMAIL=your-email@example.com
CLOUDFLARE_API_TOKEN=your_api_token

# ACME Configuration
ACME_CA_URL=https://acme-v02.api.letsencrypt.org/directory

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
├── Caddyfile           # Main config (imports sites/*.conf and catchall.conf)
├── catchall.conf       # Default 404 handler
├── sites/              # Individual proxy configs
│   ├── 1_myapp.conf
│   ├── 2_api.conf
│   └── 3_blog.conf.disabled  # Disabled proxy
└── backup/             # Timestamped backups
    └── 20240115_120000/
```

## Generated Caddyfile Example

Main Caddyfile:
```caddyfile
{
    email admin@example.com
    acme_ca https://acme-v02.api.letsencrypt.org/directory
    acme_dns cloudflare {$CLOUDFLARE_API_TOKEN}
    admin off
}

import sites/*.conf
import catchall.conf
```

Proxy config (`sites/1_myapp.conf`):
```caddyfile
# Proxy: My App (ID: 1)
# Type: reverse_proxy
myapp.example.com {
    reverse_proxy http://backend:8080 {
        header_up Host {upstream_hostport}
        header_up X-Real-IP {remote_host}
        header_up X-Forwarded-For {remote_host}
        header_up X-Forwarded-Proto {scheme}
    }
}
```

## Development

### Prerequisites

- Go 1.21+
- PostgreSQL (or SQLite for development)
- Caddy 2.10+

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
2. **Initial Sync**: On startup, waits 5 seconds then performs full sync
3. **Orphan Cleanup**: Removes config files that don't match database records
4. **Backup/Restore**: Creates backup before sync, restores on failure
5. **Caddy Reload**: Validates config then reloads Caddy after changes

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
- For SQLite, check `DB_PATH` points to a writable location

**"Caddy validation failed"**
- Check generated Caddyfile syntax: `caddy validate --config /etc/caddy/Caddyfile`
- Review logs for specific validation errors

**"Sync failed"**
- Check `/etc/caddy/backup/` for previous working configs
- Verify `CADDY_BINARY` points to correct caddy executable
- Ensure `CADDY_BASE_PATH` directory is writable

## License

See project root LICENSE file.
