# Deployment Guide

This guide covers deploying Waygates in various environments.

## Table of Contents

- [Quick Start](#quick-start)
- [Docker Compose (Recommended)](#docker-compose-recommended)
- [TLS Configuration](#tls-configuration)
- [Configuration Reference](#configuration-reference)
- [Production Considerations](#production-considerations)
- [Upgrading](#upgrading)
- [Troubleshooting](#troubleshooting)
- [Testing](#testing)

---

## Quick Start

```bash
# Pull the image
docker pull ghcr.io/aloks98/waygates:latest

# Run with Docker Compose
curl -O https://raw.githubusercontent.com/aloks98/waygates/main/docker-compose.yml
curl -O https://raw.githubusercontent.com/aloks98/waygates/main/.env.example
cp .env.example .env

# Edit .env with your settings
nano .env

# Start
docker compose up -d
```

Access the UI at `http://localhost:8080`

---

## Docker Compose (Recommended)

### 1. Create docker-compose.yml

```yaml
services:
  postgres:
    image: postgres:16-alpine
    container_name: waygates-db
    restart: unless-stopped
    environment:
      POSTGRES_USER: ${DB_USER:-waygates}
      POSTGRES_PASSWORD: ${DB_PASSWORD}
      POSTGRES_DB: ${DB_NAME:-waygates}
    volumes:
      - postgres-data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${DB_USER:-waygates}"]
      interval: 5s
      timeout: 5s
      retries: 5

  waygates:
    image: ghcr.io/aloks98/waygates:latest
    container_name: waygates
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
      - "8080:8080"
    env_file:
      - .env
    environment:
      # Override DB_HOST for Docker networking
      DB_HOST: postgres
    volumes:
      - caddy-data:/data
      - caddy-config:/config
      - caddy-etc:/etc/caddy
    depends_on:
      postgres:
        condition: service_healthy

volumes:
  postgres-data:
  caddy-data:
  caddy-config:
  caddy-etc:
```

### 2. Create .env File

```bash
cp .env.example .env
nano .env  # Edit with your settings
```

Required variables:
- `DB_PASSWORD` - Database password
- `JWT_SECRET` - Min 32 characters for JWT signing

See `.env.example` for all available options including TLS/ACME providers.

### 3. Start Services

```bash
docker compose up -d
```

---

## TLS Configuration

Waygates supports automatic TLS certificates via multiple ACME providers. Set `CADDY_ACME_PROVIDER` to enable.

### Option 1: No HTTPS (Development)

```env
CADDY_ACME_PROVIDER=off
```

### Option 2: HTTP Challenge

Requires ports 80 and 443 to be publicly accessible:

```env
CADDY_ACME_PROVIDER=http
CADDY_EMAIL=admin@example.com
```

### Option 3: DNS Challenge

For wildcard certificates or when ports 80/443 are not publicly accessible.

#### Cloudflare

```env
CADDY_ACME_PROVIDER=cloudflare
CADDY_EMAIL=admin@example.com
CLOUDFLARE_API_TOKEN=your-api-token
```

Get token from: https://dash.cloudflare.com/profile/api-tokens
Required permissions: Zone:DNS:Edit, Zone:Zone:Read

#### AWS Route53

```env
CADDY_ACME_PROVIDER=route53
CADDY_EMAIL=admin@example.com
AWS_ACCESS_KEY_ID=your-access-key
AWS_SECRET_ACCESS_KEY=your-secret-key
```

#### DigitalOcean

```env
CADDY_ACME_PROVIDER=digitalocean
CADDY_EMAIL=admin@example.com
DO_AUTH_TOKEN=your-token
```

#### Other Providers

See `.env.example` for full list: Hetzner, Porkbun, Vultr, Namecheap, OVH, Azure, DuckDNS.

---

## Configuration Reference

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DB_HOST` | Yes | - | PostgreSQL host |
| `DB_PORT` | No | `5432` | PostgreSQL port |
| `DB_USER` | No | `waygates` | Database user |
| `DB_PASSWORD` | Yes | - | Database password |
| `DB_NAME` | No | `waygates` | Database name |
| `JWT_SECRET` | Yes | - | JWT signing key (min 32 chars) |
| `JWT_ACCESS_EXPIRY` | No | `15m` | Access token expiry |
| `JWT_REFRESH_EXPIRY` | No | `168h` | Refresh token expiry |
| `CADDY_ACME_PROVIDER` | No | `off` | ACME provider |
| `CADDY_EMAIL` | No | - | Email for ACME certs |
| `SERVER_HOST` | No | `0.0.0.0` | Backend bind address |
| `SERVER_PORT` | No | `8080` | Backend port |
| `DEFAULT_USER_EMAIL` | No | - | Auto-create admin email |
| `DEFAULT_USER_PASSWORD` | No | - | Auto-create admin password |
| `BCRYPT_COST` | No | `12` | Password hashing cost |
| `LOG_LEVEL` | No | `info` | Log level |
| `LOG_FORMAT` | No | `json` | Log format (json/console) |

### Ports

| Port | Protocol | Description |
|------|----------|-------------|
| 80 | TCP | HTTP (redirect to HTTPS when TLS enabled) |
| 443 | TCP/UDP | HTTPS + HTTP/3 |
| 8080 | TCP | Backend API + UI |
| Custom | TCP/UDP | L4 proxy ports (configured per proxy) |

### L4 Proxy Ports

L4 (TCP/UDP) proxies use custom ports that must be exposed in your Docker configuration. Add them to your `docker-compose.yml`:

```yaml
waygates:
  ports:
    - "80:80"
    - "443:443"
    - "8080:8080"
    # L4 Proxy ports - add as needed
    - "5432:5432"      # PostgreSQL proxy
    - "2222:2222"      # SSH gateway
    - "3306:3306"      # MySQL proxy
    - "6379:6379"      # Redis proxy
    - "25565:25565"    # Minecraft server
    - "53:53/udp"      # DNS (UDP)
```

**Note:** You only need to expose ports for L4 proxies you've configured in Waygates.

### Volumes

| Path | Description |
|------|-------------|
| `/data` | Caddy data (certificates, etc.) |
| `/config` | Caddy config |
| `/etc/caddy` | Caddyfile and site configs |

---

## Production Considerations

### Security

1. **Use strong secrets:**
   ```bash
   openssl rand -base64 48
   ```

2. **Change default credentials** after first login.

3. **Use HTTPS** - Set `CADDY_ACME_PROVIDER` to `http` or a DNS provider.

4. **Firewall rules** - Restrict access to port 8080 if needed.

### Database

1. **Use a managed database** (RDS, Cloud SQL) for production.

2. **Enable SSL:**
   ```bash
   # Add to DB connection (future enhancement)
   ```

3. **Regular backups** - Waygates stores all configurations in PostgreSQL.

### High Availability

For HA deployments:

1. **Database:** Use PostgreSQL with replication
2. **Waygates:** Run multiple replicas behind a load balancer
3. **Shared storage:** Use shared volume for `/etc/caddy` and `/data`

### Resource Limits

```yaml
waygates:
  image: ghcr.io/aloks98/waygates:latest
  deploy:
    resources:
      limits:
        cpus: '2'
        memory: 1G
      reservations:
        cpus: '0.5'
        memory: 256M
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
docker compose logs -f waygates
```

### Specific Version

```yaml
waygates:
  image: ghcr.io/aloks98/waygates:1.2.3
```

### Database Migrations

Migrations run automatically on startup.

---

## Troubleshooting

### Container won't start

```bash
# Check logs
docker compose logs waygates

# Common issues:
# - Database not ready: Check postgres health
# - Invalid JWT_SECRET: Must be at least 32 characters
# - Invalid CADDY_ACME_PROVIDER: Check spelling
```

### Database connection errors

```bash
# Test connection
docker compose exec waygates sh -c 'nc -zv $DB_HOST $DB_PORT'

# Check PostgreSQL is running
docker compose ps postgres
```

### TLS certificate issues

```bash
# Check Caddy logs
docker compose logs waygates | grep -i caddy

# Verify DNS provider credentials are set
docker compose exec waygates env | grep -E 'CADDY|CLOUDFLARE|AWS'
```

### Sync issues

```bash
# Check sync status via API
curl http://localhost:8080/api/sync/status

# Trigger manual sync
curl -X POST http://localhost:8080/api/sync/trigger \
  -H "Authorization: Bearer $TOKEN"
```

### Reset everything

```bash
# Stop and remove everything
docker compose down -v

# Start fresh
docker compose up -d
```

---

## Testing

### Proxy traffic E2E tests

`make test-traffic` builds the `waygates-test:latest` image and runs the
end-to-end traffic suite (`backend/tests/integration/traffic_*_test.go`, build
tag `traffic`). It boots the app plus backend fixtures (HTTP/HTTPS echo, TCP
echo, Postgres) on a Docker network and drives real traffic through Caddy for
both L7 and L4 proxies:

- **L7** (`TestTraffic_L7`): reverse_proxy, redirect, static, load balancing
  (round-robin), ACL basic-auth, and block_exploits. A `custom_headers` subtest
  is included but currently **skipped** pending response-header support in the
  Caddy builder.
- **L4** (`TestTraffic_L4`): `any`/TCP echo, `http` matcher, `postgres` matcher,
  TLS SNI passthrough, `remote_ip` allow/deny, and load balancing (round-robin).

Requires Docker. These tests are build-tagged with `traffic`, so they are
**excluded** from `make backend-test` and the default `go test ./...` run.

```bash
make test-traffic
```

---

## Support

- **Issues:** [GitHub Issues](https://github.com/aloks98/waygates/issues)
