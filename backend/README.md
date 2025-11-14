# Caddy Manager Backend

Go-based backend API that provides an authenticated middleware layer between the management UI and Caddy's Admin API.

## Features

- ✅ Health check endpoint
- ✅ User authentication (JWT)
- ✅ Proxy management (reverse proxy, redirect, static)
- 🚧 User management
- 🚧 Audit logging
- 🚧 OIDC authentication (Phase 2)

...

## Available Endpoints

### Public Endpoints

| Method | Endpoint             | Description                               | Status |
|--------|----------------------|-------------------------------------------|--------|
| GET    | `/api/health`        | Health check                              | ✅ Done |
| GET    | `/api/status`        | Get application and Caddy status          | ✅ Done |
| POST   | `/api/auth/register` | Register a new user                       | ✅ Done |
| POST   | `/api/auth/login`    | Log in and receive JWT tokens             | ✅ Done |

### Protected Endpoints (Auth Required)

| Method | Endpoint                | Description                 | Status |
|--------|-------------------------|-----------------------------|--------|
| GET    | `/api/proxies`          | List all proxies            | ✅ Done |
| GET    | `/api/proxies/:id`      | Get a single proxy          | ✅ Done |
| POST   | `/api/proxies`          | Create a new proxy          | ✅ Done |
| PUT    | `/api/proxies/:id`      | Update a proxy              | ✅ Done |
| DELETE | `/api/proxies/:id`      | Delete a proxy              | ✅ Done |
| POST   | `/api/proxies/:id/enable`| Enable a proxy             | ✅ Done |
| POST   | `/api/proxies/:id/disable`| Disable a proxy            | ✅ Done |

...

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

# Default User (optional, for initial setup)
# If these are set, a default user will be created on first run if no other users exist.
DEFAULT_USER_NAME=Admin
DEFAULT_USER_USERNAME=admin
DEFAULT_USER_EMAIL=admin@example.com
DEFAULT_USER_PASSWORD=password
```

...

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
