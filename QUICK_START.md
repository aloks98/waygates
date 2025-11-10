# Quick Start Guide

## Environment Setup

### 1. Copy Environment File

```bash
cp .env.example .env
```

### 2. Configure Cloudflare

Edit `.env` and set your Cloudflare credentials:

```env
CLOUDFLARE_EMAIL=your-email@example.com
CLOUDFLARE_API_TOKEN=your_actual_token_here
```

Get your API token from: https://dash.cloudflare.com/profile/api-tokens

Required permissions:
- Zone:DNS:Edit
- Zone:Zone:Read

### 3. Choose Let's Encrypt Environment

#### For Testing (Recommended for first time)

```env
ACME_CA_URL=https://acme-staging-v02.api.letsencrypt.org/directory
```

#### For Production

```env
ACME_CA_URL=https://acme-v02.api.letsencrypt.org/directory
```

Or leave it unset (defaults to production).

## Quick Commands

### Testing with Staging

```bash
# Set staging in .env
ACME_CA_URL=https://acme-staging-v02.api.letsencrypt.org/directory

# Start services
docker-compose up -d

# Check logs
docker-compose logs -f caddy

# Test the API
curl http://localhost:8080/api/health
```

### Switch to Production

```bash
# Stop services
docker-compose down

# Clear staging certificates
rm -rf caddy-data/*

# Update .env
ACME_CA_URL=https://acme-v02.api.letsencrypt.org/directory

# Start with production
docker-compose up -d
```

### Start Backend API

```bash
# Start the backend
go run backend/cmd/server/main.go

# Or use make (if available)
make run
```

## Testing the API

```bash
# Health check
curl http://localhost:8080/api/health

# List proxies
curl http://localhost:8080/api/proxies

# Create a test proxy
curl -X POST "http://localhost:8080/api/proxies" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "reverse_proxy",
    "name": "Test App",
    "hostname": "test.yourdomain.com",
    "upstreams": [
      {
        "host": "localhost",
        "port": 8080,
        "scheme": "http"
      }
    ],
    "block_exploits": true
  }'
```

## Important Notes

- **Always use staging first** to avoid Let's Encrypt rate limits
- **Clear caddy-data/** when switching between staging and production
- Staging certificates will show as "Not Secure" in browsers (expected)
- Production certificates are trusted and valid

## Next Steps

1. Read [LETSENCRYPT_SETUP.md](docs/LETSENCRYPT_SETUP.md) for detailed information
2. See [CURL_EXAMPLES.md](backend/CURL_EXAMPLES.md) for API testing examples
3. Check [BACKEND_DESIGN.md](BACKEND_DESIGN.md) for architecture details
