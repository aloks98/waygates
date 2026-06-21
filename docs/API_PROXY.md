# Proxy Management API Specification

**Version:** 1.0
**Last Updated:** 2024-11-08

> **Note:** All endpoints in this API are protected and require a valid JWT access token. See the [Authentication API documentation](./API_AUTH.md) for details on how to obtain a token.

## Overview

The Proxy Management API provides endpoints to create, read, update, and delete proxy configurations. It supports three proxy types:

1. **Reverse Proxy** - Forward requests to backend servers (with optional load balancing)
2. **Redirect** - HTTP redirects (301, 302, 307, 308)
3. **Static** - Serve static HTML/files

## Proxy Types

### Type: `reverse_proxy`

Forward incoming requests to one or more backend servers.

**Features:**
- Single upstream (simple reverse proxy)
- Multiple upstreams (automatic load balancing)
- Custom headers
- Health checks
- WebSocket support (automatic - no configuration needed)

### Type: `redirect`

Redirect requests to another URL.

**Features:**
- Permanent redirects (301, 308)
- Temporary redirects (302, 307)
- Path preservation
- Query string handling

### Type: `static`

Serve static files or HTML templates.

**Features:**
- File server
- Template rendering
- Custom index files
- Directory browsing

---

## Base URL

```
/api/proxies
```

## Authentication

All endpoints require JWT token:

```
Authorization: Bearer <jwt_token>
```

---

## Database Schema

### Proxies Table

```sql
CREATE TABLE proxies (
    id SERIAL PRIMARY KEY,

    -- Common fields
    type VARCHAR(50) NOT NULL,  -- 'reverse_proxy', 'redirect', 'static'
    name VARCHAR(255) NOT NULL,
    hostname VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,

    -- SSL/TLS
    ssl_enabled BOOLEAN DEFAULT true,
    ssl_forced BOOLEAN DEFAULT true,

    -- Type-specific configurations (stored as JSON)
    -- For reverse_proxy type
    upstreams TEXT,  -- JSON: [{"host": "...", "port": 8080, "scheme": "http"}]
    load_balancing TEXT,  -- JSON: {"strategy": "round_robin", "health_checks": {...}}
    block_exploits BOOLEAN DEFAULT true,
    custom_headers TEXT,  -- JSON: {"request": {"X-Header": "value"}, "response": {"X-Header": "value"}} (flat {"X-Header": "value"} also accepted as request headers)

    -- For redirect type
    redirect_config TEXT,  -- JSON: {"target": "...", "status_code": 301, ...}

    -- For static type
    static_config TEXT,  -- JSON: {"root_path": "...", "index_file": "index.html", ...}

    -- Status
    is_active BOOLEAN DEFAULT true,

    -- Metadata
    created_by INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE
);

-- Indexes for better query performance
CREATE INDEX idx_proxies_hostname ON proxies(hostname);
CREATE INDEX idx_proxies_type ON proxies(type);
CREATE INDEX idx_proxies_is_active ON proxies(is_active);
CREATE INDEX idx_proxies_created_by ON proxies(created_by);
CREATE INDEX idx_proxies_created_at ON proxies(created_at);
```

### Field Descriptions

| Field | Type | Nullable | Description |
|-------|------|----------|-------------|
| `id` | INTEGER | No | Primary key, auto-increment |
| `type` | VARCHAR(50) | No | Proxy type: `reverse_proxy`, `redirect`, `static` |
| `name` | VARCHAR(255) | No | Display name for the proxy |
| `hostname` | VARCHAR(255) | No | Domain/subdomain (unique) |
| `description` | TEXT | Yes | Optional description |
| `ssl_enabled` | BOOLEAN | No | Enable SSL/TLS (default: true) |
| `ssl_forced` | BOOLEAN | No | Force HTTPS redirect (default: true) |
| `upstreams` | TEXT (JSON) | Yes | Array of upstream servers (reverse_proxy only) |
| `load_balancing` | TEXT (JSON) | Yes | Load balancing config (reverse_proxy with multiple upstreams) |
| `block_exploits` | BOOLEAN | No | Block common exploits (default: true) |
| `custom_headers` | TEXT (JSON) | Yes | Custom HTTP headers (reverse_proxy only) |
| `redirect_config` | TEXT (JSON) | Yes | Redirect configuration (redirect type only) |
| `static_config` | TEXT (JSON) | Yes | Static file server config (static type only) |
| `is_active` | BOOLEAN | No | Proxy enabled/disabled status (default: true) |
| `created_by` | INTEGER | No | User ID who created the proxy |
| `created_at` | TIMESTAMP | No | Creation timestamp |
| `updated_at` | TIMESTAMP | No | Last update timestamp |

### JSON Field Structures

**upstreams** (TEXT - JSON Array):
```json
[
  {
    "host": "192.168.100.5",
    "port": 8080,
    "scheme": "http"
  }
]
```

**load_balancing** (TEXT - JSON Object):
```json
{
  "strategy": "round_robin",
  "health_checks": {
    "enabled": true,
    "path": "/health",
    "interval": "30s",
    "timeout": "5s",
    "unhealthy_threshold": 2,
    "healthy_threshold": 2
  }
}
```

**custom_headers** (TEXT - JSON Object):

Custom headers can be set in both directions: `request` headers are sent to the
upstream, and `response` headers are returned to the client. Each direction is a
map of header name to value.

```json
{
  "request": {
    "X-Custom-Header": "value",
    "X-API-Version": "v1"
  },
  "response": {
    "X-Frame-Options": "DENY"
  }
}
```

For backward compatibility, a flat map is still accepted and is interpreted as
request headers (sent to the upstream):

```json
{
  "X-Custom-Header": "value",
  "X-API-Version": "v1"
}
```

Stored values are always normalized to the nested
`{"request": {...}, "response": {...}}` shape.

**redirect_config** (TEXT - JSON Object):
```json
{
  "target": "https://new-domain.com",
  "status_code": 301,
  "preserve_path": true,
  "preserve_query": true
}
```

**static_config** (TEXT - JSON Object):
```json
{
  "root_path": "/var/www/html",
  "index_file": "index.html",
  "browse": false,
  "try_files": ["index.html"],
  "template_rendering": false
}
```

---

## Data Models

### Proxy Object (Common Fields)

```json
{
  "id": 1,
  "type": "reverse_proxy",  // "reverse_proxy", "redirect", "static"
  "name": "My Proxy",
  "hostname": "app.example.com",
  "description": "Optional description",
  "ssl_enabled": true,
  "ssl_forced": true,
  "is_active": true,
  "created_by": {
    "id": 1,
    "name": "Admin User",
    "email": "admin@example.com"
  },
  "created_at": "2024-11-08T10:30:00Z",
  "updated_at": "2024-11-08T10:30:00Z"
}
```

### Type-Specific Fields

#### Reverse Proxy Fields

```json
{
  "type": "reverse_proxy",

  // Single upstream
  "upstreams": [
    {
      "host": "192.168.100.5",
      "port": 8080,
      "scheme": "http"
    }
  ],

  // OR Multiple upstreams (triggers load balancing)
  "upstreams": [
    {
      "host": "192.168.100.5",
      "port": 8080,
      "scheme": "http"
    },
    {
      "host": "192.168.100.6",
      "port": 8080,
      "scheme": "http"
    }
  ],

  // Load balancing (only when multiple upstreams)
  "load_balancing": {
    "strategy": "round_robin",  // "round_robin", "least_conn", "ip_hash", "random"
    "health_checks": {
      "enabled": true,
      "path": "/health",
      "interval": "30s",
      "timeout": "5s",
      "unhealthy_threshold": 2,
      "healthy_threshold": 2
    }
  },

  // Features
  "block_exploits": true,
  "custom_headers": {
    "X-Custom-Header": "value"
  }
}
```

#### Redirect Fields

```json
{
  "type": "redirect",
  "redirect": {
    "target": "https://new-domain.com",
    "status_code": 301,  // 301, 302, 307, 308
    "preserve_path": true,
    "preserve_query": true
  }
}
```

#### Static Fields

```json
{
  "type": "static",
  "static": {
    "root_path": "/var/www/html",  // Path to static files
    "index_file": "index.html",     // Default file
    "browse": false,                 // Enable directory browsing
    "try_files": ["index.html"],    // Fallback files for SPA
    "template_rendering": false      // Enable Caddy templates
  }
}
```

---

## Endpoints

### 1. List All Proxies

**GET** `/api/proxies`

Lists all proxy configurations with pagination.

#### Query Parameters
...
#### Response (200 OK)
...
#### Error Responses
- **401 Unauthorized**: If the JWT token is missing or invalid.
- **500 Internal Server Error**: If there is a problem fetching the data from the database.

---

### 2. Get Proxy by ID

**GET** `/api/proxies/:id`

Retrieves the full details of a single proxy configuration.

#### Response (200 OK)
Returns the full proxy object.
```json
{
  "success": true,
  "data": {
    "id": 1,
    "type": "reverse_proxy",
    "name": "My Application",
    "hostname": "app.example.com",
    ...
  }
}
```

#### Error Responses
- **401 Unauthorized**: If the JWT token is missing or invalid.
- **404 Not Found**: If no proxy with the specified ID exists.
  ```json
  {
    "success": false,
    "error": {
      "code": "NOT_FOUND",
      "message": "Proxy not found"
    }
  }
  ```
- **500 Internal Server Error**: For any other server-side errors.

---

### 3. Create Proxy

**POST** `/api/proxies`

Creates a new proxy configuration.

#### Request Body
See the "Data Models" section for request body examples.

#### Response (201 Created)
Returns the newly created proxy object.
```json
{
  "success": true,
  "message": "Proxy created successfully",
  "data": {
    "id": 6,
    ...
  }
}
```

#### Error Responses
- **400 Bad Request**: If the request body is invalid or fails validation.
- **401 Unauthorized**: If the JWT token is missing or invalid.
- **409 Conflict**: If a proxy with the same hostname already exists.
- **500 Internal Server Error**: For database errors.
- **502 Bad Gateway**: If the configuration could not be applied to the Caddy server.
  ```json
  {
    "success": false,
    "error": {
      "code": "EXTERNAL_SERVICE_ERROR",
      "message": "caddy API error (status 400): ..."
    }
  }
  ```

---

### 4. Update Proxy

**PUT** `/api/proxies/:id`

Updates an existing proxy configuration.

#### Response (200 OK)
Returns the updated proxy object.
```json
{
  "success": true,
  "message": "Proxy updated successfully",
  "data": {
    "id": 1,
    ...
  }
}
```

#### Error Responses
- **400 Bad Request**: If the request body is invalid.
- **401 Unauthorized**: If the JWT token is missing or invalid.
- **404 Not Found**: If the proxy does not exist.
- **409 Conflict**: If the new hostname conflicts with an existing proxy.
- **500 Internal Server Error**: For database errors.
- **502 Bad Gateway**: If the configuration could not be applied to the Caddy server.

---

### 5. Delete Proxy

**DELETE** `/api/proxies/:id`

Deletes a proxy configuration from the database and the Caddy server.

#### Response (200 OK)
```json
{
  "success": true,
  "message": "Proxy deleted successfully",
  "data": null
}
```

#### Error Responses
- **401 Unauthorized**: If the JWT token is missing or invalid.
- **404 Not Found**: If the proxy does not exist.
- **500 Internal Server Error**: For any server-side errors.

---

### 6. Enable Proxy

**POST** `/api/proxies/:id/enable`

Enables a disabled proxy, applying its configuration to the Caddy server.

#### Response (200 OK)
```json
{
  "success": true,
  "message": "Proxy enabled successfully",
  "data": null
}
```

#### Error Responses
- **400 Bad Request**: If the proxy is already enabled.
- **401 Unauthorized**: If the JWT token is missing or invalid.
- **404 Not Found**: If the proxy does not exist.
- **500 Internal Server Error**: For database errors.
- **502 Bad Gateway**: If the configuration could not be applied to the Caddy server.

---

### 7. Disable Proxy

**POST** `/api/proxies/:id/disable`

Disables an active proxy, removing its configuration from the Caddy server.

#### Response (200 OK)
```json
{
  "success": true,
  "message": "Proxy disabled successfully",
  "data": null
}
```

#### Error Responses
- **400 Bad Request**: If the proxy is already disabled.
- **401 Unauthorized**: If the JWT token is missing or invalid.
- **404 Not Found**: If the proxy does not exist.
- **500 Internal Server Error**: For any server-side errors.

---

### 8. Bulk Enable Proxies

**POST** `/api/proxies/bulk/enable`

Enables multiple proxies in a single request. Requires the `proxies:update` permission.

The operation is **best-effort**: each id is processed independently and a failure on
one id never aborts the batch. The response reports the per-id outcome along with
aggregate counts.

#### Request Body
```json
{
  "ids": [1, 2, 3]
}
```

- `ids` (array of integers, required): The proxy ids to enable. Must be non-empty and
  contain at most **1000** ids.

#### Response (200 OK)
```json
{
  "success": true,
  "message": "Bulk enable completed",
  "data": {
    "requested": 3,
    "succeeded": 2,
    "failed": 1,
    "results": [
      { "id": 1, "status": "ok" },
      { "id": 2, "status": "error", "error": "proxy not found" },
      { "id": 3, "status": "ok" }
    ]
  }
}
```

- `status` is `"ok"` or `"error"`. `error` is present only when `status` is `"error"`.

#### Error Responses
- **400 Bad Request**: If `ids` is empty, contains more than 1000 entries, or the body is malformed.
- **401 Unauthorized**: If the JWT token is missing or invalid.
- **403 Forbidden**: If the user lacks the `proxies:update` permission.

---

### 9. Bulk Disable Proxies

**POST** `/api/proxies/bulk/disable`

Disables multiple proxies in a single request. Requires the `proxies:update` permission.
Same request/response shape and best-effort semantics as **Bulk Enable Proxies**.

#### Request Body
```json
{
  "ids": [1, 2, 3]
}
```

#### Response (200 OK)
```json
{
  "success": true,
  "message": "Bulk disable completed",
  "data": {
    "requested": 3,
    "succeeded": 3,
    "failed": 0,
    "results": [
      { "id": 1, "status": "ok" },
      { "id": 2, "status": "ok" },
      { "id": 3, "status": "ok" }
    ]
  }
}
```

#### Error Responses
- **400 Bad Request**: If `ids` is empty, contains more than 1000 entries, or the body is malformed.
- **401 Unauthorized**: If the JWT token is missing or invalid.
- **403 Forbidden**: If the user lacks the `proxies:update` permission.

---

### 10. Bulk Delete Proxies

**POST** `/api/proxies/bulk/delete`

Deletes multiple proxies in a single request. Requires the `proxies:delete` permission.
Same request/response shape and best-effort semantics as **Bulk Enable Proxies**.

#### Request Body
```json
{
  "ids": [1, 2, 3]
}
```

#### Response (200 OK)
```json
{
  "success": true,
  "message": "Bulk delete completed",
  "data": {
    "requested": 3,
    "succeeded": 2,
    "failed": 1,
    "results": [
      { "id": 1, "status": "ok" },
      { "id": 2, "status": "error", "error": "proxy not found" },
      { "id": 3, "status": "ok" }
    ]
  }
}
```

#### Error Responses
- **400 Bad Request**: If `ids` is empty, contains more than 1000 entries, or the body is malformed.
- **401 Unauthorized**: If the JWT token is missing or invalid.
- **403 Forbidden**: If the user lacks the `proxies:delete` permission.

---

### 11. Export Proxies

**GET** `/api/proxies/export`

Returns proxies as a JSON array of portable export objects suitable for re-import.
Server-managed fields (`id`, `created_at`, `updated_at`, `created_by`, `ssl_forced`)
are dropped; everything needed to recreate the proxy is kept, including `is_active`
so an exported inactive proxy imports inactive. Requires the `proxies:read` permission.

#### Query Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `ids` | string | Optional comma-separated proxy ids (e.g. `?ids=1,2,3`). When provided, exactly these proxies are exported and any id that no longer exists is silently skipped. |
| `search` | string | Optional. Same as List: substring match on name/hostname. Ignored when `ids` is provided. |
| `type` | string | Optional. Same as List (`reverse_proxy`/`redirect`/`static`, supports `in:`/`not_in:`/`not:` operators). Ignored when `ids` is provided. |
| `status` | string | Optional. Same as List (`active`/`inactive`, supports `not:` operator). Ignored when `ids` is provided. |
| `ssl_enabled` | string | Optional. `true` or `false`. Ignored when `ids` is provided. |

When no `ids` are supplied, all proxies matching the given filters are exported (no pagination).

#### Response (200 OK)
```json
{
  "success": true,
  "data": [
    {
      "type": "reverse_proxy",
      "name": "My App",
      "hostname": "app.example.com",
      "ssl_enabled": true,
      "is_active": true,
      "block_exploits": true,
      "tls_insecure_skip_verify": false,
      "upstreams": ["localhost:8080"]
    }
  ]
}
```

`data` is an array of export objects. Per object: `description`, `upstreams`,
`load_balancing`, `custom_headers`, `redirect`, and `static` are omitted when empty.

#### Error Responses
- **400 Bad Request**: If `ids` contains a non-integer value, or a filter parameter is invalid.
- **401 Unauthorized**: If the JWT token is missing or invalid.
- **403 Forbidden**: If the user lacks the `proxies:read` permission.

---

## Load Balancing Details

### Strategies

| Strategy | Description | Use Case |
|----------|-------------|----------|
| `round_robin` | Distribute requests evenly in rotation | Default, works well for most cases |
| `least_conn` | Send to server with fewest active connections | Varying request durations |
| `ip_hash` | Hash client IP to determine server (sticky sessions) | Session persistence needed |
| `random` | Random selection | Simple, stateless |

### Health Checks

Health checks automatically monitor upstream server availability.

**Configuration:**

```json
{
  "health_checks": {
    "enabled": true,
    "path": "/health",           // Endpoint to check
    "interval": "30s",           // Check frequency
    "timeout": "5s",             // Request timeout
    "unhealthy_threshold": 2,    // Failures before marking unhealthy
    "healthy_threshold": 2       // Successes before marking healthy
  }
}
```

**Behavior:**
- Caddy sends GET requests to `{upstream}/{path}` every `interval`
- Expects HTTP 2xx/3xx response within `timeout`
- After `unhealthy_threshold` consecutive failures, marks upstream as down
- After `healthy_threshold` consecutive successes, marks upstream as up
- Unhealthy upstreams are excluded from load balancing

---

## Redirect Details

### Status Codes

| Code | Type | Caching | Use Case |
|------|------|---------|----------|
| 301 | Permanent | Yes | Domain migration, permanent URL change |
| 302 | Temporary | No | Temporary maintenance, A/B testing |
| 307 | Temporary (preserves method) | No | POST/PUT redirects |
| 308 | Permanent (preserves method) | Yes | POST/PUT permanent redirects |

### Path Preservation

**`preserve_path: true`** (default)
```
https://old.example.com/page/123
  ↓
https://new.example.com/page/123
```

**`preserve_path: false`**
```
https://old.example.com/page/123
  ↓
https://new.example.com
```

### Query Preservation

**`preserve_query: true`** (default)
```
https://old.example.com/page?foo=bar
  ↓
https://new.example.com/page?foo=bar
```

**`preserve_query: false`**
```
https://old.example.com/page?foo=bar
  ↓
https://new.example.com/page
```

---

## Static Site Details

### Mounting Static Files to Caddy

Before using the static proxy type, you need to mount your static files into the Waygates container so they're accessible at `root_path`.

**1. Update docker-compose.yml:**

```yaml
waygates:
  build:
    context: .
    dockerfile: Dockerfile
  volumes:
    - caddy-data:/data
    - caddy-config:/config
    - caddy-etc:/etc/caddy
    # Mount your static files
    - ./sites/my-app:/srv/my-app:ro
    - ./sites/docs:/srv/docs:ro
```

**2. Directory structure:**

```
waygates/
├── docker-compose.yml
└── sites/
    ├── my-app/           # React/Vue/Angular SPA build output
    │   ├── index.html
    │   ├── assets/
    │   └── ...
    └── docs/             # Static documentation
        ├── index.html
        └── ...
```

**3. Create proxy via API:**

```bash
curl -X POST http://localhost:8080/api/proxies \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "static",
    "name": "My SPA",
    "hostname": "app.example.com",
    "static": {
      "root_path": "/srv/my-app",
      "index_file": "index.html",
      "try_files": ["index.html"]
    }
  }'
```

**4. Restart container to pick up new volume mounts:**

```bash
docker compose restart waygates
```

**Updating files:** After copying new files to `./sites/my-app/`, no restart is needed - Caddy serves files directly from disk.

### File Serving

Caddy serves files from `root_path` directory.

**Basic configuration:**
```json
{
  "static": {
    "root_path": "/var/www/html",
    "index_file": "index.html"
  }
}
```

**Request flow:**
```
GET /about
  ↓
Looks for:
1. /var/www/html/about
2. /var/www/html/about/index.html (if about is a directory)
3. Returns 404 if not found
```

### SPA Support (Try Files)

For Single Page Applications, use `try_files` to fallback to index.html:

```json
{
  "static": {
    "root_path": "/var/www/spa",
    "index_file": "index.html",
    "try_files": ["index.html"]
  }
}
```

**Request flow:**
```
GET /dashboard/settings
  ↓
1. Try: /var/www/spa/dashboard/settings
2. Not found → Try: /var/www/spa/index.html
3. Return index.html (let SPA router handle routing)
```

### Template Rendering

Enable Caddy's template engine for dynamic content:

```json
{
  "static": {
    "root_path": "/etc/caddy/templates",
    "index_file": "maintenance.html",
    "template_rendering": true
  }
}
```

**Template example (`maintenance.html`):**
```html
<!DOCTYPE html>
<html>
<head>
    <title>Maintenance</title>
</head>
<body>
    <h1>Scheduled Maintenance</h1>
    <p>Current time: {{now | date "2006-01-02 15:04:05"}}</p>
    <p>Server: {{.Host}}</p>
</body>
</html>
```

**Available template variables:**
- `{{.Host}}` - Request hostname
- `{{.OriginalReq.URL.Path}}` - Request path
- `{{now}}` - Current timestamp
- See Caddy template docs for full list

---

## Security - Block Common Exploits

When `block_exploits: true` is set on a reverse proxy, Caddy will import security rules from `conf/snippets/security.caddy` to block common web attacks.

### How It Works

The backend generates Caddy configuration with an `import` directive:

```caddyfile
api.example.com {
    import snippets/security  # Imported when block_exploits = true

    reverse_proxy 192.168.1.100:8080
}
```

### What Gets Blocked

The security snippet blocks the following attack patterns:

**1. SQL Injection Attempts**
- Query strings containing: `union select`, `union all select`, `concat()`
- Returns: `403 Forbidden - SQL injection detected`

**2. File Injection/Traversal**
- Path traversal: `../`, `../../`, `../../../`
- Remote file inclusion: `http://`, `https://` in query parameters
- Returns: `403 Forbidden - File injection detected`

**3. Common Exploits**
- XSS attacks: `<script>`, `%3Cscript`
- PHP global exploits: `GLOBALS=`, `_REQUEST=`
- System access: `/proc/self/environ`
- Base64 encoding attempts: `base64_encode()`, `base64_decode()`
- Returns: `403 Forbidden - Common exploit detected`

**4. Spam Keywords**
- Pharmaceutical spam: `viagra`, `cialis`, `levitra`, `xanax`, `valium`
- Other spam terms: `tramadol`, `phentermin`, `ambien`
- Returns: `403 Forbidden - Spam detected`

**5. Malicious User Agents**
- Download managers: `GetRight`, `GetWeb!`, `Go!Zilla`, `Download Demon`
- Bots: `TurnitinBot`, `GrabNet`, `libwww-perl`
- Known malicious agents: `Indy Library`
- Returns: `403 Forbidden - Blocked user agent`

### Example Configuration

**Request:**
```json
{
  "type": "reverse_proxy",
  "name": "Protected API",
  "hostname": "api.example.com",
  "upstreams": [{"host": "192.168.1.100", "port": 8080, "scheme": "http"}],
  "block_exploits": true
}
```

**Generated Caddyfile:**
```caddyfile
api.example.com {
    # Security rules applied first
    import snippets/security

    # Then reverse proxy
    reverse_proxy 192.168.1.100:8080 {
        header_up X-Real-IP {remote_host}
        header_up X-Forwarded-For {remote_host}
        header_up X-Forwarded-Proto {scheme}
    }
}
```

### Testing Security Rules

Test that security rules are working:

```bash
# SQL injection - should return 403
curl "https://api.example.com/?id=1 union select password from users"

# XSS attempt - should return 403
curl "https://api.example.com/?search=<script>alert(1)</script>"

# Path traversal - should return 403
curl "https://api.example.com/?file=../../../etc/passwd"

# Bad user agent - should return 403
curl -A "libwww-perl/6.0" "https://api.example.com/"

# Normal request - should work
curl "https://api.example.com/?search=hello"
```

### Customizing Security Rules

To modify security rules:

1. Edit `/conf/snippets/security.caddy`
2. Add/remove patterns as needed
3. Validate: `make validate`
4. Apply changes: `make restart`

Changes apply to all proxies with `block_exploits: true` enabled.

### Performance Impact

- **Minimal overhead** - Pattern matching happens at Caddy level (compiled code)
- **Applied before proxying** - Malicious requests never reach your backend
- **No extra dependencies** - Uses native Caddy matchers

### Limitations

- **Not a WAF** - This is basic pattern matching, not a full Web Application Firewall
- **False positives possible** - Legitimate requests might be blocked if they contain flagged patterns
- **Query string only** - Most checks apply to URL query strings and headers, not request bodies

For advanced protection, consider:
- Cloudflare WAF (if using Cloudflare)
- Dedicated WAF solution (ModSecurity, etc.)
- `caddy-security` plugin for advanced features

---

## Common Response Format

### Success Response

```json
{
  "success": true,
  "message": "Optional success message",
  "data": {
    // Response data
  }
}
```

### Error Response

```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message",
    "details": {
      // Optional additional error details
    }
  }
}
```

## HTTP Status Codes

| Status Code | Description | Usage |
|-------------|-------------|-------|
| 200 OK | Success | GET, PUT, POST (enable/disable) |
| 201 Created | Resource created | POST (create) |
| 400 Bad Request | Invalid request | Validation errors |
| 401 Unauthorized | Not authenticated | Missing/invalid token |
| 403 Forbidden | Not authorized | Insufficient permissions |
| 404 Not Found | Resource not found | Invalid ID |
| 409 Conflict | Resource conflict | Duplicate hostname |
| 422 Unprocessable Entity | Business logic error | Cannot process valid request |
| 500 Internal Server Error | Server error | Unexpected errors |
| 502 Bad Gateway | External service error | Caddy API errors |

## Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `VALIDATION_ERROR` | 400 | Request validation failed |
| `UNAUTHORIZED` | 401 | Authentication required |
| `FORBIDDEN` | 403 | Insufficient permissions |
| `PROXY_NOT_FOUND` | 404 | Proxy doesn't exist |
| `HOSTNAME_CONFLICT` | 409 | Hostname already in use |
| `PROXY_ALREADY_ENABLED` | 409 | Proxy is already enabled |
| `PROXY_ALREADY_DISABLED` | 409 | Proxy is already disabled |
| `CADDY_API_ERROR` | 502 | Caddy Admin API error |
| `INTERNAL_ERROR` | 500 | Unexpected server error |

---

## Notes

1. **Load Balancing:** Automatically enabled when `upstreams.length > 1`
2. **Health Checks:** Recommended for production load-balanced setups
3. **Security/Exploit Blocking:** Uses reusable Caddy snippet (`conf/snippets/security.caddy`) imported when `block_exploits: true`
4. **WebSocket Support:** Automatic in Caddy's reverse_proxy - no configuration needed
5. **Redirects:** Use 301 for SEO-friendly permanent redirects
6. **Static Sites:** Perfect for documentation, landing pages, SPAs
7. **Templates:** Useful for maintenance pages, error pages
8. **Timestamps:** All in ISO 8601 format (UTC)
9. **Hostname Uniqueness:** Must be unique across all proxies
10. **SSL/TLS:** Automatic certificates via configured ACME provider (see CADDY_ACME_PROVIDER)

---

**Document Version:** 1.0
**Status:** Complete
