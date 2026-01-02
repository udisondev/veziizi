.PHONY: help up down logs db-shell migrate migrate-down migrate-status migrate-create \
        build build-api build-telegram-bot build-workers run-api run-telegram-bot run-workers \
        test test-cover test-e2e test-e2e-setup test-e2e-parallel test-e2e-containers \
        lint fmt tidy generate env-init check-env dev dev-setup dev-all back-dev \
        create-admin create-admin-dev create-test-org \
        seed-geo seed-orgs resend-telegram backfill-freight-requests

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

# Workers list
WORKERS := members invitations pending-organizations organizations \
           freight-requests orders order-creator review-receiver \
           review-analyzer reviews-projection review-activator \
           fraudster-handler order-fraud-analyzer notification-dispatcher \
           telegram-sender support-tickets rate-limiter-cleanup

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
build: build-api build-telegram-bot build-workers ## Build all binaries

build-api: ## Build API server
	go build -o bin/api ./backend/cmd/api

build-telegram-bot: ## Build Telegram bot
	go build -o bin/telegram-bot ./backend/cmd/telegram-bot

build-workers: ## Build all workers
	@for worker in $(WORKERS); do \
		go build -o bin/worker-$$worker ./backend/cmd/workers/$$worker; \
	done

# Run
run-api: ## Run API server
	go run ./backend/cmd/api

run-telegram-bot: ## Run Telegram bot for link code handling
	go run ./backend/cmd/telegram-bot

run-workers: ## Run all workers in background
	@for worker in $(WORKERS); do \
		go run ./backend/cmd/workers/$$worker & \
	done
	@echo "All workers started in background"

# Testing
test: ## Run unit tests
	go test -v ./backend/internal/...

test-cover: ## Run tests with coverage
	go test -v -coverprofile=coverage.out ./backend/internal/...
	go tool cover -html=coverage.out -o coverage.html

# E2E Tests
TEST_DATABASE_URL ?= postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/veziizi_test?sslmode=disable

test-e2e-setup: up ## Setup E2E test database
	@echo "Creating test database..."
	@docker exec veziizi-postgres psql -U $(POSTGRES_USER) -c "DROP DATABASE IF EXISTS veziizi_test" || true
	@docker exec veziizi-postgres psql -U $(POSTGRES_USER) -c "CREATE DATABASE veziizi_test"
	@echo "Running migrations on test database..."
	@goose -dir backend/migrations postgres "$(TEST_DATABASE_URL)" up
	@echo "Seeding geo data..."
	@TEST_DATABASE_URL="$(TEST_DATABASE_URL)" go run ./backend/cmd/tools/seed-geo
	@echo "Creating test admin..."
	@DATABASE_URL="$(TEST_DATABASE_URL)" go run ./backend/cmd/tools/create-admin \
		--email="admin@veziizi.local" \
		--name="Admin" \
		--password="admin123" || true
	@echo "E2E test database ready"

test-e2e: test-e2e-setup ## Run E2E tests (uses docker-compose DB)
	TEST_DATABASE_URL="$(TEST_DATABASE_URL)" go test -v -count=1 -p=1 ./backend/e2e/tests/...

test-e2e-parallel: test-e2e-setup ## Run E2E tests in parallel (uses docker-compose DB)
	TEST_DATABASE_URL="$(TEST_DATABASE_URL)" go test -v -count=1 ./backend/e2e/tests/...

test-e2e-containers: ## Run E2E tests with testcontainers (requires Docker)
	go test -v -count=1 -p=1 ./backend/e2e/tests/...

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

dev-all: check-env dev-setup up build-workers ## Start everything with hot-reload (API + workers + frontend)
	@echo "Waiting for PostgreSQL..."
	@until docker exec veziizi-postgres pg_isready -U $(POSTGRES_USER) >/dev/null 2>&1; do sleep 1; done
	@echo "PostgreSQL ready, running migrations..."
	@$(MAKE) migrate
	@echo "Seeding geo data..."
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

seed-orgs: check-env ## Seed test organizations
	go run ./backend/cmd/tools/seed-orgs

resend-telegram: check-env ## Resend failed telegram notifications
	go run ./backend/cmd/tools/resend-telegram

backfill-freight-requests: check-env ## Backfill freight requests projection
	go run ./backend/cmd/tools/backfill-freight-requests
