# Waygates

A modern reverse proxy manager with a React UI and Go backend. Manage your Caddy reverse proxy configurations through a clean web interface.

## Features

- Web UI for managing reverse proxy configurations
- REST API for automation
- JWT-based authentication with RBAC
- Automatic sync with Caddy Admin API
- Support for reverse proxy, redirect, and static file serving
- PostgreSQL for persistent storage
- Single Docker image (backend + UI)

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
- Caddy 2.x (or use official caddy image)

## Project Structure

```
waygates/
├── backend/                   # Go API server
│   ├── cmd/server/            # Main entry point
│   ├── internal/              # Internal packages
│   │   ├── api/               # HTTP handlers and routes
│   │   ├── caddy/             # Caddy Admin API client
│   │   ├── config/            # Configuration
│   │   ├── models/            # Data models
│   │   ├── repository/        # Database layer
│   │   └── service/           # Business logic
│   ├── migrations/            # Database migrations
│   └── rbac.yaml              # Role-based access control config
├── ui/                        # React frontend (Rsbuild)
│   ├── src/                   # Source code
│   └── dist/                  # Built static files (generated)
├── conf/
│   ├── Caddyfile              # Main Caddy configuration
│   └── templates/
│       └── 404.html           # Custom 404 error page
├── docs/                      # API documentation
├── docker-compose.yml         # Docker services (postgres, backend, caddy)
├── Dockerfile.backend         # Backend + UI combined image
├── Dockerfile.caddy           # Custom Caddy image with Cloudflare plugin
├── Makefile                   # Build and deployment commands
├── .env                       # Your actual credentials (not in git)
├── .env.example               # Template for environment variables
└── README.md                  # This file
```

## Architecture

The application consists of three Docker containers:

1. **Backend** (`waygates-backend`): Go API server that serves both the REST API and the React UI static files
2. **Caddy** (`waygates-caddy`): Reverse proxy with automatic HTTPS and Cloudflare DNS integration
3. **PostgreSQL** (`waygates-db`): Database for storing proxy configurations and users

## Setup Instructions

### 1. Clone or Create the Project

```bash
cd waygates
```

### 2. Get Cloudflare API Token

1. Go to [Cloudflare API Tokens](https://dash.cloudflare.com/profile/api-tokens)
2. Click **Create Token**
3. Use the **Edit Zone DNS** template
4. Set permissions:
   - **Zone:DNS:Edit**
   - **Zone:Zone:Read**
5. Set Zone Resources:
   - Include → Specific zone → `e412.in`
6. Create token and copy it

### 3. Configure Environment Variables

```bash
# Copy the example file
cp .env.example .env

# Edit with your actual credentials
nano .env
```

Update `.env` with your values:
```env
CLOUDFLARE_EMAIL=your-email@example.com
CLOUDFLARE_API_TOKEN=your_actual_cloudflare_api_token
```

### 4. Deploy

```bash
# Full deployment (validates env, builds, starts)
make deploy

# Or step by step:
make env-check    # Verify configuration
make build        # Build Caddy image
make up          # Start containers
```

### 5. Verify

```bash
# Check container status
make status

# Watch logs
make logs-follow

# Check admin API
curl http://localhost:2019/config/
```

## Makefile Commands

### Deployment

| Command | Description |
|---------|-------------|
| `make help` | Show all available commands |
| `make deploy` | Full deployment pipeline (env-check → build → up) |
| `make env-check` | Verify .env file is configured |
| `make build` | Build Caddy Docker image with Cloudflare plugin |
| `make up` | Start containers in detached mode |
| `make down` | Stop and remove containers |
| `make restart` | Restart containers |

### Monitoring

| Command | Description |
|---------|-------------|
| `make logs` | Show recent logs (last 100 lines) |
| `make logs-follow` | Follow logs in real-time |
| `make status` | Show container status and ports |

### Maintenance

| Command | Description |
|---------|-------------|
| `make validate` | Validate Caddyfile syntax |
| `make clean` | Remove containers, volumes, and images |
| `make rebuild` | Clean rebuild from scratch |

## Configuration

### Caddyfile

The main configuration file is located at `conf/Caddyfile`.

#### Current Configuration

```caddyfile
{
    admin 0.0.0.0:2019
    email {$CLOUDFLARE_EMAIL}
}

*.caddy.e412.in {
    tls {
        dns cloudflare {$CLOUDFLARE_API_TOKEN}
    }

    # Serves custom 404 page
    root * /etc/caddy/templates
    handle {
        templates
        rewrite * /404.html
        file_server
    }
}
```

#### Adding New Services

To add a reverse proxy for a specific subdomain:

```caddyfile
# Add after the *.caddy.e412.in block

app.caddy.e412.in {
    reverse_proxy localhost:3000
}

api.caddy.e412.in {
    reverse_proxy localhost:8080
}

dashboard.caddy.e412.in {
    reverse_proxy localhost:9000 {
        header_up Host {host}
        header_up X-Real-IP {remote_host}
        header_up X-Forwarded-For {remote_host}
        header_up X-Forwarded-Proto {scheme}
    }
}
```

After editing:
```bash
make restart
make logs-follow
```

### Ports Exposed

| Port | Service | Description |
|------|---------|-------------|
| 80 | HTTP | Automatic redirect to HTTPS |
| 443 | HTTPS | Secure connections (TCP) |
| 443/udp | HTTP/3 | QUIC protocol |
| 2019 | Admin API | Runtime configuration |

## Wildcard Certificate Usage

The `*.caddy.e412.in` wildcard certificate works for any single-level subdomain:

✅ **Supported:**
- `app.caddy.e412.in`
- `api.caddy.e412.in`
- `dashboard.caddy.e412.in`
- `caddy.caddy.e412.in`
- Any `*.caddy.e412.in`

❌ **Not supported (requires separate cert):**
- `api.app.caddy.e412.in` (two levels deep)
- `caddy.e412.in` (base domain, not a subdomain)

## Admin API

Access the Caddy admin API at `http://localhost:2019`

### Common API Commands

```bash
# View current configuration
curl http://localhost:2019/config/ | jq

# Reload configuration
curl -X POST http://localhost:2019/load \
  -H "Content-Type: application/json" \
  -d @conf/Caddyfile

# Check certificates
curl http://localhost:2019/pki/ca/local | jq
```

## Troubleshooting

### SSL Certificate Issues

**Error:** `no solvers available for remaining challenges`

This means DNS-01 challenge isn't working. Check:

```bash
# Verify environment variables are set
docker compose exec caddy env | grep CLOUDFLARE

# Check Cloudflare API token permissions
# Token needs: Zone:DNS:Edit and Zone:Zone:Read

# View detailed logs
make logs-follow
```

### Container Won't Start

```bash
# Validate Caddyfile syntax
make validate

# Check Docker logs
make logs

# Rebuild from scratch
make rebuild
```

### Port Already in Use

```bash
# Check what's using the port
sudo lsof -i :80
sudo lsof -i :443

# Stop other services or change ports in docker-compose.yml
```

### Environment Variables Not Loading

```bash
# Verify .env file exists and is configured
make env-check

# Check file permissions
ls -la .env

# Restart containers to reload env
make restart
```

## Common Workflows

### Daily Operations

```bash
# Check if everything is running
make status

# View recent activity
make logs

# Restart after config changes
make restart
```

### Adding a New Service

1. Edit `conf/Caddyfile`
2. Add your reverse proxy configuration
3. Restart: `make restart`
4. Monitor: `make logs-follow`

### Updating Caddy or Plugins

1. Edit `Dockerfile.caddy` if needed
2. Rebuild: `make rebuild`
3. Verify: `make status`

### Complete Reset

```bash
# Remove everything and start fresh
make clean
make deploy
```

## Security Notes

- ✅ `.env` file is gitignored (contains secrets)
- ✅ SSL certificates stored in `caddy-data/` (gitignored)
- ✅ Admin API exposed only on localhost (bind to 127.0.0.1 in production)
- ⚠️ Use firewall rules to restrict port 2019 access

## Backup

Important files to backup:

```bash
# SSL certificates and private keys
caddy-data/

# Configuration
conf/Caddyfile

# Environment (encrypted!)
.env
```

## Documentation

- **[Deployment Guide](docs/DEPLOYMENT.md)** - Full deployment instructions
- **[API Authentication](docs/API_AUTH.md)** - Authentication endpoints
- **[API Proxy](docs/API_PROXY.md)** - Proxy management endpoints
- **[OpenAPI Spec](docs/openapi.yaml)** - API specification

## Resources

- [Caddy Documentation](https://caddyserver.com/docs/)
- [Caddy Cloudflare DNS Plugin](https://github.com/caddy-dns/cloudflare)

## License

MIT License
