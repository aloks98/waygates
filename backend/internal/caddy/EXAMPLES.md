# Caddy Config Builder & Parser Examples

This document demonstrates how to use the config builder and parser functions to convert between Proxy models and Caddy JSON configuration.

## Example 1: Reverse Proxy (Simple)

### Input: Proxy Model

```go
proxy := &models.Proxy{
    ID:       1,
    Type:     models.ProxyTypeReverseProxy,
    Name:     "My Application",
    Hostname: "app.example.com",
    Upstreams: []interface{}{
        map[string]interface{}{
            "host":   "192.168.100.5",
            "port":   8080,
            "scheme": "http",
        },
    },
    BlockExploits: true,
    CustomHeaders: map[string]interface{}{
        "X-App-Version": "1.0",
    },
}
```

### Generated Caddy JSON

```go
route, _ := caddy.BuildRouteConfig(proxy)
// Result:
{
    "@id": "proxy_1",
    "match": [
        {
            "host": ["app.example.com"]
        }
    ],
    "handle": [
        {
            "handler": "reverse_proxy",
            "@id": "handler_1",
            "upstreams": [
                {"dial": "192.168.100.5:8080"}
            ],
            "headers": {
                "request": {
                    "set": {
                        "X-Real-IP": ["{http.request.remote.host}"],
                        "X-Forwarded-For": ["{http.request.remote.host}"],
                        "X-Forwarded-Proto": ["{http.request.scheme}"],
                        "X-App-Version": ["1.0"]
                    }
                }
            }
        }
    ]
}
```

### Generated Caddyfile Snippet

```go
snippet := caddy.BuildCaddyfileSnippet(proxy)
// Result:
```

```caddyfile
app.example.com {
    import snippets/security

    reverse_proxy 192.168.100.5:8080 {
        header_up X-Real-IP {remote_host}
        header_up X-Forwarded-For {remote_host}
        header_up X-Forwarded-Proto {scheme}
    }
}
```

---

## Example 2: Reverse Proxy (Load Balanced)

### Input: Proxy Model

```go
proxy := &models.Proxy{
    ID:       2,
    Type:     models.ProxyTypeReverseProxy,
    Name:     "Load Balanced API",
    Hostname: "api.example.com",
    Upstreams: []interface{}{
        map[string]interface{}{"host": "192.168.100.10", "port": 3000, "scheme": "http"},
        map[string]interface{}{"host": "192.168.100.11", "port": 3000, "scheme": "http"},
        map[string]interface{}{"host": "192.168.100.12", "port": 3000, "scheme": "http"},
    },
    LoadBalancing: map[string]interface{}{
        "strategy": "least_conn",
        "health_checks": map[string]interface{}{
            "enabled":             true,
            "path":                "/health",
            "interval":            "30s",
            "timeout":             "5s",
            "unhealthy_threshold": 2,
            "healthy_threshold":   2,
        },
    },
    BlockExploits: true,
}
```

### Generated Caddy JSON

```go
route, _ := caddy.BuildRouteConfig(proxy)
// Result includes:
{
    "handler": "reverse_proxy",
    "upstreams": [
        {"dial": "192.168.100.10:3000"},
        {"dial": "192.168.100.11:3000"},
        {"dial": "192.168.100.12:3000"}
    ],
    "load_balancing": {
        "selection_policy": "least_conn"
    },
    "health_checks": {
        "active": {
            "path": "/health",
            "interval": "30s",
            "timeout": "5s",
            "unhealthy_threshold": 2,
            "healthy_threshold": 2
        }
    }
}
```

---

## Example 3: Reverse Proxy (HTTPS Backend with Self-Signed Cert)

### Input: Proxy Model

```go
proxy := &models.Proxy{
    ID:       3,
    Type:     models.ProxyTypeReverseProxy,
    Name:     "Internal HTTPS Service",
    Hostname: "internal.example.com",
    Upstreams: []interface{}{
        map[string]interface{}{
            "host":   "192.168.100.50",
            "port":   8443,
            "scheme": "https",
        },
    },
    TLSInsecureSkipVerify: true,
    BlockExploits: true,
}
```

### Generated Caddy JSON

```go
route, _ := caddy.BuildRouteConfig(proxy)
// Result:
{
    "@id": "proxy_3",
    "match": [
        {
            "host": ["internal.example.com"]
        }
    ],
    "handle": [
        {
            "handler": "reverse_proxy",
            "@id": "handler_3",
            "upstreams": [
                {"dial": "192.168.100.50:8443"}
            ],
            "transport": {
                "protocol": "http",
                "tls": {
                    "insecure_skip_verify": true
                }
            },
            "headers": {
                "request": {
                    "set": {
                        "X-Real-IP": ["{http.request.remote.host}"],
                        "X-Forwarded-For": ["{http.request.remote.host}"],
                        "X-Forwarded-Proto": ["{http.request.scheme}"]
                    }
                }
            }
        }
    ]
}
```

### Use Case

This configuration is useful when:
- Proxying to internal services with self-signed certificates
- Development environments
- Internal corporate services
- Internal services that don't need public CA certificates

**Security Warning**: Only use `tls_insecure_skip_verify: true` for trusted internal services. This disables certificate validation and should never be used for untrusted or public services.

---

## Example 4: Redirect (301 Permanent)

### Input: Proxy Model

```go
proxy := &models.Proxy{
    ID:       3,
    Type:     models.ProxyTypeRedirect,
    Name:     "Old Domain Redirect",
    Hostname: "old.example.com",
    RedirectConfig: map[string]interface{}{
        "target":         "https://new.example.com",
        "status_code":    301,
        "preserve_path":  true,
        "preserve_query": true,
    },
}
```

### Generated Caddy JSON

```go
route, _ := caddy.BuildRouteConfig(proxy)
// Result:
{
    "@id": "proxy_3",
    "match": [{"host": ["old.example.com"]}],
    "handle": [
        {
            "handler": "static_response",
            "@id": "handler_3",
            "status_code": 301,
            "headers": {
                "response": {
                    "set": {
                        "Location": ["https://new.example.com{http.request.uri.path}{http.request.uri.query}"]
                    }
                }
            }
        }
    ]
}
```

### Generated Caddyfile Snippet

```caddyfile
old.example.com {
    redir https://new.example.com 301
}
```

---

## Example 5: Static Site (SPA)

### Input: Proxy Model

```go
proxy := &models.Proxy{
    ID:       4,
    Type:     models.ProxyTypeStatic,
    Name:     "Landing Page",
    Hostname: "landing.example.com",
    StaticConfig: map[string]interface{}{
        "root_path":          "/var/www/landing",
        "index_file":         "index.html",
        "browse":             false,
        "try_files":          []interface{}{"index.html"},
        "template_rendering": false,
    },
}
```

### Generated Caddy JSON

```go
route, _ := caddy.BuildRouteConfig(proxy)
// Result:
{
    "@id": "proxy_4",
    "match": [{"host": ["landing.example.com"]}],
    "handle": [
        {
            "handler": "file_server",
            "@id": "handler_4",
            "root": "/var/www/landing",
            "index_names": ["index.html"],
            "try_files": ["index.html"]
        }
    ]
}
```

### Generated Caddyfile Snippet

```caddyfile
landing.example.com {
    root * /var/www/landing
    file_server
}
```

---

## Example 6: Static Site with Templates

### Input: Proxy Model

```go
proxy := &models.Proxy{
    ID:       5,
    Type:     models.ProxyTypeStatic,
    Name:     "Maintenance Page",
    Hostname: "maintenance.example.com",
    StaticConfig: map[string]interface{}{
        "root_path":          "/etc/caddy/templates",
        "index_file":         "maintenance.html",
        "template_rendering": true,
    },
}
```

### Generated Caddy JSON

```go
route, _ := caddy.BuildRouteConfig(proxy)
// Result includes templates handler:
{
    "handle": [
        {
            "handler": "templates",
            "templates": {
                "file_root": "/etc/caddy/templates"
            }
        },
        {
            "handler": "file_server",
            "root": "/etc/caddy/templates",
            "index_names": ["maintenance.html"]
        }
    ]
}
```

### Generated Caddyfile Snippet

```caddyfile
maintenance.example.com {
    root * /etc/caddy/templates
    templates
    file_server
}
```

---

## Parsing Caddy Config Back to Proxy Model

### Example: Parse Reverse Proxy Route

```go
// Given a Caddy route config
route := &caddy.RouteConfig{
    ID: "proxy_1",
    Match: []caddy.MatchConfig{
        {Host: []string{"app.example.com"}},
    },
    Handle: []caddy.HandlerConfig{
        {
            Handler: "reverse_proxy",
            Upstreams: []caddy.UpstreamConfig{
                {Dial: "192.168.1.100:8080"},
            },
            Headers: &caddy.HeadersConfig{
                Request: &caddy.HeaderOps{
                    Set: map[string][]string{
                        "X-Real-IP":       {"{http.request.remote.host}"},
                        "X-Custom-Header": {"value"},
                    },
                },
            },
        },
    },
}

// Parse to Proxy model
proxy, err := caddy.ParseRouteToProxy(route)
// Result:
{
    Type: "reverse_proxy",
    Name: "proxy_1",
    Hostname: "app.example.com",
    Upstreams: []interface{}{
        map[string]interface{}{
            "host":   "192.168.1.100",
            "port":   8080,
            "scheme": "http",
        },
    },
    CustomHeaders: map[string]interface{}{
        "X-Custom-Header": "value",
    },
    BlockExploits: true,
    SSLEnabled:    true,
    SSLForced:     true,
    IsActive:      true,
}
```

---

## Simplifying for UI Display

### Example: Simplify Proxy Model for Frontend

```go
proxy := &models.Proxy{
    ID:       1,
    Type:     models.ProxyTypeReverseProxy,
    Name:     "My App",
    Hostname: "app.example.com",
    Upstreams: []interface{}{
        map[string]interface{}{"host": "backend", "port": 8080, "scheme": "http"},
    },
    CreatedBy: 1,
    CreatedAt: time.Now(),
    // ... other fields
}

simplified := caddy.SimplifyProxyForUI(proxy)
// Result - clean JSON for frontend:
{
    "id": 1,
    "type": "reverse_proxy",
    "name": "My App",
    "hostname": "app.example.com",
    "upstreams": [
        {"host": "backend", "port": 8080, "scheme": "http"}
    ],
    "block_exploits": true,
    "ssl_enabled": true,
    "ssl_forced": true,
    "is_active": true,
    "created_at": "2024-11-08T10:30:00Z",
    "updated_at": "2024-11-08T10:30:00Z"
}
// Note: Internal fields like created_by are removed
```

---

## Usage in Service Layer

### Example: Creating a Proxy and Applying to Caddy

```go
func (s *ProxyService) CreateProxy(proxy *models.Proxy) error {
    // 1. Validate
    if err := proxy.Validate(); err != nil {
        return err
    }

    // 2. Save to database
    if err := s.repo.Create(proxy); err != nil {
        return err
    }

    // 3. Generate Caddy configuration
    route, err := caddy.BuildRouteConfig(proxy)
    if err != nil {
        return err
    }

    // 4. Apply to Caddy via Admin API
    path := fmt.Sprintf("/config/apps/http/servers/srv0/routes/%d", proxy.ID)
    if err := s.caddyClient.PATCH(path, route); err != nil {
        return err
    }

    return nil
}
```

### Example: Syncing Database with Caddy Config

```go
func (s *ProxyService) SyncFromCaddy() error {
    // 1. Get current Caddy config
    config, err := s.caddyClient.GetConfig()
    if err != nil {
        return err
    }

    // 2. Parse routes
    routes := extractRoutes(config)

    for _, route := range routes {
        // 3. Convert to Proxy model
        proxy, err := caddy.ParseRouteToProxy(&route)
        if err != nil {
            continue
        }

        // 4. Update database
        s.repo.Create(proxy)
    }

    return nil
}
```

---

## Testing the Functions

You can test these functions with:

```bash
go test ./backend/internal/caddy/...
```

Or manually in a test file:

```go
package main

import (
    "encoding/json"
    "fmt"
    "github.com/aloks98/waygates/backend/internal/caddy"
    "github.com/aloks98/waygates/backend/internal/models"
)

func main() {
    // Create a test proxy
    proxy := &models.Proxy{
        ID:       1,
        Type:     models.ProxyTypeReverseProxy,
        Name:     "Test App",
        Hostname: "test.example.com",
        Upstreams: []interface{}{
            map[string]interface{}{
                "host":   "backend",
                "port":   8080,
                "scheme": "http",
            },
        },
    }

    // Build config
    route, _ := caddy.BuildRouteConfig(proxy)
    jsonBytes, _ := json.MarshalIndent(route, "", "  ")
    fmt.Println("Generated Caddy JSON:")
    fmt.Println(string(jsonBytes))

    // Generate Caddyfile snippet
    snippet := caddy.BuildCaddyfileSnippet(proxy)
    fmt.Println("\nGenerated Caddyfile:")
    fmt.Println(snippet)

    // Parse back
    parsed, _ := caddy.ParseRouteToProxy(route)
    fmt.Printf("\nParsed back to Proxy: %+v\n", parsed)

    // Simplify for UI
    simplified := caddy.SimplifyProxyForUI(proxy)
    simplifiedJSON, _ := json.MarshalIndent(simplified, "", "  ")
    fmt.Println("\nSimplified for UI:")
    fmt.Println(string(simplifiedJSON))
}
```
