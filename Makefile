.PHONY: help up down logs db-shell migrate migrate-down migrate-status migrate-create \
        build build-api build-workers run-api run-telegram-bot run-workers test lint fmt env-init generate \
        back-dev create-admin create-admin-dev dev-all dev-setup create-test-org seed-geo

# Load .env file if exists
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

# Default values
POSTGRES_USER ?= veziizi
POSTGRES_PASSWORD ?= veziizi
POSTGRES_DB ?= veziizi
POSTGRES_HOST ?= localhost
POSTGRES_PORT ?= 5432
DATABASE_URL ?= postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Docker
up: ## Start all services
	docker compose up -d

down: ## Stop all services
	docker compose down

logs: ## Show logs
	docker compose logs -f

db-shell: ## Connect to PostgreSQL shell
	docker compose exec postgres psql -U $(POSTGRES_USER) -d $(POSTGRES_DB)

# Migrations (goose)
migrate: ## Run migrations up
	goose -dir backend/migrations postgres "$(DATABASE_URL)" up

migrate-down: ## Run migrations down (one step)
	goose -dir backend/migrations postgres "$(DATABASE_URL)" down

migrate-status: ## Show migration status
	goose -dir backend/migrations postgres "$(DATABASE_URL)" status

migrate-create: ## Create new migration (use: make migrate-create name=create_users)
	@if [ -z "$(name)" ]; then echo "Usage: make migrate-create name=migration_name"; exit 1; fi
	goose -dir backend/migrations create $(name) sql

# Build
build: ## Build all binaries
	go build -o bin/api ./backend/cmd/api
	go build -o bin/telegram-bot ./backend/cmd/telegram-bot
	go build -o bin/migrator ./backend/cmd/migrator
	go build -o bin/worker-members ./backend/cmd/workers/members
	go build -o bin/worker-invitations ./backend/cmd/workers/invitations
	go build -o bin/worker-pending-organizations ./backend/cmd/workers/pending-organizations
	go build -o bin/worker-organizations ./backend/cmd/workers/organizations
	go build -o bin/worker-freight-requests ./backend/cmd/workers/freight-requests
	go build -o bin/worker-orders ./backend/cmd/workers/orders
	go build -o bin/worker-order-creator ./backend/cmd/workers/order-creator
	go build -o bin/worker-review-receiver ./backend/cmd/workers/review-receiver
	go build -o bin/worker-review-analyzer ./backend/cmd/workers/review-analyzer
	go build -o bin/worker-reviews-projection ./backend/cmd/workers/reviews-projection
	go build -o bin/worker-review-activator ./backend/cmd/workers/review-activator
	go build -o bin/worker-fraudster-handler ./backend/cmd/workers/fraudster-handler
	go build -o bin/worker-order-fraud-analyzer ./backend/cmd/workers/order-fraud-analyzer
	go build -o bin/worker-notification-dispatcher ./backend/cmd/workers/notification-dispatcher
	go build -o bin/worker-telegram-sender ./backend/cmd/workers/telegram-sender

build-api: ## Build API server
	go build -o bin/api ./backend/cmd/api

build-workers: ## Build all workers
	go build -o bin/worker-members ./backend/cmd/workers/members
	go build -o bin/worker-invitations ./backend/cmd/workers/invitations
	go build -o bin/worker-pending-organizations ./backend/cmd/workers/pending-organizations
	go build -o bin/worker-organizations ./backend/cmd/workers/organizations
	go build -o bin/worker-freight-requests ./backend/cmd/workers/freight-requests
	go build -o bin/worker-orders ./backend/cmd/workers/orders
	go build -o bin/worker-order-creator ./backend/cmd/workers/order-creator
	go build -o bin/worker-review-receiver ./backend/cmd/workers/review-receiver
	go build -o bin/worker-review-analyzer ./backend/cmd/workers/review-analyzer
	go build -o bin/worker-reviews-projection ./backend/cmd/workers/reviews-projection
	go build -o bin/worker-review-activator ./backend/cmd/workers/review-activator
	go build -o bin/worker-fraudster-handler ./backend/cmd/workers/fraudster-handler
	go build -o bin/worker-order-fraud-analyzer ./backend/cmd/workers/order-fraud-analyzer
	go build -o bin/worker-notification-dispatcher ./backend/cmd/workers/notification-dispatcher
	go build -o bin/worker-telegram-sender ./backend/cmd/workers/telegram-sender

# Run
run-api: ## Run API server
	go run ./backend/cmd/api

run-telegram-bot: ## Run Telegram bot for link code handling
	go run ./backend/cmd/telegram-bot

run-workers: ## Run all workers
	go run ./backend/cmd/workers/members &
	go run ./backend/cmd/workers/invitations &
	go run ./backend/cmd/workers/pending-organizations &
	go run ./backend/cmd/workers/organizations &
	go run ./backend/cmd/workers/freight-requests &
	go run ./backend/cmd/workers/orders &
	go run ./backend/cmd/workers/order-creator &
	go run ./backend/cmd/workers/review-receiver &
	go run ./backend/cmd/workers/review-analyzer &
	go run ./backend/cmd/workers/reviews-projection &
	go run ./backend/cmd/workers/review-activator &
	go run ./backend/cmd/workers/fraudster-handler &
	go run ./backend/cmd/workers/order-fraud-analyzer &
	go run ./backend/cmd/workers/notification-dispatcher &
	go run ./backend/cmd/workers/telegram-sender &

# Development
test: ## Run tests
	go test -v ./...

test-cover: ## Run tests with coverage
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint: ## Run linter
	golangci-lint run ./...

fmt: ## Format code
	go fmt ./...
	goimports -w .

tidy: ## Tidy go modules
	go mod tidy

generate: ## Run go generate
	go generate ./...

# Environment
env-init: ## Create .env from .env.example if not exists
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo ".env created from .env.example - please edit it with your values"; \
	else \
		echo ".env already exists"; \
	fi

check-env: ## Check if .env exists
	@if [ ! -f .env ]; then \
		echo "Error: .env file not found. Run 'make env-init' first"; \
		exit 1; \
	fi

# All-in-one development
dev-setup: ## Install dev tools (goreman, air)
	@command -v goreman >/dev/null 2>&1 || go install github.com/mattn/goreman@latest
	@command -v air >/dev/null 2>&1 || go install github.com/air-verse/air@latest
	@echo "Dev tools ready"

dev-all: check-env dev-setup up ## Start everything with hot-reload (API + workers + frontend)
	@echo "Waiting for PostgreSQL..."
	@until docker exec veziizi-postgres pg_isready -U $(POSTGRES_USER) >/dev/null 2>&1; do sleep 1; done
	@echo "PostgreSQL ready, running migrations..."
	@$(MAKE) migrate
	@echo "Seeding geo data (countries and cities)..."
	@$(MAKE) seed-geo
	@echo "Starting all services..."
	goreman -f Procfile.dev start

dev: check-env up migrate seed-geo run-api ## Start DB + API only (without workers/frontend)

# Development with hot-reload
back-dev: check-env ## Run backend with air (hot-reload)
	air

# Admin management
create-admin: check-env ## Create platform admin (interactive)
	@read -p "Email: " email; \
	read -p "Name: " name; \
	read -s -p "Password: " password; echo; \
	go run ./backend/cmd/tools/create-admin \
		--email="$$email" \
		--name="$$name" \
		--password="$$password"

create-admin-dev: check-env ## Create dev admin (admin@veziizi.local / admin123)
	go run ./backend/cmd/tools/create-admin \
		--email="admin@veziizi.local" \
		--name="Admin" \
		--password="admin123"

create-test-org: check-env ## Create test org with owner (owner@test.local / test123)
	go run ./backend/cmd/tools/create-test-org \
		--email="owner@test.local" \
		--password="test123" \
		--name="Test Owner" \
		--org="Test Organization" \
		--approve=true

seed-geo: check-env ## Seed geo data (countries and cities)
	go run ./backend/cmd/tools/seed-geo
