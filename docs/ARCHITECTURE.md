# Architecture Overview

## System Components

```
┌─────────────────────────────────────────────────────────┐
│                     Caddy Server                        │
│  ┌──────────────────────────────────────────────────┐  │
│  │  Caddyfile (Static Configuration)                │  │
│  │  - Global settings (email, ACME, DNS)            │  │
│  │  - :443 server block (srv0)                      │  │
│  │  - Base structure for dynamic routes             │  │
│  └──────────────────────────────────────────────────┘  │
│                           │                             │
│  ┌──────────────────────────────────────────────────┐  │
│  │  Admin API (:2019)                               │  │
│  │  - Dynamic route management                      │  │
│  │  - Runtime configuration                         │  │
│  └──────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
                            ▲
                            │ HTTP API calls
                            │
┌─────────────────────────────────────────────────────────┐
│              Go Backend (Port 8080)                     │
│  ┌──────────────────────────────────────────────────┐  │
│  │  REST API Handlers                               │  │
│  │  - Create/Update/Delete proxies                 │  │
│  │  - Enable/Disable proxies                       │  │
│  │  - List and search proxies                      │  │
│  └──────────────────────────────────────────────────┘  │
│                           │                             │
│  ┌──────────────────────────────────────────────────┐  │
│  │  Service Layer                                   │  │
│  │  - Business logic                                │  │
│  │  - Caddy config generation                       │  │
│  │  - Automatic sync on startup                     │  │
│  └──────────────────────────────────────────────────┘  │
│                           │                             │
│  ┌──────────────────────────────────────────────────┐  │
│  │  Repository Layer                                │  │
│  │  - Database operations (GORM)                    │  │
│  │  - Query building                                │  │
│  └──────────────────────────────────────────────────┘  │
│                           │                             │
│  ┌──────────────────────────────────────────────────┐  │
│  │  SQLite Database                                 │  │
│  │  - Proxies table (source of truth)               │  │
│  │  - Audit logs                                    │  │
│  │  - Schema migrations                             │  │
│  └──────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

## Configuration Flow

### Static Configuration (Caddyfile)

The Caddyfile provides the base structure:

```caddyfile
{
    admin 0.0.0.0:2019
    email {$CLOUDFLARE_EMAIL}
    acme_ca {$ACME_CA_URL}
    acme_dns cloudflare {$CLOUDFLARE_API_TOKEN}
}

# Creates srv0 server
:443 {
    # Routes added dynamically via Admin API
}
```

**Purpose**:
- ✅ Sets up global options (email, ACME, DNS)
- ✅ Creates the `srv0` HTTP server listening on `:443`
- ✅ Provides base structure for dynamic routes
- ✅ Persists across restarts

### Dynamic Configuration (Admin API)

The backend manages routes at runtime:

```go
// Add route
POST /config/apps/http/servers/srv0/routes
{
  "@id": "proxy_1",
  "match": [{"host": ["example.com"]}],
  "handle": [{"handler": "reverse_proxy", ...}]
}

// Update all routes
PATCH /config/apps/http/servers/srv0/routes
[route1, route2, route3, ...]
```

**Purpose**:
- ✅ Manages individual proxy routes
- ✅ Enables/disables proxies without restart
- ✅ Allows runtime configuration changes
- ⚠️ Lost on Caddy restart (requires sync)

## Data Flow

### Creating a Proxy

```
User → API (POST /api/proxies)
  ↓
Handler validates request
  ↓
Service.CreateProxy()
  ├─→ Validate proxy config
  ├─→ Check hostname uniqueness
  ├─→ Save to database
  │   └─→ if active:
  │       ├─→ Build Caddy route config
  │       ├─→ Check srv0 exists
  │       ├─→ Get current routes
  │       ├─→ Add new route
  │       └─→ PATCH routes to Caddy
  └─→ Return proxy to user
```

### Updating a Proxy

```
User → API (PUT /api/proxies/:id)
  ↓
Handler validates request
  ↓
Service.UpdateProxy()
  ├─→ Get existing proxy from DB
  ├─→ Preserve fields (is_active, ssl_*, created_*)
  ├─→ Validate new config
  ├─→ Check hostname uniqueness
  ├─→ Update database
  └─→ if active:
      ├─→ Get current routes from Caddy
      ├─→ Filter out old route by ID
      ├─→ Add updated route
      └─→ PATCH routes to Caddy
```

### Startup Sync

```
Backend Starts
  ↓
Service.NewProxyService()
  ├─→ Initialize service
  └─→ Launch syncProxiesToCaddy() goroutine
      ├─→ Query active proxies from DB
      ├─→ Build route configs for all
      ├─→ Check srv0 exists
      └─→ PATCH all routes to Caddy
```

## Database Schema

### Proxies Table

```sql
CREATE TABLE proxies (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type VARCHAR(50) NOT NULL,           -- reverse_proxy, redirect, static
    name VARCHAR(255) NOT NULL,
    hostname VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,

    -- Status flags
    ssl_enabled BOOLEAN DEFAULT TRUE,
    ssl_forced BOOLEAN DEFAULT TRUE,
    is_active BOOLEAN DEFAULT TRUE,

    -- Type-specific config (JSON)
    upstreams TEXT,                      -- JSON array
    load_balancing TEXT,                 -- JSON object
    block_exploits BOOLEAN DEFAULT TRUE,
    tls_insecure_skip_verify BOOLEAN DEFAULT FALSE,
    custom_headers TEXT,                 -- JSON object
    redirect_config TEXT,                -- JSON object
    static_config TEXT,                  -- JSON object

    -- Audit fields
    created_by INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## Route Management Strategy

### Why PATCH Entire Routes Array?

We use `PATCH /config/apps/http/servers/srv0/routes` with the complete array instead of individual operations because:

1. **Atomic Updates**: All routes updated in one operation
2. **No Index Issues**: Don't rely on array indices that can change
3. **Idempotent**: Same result regardless of current state
4. **Simpler Logic**: Read → Filter → Modify → Write

### Route Identification

Each route has a unique `@id` field:

```json
{
  "@id": "proxy_5",  // ID format: "proxy_{database_id}"
  "match": [...],
  "handle": [...]
}
```

When updating or deleting, we:
1. Get all current routes
2. Filter out routes with matching `@id`
3. Add new route (if creating/updating)
4. PATCH the entire array back

## Key Design Decisions

### 1. Database as Source of Truth

**Why**:
- Easy to backup and restore
- Survives Caddy restarts
- Can be queried and analyzed
- Supports multi-instance deployments

**How**:
- All proxy configs stored in SQLite
- Automatic sync to Caddy on startup
- Enable/disable without data loss

### 2. Hybrid Configuration (Caddyfile + Admin API)

**Why**:
- Caddyfile for static base structure
- Admin API for dynamic routes
- Best of both worlds

**Caddyfile**:
- Global options (email, ACME, DNS)
- Base server structure (`:443` block)
- Persistent across restarts

**Admin API**:
- Dynamic proxy routes
- Managed by backend
- Can be modified at runtime

### 3. Automatic Sync on Startup

**Why**:
- Caddy Admin API changes are in-memory
- Restarts lose dynamic configuration
- Need to restore from database

**How**:
- Background goroutine on backend startup
- Queries active proxies from database
- Rebuilds and applies all routes

### 4. is_active Flag

**Why**:
- Disable proxies without deleting config
- Quick enable/disable toggle
- Keep configuration for later

**Benefits**:
- Test changes before activating
- Temporarily disable without data loss
- Audit trail of configurations

## Error Handling

### Caddy Errors

When Caddy returns errors, we:
1. Wrap in `CaddyError` type
2. Return HTTP 502 Bad Gateway (not 400)
3. Preserve error message for debugging

### Database Errors

Transaction rollback on failures:
```go
if err := db.Create(proxy); err != nil {
    return err
}
if err := caddy.Apply(proxy); err != nil {
    db.Delete(proxy)  // Rollback
    return err
}
```

### Validation Errors

Validated before database operations:
- Hostname format
- Required fields per type
- Upstream configurations
- Port ranges

## Performance Considerations

### Sync on Startup

- Runs in background goroutine
- Doesn't block server startup
- Tolerates Caddy not being ready
- Retries are not implemented (startup only)

### Route Updates

- Single PATCH for all routes
- No multiple API calls
- Atomic operation
- No race conditions

### Database Queries

- Indexed on hostname (unique)
- Indexed on is_active for sync
- Pagination for list endpoints
- Search uses LIKE queries

## Security

### API Authentication

Currently not implemented. TODO:
- JWT authentication
- User management
- Role-based access control

### TLS Certificates

- Automatic via Let's Encrypt
- DNS-01 challenge with Cloudflare
- Staging/production environments
- Self-signed support for backends

### Exploit Blocking

- `block_exploits` flag tracked
- Implementation pending
- Currently documentation-only

## Future Enhancements

1. **Logging & Monitoring**
   - Structured logging (slog)
   - Metrics (Prometheus)
   - Health checks

2. **Authentication**
   - JWT tokens
   - User management
   - API keys

3. **Websocket Support**
   - Real-time config updates
   - Live log streaming

4. **Backup & Restore**
   - Database backup
   - Configuration export/import
   - Disaster recovery

5. **Multi-instance**
   - PostgreSQL support
   - Distributed locking
   - Config replication
