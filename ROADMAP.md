# Logistics Platform - Roadmap

## Project Overview
Логистическая платформа для грузоперевозок. Монорепо с Go backend и Vue.js frontend.

**Tech Stack:** Go 1.23+, Vue.js 3, Tailwind, PostgreSQL 16, watermill-sql, gorilla, squirrel, pgxscan, goose

**Architecture:** DDD, Event Sourcing, Event Driven, 12-factor app

## Current Status
**Phase:** 6 - Rating Fraud Protection
**Status:** Completed
**Last Updated:** 2025-12-19

---

## Phases

### Phase 1: Foundation
**Status:** [x] Completed

- [x] Project structure setup (monorepo /backend, /frontend)
- [x] Go modules initialization
- [x] Docker Compose (PostgreSQL)
- [x] Base packages:
  - [x] Config (12-factor: env vars with caarlos0/env + validator)
  - [x] Event store interface + PostgreSQL implementation
  - [x] Aggregate base
  - [x] Watermill setup (publisher with pgx adapters)
  - [x] HTTP server with gorilla/mux
  - [x] Session management (gorilla/sessions)
  - [x] Database connection (pgx pool)
  - [x] Migrations setup (goose)
  - [x] Transaction manager (dbtx package)
  - [x] Main entry point (cmd/api)

### Phase 2: Organization Domain
**Status:** [x] Completed

- [x] Organization aggregate + events
  - [x] OrganizationCreated
  - [x] OrganizationApproved
  - [x] OrganizationRejected
  - [x] OrganizationSuspended
  - [x] OrganizationUpdated
- [x] Member entity + events
  - [x] MemberAdded
  - [x] MemberRoleChanged
  - [x] MemberBlocked
  - [x] MemberUnblocked
- [x] Invitation entity + events
  - [x] InvitationCreated
  - [x] InvitationAccepted
  - [x] InvitationExpired
- [x] Value objects with go-enum (Country, OrganizationStatus, MemberRole, MemberStatus, InvitationStatus, CarrierProfile, Address)
- [x] Repository (uses shared event store)
- [x] Projection handlers (members_lookup, invitations_lookup)
- [x] Application service
- [x] HTTP API:
  - [x] POST /api/v1/organizations (register org + first member)
  - [x] POST /api/v1/auth/login
  - [x] POST /api/v1/auth/logout
  - [x] GET /api/v1/auth/me
  - [x] GET /api/v1/organizations/:id
  - [x] POST /api/v1/organizations/:id/invitations
  - [x] POST /api/v1/invitations/:token/accept
  - [x] PATCH /api/v1/organizations/:id/members/:memberId/role
  - [x] POST /api/v1/organizations/:id/members/:memberId/block
  - [x] POST /api/v1/organizations/:id/members/:memberId/unblock

### Phase 3: Platform Admin
**Status:** [x] Completed

- [x] PlatformAdmin entity (platform_admins table + repository)
- [x] Admin auth (separate session namespace: veziizi_admin_session)
- [x] Async watermill subscribers (separate cmd/workers processes):
  - [x] members worker (updates members_lookup)
  - [x] invitations worker (updates invitations_lookup)
  - [x] pending-organizations worker (updates pending_organizations)
- [x] Worker infrastructure (internal/pkg/worker package)
- [x] HTTP API:
  - [x] POST /api/v1/admin/auth/login
  - [x] POST /api/v1/admin/auth/logout
  - [x] GET /api/v1/admin/organizations (list pending)
  - [x] POST /api/v1/admin/organizations/:id/approve
  - [x] POST /api/v1/admin/organizations/:id/reject
  - [x] GET /api/v1/admin/organizations/:id

### Phase 4: FreightRequest Domain
**Status:** [x] Completed

- [x] Value objects:
  - [x] Route, RoutePoint, RoutePointType, Coordinates
  - [x] CargoInfo, Dimensions, CargoType, ADRClass
  - [x] VehicleRequirements, BodyType, LoadingType, Temperature
  - [x] Payment, Money, Currency, PriceType, VatType, PaymentMethod, PaymentTerms
  - [x] FreightRequestStatus, OfferStatus
- [x] FreightRequest aggregate + events
  - [x] FreightRequestCreated
  - [x] FreightRequestUpdated (increments freightVersion)
  - [x] FreightRequestReassigned
  - [x] FreightRequestCancelled
  - [x] FreightRequestExpired
  - [x] OfferMade
  - [x] OfferWithdrawn
  - [x] OfferSelected
  - [x] OfferRejected
  - [x] OfferConfirmed
  - [x] OfferDeclined
- [x] Offer entity
- [x] Application service (работает напрямую с event store)
- [x] Projection handlers + worker (freight_requests_lookup, offers_lookup)
- [x] HTTP API:
  - [x] POST /api/v1/freight-requests
  - [x] GET /api/v1/freight-requests (with filters)
  - [x] GET /api/v1/freight-requests/:id
  - [x] PATCH /api/v1/freight-requests/:id
  - [x] DELETE /api/v1/freight-requests/:id (cancel)
  - [x] POST /api/v1/freight-requests/:id/reassign
  - [x] POST /api/v1/freight-requests/:id/offers
  - [x] GET /api/v1/freight-requests/:id/offers
  - [x] DELETE /api/v1/freight-requests/:id/offers/:offerId (withdraw)
  - [x] POST /api/v1/freight-requests/:id/offers/:offerId/select
  - [x] POST /api/v1/freight-requests/:id/offers/:offerId/reject
  - [x] POST /api/v1/freight-requests/:id/offers/:offerId/confirm
  - [x] POST /api/v1/freight-requests/:id/offers/:offerId/decline

### Phase 5: Order Domain
**Status:** [x] Completed

- [x] Order aggregate + events
  - [x] OrderCreated (from confirmed offer, automatically via order-creator worker)
  - [x] MessageSent
  - [x] DocumentAttached
  - [x] DocumentRemoved
  - [x] CustomerCompleted
  - [x] CarrierCompleted
  - [x] OrderCompleted (both sides)
  - [x] OrderCancelled (with CancelledByCustomer/CancelledByCarrier status)
  - [x] ReviewLeft
- [x] Message entity
- [x] Document entity
- [x] Review entity
- [x] OrderStatus value object (active, customer_completed, carrier_completed, completed, cancelled_by_customer, cancelled_by_carrier)
- [x] File storage interface + PostgreSQL implementation
- [x] Application service (works directly with event store)
- [x] Order Creator handler (listens to OfferConfirmed, creates Order)
- [x] Projection handler + worker (orders_lookup - ID + filter columns only, no JSONB)
- [x] HTTP API:
  - [x] GET /api/v1/orders (list with filters)
  - [x] GET /api/v1/orders/:id (full order from event store)
  - [x] POST /api/v1/orders/:id/messages
  - [x] POST /api/v1/orders/:id/documents (upload)
  - [x] GET /api/v1/orders/:id/documents/:docId (download)
  - [x] DELETE /api/v1/orders/:id/documents/:docId
  - [x] POST /api/v1/orders/:id/complete
  - [x] POST /api/v1/orders/:id/cancel
  - [x] POST /api/v1/orders/:id/review

### Phase 6: Rating Fraud Protection
**Status:** [x] Completed
**Last Updated:** 2025-12-19

Система защиты рейтингов от накрутки. Реализуется поэтапно.

#### Архитектура

**Новый Review Aggregate** (отдельный от Order):
```
Order.ReviewLeft → review-receiver → Review.Received
                                          ↓
                                  review-analyzer
                                          ↓
                        Review.Analyzed (fraud check + weight calc)
                                          ↓
                    ┌─────────────────────┴─────────────────────┐
                    ↓                                           ↓
           Auto-approved                              Pending Moderation
           (no fraud signals)                         (Admin Panel)
                    ↓                                           ↓
           Review.Approved                    Review.Approved / Review.Rejected
                    ↓
           review-activator (scheduled, 7/14 дней)
                    ↓
           Review.Activated → organization_ratings (weighted)
```

**Статусы отзыва:** `pending_analysis` → `pending_moderation` / `approved` → `active` / `rejected` / `deactivated`

#### Механизмы защиты

1. **Весовая система рейтинга:**
   ```
   weight = order_amount_weight × org_age_weight × diversity_weight × reputation_weight
   ```
   - order_amount_weight: 100К+ ₽ = 1.0, 50К = 0.9, 10К = 0.7, 1К = 0.5, меньше = 0.3
   - org_age_weight: >12 мес = 1.0, 6-12 = 0.8, 3-6 = 0.6, <3 мес = 0.3
   - diversity_weight: 1-й отзыв от контрагента = 1.0, 2-й = 0.5, 3+ = 0.1
   - reputation_weight: накрутчик = 0.0, подозрительный = 0.3, нормальный = 1.0

2. **Fraud Signals (аномалии):**
   | Сигнал | Severity | Описание |
   |--------|----------|----------|
   | mutual_reviews | high | Взаимные отзывы > 5 раз за месяц |
   | fast_completion | medium | Заказ завершен < 2 часов |
   | perfect_ratings | medium | 100% пятерок от контрагента (>3 отзывов) |
   | new_org_burst | medium | Новая орг получила >10 отзывов за неделю |
   | same_ip | high | Совпадение IP при регистрации |
   | same_fingerprint | high | Совпадение device fingerprint |
   | geo_mismatch | high | Заказ завершён вдали от точки выгрузки (будущее, требует GPS) |

3. **Отложенное влияние:** 7 дней (обычные) / 14 дней (подозрительные)

4. **Репутация рецензентов:** При пометке организации как накрутчика — все её отзывы обесцениваются

#### Новые таблицы

| Таблица | Назначение |
|---------|------------|
| reviews_lookup | Отзывы с весами, fraud_score, статусами |
| review_fraud_signals | Детализация обнаруженных аномалий |
| org_interaction_stats | Статистика взаимодействий между организациями |
| org_reviewer_reputation | Репутация организации как рецензента |
| org_registration_metadata | IP/fingerprint для sock puppet detection |

#### Новые Workers

| Worker | Topic | ConsumerGroup | Назначение |
|--------|-------|---------------|------------|
| review-receiver | order.events | review_receiver | Создает Review aggregate при ReviewLeft |
| review-analyzer | review.events | review_analyzer | Fraud detection, weight calculation |
| reviews-projection | review.events | reviews_projection | Обновляет lookup таблицы |
| review-activator | cron (1 мин) | - | Активирует одобренные отзывы |
| fraudster-handler | organization.events | fraudster_handler | Деактивирует отзывы накрутчиков |

#### Подфазы

- [x] **6.1 Инфраструктура** — миграция, Review aggregate, events, values, ReviewService
- [x] **6.2 Review Receiver** — worker слушает order.events, создает Review
- [x] **6.3 Review Analyzer** — fraud detection, weight calculation (Analyzer service, FraudDataProjection, review-analyzer worker)
- [x] **6.4 Reviews Projection** — ReviewsProjection, ReviewsProjectionHandler, reviews-projection worker (обновляет reviews_lookup, review_fraud_signals, org_interaction_stats, org_reviewer_reputation, organization_ratings)
- [x] **6.5 Review Activator** — scheduled worker для активации (RunScheduled в worker package, review-activator worker)
- [x] **6.6 Admin Moderation** — API endpoints (AdminHandler), frontend (AdminReviewsView.vue) для модерации
- [x] **6.7 Reviewer Reputation** — пометка накрутчиков, деактивация отзывов
- [x] **6.8 Registration Metadata** — сбор IP/fingerprint при регистрации (FingerprintJS на frontend, хранение в MemberAdded event, login history tracking)

#### Будущие улучшения (требуют GPS)

- [ ] geo_mismatch signal — проверка что заказ завершён в точке выгрузки
- [ ] GPS-трекинг маршрута перевозчика

---

### Phase 7: Notifications
**Status:** [ ] Not Started
**Depends on:** Phase 6 (notifications about fraud/moderation)

- [ ] Notification service interface
- [ ] NotificationSubscription entity
- [ ] Telegram notifier (cmd/telegram-notifier)
  - [ ] Watermill subscriber
  - [ ] Bot commands (subscribe, unsubscribe, set filters)
  - [ ] Message formatting
- [ ] Event handlers:
  - [ ] FreightRequestCreated -> notify subscribers matching filters
  - [ ] OfferMade -> notify freight request owner
  - [ ] OfferSelected -> notify offer creator
  - [ ] OfferConfirmed -> notify freight request owner
  - [ ] OrderCreated -> notify both parties
  - [ ] MessageSent -> notify recipient
- [ ] HTTP API:
  - [ ] GET /api/v1/notifications/subscriptions
  - [ ] POST /api/v1/notifications/subscriptions
  - [ ] DELETE /api/v1/notifications/subscriptions/:id
  - [ ] PATCH /api/v1/notifications/subscriptions/:id

### Phase 8: Frontend (Vue.js)
**Status:** [~] In Progress (parallel with Phase 6)

- [x] Project setup (Vite, Vue 3, Tailwind 4, Vue Router, Pinia, Leaflet)
- [x] API client setup (fetch wrapper with error handling)
- [x] Auth:
  - [x] Login page
  - [x] Register organization page
  - [x] Accept invitation page
  - [x] Auth store (Pinia)
  - [x] Route guards (auth, orgActive, role, carrier, admin)
- [x] Organization:
  - [x] Dashboard
  - [x] Organization status pages (pending, rejected, suspended)
  - [ ] Members management
  - [ ] Invitations
  - [ ] Organization settings
- [ ] Freight Requests:
  - [x] Create form (wizard: route, cargo, vehicle, payment, confirmation)
  - [x] Address autocomplete (Nominatim)
  - [x] Map preview (Leaflet)
  - [x] List with filters (all/my, status)
  - [ ] Detail page
  - [ ] My requests (merged into list view)
- [ ] Offers:
  - [ ] Make offer form
  - [ ] Incoming offers list
  - [ ] My offers page
- [ ] Orders:
  - [ ] List
  - [ ] Detail page
  - [ ] Chat
  - [ ] Documents
  - [ ] Complete & review
- [x] Admin panel:
  - [x] Login
  - [x] Organizations list (pending)
  - [x] Organization detail view
- [x] Error pages (403 Forbidden, 404 Not Found)
- [x] Permission system (usePermissions composable, PermissionGuard component)

---

## Domain Model Summary

### Aggregates

**Organization**
- Entities: Member, Invitation
- Key fields: name, inn, legalName, country, status, members

**FreightRequest**
- Entities: Offer
- Key fields: customerOrgID, route, cargo, vehicleRequirements, payment, status, freightVersion, offers
- Statuses: published, selected, confirmed, cancelled, expired

**Order**
- Entities: Message, Document, Review
- Key fields: freightRequestID, offerID, customerOrgID, customerMemberID, carrierOrgID, carrierMemberID, status
- Statuses: active, customer_completed, carrier_completed, completed, cancelled_by_customer, cancelled_by_carrier
- Auto-created from OfferConfirmed event via order-creator worker
- Reviews can be left after completed OR cancelled

### Key Versioning
- `version` - aggregate version (changes on any event)
- `freightVersion` - freight data version (changes only on freight data updates, not on offers)

### Roles (Organization)
- owner - full access, cannot be removed
- administrator - manage members, invitations, reassign responsibility
- employee - create/manage freight requests and offers

---

## Decisions Log

| Date | Decision | Rationale |
|------|----------|-----------|
| 2024-12-14 | Organization universal | Any organization can be customer and make offers |
| 2024-12-14 | Member = User in org context | Email globally unique, 1 email = 1 member |
| 2024-12-14 | Offer inside FreightRequest aggregate | Offers are part of freight request lifecycle |
| 2024-12-14 | Invitation inside Organization aggregate | Invitations are part of org lifecycle |
| 2024-12-14 | Sessions (gorilla) not JWT | User preference, simpler for web |
| 2024-12-14 | PostgreSQL for files initially | With interface for future S3 migration |
| 2024-12-14 | Snapshots for Event Sourcing | Avoid replaying too many events |
| 2024-12-14 | freightVersion separate from version | Allow concurrent offers without conflict |
| 2024-12-14 | No draft status for FreightRequest | Publish immediately on create |
| 2024-12-14 | No i18n initially | Russian only for now |
| 2024-12-14 | Polling first, WebSocket later | Simpler initial implementation |
| 2024-12-14 | Notifiers as separate cmd processes | 12-factor, scalable workers |
| 2025-12-15 | Separate handlers (write) from projections (read) | Allows scaling write/read independently |
| 2025-12-15 | Each watermill subscriber as separate cmd | Can scale each worker type independently |
| 2025-12-15 | ConsumerGroup per handler for watermill | Each handler tracks its own offset |
| 2025-12-15 | Lookup tables: ID + filter columns only, no JSONB | Full data from event store, avoids duplication |
| 2025-12-15 | Order auto-created from OfferConfirmed | Decoupled from FreightRequest, uses worker pattern |
| 2025-12-15 | Factory for services and projections | Lazy-initialized, thread-safe dependency container |

---

## Session Notes

### 2024-12-14 - Initial Planning
- Defined all three aggregates: Organization, FreightRequest, Order
- Established event sourcing approach with watermill-sql
- Decided on project structure (/backend, /frontend)
- Created detailed domain model with all value objects
- Planned 7 implementation phases

### 2024-12-14 - Phase 1 Completed
- Implemented config with caarlos0/env v11 + go-playground/validator
- Created dbtx package for transaction propagation via context
- Implemented event store with EventEnvelope pattern and event registry
- Watermill publisher with native pgx adapters (BeginnerFromPgx, TxFromPgx)
- HTTP server with gorilla/mux, sessions with gorilla/sessions
- Logger writes to current.log file (JSON format)
- All imports use codeberg.org/udison/veziizi/backend/internal/...

### 2025-12-15 - Phase 2 Completed
- Organization aggregate with full event sourcing
- Value objects using go-enum for type-safe enums (Country, MemberRole, etc.)
- Member and Invitation entities inside Organization aggregate
- Projections: members_lookup, invitations_lookup (no organizations_projection - not needed)
- Application service with transactional save + publish
- Auth handlers (login/logout/me) with bcrypt password hashing
- Organization handlers (register, get, invitations, carrier profile, member management)
- Address as simple string (planned DaData/Google Places integration later)

### 2025-12-15 - Phase 3 Completed
- Platform admin with separate session namespace (veziizi_admin_session)
- Migration 00003_platform_admin.sql (platform_admins + pending_organizations tables)
- Admin repository using pgxscan
- Refactored projection updates to async watermill subscribers:
  - Separated handlers (write) from projections (read) for scalability
  - Each handler runs as separate process (cmd/workers/*)
  - Worker boilerplate extracted to internal/pkg/worker package
- AdminService for approve/reject with event sourcing
- AdminHandler with all HTTP endpoints
- Updated Makefile with build-workers and run-workers commands

### 2025-12-15 - Phase 5 Completed
- Order aggregate with full event sourcing
- Order auto-created via order-creator worker when OfferConfirmed event occurs
- Detailed status tracking: CustomerCompleted/CarrierCompleted → Completed, CancelledByCustomer/CancelledByCarrier
- Reviews allowed after Completed OR Cancelled
- Commands return typed errors instead of separate Can* methods
- Message, Document, Review entities (stored in aggregate, not separate tables)
- File storage interface + PostgreSQL implementation (files table)
- Lookup tables optimized: only ID + filter columns, no JSONB (full data from event store)
- Two new workers: orders (projection), order-creator (OfferConfirmed → Order)
- HTTP handlers for all order operations including document upload/download
- Migration 00005_orders.sql (files table, orders_lookup table)
- Factory pattern (internal/pkg/factory) for lazy-initialized, thread-safe dependency injection
- Worker package simplified: receives Factory instead of Deps struct
- All workers and API use Factory for services and projections

### 2025-12-19 - Phase 6 Completed
- Review aggregate с полным event sourcing (pending_analysis → pending_moderation/approved → active/rejected/deactivated)
- Fraud detection система с 6 сигналами: mutual_reviews, fast_completion, perfect_ratings, new_org_burst, same_ip, same_fingerprint
- Весовая система рейтинга: order_amount × org_age × diversity × reputation weights
- 4 новых воркера: review-receiver, review-analyzer, reviews-projection, review-activator
- Scheduled worker pattern (RunScheduled) для review-activator
- Admin moderation endpoints + AdminReviewsView.vue
- Registration metadata: FingerprintJS на frontend, member_login_history table, same_ip/same_fingerprint fraud signals
- Organization ratings projection с weighted average calculation
