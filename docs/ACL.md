# Access Control Lists (ACL)

Waygates provides a flexible Access Control List (ACL) system that allows you to protect your proxied services with various authentication methods. ACLs are implemented through reusable **ACL Groups** that can be assigned to one or more proxies.

## Table of Contents

- [Overview](#overview)
- [How ACLs Work](#how-acls-work)
- [ACL Groups](#acl-groups)
- [Authentication Options](#authentication-options)
  - [IP Rules](#ip-rules)
  - [Basic Authentication](#basic-authentication)
  - [Waygates Authentication](#waygates-authentication)
- [Assigning ACLs to Proxies](#assigning-acls-to-proxies)
- [Login Page Branding](#login-page-branding)
- [API Reference](#api-reference)
- [Configuration Examples](#configuration-examples)

---

## Overview

The ACL system provides:

- **IP-based access control**: Allow, deny, or bypass authentication based on client IP addresses
- **Basic Authentication**: Username/password authentication via HTTP Basic Auth
- **Waygates Authentication**: Session-based authentication with Waygates user accounts
- **Reusable Groups**: Define once, apply to multiple proxies
- **Union Logic**: Multiple ACL groups on a proxy combine their rules (OR semantics)
- **Static Asset Bypass**: Common static files (images, CSS, JS) automatically bypass authentication

> **Note**: External authentication providers (Authelia, Authentik) will be available in a future release.

---

## How ACLs Work

When a request is made to a protected proxy:

1. **Caddy** receives the request and checks if the proxy has ACL rules
2. For protected routes, Caddy uses `forward_auth` to send the request to Waygates for verification
3. **Waygates** evaluates the ACL rules in order:
   - **IP Deny Rules**: If the client IP matches any deny rule, access is blocked (403)
   - **IP Bypass Rules**: If the client IP matches a bypass rule, access is granted without further auth
   - **IP Allow Rules**: If the client IP matches an allow rule, access is granted
   - **Authentication**: If no IP rules match, authentication is required
4. If authentication is required, the user is redirected to the login page
5. Upon successful authentication, a session cookie is set and the user can access the protected resource

### Request Flow Diagram

```
Client Request
      │
      ▼
┌─────────────┐
│    Caddy    │
└──────┬──────┘
       │
       ▼
┌─────────────────────┐
│  IP Deny Check      │──▶ 403 Forbidden
└──────┬──────────────┘
       │ (not denied)
       ▼
┌─────────────────────┐
│  IP Bypass Check    │──▶ Access Granted
└──────┬──────────────┘
       │ (no bypass)
       ▼
┌─────────────────────┐
│  IP Allow Check     │──▶ Access Granted
└──────┬──────────────┘
       │ (no allow match)
       ▼
┌─────────────────────┐
│  Auth Required      │──▶ Login Page
└──────┬──────────────┘
       │ (valid session)
       ▼
   Access Granted
```

---

## ACL Groups

ACL Groups are reusable collections of authentication rules. Each group has:

### Properties

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Unique name for the group (required) |
| `description` | string | Optional description |
| `combination_mode` | string | How authentication methods combine (see below) |

### Combination Modes

| Mode | Description |
|------|-------------|
| `any` | **Default**. Any single authentication method grants access (OR logic) |
| `all` | All configured authentication methods must pass (AND logic) |
| `ip_bypass` | IP bypass rules can skip all other authentication |

#### Example: `any` Mode
If a group has both Basic Auth and Waygates Auth configured, a user can authenticate with **either** method.

#### Example: `all` Mode
If a group has both IP Allow rules and Basic Auth configured, a user must be from an allowed IP **and** provide valid credentials.

#### Example: `ip_bypass` Mode
If a group has IP bypass rules configured, requests from those IPs skip all authentication. Other requests still need to authenticate.

---

## Authentication Options

### IP Rules

IP rules control access based on the client's IP address. Rules are defined using CIDR notation.

#### Rule Types

| Type | Description |
|------|-------------|
| `deny` | Block access from matching IPs (highest priority, checked first) |
| `bypass` | Allow access without any authentication |
| `allow` | Allow access (may still require auth depending on combination mode) |

#### Fields

| Field | Type | Description |
|-------|------|-------------|
| `rule_type` | string | One of: `allow`, `deny`, `bypass` |
| `cidr` | string | IP address or CIDR range (e.g., `192.168.1.0/24`, `10.0.0.1/32`) |
| `description` | string | Optional description for the rule |
| `priority` | integer | Evaluation order (lower = higher priority) |

#### Examples

```
# Block a specific IP
Type: deny
CIDR: 10.0.0.100/32

# Allow internal network without auth
Type: bypass
CIDR: 192.168.1.0/24

# Allow office IP range
Type: allow
CIDR: 203.0.113.0/24
```

### Basic Authentication

HTTP Basic Authentication using username and password. Passwords are securely hashed using bcrypt.

#### Fields

| Field | Type | Description |
|-------|------|-------------|
| `username` | string | Username (unique within the group) |
| `password` | string | Password (stored as bcrypt hash) |

#### How It Works

1. When a user accesses a protected resource, the browser prompts for credentials
2. Credentials are sent via the `Authorization` header
3. Waygates validates the credentials against the stored hash
4. If valid, access is granted

#### Security Notes

- Passwords are hashed with bcrypt (cost factor 14) for Caddy compatibility
- Always use HTTPS to protect credentials in transit
- Basic Auth credentials are sent with every request

### Waygates Authentication

Session-based authentication using Waygates user accounts. This provides a more user-friendly login experience with a dedicated login page.

#### Fields

| Field | Type | Description |
|-------|------|-------------|
| `enabled` | boolean | Enable/disable Waygates auth |
| `allowed_users` | array | Specific user IDs that can access (empty = all users) |
| `allowed_roles` | array | Roles that can access (e.g., `["admin", "operator"]`) |
| `allowed_email_patterns` | array | Email patterns (e.g., `["*@company.com"]`) |
| `session_ttl` | integer | Session duration in seconds (default: 86400 = 24 hours) |

> **Note**: Two-factor authentication (2FA) support will be available in a future release.

#### How It Works

1. User accesses a protected resource
2. User is redirected to the Waygates login page
3. User authenticates with their Waygates credentials
4. A session cookie (`waygates_session`) is set
5. User can access the protected resource for the session duration

#### User Restrictions

You can restrict access to specific users by configuring:

- **Allowed Users**: List of specific user IDs
- **Allowed Roles**: List of roles (users with any of these roles can access)
- **Allowed Email Patterns**: Wildcard patterns like `*@company.com`

If none of these are configured, all authenticated Waygates users can access the resource.

---

## Assigning ACLs to Proxies

ACL groups are assigned to proxies through **Proxy ACL Assignments**.

### Assignment Properties

| Field | Type | Description |
|-------|------|-------------|
| `proxy_id` | integer | ID of the proxy |
| `acl_group_id` | integer | ID of the ACL group |
| `path_pattern` | string | URL path pattern (default: `/*`) |
| `priority` | integer | Evaluation order for multiple assignments |
| `enabled` | boolean | Enable/disable this assignment |

### Path Patterns

You can protect specific paths within a proxy:

| Pattern | Matches |
|---------|---------|
| `/*` | All paths (default) |
| `/admin/*` | All paths under `/admin/` |
| `/api/*` | All paths under `/api/` |
| `/secret` | Exactly `/secret` |

### Multiple ACL Groups (Union Logic)

When multiple ACL groups are assigned to the same proxy:

1. **Deny rules from ANY group block access** - If any group denies an IP, access is blocked
2. **Bypass rules from ANY group allow access** - If any group allows bypass for an IP, no auth is needed
3. **Any valid credential from ANY group grants access** - Users can authenticate with credentials from any assigned group

This allows flexible configurations like:
- One group for internal IPs (bypass)
- One group for contractors (basic auth)
- One group for employees (Waygates auth)

### Static Asset Bypass

The following file types automatically bypass authentication to prevent issues with page loading:

**File Extensions**: `.ico`, `.png`, `.jpg`, `.jpeg`, `.gif`, `.svg`, `.webp`, `.css`, `.js`, `.woff`, `.woff2`, `.ttf`, `.eot`, `.otf`, `.map`

**Paths**: `/favicon.ico`, `/robots.txt`, `/sitemap.xml`, `/apple-touch-icon.png`, `/manifest.json`

---

## Login Page Branding

Customize the appearance of the ACL login page with branding options.

### Branding Options

| Field | Type | Description |
|-------|------|-------------|
| `logo_url` | string | URL to your logo image |
| `primary_color` | string | Primary color for buttons (hex, e.g., `#3b82f6`) |
| `background_color` | string | Page background color (hex, e.g., `#ffffff`) |
| `title` | string | Login page title (default: "Login Required") |
| `subtitle` | string | Optional subtitle text |
| `footer_text` | string | Optional footer text |
| `custom_css` | string | Custom CSS for advanced styling |

---

## API Reference

### ACL Group Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/acl/groups` | List all ACL groups (paginated) |
| `POST` | `/api/acl/groups` | Create a new ACL group |
| `GET` | `/api/acl/groups/:id` | Get ACL group details |
| `PUT` | `/api/acl/groups/:id` | Update an ACL group |
| `DELETE` | `/api/acl/groups/:id` | Delete an ACL group |
| `GET` | `/api/acl/groups/:id/usage` | Get proxies using this group |

### IP Rules

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/acl/groups/:id/ip-rules` | List IP rules for a group |
| `POST` | `/api/acl/groups/:id/ip-rules` | Add an IP rule |
| `PUT` | `/api/acl/ip-rules/:id` | Update an IP rule |
| `DELETE` | `/api/acl/ip-rules/:id` | Delete an IP rule |

### Basic Auth Users

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/acl/groups/:id/basic-auth` | List basic auth users |
| `POST` | `/api/acl/groups/:id/basic-auth` | Add a basic auth user |
| `PUT` | `/api/acl/basic-auth/:id` | Update password |
| `DELETE` | `/api/acl/basic-auth/:id` | Delete a user |

### Waygates Auth

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/acl/groups/:id/waygates-auth` | Get Waygates auth config |
| `PUT` | `/api/acl/groups/:id/waygates-auth` | Update Waygates auth config |

### Proxy ACL Assignments

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/proxies/:id/acl` | List ACL assignments for a proxy |
| `POST` | `/api/proxies/:id/acl` | Assign an ACL group to a proxy |
| `PUT` | `/api/proxies/:id/acl/:assignmentId` | Update an assignment |
| `DELETE` | `/api/proxies/:id/acl/:groupId` | Remove an ACL group from a proxy |

### Branding

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/acl/branding` | Get branding configuration |
| `PUT` | `/api/acl/branding` | Update branding configuration |

### Sessions

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/acl/sessions` | List active sessions |
| `DELETE` | `/api/acl/sessions/:id` | Revoke a session |

---

## Configuration Examples

### Example 1: Internal-Only Access

Allow access only from your internal network without requiring authentication.

```json
{
  "name": "Internal Network Only",
  "description": "Allow access from internal IPs only",
  "combination_mode": "ip_bypass",
  "ip_rules": [
    {
      "rule_type": "bypass",
      "cidr": "192.168.0.0/16",
      "description": "Internal network"
    },
    {
      "rule_type": "bypass",
      "cidr": "10.0.0.0/8",
      "description": "VPN network"
    }
  ]
}
```

### Example 2: Basic Auth for External Access

Protect a service with basic authentication for external users, while allowing internal users to bypass.

```json
{
  "name": "External Basic Auth",
  "description": "Basic auth for external, bypass for internal",
  "combination_mode": "ip_bypass",
  "ip_rules": [
    {
      "rule_type": "bypass",
      "cidr": "192.168.1.0/24",
      "description": "Office network"
    }
  ],
  "basic_auth_users": [
    {
      "username": "contractor1",
      "password": "secure-password-here"
    },
    {
      "username": "partner",
      "password": "another-secure-password"
    }
  ]
}
```

### Example 3: Waygates Auth with Role Restriction

Allow only admin users to access a sensitive service.

```json
{
  "name": "Admin Only",
  "description": "Restrict to admin users",
  "combination_mode": "any",
  "waygates_auth": {
    "enabled": true,
    "allowed_roles": ["admin"],
    "session_ttl": 3600
  }
}
```

### Example 4: Block Specific IPs

Block known malicious IPs while allowing everyone else.

```json
{
  "name": "IP Blocklist",
  "description": "Block specific IPs",
  "combination_mode": "any",
  "ip_rules": [
    {
      "rule_type": "deny",
      "cidr": "203.0.113.0/24",
      "description": "Known bad actors"
    },
    {
      "rule_type": "deny",
      "cidr": "198.51.100.50/32",
      "description": "Blocked IP"
    }
  ]
}
```

### Example 5: Company Email Domain Only

Allow only users with a specific email domain to access.

```json
{
  "name": "Company Employees",
  "description": "Only @company.com emails",
  "combination_mode": "any",
  "waygates_auth": {
    "enabled": true,
    "allowed_email_patterns": ["*@company.com"],
    "session_ttl": 86400
  }
}
```

### Example 6: Multi-Group Assignment

Assign multiple ACL groups to a single proxy for layered access control.

**Group 1: Internal Bypass**
```json
{
  "name": "Internal Bypass",
  "ip_rules": [
    { "rule_type": "bypass", "cidr": "192.168.0.0/16" }
  ]
}
```

**Group 2: Basic Auth Contractors**
```json
{
  "name": "Contractors",
  "basic_auth_users": [
    { "username": "contractor1", "password": "..." }
  ]
}
```

**Group 3: Employee Login**
```json
{
  "name": "Employees",
  "waygates_auth": {
    "enabled": true,
    "allowed_email_patterns": ["*@company.com"]
  }
}
```

**Result**: Internal users bypass auth, contractors use basic auth, employees use Waygates login.

---

## Troubleshooting

### Common Issues

**Issue**: Users are prompted for login on every request

**Solution**: Ensure cookies are enabled and the domain is correctly configured. Check that the session TTL is set appropriately.

**Issue**: Static assets (CSS, JS) are not loading on protected pages

**Solution**: Static assets should automatically bypass authentication. If issues persist, check that your assets use standard file extensions.

**Issue**: IP rules not working as expected

**Solution**: Verify CIDR notation is correct. Use `/32` for single IPs. Remember that deny rules are evaluated first across all groups.

**Issue**: Basic auth not working

**Solution**: Ensure the username/password are correct. Check that the ACL group is properly assigned to the proxy and enabled.

---

## Security Best Practices

1. **Use HTTPS**: Always enable TLS for proxies with authentication
2. **Strong Passwords**: Use strong, unique passwords for basic auth users
3. **Limit IP Bypass**: Only bypass authentication for trusted networks
4. **Session TTL**: Set appropriate session timeouts (shorter for sensitive services)
5. **Regular Audits**: Review ACL configurations and active sessions periodically
6. **Deny First**: Use deny rules to block known bad actors before allowing access
7. **Principle of Least Privilege**: Grant the minimum access required
