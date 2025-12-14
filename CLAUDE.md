# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Language

Communicate in Russian (Русский язык).

## Commands

```bash
# Development
make dev              # Start PostgreSQL, run migrations, start API server
make run-api          # Run API server only
make up / make down   # Start/stop Docker services

# Database
make migrate                         # Run migrations up
make migrate-down                    # Rollback one migration
make migrate-create name=foo         # Create new migration
make db-shell                        # Connect to PostgreSQL

# Build & Test
make build            # Build all binaries to bin/
make build-api        # Build API only
make test             # Run tests
make lint             # Run golangci-lint
go build ./...        # Quick compilation check
```

## Architecture

Логистическая платформа с Event Sourcing и DDD. Монорепо: `/backend` (Go), `/frontend` (Vue.js - планируется).

### Domain Aggregates

- **Organization** — организация с Members и Invitations. Может быть customer и/или carrier (через CarrierProfile)
- **FreightRequest** — заявка на перевозку с Offers внутри. Два версионирования: `version` (aggregate) и `freightVersion` (только при изменении данных заявки)
- **Order** — заказ (после подтверждения оффера). Содержит Messages, Documents, Reviews

### Key Patterns

**Event Store** (`backend/internal/infrastructure/persistence/eventstore/`):
- Events implement `Event` interface, embed `BaseEvent`
- Register events via `RegisterEventType[T](eventType)` for deserialization
- `EventEnvelope` wraps events for storage with metadata
- Optimistic locking via UNIQUE constraint on `(aggregate_id, version)`
- Snapshots every N events (configurable)

**Transaction Propagation** (`backend/internal/pkg/dbtx/`):
- `TxExecutor.InTx(ctx, fn)` — creates tx or savepoint if already in tx
- `dbtx.FromCtx(ctx)` — get tx from context
- All repositories use `TxManager` interface, auto-detect tx in context

**Watermill Publisher** (`backend/internal/infrastructure/messaging/`):
- Uses `sql.BeginnerFromPgx(pool)` for default publisher
- Uses `sql.TxFromPgx(tx)` when tx in context (atomic with event store)

### Code Style

- Error wrapping: `fmt.Errorf("context: %w", err)`
- Single error return: `if err := ...; err != nil`
- Use `any` instead of `interface{}`
- Use `for range N` instead of `for i := 0; i < N; i++`
- **Never ignore errors** — at minimum log them with `slog.Error()`
- Logging: use `slog` directly (configured in main), never pass logger as dependency. Logs go to `current.log`
- **Always use latest library versions** — search GitHub/GitLab tags for Go libraries to find current versions

## Project Status

See `ROADMAP.md` for current phase and task status.
