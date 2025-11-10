# TLS Insecure Skip Verify

## Overview

The `tls_insecure_skip_verify` option allows proxying to HTTPS backends that use self-signed certificates or certificates from non-public Certificate Authorities.

## When to Use

Use this option when proxying to:
- **Homelab services** with self-signed certificates
- **Internal corporate services** with internal CA certificates
- **Development environments** where certificate validation is not critical
- **Temporary setups** during testing or migration

## Security Warning

⚠️ **WARNING**: This option disables TLS certificate verification, making the connection vulnerable to man-in-the-middle attacks.

**Never use this for:**
- Public-facing services
- Untrusted networks
- Production environments with sensitive data
- Services you don't control

## Usage

### API Request

```json
{
  "type": "reverse_proxy",
  "name": "Internal Service",
  "hostname": "internal.example.com",
  "upstreams": [
    {
      "host": "192.168.1.100",
      "port": 8443,
      "scheme": "https"
    }
  ],
  "tls_insecure_skip_verify": true,
  "block_exploits": true
}
```

### cURL Example

```bash
curl -X POST "http://localhost:8080/api/proxies" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "reverse_proxy",
    "name": "Internal HTTPS Service",
    "hostname": "internal.example.com",
    "upstreams": [
      {
        "host": "192.168.1.100",
        "port": 8443,
        "scheme": "https"
      }
    ],
    "tls_insecure_skip_verify": true,
    "block_exploits": true
  }'
```

## Generated Caddy Configuration

When `tls_insecure_skip_verify` is enabled, the following transport configuration is added to the reverse_proxy handler:

```json
{
  "handler": "reverse_proxy",
  "transport": {
    "protocol": "http",
    "tls": {
      "insecure_skip_verify": true
    }
  },
  "upstreams": [...]
}
```

## Alternative: Use Proper Certificates

Instead of skipping verification, consider:

1. **Install CA certificate** on the Caddy server if using internal CA
2. **Use Let's Encrypt** for public services
3. **Use mkcert** for local development
4. **Configure internal CA** properly

### Example: Using mkcert for Development

```bash
# Install mkcert
brew install mkcert  # macOS
# or
apt install mkcert   # Linux

# Install local CA
mkcert -install

# Generate certificate
mkcert internal.local "*.internal.local"

# Use the generated cert on your backend service
```

## Database Schema

The feature adds a new column to the `proxies` table:

```sql
ALTER TABLE proxies ADD COLUMN tls_insecure_skip_verify BOOLEAN NOT NULL DEFAULT FALSE;
```

## Default Value

- **Default**: `false` (certificate verification enabled)
- Only set to `true` when explicitly needed
- Applies to all upstreams in the proxy configuration

## Best Practices

1. ✅ **Document why** you're skipping verification in the proxy name or description
2. ✅ **Use only for trusted internal networks**
3. ✅ **Plan to replace** with proper certificates when possible
4. ✅ **Audit regularly** which proxies have this enabled
5. ❌ **Don't use as default** - always prefer proper certificate validation
6. ❌ **Don't use on public networks** - risk of MITM attacks

## Troubleshooting

### Error: "x509: certificate signed by unknown authority"

This error occurs when the backend uses a self-signed certificate. Solution:
- Enable `tls_insecure_skip_verify: true` for trusted internal services
- OR install the CA certificate properly

### Error: "x509: certificate has expired"

The backend certificate has expired. Solution:
- Renew the backend certificate
- Or temporarily enable `tls_insecure_skip_verify: true`

### Connection still fails with skip verify enabled

Check:
1. Backend is actually using HTTPS (not HTTP)
2. Port is correct (usually 443 or 8443)
3. Firewall allows the connection
4. Backend service is running

## References

- [Caddy Reverse Proxy Transport](https://caddyserver.com/docs/caddyfile/directives/reverse_proxy#transports)
- [Caddy TLS Config](https://caddyserver.com/docs/json/apps/http/servers/tls_connection_policies/)
- [Go TLS InsecureSkipVerify](https://pkg.go.dev/crypto/tls#Config)
