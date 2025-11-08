# Proxy Management API Specification

**Version:** 1.0
**Last Updated:** 2024-11-08

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
    id INTEGER PRIMARY KEY AUTOINCREMENT,

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
    custom_headers TEXT,  -- JSON: {"X-Header": "value"}

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
```json
{
  "X-Custom-Header": "value",
  "X-API-Version": "v1"
}
```

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
  "hostname": "app.caddy.e412.in",
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

#### Query Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `page` | integer | No | 1 | Page number |
| `limit` | integer | No | 20 | Items per page (max: 100) |
| `search` | string | No | - | Search by hostname or name |
| `type` | string | No | - | Filter by type: `reverse_proxy`, `redirect`, `static` |
| `status` | string | No | - | Filter by status: `active`, `inactive` |
| `sort` | string | No | `created_at` | Sort field |
| `order` | string | No | `desc` | Sort order: `asc`, `desc` |

#### Response (200 OK)

```json
{
  "success": true,
  "data": {
    "proxies": [
      {
        "id": 1,
        "type": "reverse_proxy",
        "name": "My Application",
        "hostname": "app.caddy.e412.in",
        "upstreams": [
          {
            "host": "192.168.100.5",
            "port": 8080,
            "scheme": "http"
          }
        ],
        "ssl_enabled": true,
        "is_active": true,
        "created_at": "2024-11-08T10:30:00Z"
      },
      {
        "id": 2,
        "type": "redirect",
        "name": "Old Domain Redirect",
        "hostname": "old.caddy.e412.in",
        "redirect": {
          "target": "https://new.caddy.e412.in",
          "status_code": 301,
          "preserve_path": true
        },
        "ssl_enabled": true,
        "is_active": true,
        "created_at": "2024-11-08T11:00:00Z"
      }
    ],
    "pagination": {
      "current_page": 1,
      "total_pages": 1,
      "total_items": 2,
      "items_per_page": 20,
      "has_next": false,
      "has_prev": false
    }
  }
}
```

---

### 2. Get Proxy by ID

**GET** `/api/proxies/:id`

#### Response (200 OK)

Returns full proxy details based on type.

---

### 3. Create Proxy

**POST** `/api/proxies`

#### Request Body Examples

**Example 1: Simple Reverse Proxy**

```json
{
  "type": "reverse_proxy",
  "name": "My Application",
  "hostname": "app.caddy.e412.in",
  "upstreams": [
    {
      "host": "192.168.100.5",
      "port": 8080,
      "scheme": "http"
    }
  ],
  "ssl_enabled": true,
  "ssl_forced": true,
  "block_exploits": true,
  "custom_headers": {
    "X-App-Version": "1.0"
  },
  "description": "Production application"
}
```

**Example 2: Load Balanced Reverse Proxy**

```json
{
  "type": "reverse_proxy",
  "name": "Load Balanced API",
  "hostname": "api.caddy.e412.in",
  "upstreams": [
    {
      "host": "192.168.100.10",
      "port": 3000,
      "scheme": "http"
    },
    {
      "host": "192.168.100.11",
      "port": 3000,
      "scheme": "http"
    },
    {
      "host": "192.168.100.12",
      "port": 3000,
      "scheme": "http"
    }
  ],
  "load_balancing": {
    "strategy": "least_conn",
    "health_checks": {
      "enabled": true,
      "path": "/health",
      "interval": "30s",
      "timeout": "5s",
      "unhealthy_threshold": 2,
      "healthy_threshold": 2
    }
  },
  "ssl_enabled": true,
  "ssl_forced": true,
  "description": "API with 3 backend servers"
}
```

**Example 3: Redirect**

```json
{
  "type": "redirect",
  "name": "Old Domain Redirect",
  "hostname": "old.caddy.e412.in",
  "redirect": {
    "target": "https://new.caddy.e412.in",
    "status_code": 301,
    "preserve_path": true,
    "preserve_query": true
  },
  "ssl_enabled": true,
  "description": "Permanent redirect to new domain"
}
```

**Example 4: Static HTML Site**

```json
{
  "type": "static",
  "name": "Landing Page",
  "hostname": "landing.caddy.e412.in",
  "static": {
    "root_path": "/var/www/landing",
    "index_file": "index.html",
    "browse": false,
    "try_files": ["index.html"],
    "template_rendering": false
  },
  "ssl_enabled": true,
  "description": "Static landing page"
}
```

**Example 5: Template-Based Site**

```json
{
  "type": "static",
  "name": "Maintenance Page",
  "hostname": "maintenance.caddy.e412.in",
  "static": {
    "root_path": "/etc/caddy/templates",
    "index_file": "maintenance.html",
    "browse": false,
    "template_rendering": true
  },
  "ssl_enabled": true,
  "description": "Maintenance page with Caddy templates"
}
```

#### Field Validation

**Common Fields (All Types):**

| Field | Type | Required | Validation |
|-------|------|----------|------------|
| `type` | string | Yes | `reverse_proxy`, `redirect`, `static` |
| `name` | string | Yes | 1-255 chars |
| `hostname` | string | Yes | Valid domain, unique |
| `ssl_enabled` | boolean | No | Default: `true` |
| `ssl_forced` | boolean | No | Default: `true` |
| `description` | string | No | Max 500 chars |

**Reverse Proxy Fields:**

| Field | Type | Required | Validation |
|-------|------|----------|------------|
| `upstreams` | array | Yes | Min 1, max 20 upstreams |
| `upstreams[].host` | string | Yes | Valid IP or hostname |
| `upstreams[].port` | integer | Yes | 1-65535 |
| `upstreams[].scheme` | string | No | `http` or `https`, default: `http` |
| `load_balancing.strategy` | string | No* | `round_robin`, `least_conn`, `ip_hash`, `random` |
| `load_balancing.health_checks.enabled` | boolean | No | Default: `false` |
| `load_balancing.health_checks.path` | string | No** | Valid URL path |
| `load_balancing.health_checks.interval` | string | No | Duration (e.g., `30s`) |
| `block_exploits` | boolean | No | Default: `true` |
| `custom_headers` | object | No | Max 20 headers |

\* Required if `upstreams.length > 1`
\** Required if `health_checks.enabled = true`

**Redirect Fields:**

| Field | Type | Required | Validation |
|-------|------|----------|------------|
| `redirect.target` | string | Yes | Valid URL |
| `redirect.status_code` | integer | No | 301, 302, 307, 308 (default: 302) |
| `redirect.preserve_path` | boolean | No | Default: `true` |
| `redirect.preserve_query` | boolean | No | Default: `true` |

**Static Fields:**

| Field | Type | Required | Validation |
|-------|------|----------|------------|
| `static.root_path` | string | Yes | Valid absolute path |
| `static.index_file` | string | No | Filename (default: `index.html`) |
| `static.browse` | boolean | No | Default: `false` |
| `static.try_files` | array | No | Array of filenames |
| `static.template_rendering` | boolean | No | Default: `false` |

#### Response (201 Created)

```json
{
  "success": true,
  "message": "Proxy created successfully",
  "data": {
    "id": 5,
    "type": "reverse_proxy",
    "name": "Load Balanced API",
    "hostname": "api.caddy.e412.in",
    "upstreams": [
      {
        "host": "192.168.100.10",
        "port": 3000,
        "scheme": "http"
      },
      {
        "host": "192.168.100.11",
        "port": 3000,
        "scheme": "http"
      }
    ],
    "load_balancing": {
      "strategy": "least_conn",
      "health_checks": {
        "enabled": true,
        "path": "/health",
        "interval": "30s",
        "timeout": "5s",
        "unhealthy_threshold": 2,
        "healthy_threshold": 2
      }
    },
    "ssl_enabled": true,
    "ssl_forced": true,
    "is_active": true,
    "created_by": {
      "id": 1,
      "name": "Admin User",
      "email": "admin@example.com"
    },
    "created_at": "2024-11-08T13:00:00Z",
    "updated_at": "2024-11-08T13:00:00Z"
  }
}
```

#### Error Responses

**400 Bad Request - Validation Error**

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Request validation failed",
    "details": {
      "load_balancing": "load_balancing configuration required when multiple upstreams provided",
      "upstreams": "minimum 1 upstream required for reverse_proxy type"
    }
  }
}
```

**409 Conflict**

```json
{
  "success": false,
  "error": {
    "code": "HOSTNAME_CONFLICT",
    "message": "A proxy with hostname 'app.caddy.e412.in' already exists"
  }
}
```

---

### 4. Update Proxy

**PUT** `/api/proxies/:id`

All fields are optional. Can change type if needed.

---

### 5. Delete Proxy

**DELETE** `/api/proxies/:id`

---

### 6. Enable Proxy

**POST** `/api/proxies/:id/enable`

---

### 7. Disable Proxy

**POST** `/api/proxies/:id/disable`

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
  "hostname": "api.caddy.e412.in",
  "upstreams": [{"host": "192.168.1.100", "port": 8080, "scheme": "http"}],
  "block_exploits": true
}
```

**Generated Caddyfile:**
```caddyfile
api.caddy.e412.in {
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
curl "https://api.caddy.e412.in/?id=1 union select password from users"

# XSS attempt - should return 403
curl "https://api.caddy.e412.in/?search=<script>alert(1)</script>"

# Path traversal - should return 403
curl "https://api.caddy.e412.in/?file=../../../etc/passwd"

# Bad user agent - should return 403
curl -A "libwww-perl/6.0" "https://api.caddy.e412.in/"

# Normal request - should work
curl "https://api.caddy.e412.in/?search=hello"
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
10. **SSL/TLS:** Uses wildcard certificate `*.caddy.e412.in` when enabled

---

**Document Version:** 1.0
**Status:** Complete
