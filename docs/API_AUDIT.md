# Audit Logs API Specification

**Version:** 1.0
**Last Updated:** 2024-11-08

## Overview

The Audit Logs API provides endpoints to view and search system activity logs. All actions performed by users (create, update, delete operations on proxies, users, settings, etc.) are automatically logged for security and compliance purposes.

**Key Features:**
- Automatic logging of all state-changing operations
- User attribution (who did what)
- IP address and user agent tracking
- Advanced filtering and search
- Export capabilities

---

## Base URL

```
/api/audit
```

## Authentication

All endpoints require JWT token with appropriate permissions:

```
Authorization: Bearer <jwt_token>
```

**Permission Requirements:**
- **Admin role:** Full access to all audit logs
- **User role:** Can only view their own actions

---

## Data Model

### Audit Log Object

```json
{
  "id": 1,
  "user": {
    "id": 1,
    "name": "Admin User",
    "email": "admin@example.com"
  },
  "action": "create_proxy",
  "resource_type": "proxy",
  "resource_id": 5,
  "resource_name": "My Application",
  "details": {
    "proxy": {
      "hostname": "app.example.com",
      "type": "reverse_proxy",
      "upstreams": ["192.168.100.5:8080"]
    }
  },
  "ip_address": "192.168.1.100",
  "user_agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)...",
  "status": "success",
  "error_message": null,
  "created_at": "2024-11-08T10:30:00Z"
}
```

---

## Database Schema

### Audit Logs Table

```sql
CREATE TABLE audit_logs (
    id SERIAL PRIMARY KEY,

    -- User information
    user_id INTEGER,  -- NULL for system actions or failed login attempts

    -- Action details
    action VARCHAR(100) NOT NULL,  -- Action type (e.g., 'create_proxy', 'login_failed')
    resource_type VARCHAR(50),     -- 'proxy', 'user', 'settings', 'auth', 'system', 'audit'
    resource_id INTEGER,            -- ID of affected resource (NULL for list/system operations)
    resource_name VARCHAR(255),     -- Display name of affected resource

    -- Additional context
    details TEXT,  -- JSON with action-specific details

    -- Request metadata
    ip_address VARCHAR(45),  -- IPv4 or IPv6
    user_agent TEXT,

    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'success',  -- 'success', 'failed'
    error_message TEXT,  -- Error message if status is 'failed'

    -- Timestamp
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);

-- Indexes for better query performance
CREATE INDEX idx_audit_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_action ON audit_logs(action);
CREATE INDEX idx_audit_resource_type ON audit_logs(resource_type);
CREATE INDEX idx_audit_resource_id ON audit_logs(resource_id);
CREATE INDEX idx_audit_status ON audit_logs(status);
CREATE INDEX idx_audit_created_at ON audit_logs(created_at);
CREATE INDEX idx_audit_ip_address ON audit_logs(ip_address);

-- Composite indexes for common queries
CREATE INDEX idx_audit_user_created ON audit_logs(user_id, created_at);
CREATE INDEX idx_audit_resource_type_id ON audit_logs(resource_type, resource_id);
CREATE INDEX idx_audit_action_created ON audit_logs(action, created_at);
```

### Database Field Descriptions

| Field | Type | Nullable | Description |
|-------|------|----------|-------------|
| `id` | INTEGER | No | Primary key, auto-increment |
| `user_id` | INTEGER | Yes | User who performed the action (NULL for system actions or failed logins) |
| `action` | VARCHAR(100) | No | Action type (e.g., `create_proxy`, `login_failed`) |
| `resource_type` | VARCHAR(50) | Yes | Resource type: `proxy`, `user`, `settings`, `auth`, `system`, `audit` |
| `resource_id` | INTEGER | Yes | ID of affected resource (NULL for list/system operations) |
| `resource_name` | VARCHAR(255) | Yes | Display name of affected resource |
| `details` | TEXT (JSON) | Yes | JSON with action-specific details (see Details section) |
| `ip_address` | VARCHAR(45) | Yes | Client IP address (IPv4 or IPv6) |
| `user_agent` | TEXT | Yes | Client user agent string |
| `status` | VARCHAR(20) | No | Action status: `success` or `failed` (default: `success`) |
| `error_message` | TEXT | Yes | Error message if status is `failed` |
| `created_at` | TIMESTAMP | No | Creation timestamp (auto-set) |

### API Response Field Descriptions

When retrieving audit logs via API, the `user_id` is expanded to a full user object:

| Field | Type | Description |
|-------|------|-------------|
| `id` | integer | Unique audit log ID |
| `user` | object | User who performed the action (expanded from user_id) |
| `user.id` | integer | User ID |
| `user.name` | string | User's display name |
| `user.email` | string | User's email |
| `action` | string | Action performed (see Actions table) |
| `resource_type` | string | Type of resource: `proxy`, `user`, `settings`, `auth` |
| `resource_id` | integer | ID of affected resource (null for list operations) |
| `resource_name` | string | Display name of affected resource |
| `details` | object | JSON with additional context about the action |
| `ip_address` | string | Client IP address (IPv4 or IPv6) |
| `user_agent` | string | Client user agent string |
| `status` | string | Action status: `success`, `failed` |
| `error_message` | string | Error message if status is `failed` |
| `created_at` | string | ISO 8601 timestamp (UTC) |

---

## Actions

### Proxy Actions

| Action | Description | Resource Type |
|--------|-------------|---------------|
| `create_proxy` | Created a new proxy | `proxy` |
| `update_proxy` | Updated proxy configuration | `proxy` |
| `delete_proxy` | Deleted a proxy | `proxy` |
| `enable_proxy` | Enabled a proxy | `proxy` |
| `disable_proxy` | Disabled a proxy | `proxy` |

### User Actions

| Action | Description | Resource Type |
|--------|-------------|---------------|
| `create_user` | Created a new user | `user` |
| `update_user` | Updated user information | `user` |
| `delete_user` | Deleted a user | `user` |
| `update_user_password` | Changed user password | `user` |
| `update_user_role` | Changed user role | `user` |

### Authentication Actions

| Action | Description | Resource Type |
|--------|-------------|---------------|
| `login_success` | Successful login | `auth` |
| `login_failed` | Failed login attempt | `auth` |
| `password_reset_request` | Requested password reset | `auth` |
| `password_reset_complete` | Completed password reset | `auth` |

### System Actions

| Action | Description | Resource Type |
|--------|-------------|---------------|
| `update_settings` | Updated system settings | `settings` |
| `export_audit_logs` | Exported audit logs | `audit` |
| `caddy_reload` | Reloaded Caddy configuration | `system` |

---

## Endpoints

### 1. List Audit Logs

**GET** `/api/audit`

Retrieve audit logs with filtering, search, and pagination.

#### Query Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `page` | integer | No | 1 | Page number |
| `limit` | integer | No | 50 | Items per page (max: 500) |
| `user_id` | integer | No | - | Filter by user ID |
| `action` | string | No | - | Filter by action (exact match) |
| `resource_type` | string | No | - | Filter by resource type |
| `resource_id` | integer | No | - | Filter by resource ID |
| `status` | string | No | - | Filter by status: `success`, `failed` |
| `ip_address` | string | No | - | Filter by IP address |
| `date_from` | string | No | - | Start date (ISO 8601) |
| `date_to` | string | No | - | End date (ISO 8601) |
| `search` | string | No | - | Search in resource_name, details, error_message |
| `sort` | string | No | `created_at` | Sort field: `created_at`, `action`, `user_id` |
| `order` | string | No | `desc` | Sort order: `asc`, `desc` |

#### Request Example

```http
GET /api/audit?page=1&limit=50&action=create_proxy&status=success&date_from=2024-11-01T00:00:00Z&date_to=2024-11-08T23:59:59Z
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

#### Response (200 OK)

```json
{
  "success": true,
  "data": {
    "logs": [
      {
        "id": 145,
        "user": {
          "id": 1,
          "name": "Admin User",
          "email": "admin@example.com"
        },
        "action": "create_proxy",
        "resource_type": "proxy",
        "resource_id": 5,
        "resource_name": "Production API",
        "details": {
          "proxy": {
            "hostname": "api.example.com",
            "type": "reverse_proxy",
            "upstreams": [
              {"host": "192.168.100.10", "port": 3000},
              {"host": "192.168.100.11", "port": 3000}
            ],
            "load_balancing": {
              "strategy": "least_conn",
              "health_checks": {"enabled": true}
            }
          }
        },
        "ip_address": "192.168.1.100",
        "user_agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
        "status": "success",
        "error_message": null,
        "created_at": "2024-11-08T10:30:15Z"
      },
      {
        "id": 144,
        "user": {
          "id": 2,
          "name": "John Doe",
          "email": "john@example.com"
        },
        "action": "update_proxy",
        "resource_type": "proxy",
        "resource_id": 3,
        "resource_name": "Dashboard",
        "details": {
          "changes": {
            "upstream_port": {
              "old": 8080,
              "new": 8081
            }
          }
        },
        "ip_address": "192.168.1.105",
        "user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
        "status": "success",
        "error_message": null,
        "created_at": "2024-11-08T09:15:30Z"
      }
    ],
    "pagination": {
      "current_page": 1,
      "total_pages": 3,
      "total_items": 145,
      "items_per_page": 50,
      "has_next": true,
      "has_prev": false
    },
    "summary": {
      "total_actions": 145,
      "success_count": 142,
      "failed_count": 3,
      "unique_users": 5,
      "date_range": {
        "from": "2024-11-01T00:00:00Z",
        "to": "2024-11-08T23:59:59Z"
      }
    }
  }
}
```

#### Error Responses

**401 Unauthorized**
```json
{
  "success": false,
  "error": {
    "code": "UNAUTHORIZED",
    "message": "Invalid or expired token"
  }
}
```

**403 Forbidden**
```json
{
  "success": false,
  "error": {
    "code": "FORBIDDEN",
    "message": "You don't have permission to view audit logs"
  }
}
```

**400 Bad Request**
```json
{
  "success": false,
  "error": {
    "code": "INVALID_QUERY_PARAMS",
    "message": "Invalid query parameters",
    "details": {
      "date_from": "must be valid ISO 8601 date",
      "limit": "must be between 1 and 500"
    }
  }
}
```

---

### 2. Get Audit Log by ID

**GET** `/api/audit/:id`

Retrieve a specific audit log entry.

#### Path Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `id` | integer | Yes | Audit log ID |

#### Request Example

```http
GET /api/audit/145
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

#### Response (200 OK)

```json
{
  "success": true,
  "data": {
    "id": 145,
    "user": {
      "id": 1,
      "name": "Admin User",
      "email": "admin@example.com"
    },
    "action": "create_proxy",
    "resource_type": "proxy",
    "resource_id": 5,
    "resource_name": "Production API",
    "details": {
      "proxy": {
        "hostname": "api.example.com",
        "type": "reverse_proxy",
        "upstreams": [
          {"host": "192.168.100.10", "port": 3000},
          {"host": "192.168.100.11", "port": 3000}
        ],
        "load_balancing": {
          "strategy": "least_conn",
          "health_checks": {"enabled": true}
        }
      },
      "request_body": {
        "name": "Production API",
        "hostname": "api.example.com"
      }
    },
    "ip_address": "192.168.1.100",
    "user_agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
    "status": "success",
    "error_message": null,
    "created_at": "2024-11-08T10:30:15Z"
  }
}
```

#### Error Responses

**404 Not Found**
```json
{
  "success": false,
  "error": {
    "code": "AUDIT_LOG_NOT_FOUND",
    "message": "Audit log with ID 145 not found"
  }
}
```

---

### 3. Get Audit Summary

**GET** `/api/audit/summary`

Get aggregated statistics about audit logs.

#### Query Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `date_from` | string | No | 30 days ago | Start date (ISO 8601) |
| `date_to` | string | No | now | End date (ISO 8601) |
| `user_id` | integer | No | - | Filter by user ID |
| `resource_type` | string | No | - | Filter by resource type |

#### Request Example

```http
GET /api/audit/summary?date_from=2024-11-01T00:00:00Z&date_to=2024-11-08T23:59:59Z
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

#### Response (200 OK)

```json
{
  "success": true,
  "data": {
    "period": {
      "from": "2024-11-01T00:00:00Z",
      "to": "2024-11-08T23:59:59Z",
      "days": 8
    },
    "totals": {
      "total_actions": 523,
      "success_count": 518,
      "failed_count": 5,
      "unique_users": 8,
      "unique_ips": 12
    },
    "by_action": [
      {
        "action": "create_proxy",
        "count": 45,
        "percentage": 8.6
      },
      {
        "action": "update_proxy",
        "count": 120,
        "percentage": 22.9
      },
      {
        "action": "login_success",
        "count": 156,
        "percentage": 29.8
      },
      {
        "action": "view_proxy",
        "count": 89,
        "percentage": 17.0
      }
    ],
    "by_user": [
      {
        "user": {
          "id": 1,
          "name": "Admin User",
          "email": "admin@example.com"
        },
        "count": 234,
        "percentage": 44.7
      },
      {
        "user": {
          "id": 2,
          "name": "John Doe",
          "email": "john@example.com"
        },
        "count": 156,
        "percentage": 29.8
      }
    ],
    "by_resource_type": [
      {
        "resource_type": "proxy",
        "count": 289,
        "percentage": 55.3
      },
      {
        "resource_type": "auth",
        "count": 178,
        "percentage": 34.0
      },
      {
        "resource_type": "user",
        "count": 56,
        "percentage": 10.7
      }
    ],
    "timeline": [
      {
        "date": "2024-11-01",
        "count": 67,
        "success": 65,
        "failed": 2
      },
      {
        "date": "2024-11-02",
        "count": 73,
        "success": 73,
        "failed": 0
      },
      {
        "date": "2024-11-03",
        "count": 81,
        "success": 80,
        "failed": 1
      }
    ],
    "top_failures": [
      {
        "action": "create_proxy",
        "error_message": "Hostname already exists",
        "count": 3
      },
      {
        "action": "login_failed",
        "error_message": "Invalid credentials",
        "count": 2
      }
    ]
  }
}
```

---

### 4. Export Audit Logs

**POST** `/api/audit/export`

Export audit logs to CSV or JSON format. This creates an audit log entry itself.

#### Request Body

```json
{
  "format": "csv",
  "filters": {
    "date_from": "2024-11-01T00:00:00Z",
    "date_to": "2024-11-08T23:59:59Z",
    "user_id": null,
    "action": null,
    "status": null
  }
}
```

#### Field Validation

| Field | Type | Required | Validation |
|-------|------|----------|------------|
| `format` | string | Yes | `csv` or `json` |
| `filters` | object | No | Same as List filters |

#### Request Example

```http
POST /api/audit/export
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
Content-Type: application/json

{
  "format": "csv",
  "filters": {
    "date_from": "2024-11-01T00:00:00Z",
    "date_to": "2024-11-08T23:59:59Z",
    "action": "create_proxy"
  }
}
```

#### Response (200 OK)

```json
{
  "success": true,
  "message": "Export completed successfully",
  "data": {
    "export_id": "audit_export_20241108_103045",
    "format": "csv",
    "record_count": 45,
    "file_size": 15234,
    "download_url": "/api/audit/export/audit_export_20241108_103045/download",
    "expires_at": "2024-11-09T10:30:45Z"
  }
}
```

**CSV Format Example:**
```csv
id,timestamp,user_email,user_name,action,resource_type,resource_id,resource_name,status,ip_address
145,2024-11-08T10:30:15Z,admin@example.com,Admin User,create_proxy,proxy,5,Production API,success,192.168.1.100
144,2024-11-08T09:15:30Z,john@example.com,John Doe,update_proxy,proxy,3,Dashboard,success,192.168.1.105
```

**JSON Format Example:**
```json
[
  {
    "id": 145,
    "timestamp": "2024-11-08T10:30:15Z",
    "user": {
      "id": 1,
      "email": "admin@example.com",
      "name": "Admin User"
    },
    "action": "create_proxy",
    "resource_type": "proxy",
    "resource_id": 5,
    "resource_name": "Production API",
    "status": "success",
    "ip_address": "192.168.1.100"
  }
]
```

---

### 5. Download Export

**GET** `/api/audit/export/:export_id/download`

Download a previously created export file.

#### Path Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `export_id` | string | Yes | Export ID from export response |

#### Request Example

```http
GET /api/audit/export/audit_export_20241108_103045/download
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

#### Response (200 OK)

Returns file with appropriate Content-Type:
- CSV: `text/csv; charset=utf-8`
- JSON: `application/json`

Headers:
```
Content-Type: text/csv; charset=utf-8
Content-Disposition: attachment; filename="audit_logs_20241108_103045.csv"
Content-Length: 15234
```

#### Error Responses

**404 Not Found**
```json
{
  "success": false,
  "error": {
    "code": "EXPORT_NOT_FOUND",
    "message": "Export file not found or has expired"
  }
}
```

---

### 6. Get Audit Logs for Resource

**GET** `/api/audit/resource/:resource_type/:resource_id`

Get all audit logs related to a specific resource.

#### Path Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `resource_type` | string | Yes | `proxy`, `user`, etc. |
| `resource_id` | integer | Yes | Resource ID |

#### Request Example

```http
GET /api/audit/resource/proxy/5
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

#### Response (200 OK)

```json
{
  "success": true,
  "data": {
    "resource": {
      "type": "proxy",
      "id": 5,
      "name": "Production API",
      "current_status": "active"
    },
    "logs": [
      {
        "id": 145,
        "action": "create_proxy",
        "user": {
          "id": 1,
          "name": "Admin User"
        },
        "created_at": "2024-11-08T10:30:15Z",
        "status": "success"
      },
      {
        "id": 167,
        "action": "enable_proxy",
        "user": {
          "id": 1,
          "name": "Admin User"
        },
        "created_at": "2024-11-08T11:45:00Z",
        "status": "success"
      }
    ],
    "summary": {
      "total_actions": 2,
      "created_by": {
        "id": 1,
        "name": "Admin User",
        "email": "admin@example.com"
      },
      "created_at": "2024-11-08T10:30:15Z",
      "last_modified_at": "2024-11-08T11:45:00Z"
    }
  }
}
```

---

## Details Object Structures by Action

The `details` field contains action-specific information in JSON format. Below are ALL possible structures organized by action type.

---

### Proxy Actions

#### `create_proxy`

```json
{
  "proxy": {
    "type": "reverse_proxy",
    "hostname": "api.example.com",
    "upstreams": [
      {
        "host": "192.168.100.10",
        "port": 3000,
        "scheme": "http"
      }
    ],
    "ssl_enabled": true,
    "ssl_forced": true,
    "websocket_support": false,
    "block_exploits": true,
    "custom_headers": {
      "X-API-Version": "v1"
    }
  }
}
```

For load-balanced proxy:
```json
{
  "proxy": {
    "type": "reverse_proxy",
    "hostname": "api.example.com",
    "upstreams": [
      {"host": "192.168.100.10", "port": 3000, "scheme": "http"},
      {"host": "192.168.100.11", "port": 3000, "scheme": "http"}
    ],
    "load_balancing": {
      "strategy": "least_conn",
      "health_checks": {
        "enabled": true,
        "path": "/health",
        "interval": "30s"
      }
    }
  }
}
```

For redirect:
```json
{
  "proxy": {
    "type": "redirect",
    "hostname": "old.example.com",
    "redirect": {
      "target": "https://new.example.com",
      "status_code": 301,
      "preserve_path": true,
      "preserve_query": true
    }
  }
}
```

For static site:
```json
{
  "proxy": {
    "type": "static",
    "hostname": "landing.example.com",
    "static": {
      "root_path": "/var/www/landing",
      "index_file": "index.html",
      "browse": false,
      "try_files": ["index.html"],
      "template_rendering": false
    }
  }
}
```

---

#### `update_proxy`

```json
{
  "proxy_id": 5,
  "hostname": "api.example.com",
  "changes": {
    "upstreams": {
      "old": [
        {"host": "192.168.100.10", "port": 8080}
      ],
      "new": [
        {"host": "192.168.100.10", "port": 8081}
      ]
    },
    "description": {
      "old": "Old description",
      "new": "Updated description"
    }
  }
}
```

Multiple field changes:
```json
{
  "proxy_id": 5,
  "hostname": "api.example.com",
  "changes": {
    "name": {
      "old": "Old Name",
      "new": "New Name"
    },
    "upstreams": {
      "old": [{"host": "192.168.100.10", "port": 8080}],
      "new": [
        {"host": "192.168.100.10", "port": 8080},
        {"host": "192.168.100.11", "port": 8080}
      ]
    },
    "load_balancing": {
      "old": null,
      "new": {
        "strategy": "round_robin",
        "health_checks": {"enabled": true}
      }
    },
    "websocket_support": {
      "old": false,
      "new": true
    },
    "custom_headers": {
      "old": {},
      "new": {"X-Custom": "value"}
    }
  }
}
```

Type change (e.g., reverse_proxy to redirect):
```json
{
  "proxy_id": 5,
  "hostname": "api.example.com",
  "changes": {
    "type": {
      "old": "reverse_proxy",
      "new": "redirect"
    },
    "configuration": {
      "old": {
        "upstreams": [{"host": "192.168.100.10", "port": 8080}]
      },
      "new": {
        "redirect": {
          "target": "https://new-location.com",
          "status_code": 301
        }
      }
    }
  }
}
```

---

#### `delete_proxy`

```json
{
  "deleted_proxy": {
    "id": 5,
    "hostname": "api.example.com",
    "type": "reverse_proxy",
    "name": "Production API",
    "was_active": true,
    "upstreams": [
      {"host": "192.168.100.10", "port": 3000}
    ]
  }
}
```

For redirect type:
```json
{
  "deleted_proxy": {
    "id": 7,
    "hostname": "old.example.com",
    "type": "redirect",
    "name": "Old Domain Redirect",
    "was_active": true,
    "redirect_target": "https://new.example.com"
  }
}
```

---

#### `enable_proxy`

```json
{
  "proxy_id": 5,
  "hostname": "api.example.com",
  "name": "Production API",
  "previous_status": "inactive",
  "new_status": "active"
}
```

---

#### `disable_proxy`

```json
{
  "proxy_id": 5,
  "hostname": "api.example.com",
  "name": "Production API",
  "previous_status": "active",
  "new_status": "inactive",
  "reason": "Scheduled maintenance"
}
```

---

### User Actions

#### `create_user`

```json
{
  "user": {
    "id": 10,
    "email": "newuser@example.com",
    "name": "New User",
    "role": "user",
    "is_active": true
  }
}
```

---

#### `update_user`

Single field change:
```json
{
  "user_id": 10,
  "email": "user@example.com",
  "changes": {
    "name": {
      "old": "Old Name",
      "new": "New Name"
    }
  }
}
```

Multiple field changes:
```json
{
  "user_id": 10,
  "email": "user@example.com",
  "changes": {
    "name": {
      "old": "Old Name",
      "new": "New Name"
    },
    "email": {
      "old": "oldmail@example.com",
      "new": "newmail@example.com"
    }
  }
}
```

---

#### `delete_user`

```json
{
  "deleted_user": {
    "id": 10,
    "email": "user@example.com",
    "name": "User Name",
    "role": "user",
    "was_active": true,
    "created_at": "2024-10-01T10:00:00Z"
  }
}
```

---

#### `update_user_password`

```json
{
  "user_id": 10,
  "email": "user@example.com",
  "password_changed_by": "self",
  "force_logout": true
}
```

When admin changes user password:
```json
{
  "user_id": 10,
  "email": "user@example.com",
  "password_changed_by": "admin",
  "admin_id": 1,
  "admin_email": "admin@example.com",
  "force_logout": true,
  "require_password_change": true
}
```

---

#### `update_user_role`

```json
{
  "user_id": 10,
  "email": "user@example.com",
  "name": "User Name",
  "changes": {
    "role": {
      "old": "user",
      "new": "admin"
    }
  }
}
```

---

### Authentication Actions

#### `login_success`

```json
{
  "login_method": "password",
  "user_id": 1,
  "email": "admin@example.com",
  "session_id": "sess_abc123def456"
}
```

---

#### `login_failed`

```json
{
  "login_method": "password",
  "attempted_email": "admin@example.com",
  "failure_reason": "invalid_password",
  "attempt_count": 3,
  "account_locked": false
}
```

Account locked after multiple failures:
```json
{
  "login_method": "password",
  "attempted_email": "user@example.com",
  "failure_reason": "too_many_attempts",
  "attempt_count": 5,
  "account_locked": true,
  "locked_until": "2024-11-08T11:00:00Z"
}
```

User not found:
```json
{
  "login_method": "password",
  "attempted_email": "nonexistent@example.com",
  "failure_reason": "user_not_found",
  "attempt_count": 1,
  "account_locked": false
}
```

---

#### `password_reset_request`

```json
{
  "user_id": 10,
  "email": "user@example.com",
  "reset_token_sent": true,
  "reset_token_expires_at": "2024-11-08T12:00:00Z"
}
```

---

#### `password_reset_complete`

```json
{
  "user_id": 10,
  "email": "user@example.com",
  "reset_method": "email_token",
  "force_logout_all_sessions": true
}
```

---

### System Actions

#### `update_settings`

```json
{
  "settings_section": "security",
  "changes": {
    "session_timeout": {
      "old": "24h",
      "new": "12h"
    },
    "max_login_attempts": {
      "old": 5,
      "new": 3
    }
  }
}
```

General settings:
```json
{
  "settings_section": "general",
  "changes": {
    "site_name": {
      "old": "Caddy Manager",
      "new": "My Proxy Manager"
    },
    "audit_retention_days": {
      "old": 90,
      "new": 180
    }
  }
}
```

---

#### `export_audit_logs`

```json
{
  "export_format": "csv",
  "filters": {
    "date_from": "2024-11-01T00:00:00Z",
    "date_to": "2024-11-08T23:59:59Z",
    "action": "create_proxy",
    "user_id": null
  },
  "record_count": 45,
  "file_size": 15234,
  "export_id": "audit_export_20241108_103045"
}
```

---

#### `caddy_reload`

```json
{
  "reason": "proxy_configuration_changed",
  "triggered_by_action": "update_proxy",
  "triggered_by_resource_id": 5,
  "reload_success": true,
  "reload_duration_ms": 234
}
```

Failed reload:
```json
{
  "reason": "proxy_configuration_changed",
  "triggered_by_action": "create_proxy",
  "triggered_by_resource_id": 8,
  "reload_success": false,
  "reload_duration_ms": 89,
  "error": "invalid route configuration",
  "rollback_performed": true
}
```

---

## Field Descriptions for Details

### Common Fields Across Actions

| Field | Type | Description |
|-------|------|-------------|
| `proxy` | object | Full proxy configuration (create actions) |
| `changes` | object | Old and new values for updated fields |
| `deleted_*` | object | Snapshot of deleted resource |
| `*_id` | integer | Resource ID being acted upon |
| `previous_status` | string | Status before action |
| `new_status` | string | Status after action |
| `reason` | string | Optional reason for action |

### Change Object Structure

All `changes` objects follow this pattern:
```json
{
  "field_name": {
    "old": "previous value",
    "new": "new value"
  }
}
```

### Special Cases

- **Passwords:** Never logged in details, only metadata about password changes
- **Tokens:** Never logged in details
- **Sensitive Data:** PII and credentials are never included
- **Null Values:** When a field is added (wasn't set before), `old` is `null`
- **Deletions:** When a field is removed, `new` is `null`

---

## Best Practices

### For Admins

1. **Regular Review:** Review audit logs weekly for suspicious activity
2. **Export:** Export logs monthly for long-term retention
3. **Monitor Failed Actions:** Pay attention to failed login attempts and failed operations
4. **Track Changes:** Use audit logs to track who made changes to critical proxies

### For Developers

1. **Search Performance:** Use specific filters instead of broad searches
2. **Pagination:** Use reasonable page sizes (50-100 items)
3. **Date Ranges:** Always specify date ranges for better performance
4. **Export:** Use export for bulk data instead of paginating through large result sets

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

---

## HTTP Status Codes

| Status Code | Description | Usage |
|-------------|-------------|-------|
| 200 OK | Success | GET, POST |
| 400 Bad Request | Invalid request | Validation errors |
| 401 Unauthorized | Not authenticated | Missing/invalid token |
| 403 Forbidden | Not authorized | Insufficient permissions |
| 404 Not Found | Resource not found | Invalid ID |
| 500 Internal Server Error | Server error | Unexpected errors |

---

## Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `VALIDATION_ERROR` | 400 | Request validation failed |
| `UNAUTHORIZED` | 401 | Authentication required |
| `FORBIDDEN` | 403 | Insufficient permissions |
| `AUDIT_LOG_NOT_FOUND` | 404 | Audit log doesn't exist |
| `EXPORT_NOT_FOUND` | 404 | Export file not found or expired |
| `INTERNAL_ERROR` | 500 | Unexpected server error |

---

## Notes

1. **Retention:** Audit logs are retained for 90 days by default (configurable)
2. **Immutability:** Audit logs cannot be modified or deleted by users
3. **Performance:** Indexes on `user_id`, `created_at`, `action`, `resource_type`
4. **Privacy:** User passwords and tokens are never logged in details
5. **Timestamps:** All in ISO 8601 format (UTC)
6. **Export Expiry:** Export files expire after 24 hours
7. **Rate Limiting:** Export is rate-limited to 10 requests per hour per user

---

**Document Version:** 1.0
**Status:** Complete
