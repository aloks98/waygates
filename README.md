# Homelab Reverse Proxy

A Docker-based Caddy reverse proxy setup with automatic wildcard SSL certificates using Cloudflare DNS challenge.

## Features

- 🔒 Automatic wildcard SSL certificates via Let's Encrypt
- ☁️ Cloudflare DNS-01 challenge for `*.caddy.e412.in`
- 🎨 Custom 404 error page with templates
- 🔧 Admin API for runtime configuration
- 🐳 Docker Compose for easy deployment
- 📝 Makefile for simplified operations

## Prerequisites

- Docker (with Docker Compose V2 integrated)
- A domain managed by Cloudflare (e.g., `e412.in`)
- Cloudflare API token with DNS edit permissions

> **Note:** This project uses `docker compose` (V2) instead of the deprecated `docker-compose` (V1).

## Project Structure

```
homelab-proxy/
├── conf/
│   ├── Caddyfile              # Main Caddy configuration
│   └── templates/
│       └── 404.html           # Custom 404 error page
├── caddy-data/                # SSL certificates (auto-generated)
├── caddy_config/              # Runtime config (auto-generated)
├── docker-compose.yml         # Docker services definition
├── Dockerfile.caddy           # Custom Caddy image with Cloudflare plugin
├── Makefile                   # Build and deployment commands
├── .env                       # Your actual credentials (not in git)
├── .env.example               # Template for environment variables
├── .gitignore                 # Git ignore rules
└── README.md                  # This file
```

## Setup Instructions

### 1. Clone or Create the Project

```bash
cd homelab-proxy
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

## Resources

- [Caddy Documentation](https://caddyserver.com/docs/)
- [Caddy Cloudflare DNS Plugin](https://github.com/caddy-dns/cloudflare)
- [Caddyfile Syntax](https://caddyserver.com/docs/caddyfile)
- [Cloudflare API Tokens](https://dash.cloudflare.com/profile/api-tokens)

## License

This is a personal homelab setup. Feel free to use and modify as needed.
