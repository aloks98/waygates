# Waygates REST API Reference

Waygates exposes a JSON REST API served by the Go backend on port 8080. All management endpoints live under the `/api` prefix. The React UI is served at the root and communicates exclusively with this API.

**Base URL**: `http(s)://<host>:8080/api`

---

## Authentication

All protected endpoints require a JWT access token in the `Authorization` header:

```
Authorization: Bearer <access_token>
```

Tokens are obtained via `POST /api/auth/login` or `POST /api/auth/register`. Tokens are JWTs issued by the `goauth` library and carry the user ID and role claims. The access token is short-lived; use the refresh token to obtain a new pair.

---

## Standard Response Envelope

Every response (success and error) is wrapped in the same envelope.

**Success** (`200 OK` or `201 Created`):

```json
{
  "success": true,
  "message": "Human-readable note (may be empty)",
  "data": { ... }
}
```

**Error** (`400`, `401`, `403`, `404`, `409`, `500`, `502`):

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Description of the problem",
    "details": null
  }
}
```

**Error codes**: `VALIDATION_ERROR` (400), `UNAUTHORIZED` (401), `FORBIDDEN` (403), `NOT_FOUND` (404), `CONFLICT` (409), `INTERNAL_ERROR` (500), `EXTERNAL_SERVICE_ERROR` (502).

**Paginated lists** include this shape inside `data`:

```json
{
  "items": [...],
  "total": 42,
  "page": 1,
  "limit": 20,
  "total_pages": 3
}
```

---

## Quickstart

```bash
# 1. Register (first user automatically becomes admin)
curl -s -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"name":"Admin","username":"admin","email":"admin@example.com","password":"password123"}'

# 2. Login and capture tokens
RESP=$(curl -s -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"identifier":"admin","password":"password123"}')
TOKEN=$(echo $RESP | jq -r '.data.access_token')
REFRESH_TOKEN=$(echo $RESP | jq -r '.data.refresh_token')

# 3. Authenticated call
curl -s http://localhost:8080/api/proxies \
  -H "Authorization: Bearer $TOKEN" | jq
```

---

## RBAC / Roles

Permissions are checked per-endpoint by the `goauth` middleware. Role templates defined in `backend/rbac.yaml`:

| Role | Key permissions |
|------|----------------|
| `admin` | All permissions (`*`) |
| `operator` | `proxies:*`, `l4proxies:*`, `acl:*`, `settings:read/write`, `sync:*`, `audit_logs:read`, `metrics:read` |
| `viewer` | `proxies:read`, `l4proxies:read`, `acl:read`, `settings:read`, `sync:read`, `metrics:read` |

The first registered user is automatically assigned `admin`; subsequent users get `operator`.

---

## 1. Health & Status

These endpoints are public (no authentication required).

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/health` | Service health check |
| `GET` | `/api/status` | Application and Caddy status |

### GET /api/health

Returns overall health and per-component status.

**Response `data`**:
```json
{
  "status": "healthy",
  "service": "caddy-manager-backend",
  "version": "1.0.0",
  "uptime": "2h30m15s",
  "time": "2024-01-01T12:00:00Z",
  "components": {
    "database": "healthy"
  }
}
```

`status` is `"degraded"` when the database is unreachable.

### GET /api/status

Returns Caddy liveness and whether any user account has been created (useful for the initial setup flow).

**Response `data`**:
```json
{
  "caddy_status": "healthy",
  "user_setup_complete": true
}
```

---

## 2. Auth & User

Rate-limited endpoints (register, login, ACL login): 10 requests per IP per minute.

| Method | Path | Auth required | Description |
|--------|------|---------------|-------------|
| `POST` | `/api/auth/register` | No | Register a new user |
| `POST` | `/api/auth/login` | No | Login, get token pair |
| `POST` | `/api/auth/refresh` | No | Rotate access + refresh tokens |
| `POST` | `/api/auth/logout` | Yes | Revoke tokens |
| `GET` | `/api/auth/me` | Yes | Get current user + permissions |
| `POST` | `/api/auth/change-password` | Yes | Change own password |

### POST /api/auth/register

**Request body**:
```json
{
  "name": "Alice Smith",
  "username": "alice",
  "email": "alice@example.com",
  "password": "securepass123"
}
```

**Response**: `201 Created`, `data` is the created `User` object (`id`, `name`, `username`, `email`, `created_at`, `updated_at`).

The first registered user is assigned `admin`; all subsequent users get `operator`.

### POST /api/auth/login

**Request body**:
```json
{
  "identifier": "alice",
  "password": "securepass123"
}
```

`identifier` accepts either username or email.

**Response `data`**:
```json
{
  "access_token": "eyJ...",
  "refresh_token": "eyJ..."
}
```

### POST /api/auth/refresh

**Request body**:
```json
{ "refresh_token": "eyJ..." }
```

**Response `data`**: Same shape as login (`access_token`, `refresh_token`). Refresh tokens are rotated on use.

### POST /api/auth/logout

Revokes the access token from the `Authorization` header. Optionally revokes the refresh token if provided in the body.

**Request body** (optional):
```json
{ "refresh_token": "eyJ..." }
```

### GET /api/auth/me

**Response `data`**:
```json
{
  "id": 1,
  "name": "Alice Smith",
  "username": "alice",
  "email": "alice@example.com",
  "role": "Administrator",
  "permissions": ["proxies:read", "proxies:create", "..."]
}
```

### POST /api/auth/change-password

**Request body**:
```json
{
  "current_password": "oldpass",
  "new_password": "newpass123"
}
```

New password must be at least 8 characters.

---

## 3. ACL Forward Auth & OAuth

These endpoints serve both Caddy's `forward_auth` integration and the end-user ACL login page. They are **public** (no Waygates JWT required) except where noted.

### 3.1 Forward Auth (called by Caddy)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/auth/acl/verify` | Forward auth gate called by Caddy on every protected request |
| `POST` | `/api/auth/acl/login` | Login using Waygates credentials; sets `waygates_acl_session` cookie |
| `POST` | `/api/auth/acl/logout` | Revoke ACL session and clear cookie |
| `GET` | `/api/auth/acl/session` | Return current ACL session info |
| `GET` | `/api/acl/options` | Return available auth methods for the login page |
| `GET` | `/api/acl/branding` | Return login page branding config |

#### GET /api/auth/acl/verify

Called by Caddy's `forward_auth` directive. Uses the `waygates_acl_session` cookie and/or `Authorization: Basic ...` header to evaluate ACL rules for the proxied request.

- Returns `200` when access is granted (Caddy forwards the request).
- Returns `401` with `X-Auth-Redirect` header when authentication is required.
- Returns `403` with optional `X-Auth-Denial-Reason`, `X-Auth-User-Email`, `X-Auth-Provider` headers when access is explicitly denied.

Relies on `X-Forwarded-Host`, `X-Forwarded-Uri`, `X-Forwarded-Method`, and `X-Waygates-Client-IP` headers set by Caddy.

#### POST /api/auth/acl/login

**Request body**:
```json
{
  "email": "user@example.com",
  "password": "pass",
  "redirect": "https://app.example.com/dashboard"
}
```

On success, sets the `waygates_acl_session` cookie (HttpOnly, SameSite=Lax, domain derived from `redirect` URL) and returns:

```json
{
  "success": true,
  "redirect_url": "https://app.example.com/dashboard",
  "message": "Login successful"
}
```

#### GET /api/auth/acl/session

Returns session state. `data.authenticated` is `false` when no valid session exists.

```json
{
  "authenticated": true,
  "user_id": 1,
  "username": "alice",
  "email": "alice@example.com",
  "expires_at": "2024-01-02T12:00:00Z",
  "oauth_email": "",
  "oauth_provider": ""
}
```

#### GET /api/acl/options

Returns the auth methods available for the current ACL context (used by the login page to decide which login forms to show).

#### GET /api/acl/branding

Returns ACL login page branding (`logo_url`, `primary_color`, `background_color`, `title`, `subtitle`, `footer_text`, `custom_css`).

### 3.2 OAuth Flow

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/auth/oauth/providers` | List configured OAuth providers |
| `GET` | `/auth/oauth/{provider}` | Initiate OAuth flow (redirect to provider) |
| `GET` | `/auth/oauth/{provider}/callback` | OAuth callback (redirect back after auth) |

Note: The OAuth initiation and callback routes are at `/auth/oauth/...` (no `/api` prefix) because OAuth providers redirect back to a bare path, not an API path.

Supported providers: `google`, `github`, `microsoft`, `gitlab`.

#### GET /api/auth/oauth/providers

Returns a list of configured (env-var-present) providers:

```json
[
  { "id": "google", "name": "Google", "available": true, "enabled": true }
]
```

#### GET /auth/oauth/{provider}

Initiates PKCE OAuth flow. Query params:
- `redirect` (optional): URL to redirect to after successful login. Must be a relative path or a hostname matching a configured proxy.

Stores state and PKCE cookies, then issues `302` redirect to the provider's authorization URL.

#### GET /auth/oauth/{provider}/callback

Handled internally after provider redirects back. Validates state/PKCE, exchanges code for token, fetches user info, creates or finds the Waygates user account, creates an ACL session, sets `waygates_acl_session` cookie, then redirects to the original URL.

---

## 4. Proxies (HTTP/L7)

All proxy endpoints require authentication.

| Method | Path | RBAC permission | Description |
|--------|------|----------------|-------------|
| `GET` | `/api/proxies` | `proxies:read` | List proxies (paginated, filterable) |
| `GET` | `/api/proxies/stats` | `proxies:read` | Aggregate stats |
| `GET` | `/api/proxies/export` | `proxies:read` | Export proxies as JSON array |
| `GET` | `/api/proxies/{id}` | `proxies:read` | Get single proxy |
| `POST` | `/api/proxies` | `proxies:create` | Create proxy |
| `POST` | `/api/proxies/import` | `proxies:create` | Bulk import proxies |
| `PUT` | `/api/proxies/{id}` | `proxies:update` | Update proxy |
| `POST` | `/api/proxies/{id}/enable` | `proxies:update` | Enable proxy |
| `POST` | `/api/proxies/{id}/disable` | `proxies:update` | Disable proxy |
| `POST` | `/api/proxies/bulk/enable` | `proxies:update` | Bulk enable |
| `POST` | `/api/proxies/bulk/disable` | `proxies:update` | Bulk disable |
| `DELETE` | `/api/proxies/{id}` | `proxies:delete` | Delete proxy |
| `POST` | `/api/proxies/bulk/delete` | `proxies:delete` | Bulk delete |

### Proxy Data Model

| Field | Type | Notes |
|-------|------|-------|
| `id` | int | Auto-increment primary key |
| `type` | string | `"reverse_proxy"`, `"redirect"`, or `"static"` |
| `name` | string | Display name, max 255 chars |
| `hostname` | string | Unique; no scheme, port, or path. Required unless `hostname_label` is used (see below) — in that case it is server-computed and read-only. |
| `description` | string\|null | Optional |
| `group_id` | int\|null | The proxy's `ProxyGroup` (config inheritance) — **not** an ACL group. `null` means ungrouped. |
| `hostname_label` | string\|null | A single DNS label (no dots). Valid **iff** the proxy has a `group_id` **and** that group has a `base_domain`; in that case `hostname` is server-computed as `<hostname_label>.<group's base_domain>` and any `hostname` sent in the request body is ignored. |
| `group_name` | string\|null | Computed; the group's name, when grouped (list only) |
| `ssl_enabled` | bool\|null | Tri-state: `true`/`false` is explicit; `null` means **inherit** — from the group's `ssl_enabled` if grouped and the group has an opinion, else the system default `true` |
| `ssl_forced` | bool\|null | Same tri-state/inherit semantics as `ssl_enabled`; system default `true` |
| `is_active` | bool | Default `true` on create. **Not** inheritable — always explicit. |
| `block_exploits` | bool\|null | Same tri-state/inherit semantics as `ssl_enabled`; system default `true` |
| `tls_insecure_skip_verify` | bool\|null | Same tri-state/inherit semantics as `ssl_enabled`; system default `false` |
| `upstreams` | array | Required for `reverse_proxy`; each item: `{host, port, scheme}` |
| `load_balancing` | object\|null | `{strategy, health_checks}` |
| `redirect` | object\|null | Required for `redirect`: `{target, status_code, preserve_path, preserve_query}` |
| `static` | object\|null | Required for `static`: `{root_path, index_file, browse, template_rendering, try_files}` |
| `custom_headers` | object\|null | `{request: {k:v}, response: {k:v}}`. Legacy flat map accepted (treated as request headers). Merged with the group's headers when grouped (proxy wins per key). |
| `acl_group_count` | int | Computed; number of ACL groups assigned (list only) |
| `acl_group_names` | string[] | Computed; ACL group names (list only) |
| `created_at` | RFC3339 | |
| `updated_at` | RFC3339 | |

**`tls_insecure_skip_verify`**: When `true`, Caddy skips TLS certificate verification when connecting to the upstream. Use only for internal services with self-signed certificates.

**Inheritance**: `ssl_enabled`, `ssl_forced`, `block_exploits`, and `tls_insecure_skip_verify` are tri-state. The list/detail responses above always return the **raw** value as stored (`null` for "this proxy has no opinion"); see `GET /api/proxies/{id}` below for the **resolved** value actually served.

### GET /api/proxies/{id} — effective config and \_source

The detail response returns the raw proxy fields (as in the table above) **plus** an `effective` object: the fully-resolved tri-state settings (what Caddy actually serves), and a `_source` map telling the UI where each resolved value came from — `"proxy"` (this proxy set it explicitly), `"group"` (inherited from the group), or `"default"` (neither has an opinion; the system default applies). Without `_source`, an inherited "on" and an explicit "on" are indistinguishable in the edit form.

```json
{
  "success": true,
  "data": {
    "id": 12,
    "hostname": "abc.internal.example.com",
    "group_id": 3,
    "hostname_label": "abc",
    "ssl_enabled": null,
    "block_exploits": false,
    "...": "...other raw Proxy fields...",
    "effective": {
      "ssl_enabled": true,
      "ssl_forced": true,
      "block_exploits": false,
      "tls_insecure_skip_verify": false,
      "custom_headers": { "request": {}, "response": {} },
      "_source": {
        "ssl_enabled": "group",
        "ssl_forced": "default",
        "block_exploits": "proxy",
        "tls_insecure_skip_verify": "default"
      }
    }
  },
  "error": null
}
```

### GET /api/proxies — Query Parameters

| Param | Description |
|-------|-------------|
| `page` | Page number (default 0) |
| `limit` | Page size 0-100 (default: service default) |
| `search` | Text search on name/hostname |
| `sort` | Field to sort by (includes `group_id`) |
| `order` | `asc` or `desc` |
| `type` | Filter by type, supports operators: `eq:reverse_proxy`, `in:reverse_proxy,redirect`, `not:static` |
| `status` | `active` or `inactive`; supports `eq:` and `not:` operators |
| `ssl_enabled` | `true` or `false`. Matches the **effective** (resolved) value, not just proxies with an explicit column value — an ungrouped proxy with no opinion, or a proxy inheriting `true` from its group, both match `ssl_enabled=true`. |
| `target` | Search in upstreams / redirect config |
| `group` | Filter by `ProxyGroup`. Supports `eq:<id>`, `in:<id1>,<id2>`, `not:<id>` (excludes that group but **still includes ungrouped proxies**), and `eq:none` (ungrouped only) |

### POST /api/proxies — Create

```json
{
  "type": "reverse_proxy",
  "name": "My Backend",
  "hostname": "api.example.com",
  "upstreams": [
    { "host": "192.168.1.100", "port": 8080, "scheme": "http" }
  ],
  "block_exploits": true,
  "tls_insecure_skip_verify": false,
  "ssl_enabled": true,
  "ssl_forced": true,
  "custom_headers": {
    "request": { "X-Real-IP": "{http.request.remote.host}" },
    "response": { "X-Frame-Options": "SAMEORIGIN" }
  }
}
```

Returns `201 Created` with the full `Proxy` object.

**Group-addressed create** — omit `hostname` and supply `group_id` + `hostname_label` instead, when the target group has a `base_domain`:

```json
{
  "type": "reverse_proxy",
  "name": "Internal API",
  "group_id": 3,
  "hostname_label": "api",
  "upstreams": [{ "host": "192.168.1.100", "port": 8080, "scheme": "http" }]
}
```

The server computes `hostname` as `<hostname_label>.<group.base_domain>`. Sending `hostname_label` without a `group_id`, or against a group with no `base_domain`, is a `400`; so is a `group_id` for a group **with** a `base_domain` but no `hostname_label`.

### PUT /api/proxies/{id} — Update

Same body shape as create. `group_id` / `hostname_label` can be changed to move a proxy between groups or detach it (set both to `null`) — detaching does **not** change the proxy's existing `hostname`, it just stops inheriting.

> **⚠️ Breaking change**: `ssl_enabled`, `ssl_forced`, `block_exploits`, and `tls_insecure_skip_verify` used to mean "keep the current value" when omitted from a `PUT` body (and `ssl_forced` could not be changed via the API at all — it was always force-preserved from the existing row). **An omitted field on `PUT /api/proxies/{id}` now means "inherit from the group, or the system default if ungrouped"** — `null` cannot mean both "keep existing" and "inherit", and proxy-group inheritance needs the real meaning. The UI always sends all four booleans explicitly (`null` when the user picked "Inherit") so the wire contract is unambiguous; API clients that previously omitted these fields to leave them unchanged must now send the current *effective* value explicitly to preserve prior behavior.

### POST /api/proxies/import

Accepts the same fields as create, including `group_id` / `hostname_label` and the tri-state booleans (`null` = inherit, consistent with create and update).

```json
{
  "dry_run": false,
  "proxies": [ { ...proxy fields... }, ... ]
}
```

Max 1 000 items per request. `dry_run: true` validates without persisting. Response is an import report with per-item results and a summary.

### GET /api/proxies/export

Returns an array of proxy export objects (suitable for re-import). Accepts the same filters as the list endpoint. Optional `ids` query param for selective export: `?ids=1,2,3`.

### POST /api/proxies/bulk/enable | /bulk/disable | /bulk/delete

```json
{ "ids": [1, 2, 3] }
```

Max 1 000 IDs. Response includes `requested`, `succeeded`, and `failed` counts.

### Proxy ACL Assignment (nested)

See section 6.4 below.

### GET /api/proxies/{id}/config-preview

`caddy_config:read` — Returns the generated Caddy JSON snippet for this single proxy (including ACL handlers). Does not write or reload Caddy. See section 11.

---

## 5. L4 Proxies (TCP/UDP)

| Method | Path | RBAC permission | Description |
|--------|------|----------------|-------------|
| `GET` | `/api/l4-proxies` | `l4proxies:read` | List L4 proxies (paginated) |
| `GET` | `/api/l4-proxies/stats` | `l4proxies:read` | Aggregate stats |
| `GET` | `/api/l4-proxies/export` | `l4proxies:read` | Export as JSON array |
| `GET` | `/api/l4-proxies/{id}` | `l4proxies:read` | Get single L4 proxy |
| `POST` | `/api/l4-proxies` | `l4proxies:create` | Create L4 proxy |
| `PUT` | `/api/l4-proxies/{id}` | `l4proxies:update` | Update L4 proxy |
| `PATCH` | `/api/l4-proxies/{id}/toggle` | `l4proxies:update` | Toggle active state |
| `POST` | `/api/l4-proxies/bulk/enable` | `l4proxies:update` | Bulk enable |
| `POST` | `/api/l4-proxies/bulk/disable` | `l4proxies:update` | Bulk disable |
| `DELETE` | `/api/l4-proxies/{id}` | `l4proxies:delete` | Delete L4 proxy |
| `POST` | `/api/l4-proxies/bulk/delete` | `l4proxies:delete` | Bulk delete |

### L4Proxy Data Model

| Field | Type | Notes |
|-------|------|-------|
| `id` | int | |
| `name` | string | Max 255 chars |
| `description` | string\|null | |
| `listen_port` | int | 1-65535 |
| `protocol` | string | `"tcp"` or `"udp"` |
| `is_active` | bool | Default `true` |
| `routes` | L4Route[] | Routing rules, cascading delete |
| `created_at` | RFC3339 | |
| `updated_at` | RFC3339 | |

### L4Route Fields

| Field | Type | Notes |
|-------|------|-------|
| `id` | int | |
| `priority` | int | Lower = higher priority |
| `matcher_type` | string | `any`, `tls`, `ssh`, `postgres`, `http`, `rdp`, `socks5`, `remote_ip`, `regexp` |
| `sni_hostnames` | string[] | Required when `matcher_type=tls` |
| `allowed_ip_ranges` | string[] | Used with `remote_ip` matcher |
| `regex_pattern` | string\|null | Required when `matcher_type=regexp` |
| `upstreams` | L4Upstream[] | At least one required; each: `{host, port, weight?}` |
| `load_balancing_policy` | string | `round_robin`, `least_conn`, `random`, `first`, `ip_hash` |
| `tls_terminate` | bool | Terminate TLS at Caddy |
| `tls_passthrough` | bool | Pass TLS through to upstream (mutually exclusive with `tls_terminate`) |
| `proxy_protocol_version` | string\|null | `"1"` or `"2"` |

### GET /api/l4-proxies/stats Response

```json
{
  "total_proxies": 5,
  "active_proxies": 4,
  "tcp_proxies": 3,
  "udp_proxies": 2,
  "total_routes": 8,
  "total_upstreams": 12
}
```

### Bulk operations

Same `{ "ids": [...] }` shape as HTTP proxies. Max 1 000 IDs.

---

## 6. Access Control (ACL)

### 6.1 ACL Groups

| Method | Path | RBAC permission | Description |
|--------|------|----------------|-------------|
| `GET` | `/api/acl/groups` | `acl:read` | List ACL groups (paginated) |
| `POST` | `/api/acl/groups` | `acl:create` | Create ACL group |
| `GET` | `/api/acl/groups/{id}` | `acl:read` | Get ACL group |
| `PUT` | `/api/acl/groups/{id}` | `acl:update` | Update ACL group |
| `DELETE` | `/api/acl/groups/{id}` | `acl:delete` | Delete ACL group |
| `GET` | `/api/acl/groups/{id}/usage` | `acl:read` | List proxies using this group |

### ACLGroup Data Model

| Field | Type | Notes |
|-------|------|-------|
| `id` | int | |
| `name` | string | Unique, max 255 chars |
| `description` | string\|null | |
| `combination_mode` | string | `"any"` (default), `"all"`, or `"ip_bypass"` |
| `ip_rules` | ACLIPRule[] | |
| `basic_auth_users` | ACLBasicAuthUser[] | Passwords are never returned |
| `external_providers` | ACLExternalProvider[] | |
| `waygates_auth` | ACLWaygatesAuth\|null | |
| `oauth_provider_restrictions` | ACLOAuthProviderRestriction[] | |
| `created_at` | RFC3339 | |
| `updated_at` | RFC3339 | |

**Combination modes**:
- `any`: Access is granted if any configured auth method passes.
- `all`: All configured auth methods must pass.
- `ip_bypass`: IP allow/bypass rules can grant access without requiring other auth methods.

### POST /api/acl/groups

```json
{
  "name": "Internal Users",
  "description": "Employees on the internal network",
  "combination_mode": "any"
}
```

### 6.2 IP Rules

| Method | Path | RBAC permission | Description |
|--------|------|----------------|-------------|
| `GET` | `/api/acl/groups/{id}/ip-rules` | `acl:read` | List IP rules for group |
| `POST` | `/api/acl/groups/{id}/ip-rules` | `acl:update` | Add IP rule to group |
| `PUT` | `/api/acl/ip-rules/{id}` | `acl:update` | Update IP rule |
| `DELETE` | `/api/acl/ip-rules/{id}` | `acl:delete` | Delete IP rule |

### ACLIPRule Fields

| Field | Type | Notes |
|-------|------|-------|
| `id` | int | |
| `acl_group_id` | int | |
| `rule_type` | string | `"allow"`, `"deny"`, or `"bypass"` |
| `cidr` | string | e.g. `"10.0.0.0/8"` |
| `description` | string\|null | |
| `priority` | int | Default 0; higher = checked first |

### 6.3 Basic Auth Users

| Method | Path | RBAC permission | Description |
|--------|------|----------------|-------------|
| `GET` | `/api/acl/groups/{id}/basic-auth` | `acl:read` | List basic auth users |
| `POST` | `/api/acl/groups/{id}/basic-auth` | `acl:update` | Add basic auth user |
| `PUT` | `/api/acl/basic-auth/{id}` | `acl:update` | Update user (username/password) |
| `DELETE` | `/api/acl/basic-auth/{id}` | `acl:delete` | Remove user |

Passwords are bcrypt-hashed; the hash is never returned. `username` is unique within a group.

### 6.4 Proxy ACL Assignments

| Method | Path | RBAC permission | Description |
|--------|------|----------------|-------------|
| `GET` | `/api/proxies/{id}/acl` | `acl:read` | List ACL assignments for a proxy |
| `POST` | `/api/proxies/{id}/acl` | `acl:update` | Assign an ACL group to a proxy |
| `PUT` | `/api/proxies/{id}/acl/{assignmentId}` | `acl:update` | Update assignment (path/priority/enabled) |
| `DELETE` | `/api/proxies/{id}/acl/{groupId}` | `acl:delete` | Remove ACL group from proxy |

### ProxyACLAssignment Fields

| Field | Type | Notes |
|-------|------|-------|
| `id` | int | |
| `proxy_id` | int | |
| `acl_group_id` | int | |
| `path_pattern` | string | Default `"/*"`. Caddy route matcher pattern. |
| `priority` | int | Default 0 |
| `enabled` | bool | Default `true` |

### 6.5 External Providers (Authelia / Authentik / Custom)

| Method | Path | RBAC permission | Description |
|--------|------|----------------|-------------|
| `GET` | `/api/acl/groups/{id}/providers` | `acl:read` | List external providers |
| `POST` | `/api/acl/groups/{id}/providers` | `acl:update` | Add external provider |
| `PUT` | `/api/acl/providers/{id}` | `acl:update` | Update provider |
| `DELETE` | `/api/acl/providers/{id}` | `acl:delete` | Remove provider |

### ACLExternalProvider Fields

| Field | Type | Notes |
|-------|------|-------|
| `id` | int | |
| `acl_group_id` | int | |
| `provider_type` | string | `"authelia"`, `"authentik"`, or `"custom"` |
| `name` | string | |
| `verify_url` | string | URL Caddy calls to verify the request |
| `auth_redirect_url` | string\|null | Redirect URL when auth fails |
| `headers_to_copy` | string[] | Headers to copy from the verify response |

### 6.6 Waygates Auth (per group)

| Method | Path | RBAC permission | Description |
|--------|------|----------------|-------------|
| `GET` | `/api/acl/groups/{id}/waygates-auth` | `acl:read` | Get Waygates auth config |
| `PUT` | `/api/acl/groups/{id}/waygates-auth` | `acl:update` | Update Waygates auth config |

### ACLWaygatesAuth Fields

| Field | Type | Notes |
|-------|------|-------|
| `enabled` | bool | |
| `allowed_users` | string[] | Specific Waygates usernames |
| `allowed_roles` | string[] | Waygates roles |
| `allowed_email_patterns` | string[] | Glob patterns on email |
| `require_2fa` | bool | |
| `session_ttl` | int | Seconds; default 86400 |
| `allowed_emails` | string[] | Specific email addresses |
| `allowed_domains` | string[] | Email domains e.g. `"@company.com"` |
| `allowed_providers` | string[] | OAuth provider IDs e.g. `"google"` |

### 6.7 OAuth Provider Restrictions (per group)

| Method | Path | RBAC permission | Description |
|--------|------|----------------|-------------|
| `GET` | `/api/acl/groups/{id}/oauth-restrictions` | `acl:read` | Get all provider restrictions |
| `PUT` | `/api/acl/groups/{id}/oauth-restrictions/{provider}` | `acl:update` | Set restriction for a provider |
| `DELETE` | `/api/acl/groups/{id}/oauth-restrictions/{provider}` | `acl:delete` | Remove restriction for a provider |

### ACLOAuthProviderRestriction Fields

| Field | Type | Notes |
|-------|------|-------|
| `provider` | string | Provider ID e.g. `"google"` |
| `allowed_emails` | string[] | |
| `allowed_domains` | string[] | e.g. `"@company.com"` |
| `enabled` | bool | |

### 6.8 Branding

| Method | Path | Auth | RBAC | Description |
|--------|------|------|------|-------------|
| `GET` | `/api/acl/branding` | No | — | Get branding config (public) |
| `PUT` | `/api/acl/branding` | Yes | `acl:update` | Update branding config |

### ACLBranding Fields

```json
{
  "logo_url": "https://example.com/logo.png",
  "primary_color": "#3b82f6",
  "background_color": "#ffffff",
  "title": "Login Required",
  "subtitle": null,
  "footer_text": null,
  "custom_css": null
}
```

---

## 7. Audit Logs

| Method | Path | RBAC permission | Description |
|--------|------|----------------|-------------|
| `GET` | `/api/audit-logs` | `audit_logs:read` | List audit logs (paginated, filterable) |
| `GET` | `/api/audit-logs/stats` | `audit_logs:read` | Aggregate statistics |
| `GET` | `/api/audit-logs/export` | `audit_logs:read` | Export as CSV (up to 10 000 rows) |
| `GET` | `/api/audit-logs/config` | `settings:read` | Get audit event toggle config |
| `PUT` | `/api/audit-logs/config` | `settings:write` | Update audit event toggle config |
| `GET` | `/api/audit-logs/event-groups` | `settings:read` | List available event groups |
| `GET` | `/api/audit-logs/{id}` | `audit_logs:read` | Get single audit log entry |

### AuditLog Fields

| Field | Type | Notes |
|-------|------|-------|
| `id` | int | |
| `user_id` | int\|null | |
| `action` | string | e.g. `"proxy.create"`, `"auth.login"` |
| `resource_type` | string\|null | `"proxy"`, `"user"`, `"settings"`, `"system"`, `"acl"` |
| `resource_id` | int\|null | |
| `resource_name` | string\|null | |
| `details` | object\|null | Free-form context |
| `ip_address` | string\|null | |
| `user_agent` | string\|null | |
| `status` | string | `"success"` or `"failure"` |
| `error_message` | string\|null | |
| `created_at` | RFC3339 | |

### GET /api/audit-logs — Query Parameters

| Param | Format | Description |
|-------|--------|-------------|
| `page` | int ≥1 | Default 1 |
| `limit` | 1-100 | Default 20 |
| `search` | string | Text search |
| `action` | `[operator:]value` | Filter by action; operators: `eq`, `not`, `in`, `not_in` |
| `resource_type` | `[operator:]value` | Filter by resource type |
| `status` | `[eq\|not:]success\|failure` | Filter by status |
| `ip_address` | `[eq\|not\|contains\|starts_with\|ends_with:]value` | Filter by IP |
| `user_id` | int | Filter by user ID |
| `date_from` | ISO 8601 or `YYYY-MM-DD` | |
| `date_to` | ISO 8601 or `YYYY-MM-DD` | End of day inclusive |
| `sort` | field name | |
| `order` | `asc`\|`desc` | |

### GET /api/audit-logs/export

Returns a CSV file (`Content-Type: text/csv`) with columns: `ID, Timestamp, Action, Resource Type, Resource ID, Resource Name, User ID, Status, IP Address, User Agent, Error Message`. Max 10 000 rows; accepts same query filters as the list endpoint.

### Audit Config (PUT /api/audit-logs/config)

A flat object of boolean flags, one per event type. See `AuditConfig` in `backend/internal/models/audit_log.go` for all keys (e.g., `proxy_create`, `auth_login`, `acl_group_create`). All flags default to `true`.

---

## 8. Settings

| Method | Path | RBAC permission | Description |
|--------|------|----------------|-------------|
| `GET` | `/api/settings` | `settings:read` | Get all settings as `{key: value}` map |
| `GET` | `/api/settings/{key}` | `settings:read` | Get single setting by key |
| `PUT` | `/api/settings/{key}` | `settings:write` | Update single setting |
| `GET` | `/api/settings/404` | `settings:read` | Get 404 / not-found page config |
| `PUT` | `/api/settings/404` | `settings:write` | Update 404 config |
| `GET` | `/api/settings/metrics-publish` | `settings:read` | Get external metrics publish config |
| `PUT` | `/api/settings/metrics-publish` | `settings:write` | Update external metrics publish config |

Sensitive setting keys (e.g., `metrics.basic_auth_hash`) are never returned via the generic endpoints; they behave as `404 Not Found`.

### PUT /api/settings/{key}

```json
{ "value": "new-value" }
```

### GET /api/settings/404 Response

```json
{
  "mode": "default",
  "redirect_url": ""
}
```

`mode` is `"default"` (Caddy's built-in 404) or `"redirect"`. When `redirect`, `redirect_url` must be set.

### PUT /api/settings/404

```json
{
  "mode": "redirect",
  "redirect_url": "https://example.com/not-found"
}
```

### GET /api/settings/metrics-publish Response

```json
{
  "enabled": false,
  "host": "metrics.example.com",
  "path": "/metrics",
  "basic_auth_user": "prometheus",
  "has_basic_auth": true,
  "allowed_cidrs": ["10.0.0.0/8"]
}
```

The bcrypt hash is never returned; `has_basic_auth` indicates whether one is stored.

### PUT /api/settings/metrics-publish

```json
{
  "enabled": true,
  "host": "metrics.example.com",
  "path": "/metrics",
  "basic_auth_user": "prometheus",
  "basic_auth_password": "plaintext-password",
  "allowed_cidrs": ["10.0.0.0/8"]
}
```

`basic_auth_password` is optional if a password was previously stored. `host` and `basic_auth_user` (and a stored/supplied password) are required when `enabled: true`.

---

## 9. Sync

| Method | Path | RBAC permission | Description |
|--------|------|----------------|-------------|
| `GET` | `/api/sync/status` | `sync:read` | Get current sync status |
| `POST` | `/api/sync/trigger` | `sync:trigger` | Trigger a full sync immediately |

Sync runs automatically every 60 seconds in the background. A manual trigger via `POST /api/sync/trigger` blocks until the sync completes (or fails) and returns the resulting status.

### Sync Status Response

```json
{
  "last_sync_at": "2024-01-01T12:00:00Z",
  "last_sync_status": "success",
  "last_error": null,
  "syncing": false
}
```

---

## 10. Caddy Logs

| Method | Path | RBAC permission | Description |
|--------|------|----------------|-------------|
| `GET` | `/api/caddy-logs` | `caddy_logs:read` | Snapshot of recent log lines |
| `GET` | `/api/caddy-logs/stream` | `caddy_logs:read` | SSE stream of live log lines |

### GET /api/caddy-logs — Query Parameters

| Param | Description |
|-------|-------------|
| `source` | `"runtime"` (default) or `"access"` |
| `limit` | Lines to return; default 200, max 1000 |

`data` is an array of raw log line strings.

### GET /api/caddy-logs/stream

Server-Sent Events stream. Each event has the form `data: <raw log line>\n\n`. The stream backfills the last 500 lines from the file, then follows in real time. Closes when the client disconnects.

Query param: `source` (`"runtime"` or `"access"`).

Response headers: `Content-Type: text/event-stream`, `Cache-Control: no-cache`, `Connection: keep-alive`.

---

## 11. Caddy Config Preview

These endpoints return the generated Caddy JSON config **without** writing it to disk or reloading Caddy. Useful for debugging.

| Method | Path | RBAC permission | Description |
|--------|------|----------------|-------------|
| `GET` | `/api/caddy-config` | `caddy_config:read` | Full generated Caddy JSON config |
| `GET` | `/api/proxies/{id}/config-preview` | `caddy_config:read` | Per-proxy Caddy JSON snippet |

Both endpoints return the raw Caddy JSON config object inside `data`.

---

## 12. Traffic Metrics

| Method | Path | RBAC permission | Description |
|--------|------|----------------|-------------|
| `GET` | `/api/metrics/traffic` | `metrics:read` | Traffic time-series data |

### GET /api/metrics/traffic — Query Parameters

| Param | Values | Description |
|-------|--------|-------------|
| `range` | `1h` (default), `24h`, `7d` | Time range for the time-series |

Metrics are scraped from Caddy's Prometheus endpoint every 30 seconds by the background `MetricsScraperService`. `data` is a time-series array suitable for charting.

---

## Common HTTP Status Codes

| Status | Meaning |
|--------|---------|
| 200 | Success |
| 201 | Resource created |
| 400 | Invalid request (body, params, validation) |
| 401 | Missing or invalid token |
| 403 | Valid token but insufficient permissions |
| 404 | Resource not found (or sensitive key hidden) |
| 409 | Conflict (e.g. duplicate hostname) |
| 429 | Rate limited (auth endpoints: 10 req/IP/min) |
| 500 | Internal server error |
| 502 | Caddy error (upstream unreachable or config rejected) |

---

## Self-Check Against routes.go

All routes registered in `backend/internal/api/routes/routes.go` are documented above:

- Public group: `GET /api/health`, `GET /api/status`, `POST /api/auth/register`, `POST /api/auth/login`, `POST /api/auth/refresh`, `GET /api/auth/acl/verify`, `POST /api/auth/acl/login`, `POST /api/auth/acl/logout`, `GET /api/auth/acl/session`, `GET /api/auth/oauth/providers`, `GET /auth/oauth/{provider}`, `GET /auth/oauth/{provider}/callback`, `GET /api/acl/branding`, `GET /api/acl/options` — **14 endpoints**
- Protected group: `POST /api/auth/logout`, `GET /api/auth/me`, `POST /api/auth/change-password` + all proxy, settings, sync, audit-log, ACL, caddy-logs, caddy-config, metrics, and L4-proxy routes — **74 endpoints**

**Total documented: 88 endpoints** (counting each registered `Method + Path` combination as one endpoint).
