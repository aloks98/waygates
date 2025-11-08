# Caddy Manager Backend

Go-based backend API that provides an authenticated middleware layer between the management UI and Caddy's Admin API.

## Features

- ✅ Health check endpoint
- 🚧 User authentication (JWT)
- 🚧 Proxy management (reverse proxy, redirect, static)
- 🚧 User management
- 🚧 Audit logging
- 🚧 OIDC authentication (Phase 2)

## Project Structure

```
backend/
├── cmd/
│   └── server/
│       └── main.go                 # Application entry point
├── internal/
│   ├── api/
│   │   ├── handlers/               # HTTP request handlers
│   │   │   └── health.go          # ✅ Health check handler
│   │   ├── middleware/             # Custom middleware (auth, logging, etc.)
│   │   └── routes/                 # Route definitions
│   │       └── routes.go          # ✅ Router setup
│   ├── config/                     # Configuration management
│   │   └── config.go              # ✅ Config loading with Viper
│   ├── utils/                      # Utility functions
│   │   └── response.go            # ✅ Standard API responses
│   ├── models/                     # Data models (to be implemented)
│   ├── repository/                 # Database layer (to be implemented)
│   ├── service/                    # Business logic (to be implemented)
│   └── caddy/                      # Caddy Admin API client (to be implemented)
├── migrations/                     # Database migrations
└── tests/                          # Tests
```

## Tech Stack

- **Go**: 1.21+
- **Web Framework**: Chi v5
- **ORM**: GORM
- **Database**: SQLite (Phase 1) / PostgreSQL (Phase 2)
- **Logging**: Zap (structured JSON logging)
- **Config**: Viper
- **Auth**: JWT (golang-jwt/jwt)
- **Validation**: go-playground/validator

## Prerequisites

- Go 1.21 or higher
- SQLite (included) or PostgreSQL (optional)
- Caddy server running with Admin API on port 2019

## Quick Start

### 1. Install Dependencies

Dependencies are already installed via `go.mod`. To verify:

```bash
go mod download
go mod verify
```

### 2. Configure Environment

Copy the example environment file and update it:

```bash
# .env already configured in project root
# Edit if needed:
vim ../.env
```

Required environment variables:

```bash
JWT_SECRET=your-secret-key-here  # Required!
SERVER_PORT=8080
CADDY_ADMIN_URL=http://localhost:2019
```

### 3. Run the Server

From the **project root** directory:

```bash
go run backend/cmd/server/main.go
```

Or build and run:

```bash
go build -o caddy-manager backend/cmd/server/main.go
./caddy-manager
```

### 4. Test the Health Endpoint

```bash
curl http://localhost:8080/api/health
```

Expected response:

```json
{
  "success": true,
  "data": {
    "status": "healthy",
    "service": "caddy-manager-backend",
    "version": "1.0.0",
    "uptime": "1m30s",
    "time": "2024-11-08T10:30:00Z"
  }
}
```

## Available Endpoints

### Public Endpoints (No Auth Required)

| Method | Endpoint | Description | Status |
|--------|----------|-------------|--------|
| GET | `/api/health` | Health check | ✅ Done |

### Protected Endpoints (Coming Soon)

Authentication, proxy management, user management, and audit log endpoints will be added as development progresses.

## Development

### Run with Auto-Reload

Install air for live reloading:

```bash
go install github.com/air-verse/air@latest
```

Create `.air.toml` in backend/ directory or run:

```bash
air -c .air.toml
```

### Project Guidelines

1. **Always run from project root** - The `.env` file is in the root directory
2. **Use structured logging** - All logs use Zap with JSON format in production
3. **Standard responses** - Use `utils.Success()`, `utils.Error()` helpers
4. **Configuration** - Add new config to `internal/config/config.go`
5. **Handlers** - Keep handlers thin, business logic goes in services

### Adding New Endpoints

1. Create handler in `internal/api/handlers/`
2. Define routes in `internal/api/routes/routes.go`
3. Add business logic in `internal/service/`
4. Add data access in `internal/repository/`

## Configuration Reference

All configuration is loaded from environment variables (see `.env` file in project root):

```bash
# Server
SERVER_HOST=0.0.0.0
SERVER_PORT=8080

# Database
DB_TYPE=sqlite
DB_PATH=./backend/data/caddy-manager.db

# Caddy
CADDY_ADMIN_URL=http://localhost:2019
CADDY_TIMEOUT=10s

# JWT
JWT_SECRET=required-secret-key
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=168h

# Security
BCRYPT_COST=10
CORS_ORIGINS=http://localhost:3000,http://localhost:8080

# Logging
LOG_LEVEL=info          # debug, info, warn, error
LOG_FORMAT=json         # json or console
```

## Next Steps

As you implement features, refer to:

- **API Documentation**: `/docs/API_PROXY.md` - Proxy management API spec
- **API Documentation**: `/docs/API_AUDIT.md` - Audit logging API spec
- **Design Document**: `/BACKEND_DESIGN.md` - Overall architecture

### Immediate TODOs

1. Implement database models and migrations
2. Create Caddy Admin API client
3. Implement authentication (JWT)
4. Add proxy management endpoints
5. Add user management
6. Implement audit logging

## Troubleshooting

**"JWT_SECRET is required"**
- Make sure you're running from the project root (not from `backend/` directory)
- Verify `.env` file exists in project root with `JWT_SECRET` set

**"Cannot connect to database"**
- Check `DB_PATH` points to a writable location
- For PostgreSQL, verify connection details

**"Cannot reach Caddy Admin API"**
- Verify Caddy is running: `curl http://localhost:2019/config/`
- Check `CADDY_ADMIN_URL` in `.env`

## License

See project root LICENSE file.
