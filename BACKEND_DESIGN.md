# Caddy Management Backend - Design Document

## Overview

A Golang-based backend service that acts as an authenticated middleware between a management UI and Caddy's Admin API, similar to Nginx Proxy Manager but designed specifically for Caddy.

## Architecture

### High-Level Architecture

```
┌─────────────────┐
│   Management    │
│       UI        │
└────────┬────────┘
         │ (HTTP/REST)
         │
┌────────▼────────┐
│   Go Backend    │
│  ┌───────────┐  │
│  │   Auth    │  │
│  │  Layer    │  │
│  └─────┬─────┘  │
│  ┌─────▼─────┐  │
│  │  Business │  │
│  │   Logic   │  │
│  └─────┬─────┘  │
│  ┌─────▼─────┐  │
│  │  Caddy    │  │
│  │  Client   │  │
│  └───────────┘  │
│        │        │
│  ┌─────▼─────┐  │
│  │ Database  │  │
│  │ (SQLite/  │  │
│  │PostgreSQL)│  │
│  └───────────┘  │
└────────┬────────┘
         │ (HTTP)
         │
┌────────▼────────┐
│  Caddy Admin    │
│      API        │
│   (Port 2019)   │
└─────────────────┘
```

## Tech Stack

### Core Technologies

- **Language**: Go 1.21+
- **Web Framework**: Chi (lightweight, idiomatic)
- **Database**:
  - Phase 1: SQLite (simple, embedded)
  - Phase 2: PostgreSQL (production-ready)
- **ORM**: GORM (popular, feature-rich)
- **Authentication**:
  - Phase 1: JWT (JSON Web Tokens)
  - Phase 2: OIDC (OpenID Connect)
- **Validation**: go-playground/validator
- **Configuration**: Viper
- **Logging**: Zap (structured logging)
- **HTTP Client**: Standard library + custom Caddy client

### Additional Libraries

- **Password Hashing**: bcrypt
- **Middleware**: chi/middleware
- **Environment**: godotenv
- **Migration**: golang-migrate
- **Testing**: testify

## Project Structure

```
backend/
├── cmd/
│   └── server/
│       └── main.go                 # Application entry point
├── internal/
│   ├── api/
│   │   ├── handlers/               # HTTP handlers
│   │   │   ├── auth.go
│   │   │   ├── proxy.go
│   │   │   ├── user.go
│   │   │   └── audit.go
│   │   ├── middleware/             # Custom middleware
│   │   │   ├── auth.go
│   │   │   ├── logger.go
│   │   │   └── cors.go
│   │   └── routes/                 # Route definitions
│   │       └── routes.go
│   ├── models/                     # Data models
│   │   ├── user.go
│   │   ├── proxy.go
│   │   └── audit.go
│   ├── repository/                 # Database layer
│   │   ├── user_repo.go
│   │   ├── proxy_repo.go
│   │   └── audit_repo.go
│   ├── service/                    # Business logic
│   │   ├── auth_service.go
│   │   ├── proxy_service.go
│   │   ├── user_service.go
│   │   └── audit_service.go
│   ├── caddy/                      # Caddy Admin API client
│   │   ├── client.go
│   │   ├── config.go
│   │   └── routes.go
│   ├── config/                     # Configuration
│   │   └── config.go
│   └── utils/                      # Utilities
│       ├── jwt.go
│       ├── password.go
│       └── response.go
├── migrations/                     # Database migrations
│   ├── 001_create_users.sql
│   ├── 002_create_proxies.sql
│   └── 003_create_audit_logs.sql
├── pkg/                            # Public packages (if any)
├── tests/                          # Integration tests
├── .env.example
├── .gitignore
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## Database Schema

### Users Table

```sql
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'user', -- 'admin', 'user'
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_login_at TIMESTAMP
);
```

### Proxies Table

```sql
CREATE TABLE proxies (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(255) NOT NULL,
    hostname VARCHAR(255) UNIQUE NOT NULL,
    upstream_host VARCHAR(255) NOT NULL,
    upstream_port INTEGER NOT NULL,
    upstream_scheme VARCHAR(10) DEFAULT 'http', -- 'http', 'https'

    -- SSL/TLS
    ssl_enabled BOOLEAN DEFAULT true,
    ssl_forced BOOLEAN DEFAULT true,

    -- Advanced settings
    websocket_support BOOLEAN DEFAULT false,
    block_exploits BOOLEAN DEFAULT true,

    -- Custom headers (JSON)
    custom_headers TEXT, -- JSON: {"X-Custom": "value"}

    -- Metadata
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    created_by INTEGER REFERENCES users(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Audit Logs Table

```sql
CREATE TABLE audit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER REFERENCES users(id),
    action VARCHAR(100) NOT NULL, -- 'create_proxy', 'update_proxy', 'delete_proxy', etc.
    resource_type VARCHAR(50), -- 'proxy', 'user', 'settings'
    resource_id INTEGER,
    details TEXT, -- JSON with additional context
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_audit_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_created_at ON audit_logs(created_at);
CREATE INDEX idx_audit_action ON audit_logs(action);
```

## API Endpoints

### Authentication

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| POST | `/api/auth/login` | Login with email/password | No |
| POST | `/api/auth/logout` | Logout (invalidate token) | Yes |
| POST | `/api/auth/refresh` | Refresh JWT token | Yes |
| GET | `/api/auth/me` | Get current user info | Yes |

### Users

| Method | Endpoint | Description | Auth Required | Role |
|--------|----------|-------------|---------------|------|
| GET | `/api/users` | List all users | Yes | Admin |
| GET | `/api/users/:id` | Get user by ID | Yes | Admin/Self |
| POST | `/api/users` | Create new user | Yes | Admin |
| PUT | `/api/users/:id` | Update user | Yes | Admin/Self |
| DELETE | `/api/users/:id` | Delete user | Yes | Admin |
| PUT | `/api/users/:id/password` | Change password | Yes | Admin/Self |

### Proxy Hosts

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| GET | `/api/proxies` | List all proxy hosts | Yes |
| GET | `/api/proxies/:id` | Get proxy by ID | Yes |
| POST | `/api/proxies` | Create new proxy | Yes |
| PUT | `/api/proxies/:id` | Update proxy | Yes |
| DELETE | `/api/proxies/:id` | Delete proxy | Yes |
| POST | `/api/proxies/:id/enable` | Enable proxy | Yes |
| POST | `/api/proxies/:id/disable` | Disable proxy | Yes |

### Audit Logs

| Method | Endpoint | Description | Auth Required | Role |
|--------|----------|-------------|---------------|------|
| GET | `/api/audit` | List audit logs | Yes | Admin |
| GET | `/api/audit/:id` | Get specific log | Yes | Admin |

### Health & Status

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| GET | `/api/health` | Health check | No |
| GET | `/api/status` | System status | Yes |
| GET | `/api/caddy/status` | Caddy connection status | Yes |

## Data Flow

### Creating a Proxy Host

```
1. UI sends POST /api/proxies with proxy details
   ↓
2. Backend validates request (auth, data)
   ↓
3. Backend checks if hostname already exists in DB
   ↓
4. Backend transforms UI data → Caddy route format
   ↓
5. Backend calls Caddy Admin API to add route
   ↓
6. If success: Save proxy to database
   ↓
7. Create audit log entry
   ↓
8. Return simplified response to UI
```

### Response Transformation

**Caddy Admin API Response (Raw):**
```json
{
  "@id": "proxy_123",
  "match": [{"host": ["app.example.com"]}],
  "handle": [
    {
      "handler": "reverse_proxy",
      "upstreams": [{"dial": "192.168.1.100:8080"}],
      "headers": {...}
    }
  ],
  "terminal": true
}
```

**Backend Response to UI (Simplified):**
```json
{
  "id": 1,
  "name": "My App",
  "hostname": "app.example.com",
  "upstream": "192.168.1.100",
  "port": 8080,
  "ssl_enabled": true,
  "is_active": true,
  "created_at": "2024-11-08T00:00:00Z"
}
```

## Authentication Flow

### JWT-based Authentication (Phase 1)

1. **Login:**
   - User sends credentials
   - Backend validates against database
   - Generate JWT token (access + refresh)
   - Return tokens to client

2. **Protected Request:**
   - Client sends JWT in `Authorization: Bearer <token>` header
   - Middleware validates token
   - Extract user info from token
   - Attach to request context
   - Proceed to handler

3. **Token Refresh:**
   - Client sends refresh token
   - Validate refresh token
   - Generate new access token
   - Return new tokens

### OIDC Integration (Phase 2)

- Support external identity providers (Google, GitHub, Okta, etc.)
- Implement OAuth2/OIDC flow
- Map OIDC claims to user roles
- Maintain session management

## Configuration

### Environment Variables

```env
# Server
SERVER_PORT=8080
SERVER_HOST=0.0.0.0

# Database
DB_TYPE=sqlite # sqlite or postgres
DB_PATH=./data/caddy-manager.db
# For PostgreSQL:
# DB_HOST=localhost
# DB_PORT=5432
# DB_USER=caddymanager
# DB_PASSWORD=secret
# DB_NAME=caddymanager

# Caddy
CADDY_ADMIN_URL=http://localhost:2019
CADDY_TIMEOUT=10s

# JWT
JWT_SECRET=your-secret-key
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=7d

# Security
BCRYPT_COST=10
CORS_ORIGINS=http://localhost:3000

# Logging
LOG_LEVEL=info # debug, info, warn, error
LOG_FORMAT=json # json or console
```

## Security Considerations

1. **Password Security:**
   - Use bcrypt with appropriate cost (10-12)
   - Enforce password complexity rules
   - Implement rate limiting on login

2. **JWT Security:**
   - Short-lived access tokens (15 min)
   - Longer refresh tokens (7 days)
   - Store tokens securely on client
   - Implement token blacklist for logout

3. **API Security:**
   - CORS configuration
   - Rate limiting
   - Input validation
   - SQL injection prevention (via ORM)
   - XSS prevention

4. **Audit Logging:**
   - Log all sensitive operations
   - Include user, IP, timestamp
   - Immutable logs

## Development Phases

### Phase 1: Core Functionality (MVP)

- [x] Design architecture
- [ ] Setup project structure
- [ ] Implement database models & migrations
- [ ] Build Caddy Admin API client
- [ ] Implement authentication (JWT)
- [ ] User management endpoints
- [ ] Proxy management endpoints
- [ ] Basic audit logging
- [ ] Health check endpoints
- [ ] Basic error handling

### Phase 2: Enhanced Features

- [ ] OIDC authentication
- [ ] Advanced proxy settings
- [ ] SSL certificate management
- [ ] Access lists / IP filtering
- [ ] Enhanced audit logs (filtering, search)
- [ ] Rate limiting
- [ ] Backup/restore functionality

### Phase 3: Production Ready

- [ ] Comprehensive testing
- [ ] Performance optimization
- [ ] Docker support
- [ ] Documentation
- [ ] Monitoring & metrics
- [ ] CI/CD pipeline

## Caddy Integration Points

### Operations Mapping

| Backend Operation | Caddy Admin API Call |
|-------------------|---------------------|
| Create Proxy | POST /config/apps/http/servers/srv0/routes/0 |
| Update Proxy | PATCH /config/apps/http/servers/srv0/routes |
| Delete Proxy | DELETE /id/{proxy_id} |
| List Proxies | GET /config/apps/http/servers/srv0/routes |
| Get Proxy | GET /id/{proxy_id} |
| Check Health | GET /config/ |

## Next Steps

1. Initialize Go module and project structure
2. Setup database and migrations
3. Implement Caddy client library
4. Build authentication layer
5. Create proxy management endpoints
6. Add audit logging
7. Build simple CLI/UI for testing

---

**Document Version:** 1.0
**Last Updated:** 2024-11-08
**Status:** Draft
