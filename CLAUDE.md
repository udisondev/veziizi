# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Language

Communicate in Russian (Русский язык).

## Module Path

`codeberg.org/udison/veziizi` — все импорты начинаются с этого пути.

## Prerequisites

Необходимые инструменты (устанавливаются через `make dev-setup`):
- `goreman` — запуск всех сервисов из Procfile.dev
- `air` — hot-reload для Go
- `goose` — миграции БД
- `golangci-lint` — линтер

## Commands

```bash
# Development
make dev              # Start PostgreSQL, run migrations, start API server (all-in-one)
make dev-all          # Full stack with hot-reload: API + all workers (uses goreman)
make run-api          # Run API server only
make run-workers      # Run all workers in background
make run-telegram     # Run Telegram notifier
make back-dev         # Run API with air (hot-reload)
make up / make down   # Start/stop Docker services

# Database
make migrate                         # Run migrations up
make migrate-down                    # Rollback one migration
make migrate-create name=foo         # Create new migration
make db-shell                        # Connect to PostgreSQL

# Build & Test
make build            # Build all binaries to bin/ (api, workers, telegram-notifier, migrator)
make build-api        # Build API only
make build-workers    # Build all workers
make test             # Run tests
make test-cover       # Run tests with coverage
make lint             # Run golangci-lint
go build ./...        # Quick compilation check
go test ./backend/internal/application/organization/...  # Run tests for specific package

# Code Generation
make generate         # Run go generate (enums via go-enum)

# Admin Tools
make create-admin          # Create platform admin (interactive)
make create-admin-dev      # Create dev admin (admin@veziizi.local / admin123)
make create-test-org       # Create test org (owner@test.local / test123)
```

## Environment

Скопировать `.env.example` в `.env` перед запуском (или `make env-init`):
```
DATABASE_URL=postgres://veziizi:veziizi@localhost:5432/veziizi?sslmode=disable
SESSION_KEY=32-byte-key-for-sessions
ADMIN_SESSION_KEY=32-byte-key-for-admin-sessions
```

## Architecture

Логистическая платформа с Event Sourcing и DDD. Монорепо: `/backend` (Go), `/frontend` (Vue.js).

### Domain Aggregates

- **Organization** — организация с Members и Invitations. Может быть customer и/или carrier (через CarrierProfile)
- **FreightRequest** — заявка на перевозку с Offers внутри. Два версионирования: `version` (aggregate) и `freightVersion` (только при изменении данных заявки)
- **Order** — заказ (после подтверждения оффера). Содержит Messages, Documents, Reviews. Создаётся автоматически через order-creator worker при OfferConfirmed

### Key Patterns

**Event Store** (`backend/internal/infrastructure/persistence/eventstore/`):
- Events implement `Event` interface, embed `BaseEvent`
- Register events via `RegisterEventType[T](eventType)` for deserialization
- `EventEnvelope` wraps events for storage with metadata
- Optimistic locking via UNIQUE constraint on `(aggregate_id, version)`

**Factory** (`backend/internal/pkg/factory/`):
- Lazy-initialized, thread-safe dependency container (sync.Once)
- Creates services: `OrganizationService()`, `AdminService()`, `FreightRequestService()`, `OrderService()`
- Creates projections: `MembersProjection()`, `InvitationsProjection()`, `FreightRequestsProjection()`, `OrdersProjection()`
- Used by both API and workers

**Transaction Propagation** (`backend/internal/pkg/dbtx/`):
- `TxExecutor.InTx(ctx, fn)` — creates tx or savepoint if already in tx
- `dbtx.FromCtx(ctx)` — get tx from context
- All repositories use `TxManager` interface, auto-detect tx in context

**Watermill Publisher** (`backend/internal/infrastructure/messaging/`):
- Uses `sql.BeginnerFromPgx(pool)` for default publisher
- Uses `sql.TxFromPgx(tx)` when tx in context (atomic with event store)

**Worker Package** (`backend/internal/pkg/worker/`):
- Boilerplate for async watermill subscribers
- Each worker runs as separate process (`backend/cmd/workers/*`)
- Handler receives `*factory.Factory` for dependency access
- Each worker has its own ConsumerGroup for independent offset tracking

**Worker Topics & Consumers:**
| Worker | Topic | ConsumerGroup | Purpose |
|--------|-------|---------------|---------|
| members | organization.events | members | Update members_lookup |
| invitations | organization.events | invitations | Update invitations_lookup |
| pending-organizations | organization.events | pending_organizations | Update pending_organizations |
| freight-requests | freightrequest.events | freight_requests | Update freight_requests_lookup, offers_lookup |
| orders | order.events | orders | Update orders_lookup |
| order-creator | freightrequest.events | order_creator | Create Order on OfferConfirmed |

**Event Imports per Worker:**
- `members`, `invitations`, `pending-organizations`: `organization/events`
- `freight-requests`: `freightrequest/events`
- `orders`: `order/events`
- `order-creator`: `freightrequest/events`, `order/events` (слушает freightrequest, создаёт order)

**Lookup Tables (Projections)**:
- Store only ID + filter columns (status, org_id, etc.), no JSONB
- Full data loaded from event store via service.Get() when needed
- Examples: `freight_requests_lookup`, `offers_lookup`, `orders_lookup`

### Code Style

- Error wrapping: `fmt.Errorf("context: %w", err)`
- Single error return: `if err := ...; err != nil`
- Use `any` instead of `interface{}`
- Use `for range N` instead of `for i := 0; i < N; i++`
- **Never ignore errors** — at minimum log them with `slog.Error()`
- Logging: use `slog` directly (configured in main), never pass logger as dependency. Logs go to `current.log`
- **Always use latest library versions** — search GitHub/GitLab tags for Go libraries to find current versions

### Adding New Components

**New Event:**
1. Add event type constant in `domain/<name>/events/events.go`
2. Create event struct with `BaseEvent` embed
3. Add `RegisterEventType[T]` in `init()`
4. Implement `apply()` in aggregate

**New Worker:**
1. Create `backend/cmd/workers/<name>/main.go`
2. **CRITICAL:** Import events package for registration: `_ ".../<domain>/events"` (без этого десериализация не работает!)
3. Add Handler in `infrastructure/handlers/`
4. Add build command in `Makefile`
5. Add to `make run-workers` and `Procfile.dev`

**New Aggregate:**
1. `domain/<name>/aggregate.go` — struct + `New()`, `NewFromEvents()`
2. `domain/<name>/events/` — events with `init()` registration
3. `domain/<name>/values/` — value objects (optional)
4. `domain/<name>/entities/` — entities inside aggregate (optional)
5. `application/<name>/service.go` — use cases
6. `infrastructure/projections/<name>.go` — lookup table
7. `interfaces/http/handlers/<name>.go` — HTTP handlers
8. `migrations/` — lookup table migration
9. `pkg/factory/` — add service and projection getters

### Event Registration (ВАЖНО!)

Workers **должны** импортировать events packages через blank import для регистрации типов событий:
```go
import (
    _ "codeberg.org/udison/veziizi/backend/internal/domain/organization/events"
    _ "codeberg.org/udison/veziizi/backend/internal/domain/freightrequest/events"
    _ "codeberg.org/udison/veziizi/backend/internal/domain/order/events"
)
```
Без этого `eventstore.EventEnvelope.UnmarshalEvent()` вернёт ошибку "unknown event type".

### Project Structure

```
backend/
├── cmd/
│   ├── api/              # HTTP API server
│   ├── tools/            # CLI utilities (create-admin, create-test-org)
│   └── workers/          # Async event handlers
│       ├── members/
│       ├── invitations/
│       ├── pending-organizations/
│       ├── freight-requests/
│       ├── orders/
│       └── order-creator/
├── internal/
│   ├── application/      # Application services (use cases)
│   │   ├── organization/
│   │   ├── admin/
│   │   ├── freightrequest/
│   │   └── order/
│   ├── domain/           # Aggregates, entities, events, value objects
│   │   ├── organization/
│   │   ├── freightrequest/
│   │   └── order/
│   ├── infrastructure/
│   │   ├── handlers/     # Watermill event handlers (write side)
│   │   ├── messaging/    # Watermill publisher
│   │   ├── persistence/  # Event store, repositories, file storage (DB/S3)
│   │   └── projections/  # Read models (lookup tables)
│   ├── interfaces/http/  # HTTP handlers, session, server
│   └── pkg/              # Shared packages (aggregate, config, dbtx, factory, worker)
└── migrations/           # Goose SQL migrations

frontend/
├── src/
│   ├── api/              # API client (fetch wrapper)
│   ├── components/       # Vue components
│   │   ├── freight-request/  # Wizard steps, shared components
│   │   └── ui/           # Reusable UI (AppHeader, PermissionGuard)
│   ├── composables/      # Vue composables (usePermissions, useAddressSearch)
│   ├── router/           # Vue Router + guards (auth, orgActive, role, carrier, admin)
│   ├── stores/           # Pinia stores (auth, admin)
│   ├── types/            # TypeScript interfaces
│   └── views/            # Page components
│       └── admin/        # Admin panel views
└── package.json
```

## Frontend

**Stack:** Vue 3, Vite, Tailwind 4, Pinia, Vue Router, Leaflet (карты), maska (маски ввода)

```bash
cd frontend
npm install           # Install dependencies
npm run dev           # Dev server (http://localhost:5173)
npm run build         # Production build
```

**Route Guards** (`router/guards.ts`):
- `authGuard` — проверка авторизации (глобальный)
- `orgActiveGuard` — редирект на статусные страницы если org не active (глобальный)
- `roleGuard(['owner', 'administrator'])` — проверка роли
- `carrierGuard` — проверка CarrierProfile
- `adminGuard` — проверка admin сессии

**Permission System:**
- `usePermissions()` — composable для проверки прав
- `<PermissionGuard>` — компонент для условного рендеринга

## Project Status

See `ROADMAP.md` for current phase and task status.
