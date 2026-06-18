# AGENTS.md - Project Guidelines

This document provides guidelines for working on the Waygates project, a modern reverse proxy manager with a React UI and Go backend.

## Project Overview

Waygates is a single Docker container application that combines:
- **Go Backend** (Port 8080): REST API + static UI files + Caddy management
- **Caddy** (Ports 80, 443): Reverse proxy with automatic HTTPS
- **React Frontend**: Management interface

The application manages Caddy reverse proxy configurations through a web interface with authentication, RBAC, and audit logging.

## Technology Stack

### Backend (Go)
- **Go 1.25** with modules
- **chi/v5** - HTTP router
- **GORM** - ORM (PostgreSQL)
- **goauth** - JWT authentication with RBAC
- **zap** - Structured logging
- **golang-migrate** - Database migrations
- **testify** - Testing assertions
- **testcontainers-go** - Integration tests

### Frontend (React/TypeScript)
- **React 19** with TypeScript
- **Vite** - Build tool
- **TanStack React Router** - Routing
- **TanStack React Query** - Server state
- **TanStack React Form** + **Zod** - Form handling/validation
- **Zustand** - Client state (auth)
- **ky** - HTTP client
- **Tailwind CSS 4** - Styling
- **oxlint** + **oxfmt** - Linter/formatter

## Directory Structure

```
waygates/
├── backend/
│   ├── cmd/server/main.go           # Entry point
│   ├── internal/
│   │   ├── api/
│   │   │   ├── handlers/            # HTTP handlers (one per domain)
│   │   │   ├── middleware/          # HTTP middleware
│   │   │   └── routes/              # Route definitions
│   │   ├── auth/                    # Authentication (goauth wrapper)
│   │   ├── caddy/
│   │   │   ├── caddyfile/           # Caddyfile generation
│   │   │   ├── file_manager.go      # File operations
│   │   │   └── reloader.go          # Caddy reload logic
│   │   ├── config/                  # Configuration (env vars)
│   │   ├── database/                # DB connection & migrations
│   │   ├── models/                  # GORM models
│   │   ├── repository/              # Data access layer
│   │   ├── service/                 # Business logic
│   │   ├── utils/                   # Response formatting
│   │   └── validation/              # Request validation
│   ├── migrations/                  # SQL migrations (up/down)
│   ├── templates/                   # HTML templates
│   └── rbac.yaml                    # RBAC configuration
│
├── ui/
│   ├── src/
│   │   ├── components/              # React components by domain
│   │   ├── hooks/                   # Custom hooks (use-*.ts)
│   │   ├── routes/                  # Page routes
│   │   ├── stores/                  # Zustand stores
│   │   ├── types/                   # TypeScript definitions
│   │   └── lib/                     # Utilities (api, router, validation)
│   ├── .oxlintrc.json               # oxlint (lint) config
│   ├── .oxfmtrc.json                # oxfmt (format) config
│   └── vite.config.ts               # Build config (Vite)
│
├── docs/                            # API documentation
├── docker/entrypoint.sh             # Container startup
├── Dockerfile                       # Multi-stage build
├── docker-compose.yml               # Local deployment
└── Makefile                         # Build commands
```

## Common Commands

### Development
```bash
make backend-run              # Run backend locally
make backend-build            # Build backend binary
pnpm --dir ui dev             # Run frontend dev server (port 8008)
pnpm --dir ui build           # Build frontend
```

### Testing
```bash
make backend-test             # Run all backend tests
make backend-test-coverage    # Run tests with coverage
go test ./... -short          # Unit tests only (skip integration)
```

### Code Quality
```bash
make lint                     # Lint backend + frontend
make lint-backend             # Lint Go backend only (golangci-lint)
make lint-ui                  # Lint + format UI (oxlint + oxfmt, via pnpm check:fix)
make format                   # Format backend + frontend (runs lint-ui for UI)
make format-backend           # Format Go code (gofmt + goimports)
make check                    # Run all checks (lint + tests)
```

### Docker
```bash
make build                    # Build Docker image
make up                       # Start containers
make down                     # Stop containers
make logs                     # View logs
make rebuild                  # Rebuild and restart
```

### Database
```bash
make migrate-create NAME=description    # Create new migration
```

## Code Conventions

### Go Backend

#### Error Handling
Always wrap errors with context:
```go
if err != nil {
    return fmt.Errorf("failed to create proxy: %w", err)
}
```

#### Logging
Use structured logging with named loggers:
```go
logger.Info("proxy created",
    zap.String("name", proxy.Name),
    zap.Int64("id", proxy.ID),
)
```

#### Repository Pattern
- Interfaces in `repository/interfaces.go`
- Implementations in `repository/{domain}_repository.go`
- Services depend on repository interfaces

#### Handler Pattern
```go
type ProxyHandler struct {
    service ProxyService
    logger  *zap.Logger
}

func NewProxyHandler(service ProxyService, logger *zap.Logger) *ProxyHandler {
    return &ProxyHandler{service: service, logger: logger}
}

func (h *ProxyHandler) Create(w http.ResponseWriter, r *http.Request) {
    // Implementation
}
```

#### Test Files
- Place tests alongside implementation: `*_test.go`
- Integration tests: `*_integration_test.go`
- Use testify for assertions
- Use testcontainers for database tests

### React Frontend

#### File Naming
- Components: `kebab-case.tsx` (e.g., `proxy-data-grid.tsx`)
- Hooks: `use-{domain}.ts` (e.g., `use-proxies.ts`)
- Types: `{domain}.ts` in `types/`

#### Component Structure
```tsx
interface ProxyFormProps {
  proxy?: Proxy;
  onSubmit: (data: ProxyFormData) => void;
}

export function ProxyForm({ proxy, onSubmit }: ProxyFormProps) {
  // Implementation
}
```

#### Data Fetching
Use custom hooks with React Query:
```tsx
export function useProxies() {
  return useQuery({
    queryKey: ["proxies"],
    queryFn: () => api.get("proxies").json<ProxyListResponse>(),
  });
}
```

#### Form Validation
Use Zod schemas in `lib/form-validation.ts`:
```tsx
export const proxySchema = z.object({
  name: z.string().min(1, "Name is required"),
  target: z.string().url("Must be a valid URL"),
});
```

#### Data Tables
**Always use `DataGrid` from `@e412/titanium` for displaying tabular data.** Do not use the basic `Table` component directly. DataGrid provides consistent styling, skeleton loading states, sorting, pagination, and row click handling.

```tsx
import {
  DataGrid,
  DataGridColumnHeader,
  DataGridContainer,
  DataGridPagination,
  DataGridTable,
} from '@e412/titanium';
import { type ColumnDef, getCoreRowModel, useReactTable } from '@tanstack/react-table';

const columns = useMemo<ColumnDef<MyData>[]>(() => [
  {
    accessorKey: 'name',
    header: ({ column }) => <DataGridColumnHeader column={column} title="Name" />,
    cell: ({ row }) => <span>{row.getValue('name')}</span>,
    meta: {
      skeleton: <Skeleton className="h-5 w-32" />,
    },
  },
], []);

const table = useReactTable({
  data,
  columns,
  getCoreRowModel: getCoreRowModel(),
});

return (
  <DataGrid
    table={table}
    recordCount={data.length}
    isLoading={isLoading}
    loadingMode="skeleton"
    emptyMessage="No data found"
    onRowClick={handleRowClick} // Optional: receives row data directly
  >
    <DataGridContainer>
      <DataGridTable />
      <DataGridPagination sizes={[10, 20, 50]} /> {/* Optional */}
    </DataGridContainer>
  </DataGrid>
);
```

## Testing Guidelines

### Backend Tests

#### Unit Tests
Mock repositories/services for isolated testing:
```go
func TestProxyService_Create(t *testing.T) {
    mockRepo := &MockProxyRepository{}
    service := NewProxyService(mockRepo, logger)

    // Test implementation
}
```

#### Integration Tests
Use testcontainers for real database:
```go
func TestProxyHandler_Integration(t *testing.T) {
    ctx := context.Background()
    container, db := setupTestContainer(ctx, t)
    defer container.Terminate(ctx)

    // Test with real database
}
```

#### Coverage Target
Aim for >80% code coverage on business logic.

## Database Changes

1. Create migration: `make migrate-create NAME=add_column_to_proxies`
2. Edit `migrations/{timestamp}_add_column_to_proxies.up.sql`
3. Edit `migrations/{timestamp}_add_column_to_proxies.down.sql`
4. Migrations run automatically on startup

Migration file format:
```sql
-- up.sql
ALTER TABLE proxies ADD COLUMN new_field VARCHAR(255);

-- down.sql
ALTER TABLE proxies DROP COLUMN new_field;
```

## API Guidelines

### Endpoint Structure
- Base URL: `/api`
- Protected routes require `Authorization: Bearer {token}`
- Permission checks via RBAC middleware

### Response Format
```json
{
  "success": true,
  "data": { },
  "error": null
}
```

### Adding New Endpoints

1. Add handler in `handlers/{domain}_handler.go`
2. Add service methods in `service/{domain}_service.go`
3. Add repository methods in `repository/{domain}_repository.go`
4. Register routes in `routes/routes.go`
5. Add RBAC permissions in `rbac.yaml` if needed
6. Update documentation in `docs/`

### RBAC Permissions
Defined in `backend/rbac.yaml`:
- `proxies:read`, `proxies:create`, `proxies:update`, `proxies:delete`
- `acl:read`, `acl:create`, `acl:update`, `acl:delete`
- `settings:read`, `settings:write`
- `audit:read`
- `sync:read`, `sync:trigger`

## Configuration

All configuration via environment variables (see `.env.example`):

### Required
- `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD`
- `JWT_SECRET` (generate with `openssl rand -base64 32`)

### Optional
- `SERVER_PORT` (default: 8080)
- `LOG_LEVEL` (default: info)
- `JWT_ACCESS_EXPIRY`, `JWT_REFRESH_EXPIRY`
- `BCRYPT_COST` (default: 12)
- TLS/ACME settings for automatic HTTPS

## Security Considerations

- Never hardcode secrets; use environment variables
- Use parameterized queries (GORM handles this)
- Validate all user input (go-playground/validator + Zod)
- Follow RBAC; check permissions in handlers
- Log security events to audit log
- Use HTTPS in production (Caddy auto-manages)

## Sync Service

The sync service (`service/sync_service.go`) runs every 60 seconds:
1. Reads proxies from database
2. Generates Caddyfile configurations
3. Writes to `sites/` directory
4. Reloads Caddy

Manual sync: `POST /api/sync/trigger` (requires `sync:trigger` permission)

## Key Files Reference

| Purpose | Location |
|---------|----------|
| Main entry point | `backend/cmd/server/main.go` |
| Route registration | `backend/internal/api/routes/routes.go` |
| RBAC config | `backend/rbac.yaml` |
| Database models | `backend/internal/models/` |
| Caddyfile generation | `backend/internal/caddy/caddyfile/` |
| API client | `ui/src/lib/api.ts` |
| Auth store | `ui/src/stores/auth.ts` |
| Form schemas | `ui/src/lib/form-validation.ts` |
| Environment template | `.env.example` |

## Development Workflow

1. Create feature branch from main
2. Implement backend changes (service -> repository -> handler)
3. Add/update tests
4. Implement frontend changes
5. **Run lint and tests** (see Post-Implementation Checklist below)
6. Update documentation if needed
7. Create PR with clear description

## Post-Implementation Checklist

**IMPORTANT: After implementing any feature, you MUST lint and format the code.**

### Backend Changes
```bash
make format-backend           # Format Go code (gofmt + goimports)
make lint-backend             # Run golangci-lint on Go code
make backend-test             # Run backend tests
```

### Frontend Changes
```bash
make lint-ui                  # Lint + format UI (oxlint + oxfmt, via pnpm check:fix)
```

### Both Backend and Frontend Changes
```bash
make format                   # Format both backend and frontend
make lint                     # Lint both backend and frontend
make check                    # Run all checks (lint + tests)
```

Fix any linting or formatting errors before committing. Do not skip this step.
