.PHONY: help build up down restart logs logs-follow status clean rebuild validate env-check

# Default target
help:
	@echo "Available commands:"
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
	@echo "  make env-check     - Check if .env file exists and is configured"
	@echo "  make deploy        - Full deployment (env-check, build, up)"

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

# Build the Docker image
build:
	@echo "Building Caddy with Cloudflare DNS plugin..."
	docker compose build

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
