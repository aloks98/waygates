# Deployment Guide

This guide covers deploying Waygates in various environments.

## Table of Contents

- [Quick Start](#quick-start)
- [Docker Compose (Recommended)](#docker-compose-recommended)
- [Manual Docker Setup](#manual-docker-setup)
- [Building from Source](#building-from-source)
- [Configuration](#configuration)
- [Caddy Setup](#caddy-setup)
- [Production Considerations](#production-considerations)
- [Upgrading](#upgrading)

---

## Quick Start

The fastest way to get Waygates running:

```bash
# Pull the image
docker pull ghcr.io/aloks98/waygates:latest

# Run with minimal config
docker run -d \
  --name waygates \
  -p 8080:8080 \
  -e DATABASE_URL=postgres://user:pass@host:5432/waygates \
  -e JWT_SECRET=your-secret-key-min-32-chars \
  ghcr.io/aloks98/waygates:latest
```

Access the UI at `http://localhost:8080`

---

## Docker Compose (Recommended)

### 1. Create Project Directory

```bash
mkdir waygates && cd waygates
```

### 2. Create docker-compose.yml

```yaml
services:
  postgres:
    image: postgres:16-alpine
    container_name: waygates-db
    restart: unless-stopped
    environment:
      POSTGRES_USER: ${DB_USER:-postgres}
      POSTGRES_PASSWORD: ${DB_PASSWORD:-postgres}
      POSTGRES_DB: ${DB_NAME:-waygates}
    volumes:
      - postgres-data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${DB_USER:-postgres} -d ${DB_NAME:-waygates}"]
      interval: 5s
      timeout: 5s
      retries: 5

  backend:
    image: ghcr.io/aloks98/waygates:latest
    container_name: waygates-backend
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      # Database
      DB_HOST: postgres
      DB_PORT: 5432
      DB_USER: ${DB_USER:-postgres}
      DB_PASSWORD: ${DB_PASSWORD:-postgres}
      DB_NAME: ${DB_NAME:-waygates}

      # Caddy Admin API
      CADDY_ADMIN_URL: http://caddy:2019

      # Authentication
      JWT_SECRET: ${JWT_SECRET}
      JWT_ACCESS_EXPIRY: 15m
      JWT_REFRESH_EXPIRY: 168h

      # First user credentials (created on startup)
      DEFAULT_USER_EMAIL: ${DEFAULT_USER_EMAIL:-admin@example.com}
      DEFAULT_USER_PASSWORD: ${DEFAULT_USER_PASSWORD:-changeme}
    depends_on:
      postgres:
        condition: service_healthy
      caddy:
        condition: service_started

  caddy:
    image: caddy:2-alpine
    container_name: waygates-caddy
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
      - "443:443/udp"
      - "2019:2019"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile:ro
      - caddy-data:/data
      - caddy-config:/config

volumes:
  postgres-data:
  caddy-data:
  caddy-config:
```

### 3. Create Caddyfile

```caddyfile
{
    admin 0.0.0.0:2019
}

# Default response for unmatched hosts
:80, :443 {
    respond "Not Found" 404
}
```

### 4. Create .env File

```bash
# Database
DB_USER=postgres
DB_PASSWORD=your-secure-password
DB_NAME=waygates

# Authentication (REQUIRED - min 32 characters)
JWT_SECRET=your-very-long-secret-key-at-least-32-characters

# Default admin user
DEFAULT_USER_EMAIL=admin@example.com
DEFAULT_USER_PASSWORD=your-admin-password
```

### 5. Start Services

```bash
docker compose up -d
```

### 6. Access

- **UI**: http://localhost:8080
- **API**: http://localhost:8080/api
- **Caddy Admin**: http://localhost:2019

---

## Manual Docker Setup

### Prerequisites

- PostgreSQL 14+ database
- Caddy 2.x server (or compatible reverse proxy)

### Run Waygates

```bash
docker run -d \
  --name waygates \
  -p 8080:8080 \
  -e DB_HOST=your-postgres-host \
  -e DB_PORT=5432 \
  -e DB_USER=postgres \
  -e DB_PASSWORD=your-password \
  -e DB_NAME=waygates \
  -e CADDY_ADMIN_URL=http://your-caddy-host:2019 \
  -e JWT_SECRET=your-secret-key-min-32-chars \
  -e DEFAULT_USER_EMAIL=admin@example.com \
  -e DEFAULT_USER_PASSWORD=changeme \
  ghcr.io/aloks98/waygates:latest
```

---

## Building from Source

### Prerequisites

- Go 1.25+
- Node.js 22+ with pnpm
- Docker (optional, for containerized build)

### Option 1: Docker Build

```bash
# Clone repository
git clone https://github.com/aloks98/waygates.git
cd waygates

# Build image
docker build -f Dockerfile.backend -t waygates:local .
```

### Option 2: Local Build

```bash
# Clone repository
git clone https://github.com/aloks98/waygates.git
cd waygates

# Build UI
cd ui
pnpm install
pnpm build
cd ..

# Build backend
go build -o bin/waygates ./backend/cmd/server

# Run
./bin/waygates
```

---

## Configuration

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DB_HOST` | Yes | - | PostgreSQL host |
| `DB_PORT` | No | `5432` | PostgreSQL port |
| `DB_USER` | Yes | - | Database user |
| `DB_PASSWORD` | Yes | - | Database password |
| `DB_NAME` | No | `waygates` | Database name |
| `JWT_SECRET` | Yes | - | JWT signing key (min 32 chars) |
| `JWT_ACCESS_EXPIRY` | No | `15m` | Access token expiry |
| `JWT_REFRESH_EXPIRY` | No | `168h` | Refresh token expiry (7 days) |
| `CADDY_ADMIN_URL` | Yes | - | Caddy Admin API URL |
| `CADDY_TIMEOUT` | No | `10s` | Caddy API timeout |
| `SERVER_HOST` | No | `0.0.0.0` | Server bind address |
| `SERVER_PORT` | No | `8080` | Server port |
| `DEFAULT_USER_EMAIL` | No | - | Auto-create admin user email |
| `DEFAULT_USER_PASSWORD` | No | - | Auto-create admin user password |
| `DEFAULT_USER_NAME` | No | `Admin` | Auto-create admin user name |
| `BCRYPT_COST` | No | `12` | Password hashing cost |
| `CORS_ORIGINS` | No | `*` | Allowed CORS origins |
| `LOG_LEVEL` | No | `info` | Log level (debug/info/warn/error) |
| `LOG_FORMAT` | No | `json` | Log format (json/console) |
| `UI_ENABLED` | No | `true` | Serve UI static files |
| `UI_PATH` | No | `/app/ui` | Path to UI static files |

### Database Connection

You can use either individual variables or a connection URL:

```bash
# Individual variables
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=secret
DB_NAME=waygates

# Or connection URL
DATABASE_URL=postgres://postgres:secret@localhost:5432/waygates?sslmode=disable
```

---

## Caddy Setup

Waygates manages Caddy reverse proxy configurations via the Caddy Admin API. You have several options:

### Option 1: Official Caddy Image (Simple)

Use the official Caddy image if you don't need custom plugins:

```yaml
caddy:
  image: caddy:2-alpine
  ports:
    - "80:80"
    - "443:443"
    - "2019:2019"
  volumes:
    - ./Caddyfile:/etc/caddy/Caddyfile:ro
    - caddy-data:/data
```

### Option 2: Custom Caddy with Plugins

Build a custom Caddy image with plugins (e.g., Cloudflare DNS for wildcard certs):

**Dockerfile.caddy:**
```dockerfile
FROM caddy:2-builder AS builder

RUN xcaddy build \
    --with github.com/caddy-dns/cloudflare

FROM caddy:2

COPY --from=builder /usr/bin/caddy /usr/bin/caddy
```

**Build and use:**
```bash
docker build -f Dockerfile.caddy -t caddy-custom .
```

**docker-compose.yml:**
```yaml
caddy:
  build:
    context: .
    dockerfile: Dockerfile.caddy
  environment:
    - CLOUDFLARE_API_TOKEN=${CLOUDFLARE_API_TOKEN}
```

### Option 3: Existing Caddy Server

If you have an existing Caddy server, just point Waygates to its Admin API:

```bash
CADDY_ADMIN_URL=http://192.168.1.100:2019
```

Ensure the Caddy Admin API is accessible from the Waygates container.

### Default 404 Page

Waygates includes a branded 404 page for unconfigured routes. To enable it, you need a wildcard domain with DNS challenge for SSL certificates.

**Enable in Caddyfile:**

Edit `conf/Caddyfile` and uncomment the wildcard block, replacing with your domain:

```caddyfile
*.example.com {
    tls {
        dns cloudflare {$CLOUDFLARE_API_TOKEN}
    }

    root * /etc/caddy/templates
    templates
    rewrite * /404.html
    file_server
}
```

This serves the branded 404 page for any `*.example.com` subdomain that doesn't have a configured proxy route.

**Why a wildcard domain is required:**
- SSL certificates require a fully qualified domain name
- A catch-all like `:443` can't obtain certificates
- The wildcard certificate covers all subdomains under your domain

The template is located at `conf/templates/404.html` and can be customized.

---

## Production Considerations

### Security

1. **Use strong secrets:**
   ```bash
   # Generate a secure JWT secret
   openssl rand -base64 48
   ```

2. **Restrict Caddy Admin API:**
   ```caddyfile
   {
       admin 127.0.0.1:2019  # Only localhost
   }
   ```
   Or use network policies to restrict access.

3. **Use HTTPS for Waygates UI:**
   Put Waygates behind Caddy with TLS:
   ```caddyfile
   waygates.example.com {
       reverse_proxy waygates:8080
   }
   ```

4. **Change default credentials** after first login.

### Database

1. **Use a managed database** (RDS, Cloud SQL) for production.

2. **Enable SSL:**
   ```bash
   DATABASE_URL=postgres://user:pass@host:5432/waygates?sslmode=require
   ```

3. **Regular backups** - Waygates stores all proxy configurations in PostgreSQL.

### High Availability

For HA deployments:

1. **Database:** Use PostgreSQL with replication or managed service
2. **Waygates:** Run multiple replicas behind a load balancer
3. **Caddy:** Each Caddy instance syncs from Waygates on startup

### Monitoring

1. **Health endpoint:** `GET /health`
2. **Logs:** JSON format by default, compatible with log aggregators
3. **Metrics:** Caddy exposes Prometheus metrics at `/metrics`

### Resource Limits

```yaml
backend:
  image: ghcr.io/aloks98/waygates:latest
  deploy:
    resources:
      limits:
        cpus: '1'
        memory: 512M
      reservations:
        cpus: '0.25'
        memory: 128M
```

---

## Upgrading

### Using Docker Compose

```bash
# Pull latest image
docker compose pull

# Restart with new image
docker compose up -d

# Check logs
docker compose logs -f backend
```

### Specific Version

```yaml
backend:
  image: ghcr.io/aloks98/waygates:1.2.3  # Pin to version
```

### Database Migrations

Migrations run automatically on startup. The backend will:

1. Check current migration version
2. Apply any pending migrations
3. Start the server

To check migration status:
```bash
docker compose exec backend /app/server migrate status
```

---

## Troubleshooting

### Container won't start

```bash
# Check logs
docker compose logs backend

# Common issues:
# - Database not ready: Check postgres health
# - Invalid JWT_SECRET: Must be at least 32 characters
# - Caddy unreachable: Check CADDY_ADMIN_URL
```

### Database connection errors

```bash
# Test connection
docker compose exec backend sh -c 'nc -zv $DB_HOST $DB_PORT'

# Check credentials
docker compose exec postgres psql -U postgres -c '\l'
```

### Caddy sync issues

```bash
# Check Caddy Admin API
curl http://localhost:2019/config/

# Check Waygates logs for sync errors
docker compose logs backend | grep -i caddy
```

### Reset everything

```bash
# Stop and remove everything
docker compose down -v

# Start fresh
docker compose up -d
```

---

## Support

- **Issues:** [GitHub Issues](https://github.com/aloks98/waygates/issues)
- **Documentation:** [docs/](./docs/)
