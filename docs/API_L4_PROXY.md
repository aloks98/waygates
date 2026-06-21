# L4 (TCP/UDP) Proxy Management API Specification

**Version:** 1.0
**Last Updated:** 2025-02-15

> **Note:** All endpoints in this API are protected and require a valid JWT access token. See the [Authentication API documentation](./API_AUTH.md) for details on how to obtain a token.

## Overview

The L4 Proxy Management API provides endpoints to create, read, update, and delete Layer 4 (TCP/UDP) proxy configurations. Unlike HTTP proxies that operate at Layer 7, L4 proxies route raw TCP/UDP traffic based on connection-level attributes.

**Key Features:**
- TCP and UDP protocol support
- Protocol-aware matchers (TLS/SNI, SSH, PostgreSQL, HTTP, RDP, SOCKS5)
- IP-based access control
- Multiple upstream servers with load balancing
- TLS passthrough or termination
- Proxy Protocol v1/v2 support

## Use Cases

- **Database proxying** - Route PostgreSQL, MySQL connections
- **SSH gateway** - Centralized SSH access point
- **Game servers** - Minecraft, game server proxying
- **TLS passthrough** - SNI-based routing without termination
- **DNS proxying** - UDP-based DNS forwarding
- **Generic TCP/UDP** - Any TCP or UDP service

---

## Base URL

```
/api/l4-proxies
```

## Authentication

All endpoints require JWT token:

```
Authorization: Bearer <jwt_token>
```

## RBAC Permissions

| Permission | Description |
|------------|-------------|
| `l4proxies:read` | List and view L4 proxies |
| `l4proxies:create` | Create new L4 proxies |
| `l4proxies:update` | Update existing L4 proxies |
| `l4proxies:delete` | Delete L4 proxies |

---

## Database Schema

### L4 Proxies Table

```sql
CREATE TABLE l4_proxies (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    listen_port INTEGER NOT NULL,
    protocol VARCHAR(10) NOT NULL DEFAULT 'tcp',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_by INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT uq_l4_proxies_port_protocol UNIQUE (listen_port, protocol),
    CONSTRAINT chk_l4_proxies_protocol CHECK (protocol IN ('tcp', 'udp')),
    CONSTRAINT chk_l4_proxies_port_range CHECK (listen_port >= 1 AND listen_port <= 65535)
);
```

### L4 Routes Table

```sql
CREATE TABLE l4_routes (
    id SERIAL PRIMARY KEY,
    l4_proxy_id INTEGER NOT NULL,
    priority INTEGER NOT NULL DEFAULT 0,
    matcher_type VARCHAR(50) NOT NULL DEFAULT 'any',
    sni_hostnames TEXT[] DEFAULT '{}',
    allowed_ip_ranges TEXT[] DEFAULT '{}',
    regex_pattern VARCHAR(500),
    upstreams JSONB NOT NULL DEFAULT '[]',
    load_balancing_policy VARCHAR(50) NOT NULL DEFAULT 'round_robin',
    tls_terminate BOOLEAN NOT NULL DEFAULT false,
    tls_passthrough BOOLEAN NOT NULL DEFAULT true,
    proxy_protocol_version VARCHAR(5),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_l4_routes_l4_proxy FOREIGN KEY (l4_proxy_id)
        REFERENCES l4_proxies(id) ON DELETE CASCADE
);
```

---

## Data Models

### L4Proxy Object

```json
{
  "id": 1,
  "name": "SSH Gateway",
  "description": "Central SSH access point",
  "listen_port": 2222,
  "protocol": "tcp",
  "is_active": true,
  "created_at": "2025-02-15T10:00:00Z",
  "updated_at": "2025-02-15T10:00:00Z",
  "routes": [
    {
      "id": 1,
      "l4_proxy_id": 1,
      "priority": 0,
      "matcher_type": "ssh",
      "sni_hostnames": [],
      "allowed_ip_ranges": ["10.0.0.0/8", "192.168.0.0/16"],
      "regex_pattern": null,
      "upstreams": [
        {"host": "192.168.1.100", "port": 22, "weight": 1}
      ],
      "load_balancing_policy": "round_robin",
      "tls_terminate": false,
      "tls_passthrough": true,
      "proxy_protocol_version": null,
      "created_at": "2025-02-15T10:00:00Z",
      "updated_at": "2025-02-15T10:00:00Z"
    }
  ]
}
```

### L4Route Object

| Field | Type | Description |
|-------|------|-------------|
| `id` | integer | Route ID |
| `l4_proxy_id` | integer | Parent proxy ID |
| `priority` | integer | Route priority (lower = higher priority) |
| `matcher_type` | string | Connection matcher type |
| `sni_hostnames` | string[] | SNI hostnames for TLS matcher |
| `allowed_ip_ranges` | string[] | CIDR ranges for IP filtering |
| `regex_pattern` | string | Pattern for regexp matcher |
| `upstreams` | object[] | Upstream server list |
| `load_balancing_policy` | string | Load balancing strategy |
| `tls_terminate` | boolean | Terminate TLS at proxy |
| `tls_passthrough` | boolean | Pass TLS through to upstream |
| `proxy_protocol_version` | string | Proxy Protocol version (v1/v2) |

### L4Upstream Object

```json
{
  "host": "192.168.1.100",
  "port": 5432,
  "weight": 1
}
```

---

## Constants

### Protocols

| Value | Description |
|-------|-------------|
| `tcp` | TCP protocol |
| `udp` | UDP protocol |

### Matcher Types

| Value | Description | Required Fields |
|-------|-------------|-----------------|
| `any` | Match all connections | None |
| `tls` | Match TLS connections by SNI | `sni_hostnames` |
| `ssh` | Match SSH protocol | None |
| `postgres` | Match PostgreSQL protocol | None |
| `http` | Match HTTP protocol (L4 level) | None |
| `rdp` | Match RDP protocol | None |
| `socks5` | Match SOCKS5 protocol | None |
| `remote_ip` | Match by client IP | `allowed_ip_ranges` |
| `regexp` | Match by regex pattern | `regex_pattern` |

### Load Balancing Policies

| Value | Description |
|-------|-------------|
| `round_robin` | Distribute evenly across upstreams |
| `least_conn` | Send to upstream with fewest connections |
| `random` | Random upstream selection |
| `first` | Always use first available upstream |
| `ip_hash` | Consistent hashing based on client IP |

---

## Endpoints

### List L4 Proxies

```
GET /api/l4-proxies
```

**Permission:** `l4proxies:read`

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `page` | integer | 1 | Page number |
| `limit` | integer | 20 | Items per page (max: 100) |
| `search` | string | - | Search by name/description |
| `protocol` | string | - | Filter by protocol (tcp/udp) |
| `is_active` | string | - | Filter by status (true/false) |
| `sort` | string | created_at | Sort field |
| `order` | string | desc | Sort order (asc/desc) |

**Response:**

```json
{
  "success": true,
  "data": {
    "items": [...],
    "total": 25,
    "page": 1,
    "limit": 20,
    "total_pages": 2
  }
}
```

---

### Get L4 Proxy

```
GET /api/l4-proxies/{id}
```

**Permission:** `l4proxies:read`

**Response:**

```json
{
  "success": true,
  "data": {
    "id": 1,
    "name": "PostgreSQL Proxy",
    "listen_port": 5432,
    "protocol": "tcp",
    "is_active": true,
    "routes": [...]
  }
}
```

**Error Responses:**

| Status | Code | Description |
|--------|------|-------------|
| 404 | `L4_PROXY_NOT_FOUND` | Proxy not found |

---

### Create L4 Proxy

```
POST /api/l4-proxies
```

**Permission:** `l4proxies:create`

**Request Body:**

```json
{
  "name": "PostgreSQL Proxy",
  "description": "Database connection proxy",
  "listen_port": 5432,
  "protocol": "tcp",
  "is_active": true,
  "routes": [
    {
      "priority": 0,
      "matcher_type": "postgres",
      "allowed_ip_ranges": ["10.0.0.0/8"],
      "upstreams": [
        {"host": "db-primary.internal", "port": 5432, "weight": 2},
        {"host": "db-replica.internal", "port": 5432, "weight": 1}
      ],
      "load_balancing_policy": "least_conn",
      "tls_passthrough": true
    }
  ]
}
```

**Validation Rules:**

| Field | Rules |
|-------|-------|
| `name` | Required, max 255 characters |
| `listen_port` | Required, 1-65535 |
| `protocol` | Required, "tcp" or "udp" |
| `routes[].upstreams` | At least one upstream required |
| `routes[].upstreams[].host` | Required |
| `routes[].upstreams[].port` | Required, 1-65535 |

**Error Responses:**

| Status | Code | Description |
|--------|------|-------------|
| 400 | `VALIDATION_ERROR` | Invalid input |
| 409 | `L4_PROXY_PORT_CONFLICT` | Port+protocol already in use |

---

### Update L4 Proxy

```
PUT /api/l4-proxies/{id}
```

**Permission:** `l4proxies:update`

**Request Body:** Same as Create

**Error Responses:**

| Status | Code | Description |
|--------|------|-------------|
| 404 | `L4_PROXY_NOT_FOUND` | Proxy not found |
| 409 | `L4_PROXY_PORT_CONFLICT` | Port+protocol already in use |

---

### Delete L4 Proxy

```
DELETE /api/l4-proxies/{id}
```

**Permission:** `l4proxies:delete`

**Response:**

```json
{
  "success": true,
  "data": null
}
```

---

### Toggle Active Status

```
PATCH /api/l4-proxies/{id}/toggle
```

**Permission:** `l4proxies:update`

**Response:**

```json
{
  "success": true,
  "data": {
    "id": 1,
    "is_active": false
  }
}
```

---

### Bulk Enable L4 Proxies

```
POST /api/l4-proxies/bulk/enable
```

**Permission:** `l4proxies:update`

Enables multiple L4 proxies in a single request. The operation is **best-effort**: each
id is processed independently and a failure on one id never aborts the batch.

**Request Body:**

```json
{
  "ids": [1, 2, 3]
}
```

- `ids` (array of integers, required): The proxy ids to enable. Must be non-empty and
  contain at most **1000** ids.

**Response (200 OK):**

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
      { "id": 2, "status": "error", "error": "l4 proxy not found" },
      { "id": 3, "status": "ok" }
    ]
  }
}
```

- `status` is `"ok"` or `"error"`. `error` is present only when `status` is `"error"`.

**Error Responses:**
- **400 Bad Request**: If `ids` is empty, contains more than 1000 entries, or the body is malformed.
- **401 Unauthorized**: If the JWT token is missing or invalid.
- **403 Forbidden**: If the user lacks the `l4proxies:update` permission.

---

### Bulk Disable L4 Proxies

```
POST /api/l4-proxies/bulk/disable
```

**Permission:** `l4proxies:update`

Disables multiple L4 proxies in a single request. Same request/response shape and
best-effort semantics as **Bulk Enable L4 Proxies**.

**Request Body:**

```json
{
  "ids": [1, 2, 3]
}
```

**Response (200 OK):**

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

**Error Responses:**
- **400 Bad Request**: If `ids` is empty, contains more than 1000 entries, or the body is malformed.
- **401 Unauthorized**: If the JWT token is missing or invalid.
- **403 Forbidden**: If the user lacks the `l4proxies:update` permission.

---

### Bulk Delete L4 Proxies

```
POST /api/l4-proxies/bulk/delete
```

**Permission:** `l4proxies:delete`

Deletes multiple L4 proxies in a single request. Same request/response shape and
best-effort semantics as **Bulk Enable L4 Proxies**.

**Request Body:**

```json
{
  "ids": [1, 2, 3]
}
```

**Response (200 OK):**

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
      { "id": 2, "status": "error", "error": "l4 proxy not found" },
      { "id": 3, "status": "ok" }
    ]
  }
}
```

**Error Responses:**
- **400 Bad Request**: If `ids` is empty, contains more than 1000 entries, or the body is malformed.
- **401 Unauthorized**: If the JWT token is missing or invalid.
- **403 Forbidden**: If the user lacks the `l4proxies:delete` permission.

---

### Export L4 Proxies

```
GET /api/l4-proxies/export
```

**Permission:** `l4proxies:read`

Returns L4 proxies as a JSON array of portable export objects suitable for re-import.
Server-managed fields (`id`, `l4_proxy_id`, `created_at`, `updated_at`, `created_by`)
are dropped; everything needed to recreate the proxy is kept, including `is_active`
so an exported inactive proxy imports inactive.

**Query Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `ids` | string | Optional comma-separated proxy ids (e.g. `?ids=1,2,3`). When provided, exactly these proxies are exported and any id that no longer exists is silently skipped. |
| `search` | string | Optional. Substring match on name. Ignored when `ids` is provided. |
| `protocol` | string | Optional. `tcp` or `udp`. Ignored when `ids` is provided. |
| `is_active` | string | Optional. `true` or `false`. Ignored when `ids` is provided. |

When no `ids` are supplied, all L4 proxies matching the given filters are exported (no pagination).

**Response (200 OK):**

```json
{
  "success": true,
  "data": [
    {
      "name": "TLS Router",
      "listen_port": 443,
      "protocol": "tcp",
      "is_active": true,
      "routes": [
        {
          "priority": 0,
          "matcher_type": "tls",
          "sni_hostnames": ["api.example.com"],
          "upstreams": [{"host": "api-server", "port": 8443}],
          "load_balancing_policy": "round_robin",
          "tls_terminate": false,
          "tls_passthrough": true
        }
      ]
    }
  ]
}
```

`data` is an array of export objects. Per object: `description`, `sni_hostnames`,
`allowed_ip_ranges`, `regex_pattern`, `weight`, and `proxy_protocol_version` are
omitted when empty/nil.

**Error Responses:**
- **400 Bad Request**: If `ids` contains a non-integer value, or a filter parameter is invalid.
- **401 Unauthorized**: If the JWT token is missing or invalid.
- **403 Forbidden**: If the user lacks the `l4proxies:read` permission.

---

### Get Statistics

```
GET /api/l4-proxies/stats
```

**Permission:** `l4proxies:read`

**Response:**

```json
{
  "success": true,
  "data": {
    "total_proxies": 10,
    "active_proxies": 8,
    "tcp_proxies": 7,
    "udp_proxies": 3,
    "total_routes": 15,
    "total_upstreams": 25
  }
}
```

---

## Examples

### Example 1: TLS Passthrough with SNI Routing

Route TLS traffic to different backends based on SNI hostname:

```json
{
  "name": "TLS Router",
  "listen_port": 443,
  "protocol": "tcp",
  "routes": [
    {
      "priority": 0,
      "matcher_type": "tls",
      "sni_hostnames": ["api.example.com"],
      "upstreams": [{"host": "api-server", "port": 8443}],
      "tls_passthrough": true
    },
    {
      "priority": 1,
      "matcher_type": "tls",
      "sni_hostnames": ["web.example.com"],
      "upstreams": [{"host": "web-server", "port": 443}],
      "tls_passthrough": true
    }
  ]
}
```

### Example 2: SSH Gateway with IP Restriction

```json
{
  "name": "SSH Gateway",
  "listen_port": 2222,
  "protocol": "tcp",
  "routes": [
    {
      "matcher_type": "ssh",
      "allowed_ip_ranges": ["10.0.0.0/8", "192.168.1.0/24"],
      "upstreams": [{"host": "bastion.internal", "port": 22}]
    }
  ]
}
```

### Example 3: PostgreSQL Load Balancer

```json
{
  "name": "PostgreSQL LB",
  "listen_port": 5432,
  "protocol": "tcp",
  "routes": [
    {
      "matcher_type": "postgres",
      "upstreams": [
        {"host": "pg-primary", "port": 5432, "weight": 2},
        {"host": "pg-replica-1", "port": 5432, "weight": 1},
        {"host": "pg-replica-2", "port": 5432, "weight": 1}
      ],
      "load_balancing_policy": "least_conn"
    }
  ]
}
```

### Example 4: DNS Proxy (UDP)

```json
{
  "name": "DNS Forwarder",
  "listen_port": 53,
  "protocol": "udp",
  "routes": [
    {
      "matcher_type": "any",
      "upstreams": [
        {"host": "8.8.8.8", "port": 53},
        {"host": "8.8.4.4", "port": 53}
      ],
      "load_balancing_policy": "round_robin"
    }
  ]
}
```

---

## Docker Port Mapping

L4 proxies use custom ports that must be exposed in your Docker configuration:

```yaml
# docker-compose.yml
services:
  waygates:
    ports:
      - "80:80"
      - "443:443"
      # L4 Proxy ports
      - "5432:5432"      # PostgreSQL
      - "2222:2222"      # SSH
      - "53:53/udp"      # DNS
      - "25565:25565"    # Minecraft
```

---

## Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `L4_PROXY_NOT_FOUND` | 404 | L4 proxy does not exist |
| `L4_PROXY_PORT_CONFLICT` | 409 | Port+protocol combination already in use |
| `VALIDATION_ERROR` | 400 | Request validation failed |
| `UNAUTHORIZED` | 401 | Missing or invalid authentication |
| `FORBIDDEN` | 403 | Insufficient permissions |
