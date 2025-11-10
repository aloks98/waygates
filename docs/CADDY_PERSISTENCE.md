# Caddy Configuration Persistence

## Problem

When using Caddy's Admin API to configure routes dynamically, the configuration is stored in memory. When the Caddy container restarts, all routes configured via the API are lost.

## Solution

The backend automatically syncs all active proxies from the database to Caddy on startup. This ensures that proxy configurations persist across Caddy restarts.

## How It Works

### 1. Database as Source of Truth

All proxy configurations are stored in the SQLite database (`backend/data/caddy-manager.db`). The database schema includes:

- Proxy details (hostname, upstreams, type, etc.)
- Active status (`is_active` boolean)
- SSL settings
- Custom headers
- Load balancing configuration

### 2. Automatic Sync on Backend Startup

When the backend starts (`go run backend/cmd/server/main.go`), the `ProxyService` automatically:

1. Queries all active proxies from the database
2. Builds Caddy route configurations for each
3. Applies all routes to Caddy via Admin API

**Code**: `backend/internal/service/proxy_service.go:32-66`

```go
// syncProxiesToCaddy syncs all active proxies from database to Caddy
func (s *ProxyService) syncProxiesToCaddy() {
    // Get all active proxies
    proxies, _, err := s.repo.List(repository.ProxyListParams{
        Status: "active",
        Limit:  1000,
    })

    // Build routes
    var routes []interface{}
    for _, proxy := range proxies {
        route, err := caddy.BuildRouteConfig(&proxy)
        if err != nil {
            continue
        }
        routes = append(routes, route)
    }

    // Apply all routes to Caddy
    if len(routes) > 0 {
        path := "/config/apps/http/servers/srv0/routes"
        s.caddyClient.PATCH(path, routes)
    }
}
```

### 3. Caddy Auto-Save

Caddy also has its own persistence mechanism:
- Configuration changes via Admin API are auto-saved to `/config/caddy/autosave.json`
- This is mapped to `./caddy_config/autosave.json` on the host via docker volume
- Caddy loads this on startup

**However**, we don't rely solely on Caddy's autosave because:
- The database is the authoritative source
- It's easier to manage and backup
- It provides a clean separation of concerns

## Restart Workflow

### When Backend Restarts

1. ✅ Database connection established
2. ✅ Active proxies loaded from database
3. ✅ Routes synced to Caddy automatically
4. ✅ All configurations restored

### When Caddy Restarts

1. ⚠️ Caddy loads `/config/caddy/autosave.json` (may have stale routes)
2. ✅ Backend detects Caddy is running
3. ✅ Backend re-syncs all active proxies from database
4. ✅ Fresh, correct configuration applied

### When Both Restart (docker-compose restart)

1. ✅ Caddy starts with autosave.json
2. ✅ Backend starts and syncs from database
3. ✅ All active proxies restored

## Testing

### Verify Sync on Startup

```bash
# Create some active proxies
curl -X POST http://localhost:8080/api/proxies \
  -H "Content-Type: application/json" \
  -d '{"type":"reverse_proxy","name":"Test1","hostname":"test1.local","upstreams":[{"host":"localhost","port":8001,"scheme":"http"}]}'

curl -X POST http://localhost:8080/api/proxies \
  -H "Content-Type: application/json" \
  -d '{"type":"reverse_proxy","name":"Test2","hostname":"test2.local","upstreams":[{"host":"localhost","port":8002,"scheme":"http"}]}'

# Verify routes in Caddy
curl -s http://localhost:2019/config/apps/http/servers/srv0/routes | jq 'length'
# Should show: 2

# Restart Caddy
docker-compose restart caddy

# Wait a moment, then check routes again
sleep 3
curl -s http://localhost:2019/config/apps/http/servers/srv0/routes | jq 'length'
# Should still show: 2

# Verify the routes are correct
curl -s http://localhost:2019/config/apps/http/servers/srv0/routes | jq '[.[] | {id: ."@id", host: .match[0].host[0]}]'
```

### Manual Resync

If for any reason routes get out of sync, simply restart the backend:

```bash
# Kill backend
pkill -f "go run backend"

# Restart backend
go run backend/cmd/server/main.go

# Routes will be synced automatically
```

## Database Backup

Since the database is the source of truth, make sure to back it up:

```bash
# Backup database
cp backend/data/caddy-manager.db backend/data/caddy-manager.db.backup

# Or use SQLite backup command
sqlite3 backend/data/caddy-manager.db ".backup 'backup.db'"
```

## Troubleshooting

### Routes Missing After Restart

**Check**:
1. Are proxies marked as active in database?
   ```bash
   sqlite3 backend/data/caddy-manager.db "SELECT id, name, is_active FROM proxies;"
   ```

2. Is backend running and connected to Caddy?
   ```bash
   curl http://localhost:8080/api/health
   ```

3. Check backend logs for sync errors

**Fix**:
- Enable inactive proxies: `POST /api/proxies/{id}/enable`
- Restart backend to trigger resync

### Routes in Caddy but Not in Database

This shouldn't happen, but if it does:
- Caddy has stale routes from autosave.json
- Restart Caddy to clear them
- Backend will repopulate from database

### Duplicate Routes

If you see duplicate routes:
- This indicates a bug in the update logic
- Restart both Caddy and backend to start fresh
- Report the issue with steps to reproduce

## Best Practices

1. ✅ **Always use the API** to manage proxies (don't edit Caddy config directly)
2. ✅ **Backup the database** regularly
3. ✅ **Monitor sync** on startup (check logs)
4. ✅ **Use enable/disable** instead of deleting proxies temporarily
5. ❌ **Don't edit** Caddy config files manually when using the API

## Architecture

```
┌─────────────┐
│   Database  │ ◄── Source of Truth
│   (SQLite)  │
└──────┬──────┘
       │
       ▼
┌─────────────┐     On Startup
│   Backend   │────────────────┐
│  (Go API)   │                │
└──────┬──────┘                │
       │                       ▼
       │ CRUD Operations  ┌─────────────┐
       │ via Admin API    │    Caddy    │
       └──────────────────►│   Server    │
                          └─────────────┘
                               │
                               ▼
                          autosave.json
                          (redundant backup)
```

## Related Files

- `backend/internal/service/proxy_service.go` - Sync logic
- `backend/internal/caddy/config_builder.go` - Route generation
- `backend/data/caddy-manager.db` - Database (source of truth)
- `caddy_config/autosave.json` - Caddy's auto-save (backup)

## Summary

The system ensures persistence through:
1. **Database** as the primary source of truth
2. **Automatic sync** on backend startup
3. **Caddy autosave** as a redundant backup

This provides robust configuration persistence across any restart scenario.
