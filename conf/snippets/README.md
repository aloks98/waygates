# Caddy Snippets

This directory contains reusable Caddy configuration snippets that can be imported into your Caddyfile.

## Available Snippets

### security.caddy

Blocks common web exploits and attacks including:
- SQL injection attempts
- File injection/traversal attempts
- XSS (Cross-Site Scripting) attacks
- PHP global variable exploits
- Spam keywords in query strings
- Malicious user agents

**Usage:**

```caddyfile
example.com {
    # Import security rules BEFORE other handlers
    import snippets/security

    # Your normal configuration
    reverse_proxy backend:8080
}
```

**What gets blocked:**

1. **SQL Injections:**
   - `union select`, `concat()` patterns
   - Common SQL injection query parameters

2. **File Injections:**
   - Path traversal attempts (`../`, `../../`)
   - Remote file inclusion (`http://`, `https://` in query)

3. **Common Exploits:**
   - XSS attempts (`<script>` tags)
   - PHP globals (`GLOBALS=`, `_REQUEST=`)
   - Base64 encoding attempts
   - `/proc/self/environ` access

4. **Spam Keywords:**
   - Pharmaceutical spam (viagra, cialis, etc.)
   - Common spam terms

5. **Malicious User Agents:**
   - Download managers (GetRight, Go!Zilla)
   - Bots (TurnitinBot, GrabNet)
   - Known malicious agents

**Response:**

All blocked requests receive a `403 Forbidden` response with a descriptive message indicating what was detected.

## Backend Integration

When the API receives a proxy creation/update request with `block_exploits: true`, the backend should inject the import statement into the generated Caddy configuration:

```json
{
  "type": "reverse_proxy",
  "hostname": "api.example.com",
  "block_exploits": true,
  ...
}
```

Generated Caddyfile:
```caddyfile
api.example.com {
    import snippets/security  # Added when block_exploits = true

    reverse_proxy 192.168.1.100:8080
}
```

## Testing the Security Rules

You can test the security rules with curl:

```bash
# Should be blocked (SQL injection)
curl "https://example.com/?id=1 union select * from users"

# Should be blocked (XSS)
curl "https://example.com/?search=<script>alert(1)</script>"

# Should be blocked (Path traversal)
curl "https://example.com/?file=../../../etc/passwd"

# Should be blocked (Bad user agent)
curl -A "libwww-perl/6.0" "https://example.com/"

# Should work normally
curl "https://example.com/?search=normal query"
```

## Customization

To customize the security rules:

1. Edit `conf/snippets/security.caddy`
2. Validate the configuration: `make validate`
3. Reload Caddy: `make restart`

The changes will apply to all proxies that have `block_exploits: true` enabled.
