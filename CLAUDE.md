# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Language

Communicate in Russian (Русский язык).

## Tech Stack

- **Backend:** Go 1.23+, PostgreSQL 16, watermill-sql, gorilla, squirrel, pgxscan, goose
- **Frontend:** Vue 3, Vite, Tailwind 4, Pinia, Vue Router, Leaflet, maska
- **Architecture:** DDD, Event Sourcing, Event-Driven, 12-factor app

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
make run-telegram-bot # Run Telegram bot for link codes
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
make tidy             # Tidy go modules
go build ./...        # Quick compilation check
go test ./backend/internal/application/organization/...  # Run tests for specific package

# Code Generation
make generate         # Run go generate (enums via go-enum)

# Admin Tools
make create-admin          # Create platform admin (interactive)
make create-admin-dev      # Create dev admin (admin@veziizi.local / admin123)
make create-test-org       # Create test org (owner@test.local / test123)

# Geo & Seed Data
make seed-geo         # Seed countries and cities (runs automatically with dev/dev-all)
go run ./backend/cmd/tools/seed-orgs              # Seed test organizations
go run ./backend/cmd/tools/backfill-freight-requests  # Backfill freight requests projection
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

- **Organization** — организация с Members и Invitations. Любая организация может быть заказчиком и делать офферы
- **FreightRequest** — заявка на перевозку с Offers внутри. Два версионирования: `version` (aggregate) и `freightVersion` (только при изменении данных заявки)
- **Order** — заказ (после подтверждения оффера). Содержит Messages, Documents, Reviews. Создаётся автоматически через order-creator worker при OfferConfirmed
- **Review** — отдельный агрегат для защиты рейтингов от накрутки. Создаётся из Order.ReviewLeft через review-receiver worker. Проходит анализ на фрод и модерацию
- **Notification** — уведомления с настройками предпочтений (in-app, telegram). notification-dispatcher роутит доменные события на каналы, telegram-sender отправляет в Telegram. Не имеет aggregate.go, только events/ и values/

### Key Patterns

**Event Store** (`backend/internal/infrastructure/persistence/eventstore/`):
- Events implement `Event` interface, embed `BaseEvent`
- Register events via `RegisterEventType[T](eventType)` for deserialization
- `EventEnvelope` wraps events for storage with metadata
- Optimistic locking via UNIQUE constraint on `(aggregate_id, version)`

**Factory** (`backend/internal/pkg/factory/`):
- Lazy-initialized, thread-safe dependency container (sync.Once)
- Creates services: `OrganizationService()`, `AdminService()`, `FreightRequestService()`, `OrderService()`, `ReviewService()`, `HistoryService()`, `NotificationService()`
- Creates projections: `MembersProjection()`, `InvitationsProjection()`, `OrganizationsProjection()`, `FreightRequestsProjection()`, `OrdersProjection()`, `OrganizationRatingsProjection()`, `FraudDataProjection()`, `ReviewsProjection()`, `OrderFraudProjection()`, `SessionFraudProjection()`, `GeoProjection()`, `NotificationPreferencesProjection()`, `InAppNotificationsProjection()`, `DeliveryLogProjection()`
- Creates analyzers: `ReviewAnalyzer()`, `SessionAnalyzer()`
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
- `worker.Run(Config)` — event-driven workers (watermill subscribers)
- `worker.RunScheduled(ScheduledConfig)` — scheduled workers (ticker-based, e.g. review-activator)

**Worker Topics & Consumers:**
| Worker | Topic | ConsumerGroup | Purpose |
|--------|-------|---------------|---------|
| members | organization.events | members | Update members_lookup |
| invitations | organization.events | invitations | Update invitations_lookup |
| pending-organizations | organization.events | pending_organizations | Update pending_organizations |
| organizations | organization.events | organizations_projection | Update organizations_lookup |
| freight-requests | freightrequest.events | freight_requests | Update freight_requests_lookup, offers_lookup |
| orders | order.events | orders | Update orders_lookup |
| order-creator | freightrequest.events | order_creator | Create Order on OfferConfirmed |
| review-receiver | order.events | review_receiver | Create Review on ReviewLeft |
| review-analyzer | review.events | review_analyzer | Fraud detection, weight calculation |
| reviews-projection | review.events | reviews_projection | Update reviews_lookup, fraud_signals, interaction_stats, ratings |
| review-activator | scheduled (1 min) | - | Activate approved reviews after activation_date |
| fraudster-handler | organization.events | fraudster_handler | Deactivate reviews when org marked as fraudster |
| order-fraud-analyzer | order.events | order_fraud_analyzer | Detect order fraud: cancel patterns, ghost deliveries, circular orders |
| notification-dispatcher | *.events | notification_dispatcher | Route domain events to notification channels |
| telegram-sender | notification.send | telegram_sender | Send notifications via Telegram |

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

### Security

**Dev User Switcher** (только development):
- Защищён `APP_ENV=development` (роуты не регистрируются в production)
- Middleware `DevOnly` блокирует доступ если `APP_ENV=production`
- Фронтенд проверяет `import.meta.env.DEV` (false при `npm run build`)

**HTTP Middleware** (применяется в порядке):
1. `SecurityHeaders` — заголовки безопасности (CSP, X-Frame-Options, etc.)
2. `CORS` — настройка CORS
3. `BodyLimit` — лимит размера тела запроса
4. `RequireAuth` — проверка сессии (пропускает public paths)
5. `RateLimiter` — ограничение запросов + фрод-анализ сессий
6. `CSRFProtection` — проверка X-Requested-With header

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
    _ "codeberg.org/udison/veziizi/backend/internal/domain/review/events"
    _ "codeberg.org/udison/veziizi/backend/internal/domain/notification/events"
)
```
Без этого `eventstore.EventEnvelope.UnmarshalEvent()` вернёт ошибку "unknown event type".

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
- `adminGuard` — проверка admin сессии

**Permission System:**
- `usePermissions()` — composable для проверки прав
- `<PermissionGuard>` — компонент для условного рендеринга

## Project Status

Current: Phase 6 (Rating Fraud Protection) — Completed. Phase 8 (Frontend) — In Progress.
See `ROADMAP.md` for details.
