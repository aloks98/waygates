# Curl Examples for Testing API

This document provides curl commands to test all API endpoints.

**Base URL**: `http://localhost:8080/api`

---

## 1. Authentication

First, you need to register a user and log in to get an access token.

### a. Register a new user

```bash
curl -X POST "http://localhost:8080/api/auth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Admin",
    "username": "admin",
    "email": "admin@example.com",
    "password": "password123"
  }'
```

### b. Login to get tokens

```bash
curl -X POST "http://localhost:8080/api/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "identifier": "admin",
    "password": "password123"
  }'
```

### c. Store the access token

From the login response, copy the `access_token` and store it in an environment variable for convenience.

```bash
export TOKEN="ey..."
export REFRESH_TOKEN="ey..."
```

### d. Refresh token

```bash
curl -X POST "http://localhost:8080/api/auth/refresh" \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "'"$REFRESH_TOKEN"'"
  }'
```

### e. Get current user

```bash
curl -X GET "http://localhost:8080/api/auth/me" \
  -H "Authorization: Bearer $TOKEN"
```

### f. Logout

```bash
# Logout (revoke access token only)
curl -X POST "http://localhost:8080/api/auth/logout" \
  -H "Authorization: Bearer $TOKEN"

# Logout (revoke both tokens - recommended)
curl -X POST "http://localhost:8080/api/auth/logout" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "'"$REFRESH_TOKEN"'"
  }'
```

---

## 2. Get Application Status

**GET** `/api/status`

```bash
curl -X GET "http://localhost:8080/api/status"
```

---

## 3. Proxy Management Endpoints (Authentication Required)

**Note**: All proxy endpoints require authentication. Ensure you have obtained and exported a `$TOKEN` as described in the Authentication section.

### a. List Proxies

**GET** `/api/proxies`

```bash
# List all proxies (default pagination)
curl -X GET "http://localhost:8080/api/proxies" \
  -H "Authorization: Bearer $TOKEN"

# List with pagination
curl -X GET "http://localhost:8080/api/proxies?page=1&limit=10" \
  -H "Authorization: Bearer $TOKEN"

# Search by name/hostname
curl -X GET "http://localhost:8080/api/proxies?search=example" \
  -H "Authorization: Bearer $TOKEN"

# Filter by type (reverse_proxy, redirect, static)
curl -X GET "http://localhost:8080/api/proxies?type=reverse_proxy" \
  -H "Authorization: Bearer $TOKEN"

# Filter by status
curl -X GET "http://localhost:8080/api/proxies?status=active" \
  -H "Authorization: Bearer $TOKEN"

# Sort and order
curl -X GET "http://localhost:8080/api/proxies?sort=created_at&order=desc" \
  -H "Authorization: Bearer $TOKEN"
```

### b. Get Proxy by ID

**GET** `/api/proxies/:id`

```bash
curl -X GET "http://localhost:8080/api/proxies/1" \
  -H "Authorization: Bearer $TOKEN"
```

### c. Create Reverse Proxy (Simple)

**POST** `/api/proxies`

```bash
curl -X POST "http://localhost:8080/api/proxies" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "type": "reverse_proxy",
    "name": "My Backend API",
    "hostname": "api.example.com",
    "upstreams": [
      {
        "host": "192.168.1.100",
        "port": 8080,
        "scheme": "http"
      }
    ],
    "block_exploits": true
  }'
```

### d. Create Reverse Proxy (HTTPS Backend with Self-Signed Certificate)

**POST** `/api/proxies`

```bash
curl -X POST "http://localhost:8080/api/proxies" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
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

**Note**: Set `tls_insecure_skip_verify: true` when proxying to HTTPS backends with self-signed certificates.

### e. Create Reverse Proxy (Load Balanced with Health Checks)

**POST** `/api/proxies`

```bash
curl -X POST "http://localhost:8080/api/proxies" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "type": "reverse_proxy",
    "name": "Load Balanced App",
    "hostname": "app.example.com",
    "upstreams": [
      {
        "host": "backend1.local",
        "port": 8080,
        "scheme": "http"
      },
      {
        "host": "backend2.local",
        "port": 8080,
        "scheme": "http"
      },
      {
        "host": "backend3.local",
        "port": 8080,
        "scheme": "http"
      }
    ],
    "load_balancing": {
      "strategy": "round_robin",
      "health_checks": {
        "enabled": true,
        "path": "/health",
        "interval": "30s",
        "timeout": "5s",
        "unhealthy_threshold": 3,
        "healthy_threshold": 2
      }
    },
    "block_exploits": true,
    "custom_headers": {
      "request": {
        "X-App-Version": "1.0.0"
      },
      "response": {
        "X-Frame-Options": "SAMEORIGIN"
      }
    }
  }'
```

**Note**: `custom_headers` uses the nested `{"request":{...},"response":{...}}` shape. A flat map (e.g. `{"X-App-Version":"1.0.0"}`) is still accepted for backward compatibility and is treated as request headers.

### f. Create Redirect (301 Permanent)

**POST** `/api/proxies`

```bash
curl -X POST "http://localhost:8080/api/proxies" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "type": "redirect",
    "name": "Old Domain Redirect",
    "hostname": "old.example.com",
    "redirect": {
      "target": "https://new.example.com",
      "status_code": 301,
      "preserve_path": true,
      "preserve_query": true
    }
  }'
```

### g. Create Redirect (302 Temporary, No Path Preservation)

**POST** `/api/proxies`

```bash
curl -X POST "http://localhost:8080/api/proxies" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "type": "redirect",
    "name": "Temporary Redirect",
    "hostname": "temp.example.com",
    "redirect": {
      "target": "https://target.example.com/landing",
      "status_code": 302,
      "preserve_path": false,
      "preserve_query": false
    }
  }'
```

### h. Create Static Site (Simple)

**POST** `/api/proxies`

```bash
curl -X POST "http://localhost:8080/api/proxies" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "type": "static",
    "name": "Marketing Site",
    "hostname": "marketing.example.com",
    "static": {
      "root_path": "/var/www/marketing",
      "index_file": "index.html",
      "browse": false,
      "template_rendering": false
    }
  }'
```

### i. Create Static Site (SPA with Try Files)

**POST** `/api/proxies`

```bash
curl -X POST "http://localhost:8080/api/proxies" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "type": "static",
    "name": "React App",
    "hostname": "app.example.com",
    "static": {
      "root_path": "/var/www/react-app/build",
      "index_file": "index.html",
      "browse": false,
      "template_rendering": false,
      "try_files": ["/index.html"]
    }
  }'
```

### j. Create Static Site (With Template Rendering)

**POST** `/api/proxies`

```bash
curl -X POST "http://localhost:8080/api/proxies" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "type": "static",
    "name": "Blog",
    "hostname": "blog.example.com",
    "static": {
      "root_path": "/var/www/blog",
      "index_file": "index.html",
      "browse": true,
      "template_rendering": true
    }
  }'
```

### k. Update Proxy

**PUT** `/api/proxies/:id`

```bash
# Update hostname and add custom header
curl -X PUT "http://localhost:8080/api/proxies/1" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "type": "reverse_proxy",
    "name": "My Backend API (Updated)",
    "hostname": "api-v2.example.com",
    "upstreams": [
      {
        "host": "192.168.1.200",
        "port": 9090,
        "scheme": "http"
      }
    ],
    "block_exploits": true,
    "custom_headers": {
      "request": {
        "X-API-Version": "2.0"
      },
      "response": {
        "X-Frame-Options": "SAMEORIGIN"
      }
    }
  }'
```

**Note**: `custom_headers` uses the nested `{"request":{...},"response":{...}}` shape. A flat map (e.g. `{"X-API-Version":"2.0"}`) is still accepted for backward compatibility and is treated as request headers.

### l. Delete Proxy

**DELETE** `/api/proxies/:id`

```bash
curl -X DELETE "http://localhost:8080/api/proxies/1" \
  -H "Authorization: Bearer $TOKEN"
```

### m. Enable Proxy

**POST** `/api/proxies/:id/enable`

```bash
curl -X POST "http://localhost:8080/api/proxies/1/enable" \
  -H "Authorization: Bearer $TOKEN"
```

### n. Disable Proxy

**POST** `/api/proxies/:id/disable`

```bash
curl -X POST "http://localhost:8080/api/proxies/1/disable" \
  -H "Authorization: Bearer $TOKEN"
```

---

## 4. Testing Workflow

Here's a recommended testing workflow:

### 1. Start the server
```bash
cd /Users/aloks98/waygates
go run backend/cmd/server/main.go
```

### 2. Register and Login
See Section 1 above to register a user and get an access token.
Make sure to `export` the token.

### 3. Create a test reverse proxy
```bash
curl -X POST "http://localhost:8080/api/proxies" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "type": "reverse_proxy",
    "name": "Test App",
    "hostname": "test.example.com",
    "upstreams": [
      {
        "host": "localhost",
        "port": 9999,
        "scheme": "http"
      }
    ],
    "block_exploits": true
  }'
```

### 4. List all proxies to verify creation
```bash
curl -X GET "http://localhost:8080/api/proxies" -H "Authorization: Bearer $TOKEN"
```

### 5. Get the specific proxy (replace 1 with actual ID)
```bash
curl -X GET "http://localhost:8080/api/proxies/1" -H "Authorization: Bearer $TOKEN"
```

### 6. Disable the proxy
```bash
curl -X POST "http://localhost:8080/api/proxies/1/disable" -H "Authorization: Bearer $TOKEN"
```

### 7. Re-enable the proxy
```bash
curl -X POST "http://localhost:8080/api/proxies/1/enable" -H "Authorization: Bearer $TOKEN"
```

### 8. Update the proxy
```bash
curl -X PUT "http://localhost:8080/api/proxies/1" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "type": "reverse_proxy",
    "name": "Test App (Updated)",
    "hostname": "test-v2.example.com",
    "upstreams": [
      {
        "host": "localhost",
        "port": 8888,
        "scheme": "http"
      }
    ],
    "block_exploits": true
  }'
```

### 9. Delete the proxy
```bash
curl -X DELETE "http://localhost:8080/api/proxies/1" -H "Authorization: Bearer $TOKEN"
```

---

## Response Format

### Success Response (200 OK, 201 Created)
```json
{
  "success": true,
  "message": "Proxy created successfully",
  "data": {
    "id": 1,
    "type": "reverse_proxy",
    "name": "My Backend API",
    "hostname": "api.example.com",
    "upstreams": [...],
    "is_active": true,
    "ssl_enabled": true,
    "ssl_forced": true,
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

### Error Response (400, 404, 409, 500)
```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Hostname is required",
    "details": null
  }
}
```

---

## Common Error Codes

- `VALIDATION_ERROR` (400) - Invalid request data
- `NOT_FOUND` (404) - Proxy not found
- `CONFLICT` (409) - Hostname already exists
- `INTERNAL_ERROR` (500) - Server error

---

## Pretty Print JSON Responses

Add `| jq` to the end of curl commands for formatted output:

```bash
curl -X GET "http://localhost:8080/api/proxies" | jq
```

Install jq:
```bash
# macOS
brew install jq

# Linux (Debian/Ubuntu)
sudo apt-get install jq
```