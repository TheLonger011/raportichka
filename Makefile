include .env
export

PROJECT_ROOT := $(shell pwd)
MIGRATIONS_PATH := migrations
DB_DSN := postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:5432/$(POSTGRES_DB)?sslmode=disable
DB_DSN_DOCKER := postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@raportichka-postgres:5432/$(POSTGRES_DB)?sslmode=disable

.PHONY: help
help:
	@echo "Available commands:"
	@echo "  make up                - Start all services"
	@echo "  make down              - Stop all services"
	@echo "  make restart           - Restart all services"
	@echo "  make logs              - View all logs"
	@echo "  make app-logs          - View app logs"
	@echo "  make db-logs           - View database logs"
	@echo "  make build             - Build images"
	@echo "  make rebuild           - Rebuild and start"
	@echo "  make migrate-create    - Create new migration"
	@echo "  make migrate-up        - Apply migrations locally"
	@echo "  make migrate-down      - Rollback last migration locally"
	@echo "  make migrate-docker-up - Apply migrations in Docker"
	@echo "  make db-reset          - Reset database (danger!)"
	@echo "  make seed              - Seed test data"
	@echo "  make psql              - Connect to PostgreSQL"
	@echo "  make app-shell         - Enter app container"
	@echo "  make clean             - Clean Docker resources"

# Docker compose commands
up:
	docker compose up -d
	@echo "Services started. App available at http://localhost:8800"

down:
	docker compose down

restart:
	docker compose restart

logs:
	docker compose logs -f

app-logs:
	docker compose logs -f raportichka-app

db-logs:
	docker compose logs -f raportichka-postgres

build:
	docker compose build

rebuild:
	docker compose down
	docker compose build --no-cache
	docker compose up -d

# Database migrations
migrate-create:
	@read -p "Enter migration name: " name; \
	migrate create -ext sql -dir $(MIGRATIONS_PATH) -seq $$name

migrate-up:
	migrate -path $(MIGRATIONS_PATH) -database "$(DB_DSN)" up

migrate-down:
	migrate -path $(MIGRATIONS_PATH) -database "$(DB_DSN)" down 1

migrate-force:
	@read -p "Enter version: " version; \
	migrate -path $(MIGRATIONS_PATH) -database "$(DB_DSN)" force $$version

migrate-version:
	migrate -path $(MIGRATIONS_PATH) -database "$(DB_DSN)" version

migrate-docker-up:
	docker exec -it raportichka-app sh -c "migrate -path /app/migrations -database '$(DB_DSN_DOCKER)' up"

migrate-docker-down:
	docker exec -it raportichka-app sh -c "migrate -path /app/migrations -database '$(DB_DSN_DOCKER)' down 1"

# Database reset and seed
db-reset:
	@echo "⚠️  This will delete all data! Continue? (y/N) " && read ans && [ $${ans:-N} = y ]
	docker compose down raportichka-postgres
	docker volume rm raportichka_postgres_data || true
	docker compose up -d raportichka-postgres
	@sleep 5
	make migrate-up
	make seed

seed:
	go run scripts/seed.go

# Utilities
psql:
	docker exec -it raportichka-postgres psql -U $(POSTGRES_USER) -d $(POSTGRES_DB)

app-shell:
	docker exec -it raportichka-app sh

db-shell:
	docker exec -it raportichka-postgres sh

# Cleanup
clean:
	docker compose down -v
	docker system prune -f

# Development
dev:
	@echo "Starting in development mode..."
	docker compose up --build

status:
	docker compose ps

# Backup
backup-db:
	@mkdir -p backups
	docker exec raportichka-postgres pg_dump -U $(POSTGRES_USER) $(POSTGRES_DB) > backups/backup_$$(date +%Y%m%d_%H%M%S).sql
	@echo "Backup created in backups/"

# Для локальной разработки (без Docker)
local-up:
	docker compose up -d raportichka-postgres
	@sleep 3
	make migrate-up
	go run cmd/server/main.go

local-stop:
	docker compose stop raportichka-postgres