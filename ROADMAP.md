# Logistics Platform - Roadmap

## Project Overview
Логистическая платформа для грузоперевозок. Монорепо с Go backend и Vue.js frontend.

**Tech Stack:** Go 1.23+, Vue.js 3, Tailwind, PostgreSQL 16, watermill-sql, gorilla, squirrel, pgxscan, goose

**Architecture:** DDD, Event Sourcing, Event Driven, 12-factor app

## Current Status
**Phase:** 4 - FreightRequest Domain
**Status:** Not Started
**Last Updated:** 2025-12-15

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
  - [x] CarrierProfileSet
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
  - [x] POST /api/v1/organizations/:id/carrier-profile
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
**Status:** [ ] Not Started

- [ ] Value objects:
  - [ ] Route, RoutePoint, Address, Coordinates
  - [ ] CargoInfo, Dimensions, CargoType, ADRClass
  - [ ] VehicleRequirements, BodyType, LoadingTypes
  - [ ] Payment, Money, PriceType, VatType, PaymentMethod, PaymentTerms
- [ ] FreightRequest aggregate + events
  - [ ] FreightRequestCreated
  - [ ] FreightRequestUpdated (increments freightVersion)
  - [ ] FreightRequestCancelled
  - [ ] FreightRequestExpired
  - [ ] OfferMade
  - [ ] OfferWithdrawn
  - [ ] OfferSelected
  - [ ] OfferRejected
  - [ ] OfferConfirmed
  - [ ] OfferDeclined
- [ ] Offer entity
- [ ] Repository with optimistic locking (version + freightVersion)
- [ ] Projection handlers (freight_requests_projection)
- [ ] HTTP API:
  - [ ] POST /api/v1/freight-requests
  - [ ] GET /api/v1/freight-requests (with filters)
  - [ ] GET /api/v1/freight-requests/:id
  - [ ] PATCH /api/v1/freight-requests/:id
  - [ ] DELETE /api/v1/freight-requests/:id (cancel)
  - [ ] POST /api/v1/freight-requests/:id/offers
  - [ ] DELETE /api/v1/freight-requests/:id/offers/:offerId (withdraw)
  - [ ] POST /api/v1/freight-requests/:id/offers/:offerId/select
  - [ ] POST /api/v1/freight-requests/:id/offers/:offerId/reject
  - [ ] POST /api/v1/freight-requests/:id/offers/:offerId/confirm
  - [ ] POST /api/v1/freight-requests/:id/offers/:offerId/decline

### Phase 5: Order Domain
**Status:** [ ] Not Started

- [ ] Order aggregate + events
  - [ ] OrderCreated (from confirmed offer)
  - [ ] MessageSent
  - [ ] DocumentAttached
  - [ ] CustomerCompleted
  - [ ] CarrierCompleted
  - [ ] OrderCompleted (both sides)
  - [ ] ReviewLeft
- [ ] Message entity
- [ ] Document entity
- [ ] Review entity (value object?)
- [ ] File storage interface + PostgreSQL implementation
- [ ] Repository
- [ ] HTTP API:
  - [ ] GET /api/v1/orders
  - [ ] GET /api/v1/orders/:id
  - [ ] POST /api/v1/orders/:id/messages
  - [ ] GET /api/v1/orders/:id/messages
  - [ ] POST /api/v1/orders/:id/documents
  - [ ] GET /api/v1/orders/:id/documents
  - [ ] GET /api/v1/orders/:id/documents/:docId
  - [ ] POST /api/v1/orders/:id/complete
  - [ ] POST /api/v1/orders/:id/review

### Phase 6: Notifications
**Status:** [ ] Not Started

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

### Phase 7: Frontend (Vue.js)
**Status:** [ ] Not Started

- [ ] Project setup (Vite, Vue 3, Tailwind, Vue Router, Pinia)
- [ ] API client setup
- [ ] Auth:
  - [ ] Login page
  - [ ] Register organization page
  - [ ] Accept invitation page
- [ ] Organization:
  - [ ] Dashboard
  - [ ] Members management
  - [ ] Invitations
  - [ ] Carrier profile settings
- [ ] Freight Requests:
  - [ ] Create form
  - [ ] List with filters
  - [ ] Detail page
  - [ ] My requests
- [ ] Offers:
  - [ ] Make offer form
  - [ ] Incoming offers list
  - [ ] My offers
- [ ] Orders:
  - [ ] List
  - [ ] Detail page
  - [ ] Chat
  - [ ] Documents
  - [ ] Complete & review
- [ ] Admin panel:
  - [ ] Login
  - [ ] Organizations moderation

---

## Domain Model Summary

### Aggregates

**Organization**
- Entities: Member, Invitation
- Key fields: name, inn, legalName, country, status, carrierProfile?, members

**FreightRequest**
- Entities: Offer
- Key fields: customerOrgID, route, cargo, vehicleRequirements, payment, status, freightVersion, offers
- Statuses: published, selected, confirmed, cancelled, expired

**Order**
- Entities: Message, Document, Review
- Key fields: freightRequestID, customerOrgID, customerMemberID, carrierOrgID, carrierMemberID, status
- Statuses: active, completed, cancelled

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
| 2024-12-14 | Organization universal (customer + carrier) | CarrierProfile determines if can be carrier |
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
