.PHONY: help build up down logs ps shell test migrate backup restore clean prod prod-down prod-logs dashboard htpasswd

# Default target
help:
	@echo "Receipt Manager - Docker Commands with Traefik"
	@echo "==============================================="
	@echo ""
	@echo "Development:"
	@echo "  make build       - Build all Docker images"
	@echo "  make up          - Start all services with Traefik"
	@echo "  make down        - Stop all services"
	@echo "  make logs        - View logs (follow mode)"
	@echo "  make ps          - List running containers"
	@echo "  make dashboard   - Open Traefik dashboard"
	@echo "  make shell       - Open shell in API container"
	@echo "  make test        - Run backend tests"
	@echo "  make migrate     - Run database migrations"
	@echo ""
	@echo "Production:"
	@echo "  make prod        - Deploy to production with HTTPS"
	@echo "  make prod-down   - Stop production services"
	@echo "  make prod-logs   - View production logs"
	@echo "  make htpasswd    - Generate password hash for basic auth"
	@echo ""
	@echo "Maintenance:"
	@echo "  make backup      - Backup database and storage"
	@echo "  make restore     - Restore from backup"
	@echo "  make clean       - Remove all containers and volumes (⚠️ DELETES DATA)"
	@echo "  make update      - Update all images and restart"
	@echo ""
	@echo "Traefik Commands:"
	@echo "  make traefik-logs   - View Traefik logs"
	@echo "  make traefik-debug  - View Traefik debug info"

# Development commands
build:
	docker compose build

up:
	@echo "🚀 Starting Receipt Manager with Traefik..."
	@echo ""
	@echo "Services will be available at:"
	@echo "  - API:        http://localhost/api"
	@echo "  - Webhooks:   http://localhost/webhooks"
	@echo "  - Health:     http://localhost/health"
	@echo "  - Dashboard:  http://localhost:8080 (Traefik UI)"
	@echo "  - Temporal:   http://localhost/temporal"
	@echo ""
	@docker compose up -d
	@echo ""
	@echo "✅ Services started! Waiting for health checks..."
	@sleep 5
	@docker compose ps

down:
	docker compose down

logs:
	docker compose logs -f

ps:
	docker compose ps

shell:
	docker compose exec api /bin/sh

test:
	docker compose exec api go test ./...

migrate:
	docker compose exec postgres psql -U postgres -d receipt_manager -f /docker-entrypoint-initdb.d/migrate.sql

# Traefik specific commands
dashboard:
	@echo "Opening Traefik dashboard..."
	@echo "http://localhost:8080"
	@open http://localhost:8080 2>/dev/null || xdg-open http://localhost:8080 2>/dev/null || echo "Please open: http://localhost:8080"

traefik-logs:
	docker compose logs -f traefik

traefik-debug:
	@echo "Traefik Configuration:"
	@docker compose exec traefik cat /etc/traefik/traefik.yml 2>/dev/null || echo "Config not found"
	@echo ""
	@echo "Traefik Routers:"
	@curl -s http://localhost:8080/api/http/routers 2>/dev/null | head -50 || echo "Cannot connect to API"
	@echo ""
	@echo "Traefik Services:"
	@curl -s http://localhost:8080/api/http/services 2>/dev/null | head -50 || echo "Cannot connect to API"

# Production commands
prod:
	@echo "🚀 Deploying to PRODUCTION..."
	@echo ""
	@if [ -z "$(DOMAIN)" ]; then \
		echo "⚠️  Warning: DOMAIN not set. Using localhost."; \
		echo "   Set DOMAIN=your-domain.com for HTTPS."; \
		echo ""; \
	fi
	@if [ -z "$(ACME_EMAIL)" ]; then \
		echo "⚠️  Warning: ACME_EMAIL not set. Let's Encrypt certificates may fail."; \
		echo "   Set ACME_EMAIL=your-email@example.com"; \
		echo ""; \
	fi
	@docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
	@echo ""
	@echo "✅ Production services started!"
	@echo ""
	@echo "Your application should be available at:"
	@echo "  - https://$(DOMAIN)/api (if DOMAIN is set)"
	@echo ""
	@docker compose -f docker-compose.yml -f docker-compose.prod.yml ps

prod-down:
	docker compose -f docker-compose.yml -f docker-compose.prod.yml down

prod-logs:
	docker compose -f docker-compose.yml -f docker-compose.prod.yml logs -f

htpasswd:
	@read -p "Enter username: " user; \
	read -s -p "Enter password: " pass; \
	echo ""; \
	docker run --rm httpd:2-alpine htpasswd -nbB "$$user" "$$pass" | tee -a .htpasswd; \
	echo ""; \
	echo "Add this to traefik/dynamic/middlewares.yml under basic-auth users"

# Backup and restore
backup:
	@echo "💾 Creating backup..."
	@mkdir -p backups
	@docker exec receipt-manager-postgres pg_dump -U postgres receipt_manager > backups/postgres-$$(date +%Y%m%d-%H%M%S).sql
	@docker run --rm -v receipt-manager_rustfs_data:/data -v $$(pwd)/backups:/backup alpine tar czf /backup/rustfs-$$(date +%Y%m%d-%H%M%S).tar.gz -C /data .
	@echo "✅ Backup complete! Files in ./backups/"

restore:
	@echo "📋 Available backups:"
	@ls -la backups/
	@echo ""
	@echo "To restore PostgreSQL:"
	@echo "  docker exec -i receipt-manager-postgres psql -U postgres receipt_manager < backups/postgres-YYYYMMDD-HHMMSS.sql"
	@echo ""
	@echo "To restore RustFS:"
	@echo "  docker run --rm -v receipt-manager_rustfs_data:/data -v $$(pwd)/backups:/backup alpine sh -c 'cd /data && tar xzf /backup/rustfs-YYYYMMDD-HHMMSS.tar.gz'"

# Maintenance
clean:
	@echo "⚠️  WARNING: This will DELETE ALL DATA! ⚠️"
	@echo "Press Ctrl+C to cancel, or wait 5 seconds to continue..."
	@sleep 5
	docker compose down -v
	docker system prune -f

update:
	docker compose pull
	docker compose up -d --build

# Scale workers
scale-worker:
	@read -p "Number of workers: " n; \
	docker compose up -d --scale worker=$$n

# Health check
health:
	@echo "🔍 Checking service health..."
	@curl -s http://localhost/health && echo " ✅ API is healthy" || echo " ❌ API health check failed"
	@curl -s http://localhost:8080/ping && echo " ✅ Traefik is healthy" || echo " ❌ Traefik ping failed"

# Generate TLS cert info (production)
tls-info:
	@docker compose -f docker-compose.yml -f docker-compose.prod.yml exec traefik cat /letsencrypt/acme.json 2>/dev/null | head -20 || echo "No certificates yet"
