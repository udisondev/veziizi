# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Language

Communicate in Russian (Русский язык).

## Tech Stack

- **Backend:** Go 1.23+, PostgreSQL 16, watermill-sql, gorilla, squirrel, pgxscan, goose
- **Frontend:** Vue 3, Vite, Tailwind 4, Pinia, Vue Router, Leaflet, maska
- **Architecture:** DDD, Event Sourcing, Event-Driven, 12-factor app

## Module Path

`github.com/udisondev/veziizi` — все импорты начинаются с этого пути.

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
make test             # Run unit tests
make test-cover       # Run tests with coverage
make test-e2e         # Run E2E tests (setup + sequential, uses docker-compose DB)
make test-e2e-parallel # Run E2E tests in parallel
make test-e2e-containers # Run E2E tests with testcontainers (auto-creates DB)
make lint             # Run golangci-lint
make tidy             # Tidy go modules
go build ./...        # Quick compilation check
go test ./backend/internal/application/organization/...  # Run tests for specific package
go test -v -count=1 ./backend/e2e/tests/...              # Run specific E2E test

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
SESSION_SECRET=32-byte-key-for-sessions
SESSION_ADMIN_SECRET=32-byte-key-for-admin-sessions
TELEGRAM_BOT_TOKEN=your-bot-token  # Required for telegram-bot and telegram-sender
APP_ENV=development                # development | production
```

## Architecture

Логистическая платформа с Event Sourcing и DDD. Монорепо: `/backend` (Go), `/frontend` (Vue.js).

### Domain Aggregates

- **Organization** — организация с Members и Invitations. Любая организация может быть заказчиком и делать офферы
- **FreightRequest** — заявка на перевозку с Offers внутри. После OfferConfirmed переходит в статус confirmed и содержит carrier info. Обе стороны могут явно завершить (Complete) и оставить отзыв (LeaveReview). Статусы: `published → selected → confirmed → partially_completed → completed`
- **Review** — отдельный агрегат для защиты рейтингов от накрутки. Создаётся из FreightRequest.ReviewLeft через review-receiver worker. Проходит анализ на фрод и модерацию
- **Notification** — уведомления с настройками предпочтений (in-app, telegram). notification-dispatcher роутит доменные события на каналы, telegram-sender отправляет в Telegram. Не имеет aggregate.go, только events/ и values/
- **Support** — тикеты поддержки. Агрегат SupportTicket для обращений пользователей

### Key Patterns

**Event Store** (`backend/internal/infrastructure/persistence/eventstore/`):
- Events implement `Event` interface, embed `BaseEvent`
- Register events via `RegisterEventType[T](eventType)` for deserialization
- `EventEnvelope` wraps events for storage with metadata
- Optimistic locking via UNIQUE constraint on `(aggregate_id, version)`
- Errors: `ErrAggregateNotFound`, `ErrConcurrentModification`, `ErrEventVersionConflict`

**Factory** (`backend/internal/pkg/factory/`):
- Lazy-initialized, thread-safe dependency container (sync.Once)
- Creates services: `OrganizationService()`, `AdminService()`, `FreightRequestService()`, `ReviewService()`, `HistoryService()`, `NotificationService()`, `SupportService()`
- Creates projections: `MembersProjection()`, `InvitationsProjection()`, `OrganizationsProjection()`, `FreightRequestsProjection()`, `OrganizationRatingsProjection()`, `FraudDataProjection()`, `ReviewsProjection()`, `SessionFraudProjection()`, `GeoProjection()`, `NotificationPreferencesProjection()`, `InAppNotificationsProjection()`, `DeliveryLogProjection()`, `TelegramLinkProjection()`, `FreightSubscriptionsProjection()`, `SupportTicketsProjection()`
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
| review-receiver | freightrequest.events | review_receiver | Create Review on ReviewLeft |
| review-analyzer | review.events | review_analyzer | Fraud detection, weight calculation |
| reviews-projection | review.events | reviews_projection | Update reviews_lookup, fraud_signals, interaction_stats, ratings |
| review-activator | scheduled (1 min) | - | Activate approved reviews after activation_date |
| fraudster-handler | organization.events | fraudster_handler | Deactivate reviews when org marked as fraudster |
| notification-dispatcher | freightrequest.events | notification_dispatcher | Route domain events to notification channels via rules |
| telegram-sender | notification.send | telegram_sender | Send notifications via Telegram |
| support-tickets | support.events | support_tickets | Update support_tickets_lookup |
| rate-limiter-cleanup | scheduled (5 min) | - | Clean up expired rate limiter entries |
| telegram-bot | - | - | Telegram bot for link codes (separate cmd, not a worker) |

**Notification Rules** (`backend/internal/domain/notification/rules/`):
- Rules convert domain events to notifications (in-app, telegram)
- Each rule implements `Rule` interface: `EventType()`, `Handle(event, deps)`
- Rules register in `registry.go` via `RegisterRule()`
- Adapters (`adapters.go`) provide access to projections for context (e.g., member names)

**Lookup Tables (Projections)**:
- Store only ID + filter columns (status, org_id, etc.), no JSONB
- Full data loaded from event store via service.Get() when needed
- Examples: `freight_requests_lookup`, `offers_lookup`

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

**New Notification Rule:**
1. Create `domain/notification/rules/<domain>/<event_name>.go`
2. Implement `Rule` interface: `EventType() string`, `Handle(event, deps) error`
3. Register in `rules/registry.go` via `RegisterRule(&MyRule{})`
4. Use `deps` adapters to fetch context (member names, org info)

### Event Registration (ВАЖНО!)

Workers **должны** импортировать events packages через blank import для регистрации типов событий:
```go
import (
    _ "github.com/udisondev/veziizi/backend/internal/domain/organization/events"
    _ "github.com/udisondev/veziizi/backend/internal/domain/freightrequest/events"
    _ "github.com/udisondev/veziizi/backend/internal/domain/review/events"
    _ "github.com/udisondev/veziizi/backend/internal/domain/notification/events"
    _ "github.com/udisondev/veziizi/backend/internal/domain/support/events"
)
```
Без этого `eventstore.EventEnvelope.UnmarshalEvent()` вернёт ошибку "unknown event type".

## Frontend

**Stack:** Vue 3, Vite, Tailwind 4, Pinia, Vue Router, Leaflet (карты), maska (маски ввода)

```bash
cd frontend
npm install           # Install dependencies
npm run dev           # Dev server (http://localhost:5173)
npm run build         # Production build (vue-tsc type check + vite build)
npm run preview       # Preview production build locally
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

Current: Phase 8 (Frontend) — In Progress. Phases 1-6 completed, Phase 7 (Notifications) completed.
See `ROADMAP.md` for details.

## AI Team — Роли и команды

Проект использует систему AI-агентов с разными ролями. Каждая роль имеет свою специализацию.
Контекст сохраняется в **Beads** (задачи) и **Agent Mail** (обсуждения).

### Быстрый старт

```bash
# Терминал 1: Запустить Agent Mail сервер
am

# Терминал 2: Запустить Claude Code и начать с /analyst
cd /path/to/veziizi
claude
> /analyst
```

### Команды ролей

При вызове команды роли:
1. Зарегистрируйся в Agent Mail через `macro_start_session` с указанным именем агента
2. Проверь inbox — прочитай сообщения от других ролей
3. Проверь задачи через `br ready --label "role:<my_role>"` и `br list --label "role:<my_role>"`
4. Работай в рамках своей роли
5. Перед редактированием файлов — зарезервируй их через `file_reservation_paths`
6. После завершения — передай задачу следующей роли и освободи резервы

---

### Task Labels и передача задач

**Одна Feature задача проходит через все этапы:**
```
backend → review → frontend → review → test → closed
```

**При взятии задачи:**
```bash
br update bd-xxx --assignee <role> --status in_progress
```

**При передаче следующей роли:**
```bash
br update bd-xxx --assignee <next_role> --set-labels "role:<next_role>"
```

**Labels для ролей:**
| Label | Роль |
|-------|------|
| `role:analyst` | Аналитик |
| `role:architect` | Архитектор |
| `role:security` | Security |
| `role:lead` | Тим-лид |
| `role:devops` | DevOps |
| `role:backend` | Backend разработчик |
| `role:frontend` | Frontend разработчик |
| `role:review` | Код-ревьюер |
| `role:test` | Тестировщик |

**Фильтрация своих задач:**
```bash
br ready --label "role:backend"   # готовые для backend
br list --label "role:backend"    # все backend задачи
```

**Мониторинг (для lead):**
```bash
br list --status in_progress --long  # кто над чем работает
br list --label "role:review"        # задачи на ревью
```

---

### /analyst
**Агент:** AnalystOwl  
**Роль:** Бизнес-аналитик

Ты — бизнес-аналитик проекта Veziizi (B2B платформа грузоперевозок).

**Обязанности:**
- Собирать и уточнять требования от пользователя
- Декомпозировать фичи на понятные задачи
- Создавать эпики в Beads (`br create --title "..." --priority high`)
- Передавать задачи архитектору через Agent Mail

**При старте:**
```
macro_start_session:
  agent_name: "AnalystOwl"
  task_description: "Business analysis, requirements gathering"
```

**После анализа:**
- Создай задачу: `br create --title "Название фичи" --body "Описание требований" --priority high`
- Отправь архитектору: `send_message` to ArchitectEagle с требованиями

---

### /architect
**Агент:** ArchitectEagle  
**Роль:** Архитектор

Ты — архитектор проекта Veziizi. Знаешь DDD, Event Sourcing, Go, Vue.

**Обязанности:**
- Проектировать техническое решение
- Выбирать подходы и паттерны (учитывая существующую архитектуру)
- Определять новые агрегаты, события, проекции
- Оценивать влияние на существующий код
- При необходимости запрашивать security review

**При старте:**
```
macro_start_session:
  agent_name: "ArchitectEagle"
  task_description: "Architecture design, technical decisions"
```

**После проектирования:**
- Обнови задачу с техническим описанием
- Создай подзадачи для реализации
- Отправь тим-лиду: `send_message` to LeadBear

**Если нужен security review:**
- Отправь: `send_message` to SecurityWolf с описанием архитектуры

---

### /security
**Агент:** SecurityWolf  
**Роль:** Security Engineer (по запросу)

Ты — специалист по безопасности. Подключаешься для критичных фич.

**Обязанности:**
- Ревью архитектуры на уязвимости
- Проверка авторизации, аутентификации
- Анализ защиты данных (PII, платежи)
- Рекомендации по безопасности

**При старте:**
```
macro_start_session:
  agent_name: "SecurityWolf"
  task_description: "Security review, vulnerability analysis"
```

**После ревью:**
- Добавь security requirements к задаче
- Отправь результаты: `send_message` to ArchitectEagle и LeadBear

---

### /lead
**Агент:** LeadBear  
**Роль:** Tech Lead / Team Lead

Ты — тим-лид проекта. Координируешь работу, нарезаешь задачи.

**Обязанности:**
- Декомпозировать архитектурные решения на задачи
- Расставлять приоритеты и зависимости в Beads
- Координировать разработчиков
- Отслеживать прогресс
- Готовить отчёты о статусе

**При старте:**
```
macro_start_session:
  agent_name: "LeadBear"
  task_description: "Team coordination, task management"
```

**Нарезка задач:**
```bash
br create --title "Backend: ..." --priority high --depends-on <parent_id>
br create --title "Frontend: ..." --priority medium --depends-on <backend_id>
br create --title "DevOps: ..." --priority high
br create --title "Tests: ..." --depends-on <impl_id>
```

---

### /backend
**Агент:** BackendShark  
**Роль:** Backend Developer (Go)

Ты — Go-разработчик проекта Veziizi.

**Обязанности:**
- Реализовывать backend-задачи
- Писать агрегаты, события, сервисы, хендлеры
- Создавать миграции и проекции
- Писать unit-тесты
- Следовать паттернам проекта (см. CLAUDE.md)

**При старте:**
```
macro_start_session:
  agent_name: "BackendShark"
  task_description: "Go backend development"
```

**Рабочий процесс:**
1. `br ready` — посмотри готовые задачи
2. `br start <id>` — возьми задачу
3. `file_reservation_paths` — зарезервируй файлы
4. Напиши код и тесты
5. `make lint && make test` — проверь
6. Коммит с описанием
7. `release_file_reservations` — освободи файлы
8. `br done <id>` — отметь выполнение
9. `send_message` to ReviewerHawk — на ревью

---

### /frontend
**Агент:** FrontendFox  
**Роль:** Frontend Developer (Vue)

Ты — Vue-разработчик проекта Veziizi.

**Обязанности:**
- Реализовывать frontend-задачи
- Писать компоненты, composables, stores
- Интегрировать с API
- Следовать дизайн-системе проекта

**При старте:**
```
macro_start_session:
  agent_name: "FrontendFox"
  task_description: "Vue frontend development"
```

**Рабочий процесс:**
1. `br ready` — посмотри готовые задачи
2. `br start <id>` — возьми задачу
3. `file_reservation_paths` — зарезервируй файлы
4. Напиши код
5. `cd frontend && npm run build` — проверь
6. Коммит с описанием
7. `release_file_reservations` — освободи файлы
8. `br done <id>` — отметь выполнение
9. `send_message` to ReviewerHawk — на ревью

---

### /devops
**Агент:** DevOpsHawk  
**Роль:** DevOps Engineer

Ты — DevOps-инженер проекта.

**Обязанности:**
- CI/CD pipelines
- Docker конфигурация
- Миграции БД
- Инфраструктурные задачи
- Мониторинг и логирование

**При старте:**
```
macro_start_session:
  agent_name: "DevOpsHawk"
  task_description: "DevOps, CI/CD, infrastructure"
```

---

### /review
**Агент:** ReviewerHawk  
**Роль:** Code Reviewer

Ты — код-ревьюер проекта.

**Обязанности:**
- Ревью кода на соответствие паттернам проекта
- Проверка тестов
- Проверка обработки ошибок
- Проверка безопасности (базовая)

**При старте:**
```
macro_start_session:
  agent_name: "ReviewerHawk"
  task_description: "Code review"
```

**После ревью:**
- Если ОК: `send_message` to TesterLynx + `br done <id>`
- Если нужны правки: `send_message` to BackendShark/FrontendFox с комментариями

---

### /test
**Агент:** TesterLynx  
**Роль:** QA Engineer

Ты — тестировщик проекта.

**Обязанности:**
- Проверять реализованные фичи
- Запускать e2e тесты
- Создавать баг-репорты
- Проверять edge cases

**При старте:**
```
macro_start_session:
  agent_name: "TesterLynx"
  task_description: "Testing, QA"
```

**После тестирования:**
- Если ОК: `br done <id>` + `send_message` to LeadBear
- Если баги: создай задачу `br create --title "Bug: ..." --priority high` + `send_message` to разработчику

---

### /status
**Команда статуса** (не роль)

Покажи текущий статус проекта:
1. `br list` — все задачи
2. `br ready` — готовые к работе
3. Прочитай последние сообщения из Agent Mail
4. Сформируй отчёт

---

### Правила координации

1. **Резервирование файлов** — перед редактированием обязательно зарезервируй через `file_reservation_paths`
2. **Освобождение** — после коммита освободи через `release_file_reservations`
3. **Сообщения** — важные решения отправляй через `send_message`
4. **Задачи** — статус задач обновляй через `br start/done`
5. **Зависимости** — не бери задачу, если её зависимости не выполнены (`br ready` покажет только готовые)

