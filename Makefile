.PHONY: help build up down restart logs logs-follow status clean rebuild validate env-check ui-build
.PHONY: backend-run backend-build backend-test migrate-create

# Default target
help:
	@echo "Available commands:"
	@echo ""
	@echo "Caddy (Docker):"
	@echo "  make build         - Build the Caddy Docker image"
	@echo "  make up            - Start containers in detached mode"
	@echo "  make down          - Stop and remove containers"
	@echo "  make restart       - Restart containers"
	@echo "  make logs          - Show recent logs"
	@echo "  make logs-follow   - Follow logs in real-time"
	@echo "  make status        - Show container status"
	@echo "  make clean         - Remove containers, volumes, and images"
	@echo "  make rebuild       - Clean build and restart everything"
	@echo "  make validate      - Validate Caddyfile syntax"
	@echo "  make deploy        - Full deployment (env-check, build, up)"
	@echo ""
	@echo "Backend (Go):"
	@echo "  make backend-run   - Run the Go backend server"
	@echo "  make backend-build - Build the Go backend binary"
	@echo "  make backend-test  - Run backend tests"
	@echo ""
	@echo "UI (Next.js):"
	@echo "  make ui-build      - Build the UI Docker image"
	@echo ""
	@echo "Database Migrations:"
	@echo "  make migrate-create NAME=name - Create new migration files"
	@echo "  Note: Migrations run automatically when backend starts"
	@echo ""
	@echo "Other:"
	@echo "  make env-check     - Check if .env file exists and is configured"

# Check if .env file exists and is configured
env-check:
	@if [ ! -f .env ]; then \
		echo "❌ Error: .env file not found!"; \
		echo "Please copy .env.example to .env and configure it:"; \
		echo "  cp .env.example .env"; \
		exit 1; \
	fi
	@if grep -q "your_cloudflare_api_token_here" .env; then \
		echo "⚠️  Warning: .env file contains placeholder values!"; \
		echo "Please update CLOUDFLARE_API_TOKEN in .env file"; \
		exit 1; \
	fi
	@echo "✓ Environment file configured"

# Build the Docker images
build:
	@echo "Building Caddy with Cloudflare DNS plugin..."
	docker compose build
	$(MAKE) ui-build

# Build the UI Docker image
ui-build:
	@echo "Building UI Docker image..."
	docker build -f Dockerfile.ui -t homelab-proxy-ui .
	@echo "✓ UI Docker image built: homelab-proxy-ui"

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
	@echo "Caddy admin API:"
	@echo "  http://localhost:2019/config/"

# Clean everything (containers, volumes, images)
clean:
	@echo "Cleaning up..."
	docker compose down -v
	docker rmi homelab-proxy-caddy 2>/dev/null || true
	@echo "✓ Cleanup complete"

# Rebuild everything from scratch
rebuild: clean build up

# Validate Caddyfile syntax
validate:
	@echo "Validating Caddyfile..."
	@if [ -z "$$(docker images -q homelab-proxy-caddy 2> /dev/null)" ]; then \
		echo "Custom Caddy image not found. Building first..."; \
		$(MAKE) build; \
	fi
	@docker run --rm \
		--env-file .env \
		-v $(PWD)/conf:/etc/caddy \
		homelab-proxy-caddy \
		caddy validate --config /etc/caddy/Caddyfile
	@echo "✓ Caddyfile is valid"

# Full deployment pipeline
deploy: env-check build up
	@echo ""
	@echo "✓ Deployment complete!"
	@echo ""
	@echo "Services:"
	@echo "  Admin API: http://localhost:2019"
	@echo "  HTTP:      http://localhost:80"
	@echo "  HTTPS:     https://localhost:443"
	@echo ""
	@echo "View logs with: make logs-follow"

# ============================================
# Backend Commands
# ============================================

# Run the Go backend server
backend-run: env-check
	@echo "Starting Go backend server..."
	@go run backend/cmd/server/main.go

# Build the Go backend binary
backend-build:
	@echo "Building Go backend..."
	@go build -o bin/caddy-manager backend/cmd/server/main.go
	@echo "✓ Binary created at: bin/caddy-manager"

# Run backend tests
backend-test:
	@echo "Running backend tests..."
	@go test -v ./backend/...

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
