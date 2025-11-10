# Let's Encrypt Configuration Guide

This guide explains how to configure Caddy to use Let's Encrypt staging or production environments.

## Overview

Let's Encrypt provides two environments:

1. **Production** - Issues real, trusted certificates
   - URL: `https://acme-v02.api.letsencrypt.org/directory`
   - Rate limits: 50 certificates per domain per week
   - Use for live deployments

2. **Staging** - Issues test certificates (not trusted by browsers)
   - URL: `https://acme-staging-v02.api.letsencrypt.org/directory`
   - Higher rate limits for testing
   - **Use for testing before production deployment**

## Configuration

The ACME CA server is configured via the `ACME_CA_URL` environment variable in your `.env` file.

### For Production (Default)

```env
ACME_CA_URL=https://acme-v02.api.letsencrypt.org/directory
```

Or simply omit the variable - production is the default:
```env
# ACME_CA_URL not set = production
```

### For Staging/Testing

```env
ACME_CA_URL=https://acme-staging-v02.api.letsencrypt.org/directory
```

## Usage Workflow

### 1. Initial Setup - Use Staging

When setting up for the first time, **always use staging** to avoid hitting rate limits:

```bash
# 1. Copy .env.example to .env
cp .env.example .env

# 2. Edit .env and set to staging
vim .env
```

Set in `.env`:
```env
CLOUDFLARE_EMAIL=your-email@example.com
CLOUDFLARE_API_TOKEN=your_actual_token
ACME_CA_URL=https://acme-staging-v02.api.letsencrypt.org/directory
```

### 2. Test Your Configuration

```bash
# Start Caddy with staging
docker-compose up -d caddy

# Check Caddy logs
docker-compose logs -f caddy

# Create a test proxy via API
curl -X POST "http://localhost:8080/api/proxies" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "reverse_proxy",
    "name": "Test",
    "hostname": "test.yourdomain.com",
    "upstreams": [{"host": "localhost", "port": 8080, "scheme": "http"}]
  }'

# Check if certificate was issued (staging cert)
curl -vI https://test.yourdomain.com 2>&1 | grep "issuer"
# Should show: issuer: CN=Fake LE Intermediate X1
```

### 3. Switch to Production

Once everything works with staging:

```bash
# 1. Stop Caddy
docker-compose down

# 2. Clear Caddy data (removes staging certificates)
rm -rf caddy-data/*

# 3. Update .env to use production
vim .env
```

Set in `.env`:
```env
ACME_CA_URL=https://acme-v02.api.letsencrypt.org/directory
```

```bash
# 4. Restart Caddy
docker-compose up -d caddy

# 5. Verify production certificate
curl -vI https://test.yourdomain.com 2>&1 | grep "issuer"
# Should show: issuer: C=US; O=Let's Encrypt; CN=R3
```

## Why Use Staging?

Let's Encrypt has strict rate limits on production:

- **50 certificates per domain per week**
- **5 duplicate certificates per week** (same set of names)
- **300 new orders per account per 3 hours**

If you hit these limits, you're blocked for the time period. Staging has much higher limits and is perfect for:

- Initial setup and testing
- Testing configuration changes
- Development environments
- CI/CD pipelines

## Checking Current Environment

You can check which environment Caddy is using:

```bash
# Via Caddy Admin API
curl http://localhost:2019/config/ | jq '.apps.tls.automation.policies[0].issuer.ca'

# Via environment variable
docker-compose exec caddy env | grep ACME_CA_URL
```

## Troubleshooting

### Staging Certificate Shows in Browser

**Problem**: Browser shows "Not Secure" or "Fake LE Intermediate"

**Solution**: This is expected with staging! Switch to production once testing is complete.

### Rate Limit Errors

**Error**: `too many certificates already issued`

**Solution**:
1. Switch to staging immediately
2. Wait for the rate limit window to expire
3. Clear caddy-data and restart

### Mixed Certificates

**Problem**: Some domains have staging certs, others have production

**Solution**:
```bash
# Clear all certificates
docker-compose down
rm -rf caddy-data/*

# Make sure ACME_CA_URL is set correctly in .env
# Restart
docker-compose up -d
```

## Docker Compose Configuration

Make sure your `docker-compose.yml` passes the environment variable:

```yaml
services:
  caddy:
    image: caddy:latest
    environment:
      - CLOUDFLARE_EMAIL=${CLOUDFLARE_EMAIL}
      - CLOUDFLARE_API_TOKEN=${CLOUDFLARE_API_TOKEN}
      - ACME_CA_URL=${ACME_CA_URL:-https://acme-v02.api.letsencrypt.org/directory}
    # ... rest of config
```

The `${ACME_CA_URL:-https://acme-v02.api.letsencrypt.org/directory}` syntax means:
- Use `ACME_CA_URL` if set
- Otherwise default to production URL

## Best Practices

1. **Always start with staging** for new setups
2. **Test thoroughly** with staging before production
3. **Document your environment** in deployment notes
4. **Use staging in CI/CD** to avoid rate limits
5. **Clear caddy-data** when switching environments
6. **Monitor certificate expiry** (Caddy auto-renews)

## Quick Commands

```bash
# Switch to staging
echo "ACME_CA_URL=https://acme-staging-v02.api.letsencrypt.org/directory" >> .env
docker-compose restart caddy

# Switch to production
sed -i '' 's|acme-staging-v02|acme-v02|g' .env
docker-compose down
rm -rf caddy-data/*
docker-compose up -d

# Check current CA
curl -s http://localhost:2019/config/ | jq -r '.apps.tls.automation.policies[0].issuer.ca'
```

## References

- [Let's Encrypt Rate Limits](https://letsencrypt.org/docs/rate-limits/)
- [Let's Encrypt Staging Environment](https://letsencrypt.org/docs/staging-environment/)
- [Caddy ACME Documentation](https://caddyserver.com/docs/automatic-https#acme-challenges)
