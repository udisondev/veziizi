.PHONY: help up down logs db-shell migrate migrate-down migrate-status migrate-create \
        build build-api build-workers run-api run-telegram run-workers test lint fmt env-init generate

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
	go build -o bin/telegram-notifier ./backend/cmd/telegram-notifier
	go build -o bin/migrator ./backend/cmd/migrator
	go build -o bin/worker-members ./backend/cmd/workers/members
	go build -o bin/worker-invitations ./backend/cmd/workers/invitations
	go build -o bin/worker-pending-organizations ./backend/cmd/workers/pending-organizations

build-api: ## Build API server
	go build -o bin/api ./backend/cmd/api

build-workers: ## Build all workers
	go build -o bin/worker-members ./backend/cmd/workers/members
	go build -o bin/worker-invitations ./backend/cmd/workers/invitations
	go build -o bin/worker-pending-organizations ./backend/cmd/workers/pending-organizations

# Run
run-api: ## Run API server
	go run ./backend/cmd/api

run-telegram: ## Run Telegram notifier
	go run ./backend/cmd/telegram-notifier

run-workers: ## Run all workers
	go run ./backend/cmd/workers/members &
	go run ./backend/cmd/workers/invitations &
	go run ./backend/cmd/workers/pending-organizations &

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

# All-in-one
dev: check-env up migrate run-api ## Start all services and run API
