.PHONY: help build up down restart logs logs-follow status clean rebuild validate env-check
.PHONY: backend-run backend-build backend-test backend-test-coverage test-traffic migrate-create
.PHONY: lint lint-backend lint-ui format format-backend check setup-tools

# Version stamped into the binary/image via -ldflags. Strip the leading "v" so
# the UI's "v{version}" display doesn't render a double "v". Falls back to "dev"
# outside a git checkout.
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null | sed 's/^v//' || echo dev)
LDFLAGS := -X github.com/aloks98/waygates/backend/internal/version.Version=$(VERSION)

# Default target
help:
	@echo "Available commands:"
	@echo ""
	@echo "Docker:"
	@echo "  make build         - Build the Waygates Docker image"
	@echo "  make up            - Start containers in detached mode"
	@echo "  make down          - Stop and remove containers"
	@echo "  make restart       - Restart containers"
	@echo "  make logs          - Show recent logs"
	@echo "  make logs-follow   - Follow logs in real-time"
	@echo "  make status        - Show container status"
	@echo "  make clean         - Remove containers, volumes, and images"
	@echo "  make rebuild       - Clean build and restart everything"
	@echo "  make validate      - Validate Caddy JSON config"
	@echo "  make deploy        - Full deployment (env-check, build, up)"
	@echo ""
	@echo "Backend (Go):"
	@echo "  make backend-run   - Run the Go backend server locally"
	@echo "  make backend-build - Build the Go backend binary"
	@echo "  make backend-test  - Run backend tests"
	@echo "  make backend-test-coverage - Run tests with coverage report"
	@echo "  make test-traffic  - Build image and run proxy traffic E2E suite (Docker required)"
	@echo ""
	@echo "Linting & Formatting:"
	@echo "  make lint          - Run linters on both backend and UI"
	@echo "  make lint-backend  - Run golangci-lint on backend"
	@echo "  make lint-ui       - Lint and format UI with oxlint + oxfmt"
	@echo "  make format        - Format both backend and UI code"
	@echo "  make format-backend - Format Go code with gofmt/goimports"
	@echo "  make check         - Run all checks (lint + tests)"
	@echo ""
	@echo "Database Migrations:"
	@echo "  make migrate-create NAME=name - Create new migration files"
	@echo "  Note: Migrations run automatically when backend starts"
	@echo ""
	@echo "Other:"
	@echo "  make env-check     - Check if .env file exists"
	@echo "  make setup-tools   - Install development tools (golangci-lint, goimports)"

# Check if .env file exists
env-check:
	@if [ ! -f .env ]; then \
		echo "❌ Error: .env file not found!"; \
		echo "Please copy .env.example to .env and configure it:"; \
		echo "  cp .env.example .env"; \
		exit 1; \
	fi
	@echo "✓ Environment file exists"

# Build the Docker image
build:
	@echo "Building Waygates Docker image (version $(VERSION))..."
	VERSION=$(VERSION) docker compose build

# Start containers
up:
	@echo "Starting containers..."
	docker compose up -d
	@echo "✓ Containers started"
	@echo "Run 'make logs-follow' to watch logs"

# Stop containers
down:
	@echo "Stopping containers..."
	docker compose down
	@echo "✓ Containers stopped"

# Restart containers
restart:
	@echo "Restarting containers..."
	docker compose restart
	@echo "✓ Containers restarted"

# Show logs
logs:
	docker compose logs --tail=100

# Follow logs in real-time
logs-follow:
	docker compose logs -f

# Show container status
status:
	@echo "Container status:"
	@docker compose ps
	@echo ""
	@echo "Backend API: http://localhost:8080"
	@echo "Caddy Admin: http://localhost:2019/config/"

# Clean everything (containers, volumes, images)
clean:
	@echo "Cleaning up..."
	docker compose down -v
	docker rmi waygates 2>/dev/null || true
	@echo "✓ Cleanup complete"

# Rebuild everything from scratch
rebuild: clean build up

# Validate Caddy JSON config (requires running container)
validate:
	@echo "Validating Caddy JSON config..."
	docker compose exec waygates caddy validate --config /etc/caddy/caddy.json
	@echo "✓ Caddy config is valid"

# Full deployment pipeline
deploy: env-check build up
	@echo ""
	@echo "✓ Deployment complete!"
	@echo ""
	@echo "Services:"
	@echo "  Backend API: http://localhost:8080"
	@echo "  HTTP:        http://localhost:80"
	@echo "  HTTPS:       https://localhost:443"
	@echo ""
	@echo "View logs with: make logs-follow"

# ============================================
# Backend Commands
# ============================================

# Run the Go backend server locally
backend-run: env-check
	@echo "Starting Go backend server (version $(VERSION))..."
	@go run -ldflags "$(LDFLAGS)" backend/cmd/server/main.go

# Build the Go backend binary
backend-build:
	@echo "Building Go backend (version $(VERSION))..."
	@go build -ldflags "$(LDFLAGS)" -o bin/waygates backend/cmd/server/main.go
	@echo "✓ Binary created at: bin/waygates"

# Run backend tests
backend-test:
	@echo "Running backend tests..."
	@go test -v ./backend/...

# Build the image and run the proxy traffic E2E suite (Docker required)
test-traffic:
	@echo "Building waygates-test:latest..."
	@docker build -t waygates-test:latest .
	@echo "Running proxy traffic E2E suite..."
	@cd backend && go test -tags traffic -run 'TestTraffic' ./tests/integration/ -count=1 -v

# Run backend tests with coverage (includes cross-package coverage from integration tests)
backend-test-coverage:
	@echo "Running backend tests with coverage..."
	@cd backend && go test -count=1 -coverprofile=coverage.out -covermode=atomic -coverpkg=./... ./...
	@cd backend && go tool cover -func=coverage.out | grep total
	@echo "✓ Coverage report generated at backend/coverage.out"
	@echo "Run 'go tool cover -html=backend/coverage.out' to view in browser"

# ============================================
# Database Migration Commands
# ============================================
# Note: Migrations run automatically when the backend starts

# Create new migration files
migrate-create:
	@if [ -z "$(NAME)" ]; then \
		echo "❌ Error: NAME is required"; \
		echo "Usage: make migrate-create NAME=create_users_table"; \
		exit 1; \
	fi
	@echo "Creating migration: $(NAME)"
	@bash -c 'NEXT=$$(ls backend/migrations/*.up.sql 2>/dev/null | wc -l | tr -d " "); \
		NEXT=$$((NEXT + 1)); \
		NUM=$$(printf "%06d" $$NEXT); \
		touch backend/migrations/$${NUM}_$(NAME).up.sql; \
		touch backend/migrations/$${NUM}_$(NAME).down.sql; \
		echo "✓ Created:"; \
		echo "  backend/migrations/$${NUM}_$(NAME).up.sql"; \
		echo "  backend/migrations/$${NUM}_$(NAME).down.sql"'

# ============================================
# Linting & Formatting Commands
# ============================================

# Run all linters
lint: lint-backend lint-ui

# Lint backend with golangci-lint
lint-backend:
	@echo "Linting Go backend..."
	@golangci-lint run ./backend/...
	@echo "✓ Backend lint complete"

# Lint and format UI with oxlint + oxfmt
lint-ui:
	@echo "Linting and formatting UI..."
	@cd ui && pnpm check:fix
	@echo "✓ UI check complete"

# Format all code
format: format-backend lint-ui

# Format backend with gofmt and goimports
format-backend:
	@echo "Formatting Go backend..."
	@gofmt -w -s backend/
	@goimports -w -local github.com/aloks98/waygates backend/
	@echo "✓ Backend formatted"

# Run all checks (lint + tests)
check: lint backend-test
	@echo "✓ All checks passed"

# Install development tools
setup-tools:
	@echo "Installing development tools..."
	@echo "Installing golangci-lint..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Installing goimports..."
	@go install golang.org/x/tools/cmd/goimports@latest
	@echo "Installing UI dependencies..."
	@cd ui && pnpm install
	@echo "✓ Development tools installed"
	@echo ""
	@echo "Make sure $$GOPATH/bin is in your PATH"
