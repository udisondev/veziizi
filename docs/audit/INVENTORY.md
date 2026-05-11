# Инвентаризация компонентов

> Источник: трёх параллельных Explore-агентов (HTTP, workers, domain). Числа могут отличаться от факта на ±2 — финальное число проверяется по коду.

## Сводная статистика

- **HTTP endpoints:** ~133 (11 public + 27 admin + 4 dev + ~91 auth)
- **Workers:** 16 (14 event-driven + 2 scheduled)
- **Pub/sub topics:** 6 (`organization.events`, `freightrequest.events`, `review.events`, `support.events`, `notification.email`, `notification.telegram`)
- **Aggregates:** 4 (Organization, FreightRequest, Review, SupportTicket)
- **Projections:** ~20 lookup-таблиц

## HTTP-эндпоинты (по доменам)

### Public (без auth)
| Метод | Путь | Описание |
|---|---|---|
| GET | /livez, /readyz, /healthz | Health probes |
| POST | /api/v1/organizations | Регистрация |
| GET | /api/v1/organizations/{id} | Публичный профиль |
| GET | /api/v1/organizations/{id}/rating | Рейтинг |
| GET | /api/v1/organizations/{id}/stats | Статистика |
| GET | /api/v1/organizations/{id}/reviews | Отзывы |
| POST | /api/v1/auth/login | Логин |
| POST | /api/v1/auth/forgot-password | Запрос reset |
| POST | /api/v1/auth/reset-password | Reset пароля |
| GET | /api/v1/auth/reset-password/{token} | Валидация токена |
| GET | /api/v1/support/faq | FAQ |
| GET | /api/v1/geo/* | Гео-справочник |
| GET | /api/v1/invitations/{token} | Просмотр инвайта |
| POST | /api/v1/invitations/{token}/accept | Принять инвайт |
| POST | /api/v1/notifications/email/verify | Верификация email |

### Auth (RequireAuth)
- **Organization:** GetFull, GetDashboardStats, GetPendingOffers, CreateInvitation, ListInvitations, CancelInvitation, ChangeMemberRole, BlockMember, UnblockMember, UpdateMemberInfo
- **FreightRequest:** Create, List, Get, Update, Cancel, Reassign, MakeOffer, ListOffers, WithdrawOffer, SelectOffer, RejectOffer, ConfirmOffer, DeclineOffer, UnselectOffer, Complete, LeaveReview, EditReview, CancelAfterConfirmed, ReassignCarrierMember, ListMyOffers, InviteCarrier, ListInvites, ListViewed
- **Notification:** List, GetUnreadCount, MarkAsRead, MarkAllAsRead, GetPreferences, UpdatePreferences, GenerateLinkCode, DisconnectTelegram, SetEmail, DisconnectEmail, SetMarketingConsent, ResendEmailVerification
- **Support:** CreateTicket, ListMyTickets, GetTicket, AddMessage, ReopenTicket
- **Vehicles:** Add, Update, Archive, ListByOrganization, List, Get
- **Subscriptions:** List, Create, Get, Update, Delete, SetActive
- **History:** GetOrganizationHistory, GetFreightRequestHistory

### Admin (RequireAdminAuth)
- **Auth:** Login, Logout, Me
- **Organizations:** ListPending, GetOrganization, Approve, Reject, MarkFraudster, UnmarkFraudster
- **Reviews:** ListPendingReviews, GetReview, ApproveReview, RejectReview
- **Fraudsters:** ListFraudsters
- **Support:** ListTickets, GetTicket, AddMessage, CloseTicket
- **Email templates:** List, Create, Preview, Get, Update, Delete
- **Vehicles:** ListPending, Verify, Reject

### Dev-only (APP_ENV=development)
- /api/v1/dev/status, /users, /switch, /users/{id} DELETE

## Workers

### Event-driven (14)
| Воркер | Topic | ConsumerGroup | Что делает |
|---|---|---|---|
| email-sender | notification.email | email_sender | Отправляет email |
| fraudster-handler | organization.events | fraudster_handler | Деактивирует ревью у фрод-орг |
| freight-requests | freightrequest.events | freight_requests_projection | Обновляет freight_requests_lookup, offers_lookup, freight_request_invites_log |
| invitations | organization.events | invitations_projection | invitations_lookup |
| members | organization.events | members_projection | members_lookup |
| notification-dispatcher | freightrequest.events | notification_dispatcher | Роутит события на каналы (email/telegram) |
| organizations | organization.events | organizations_projection | organizations_lookup |
| pending-organizations | organization.events | pending_organizations | pending_organizations_lookup |
| review-analyzer | review.events | review_analyzer | Фрод-анализ, расчёт веса |
| review-receiver | freightrequest.events | review_receiver | Создаёт Review aggregate из ReviewLeft |
| reviews-projection | review.events | reviews_projection | reviews_lookup, organization_ratings, fraud_signals |
| support-tickets | support.events | support_tickets | support_tickets_lookup + admin-уведомления |
| telegram-sender | notification.telegram | telegram_sender | Отправляет telegram |
| vehicles | organization.events | vehicles_projection | vehicles_lookup, pending_vehicles |

### Scheduled (2)
| Воркер | Интервал | Что делает |
|---|---|---|
| review-activator | cfg.Worker.ReviewActivatorInterval | Активирует одобренные ревью после activation_date |
| rate-limiter-cleanup | cfg.Worker.RateLimiterCleanupInterval | Чистит истёкшие rate-limit-записи |

### Pub/sub граф

| Topic | Publishers | Subscribers |
|---|---|---|
| organization.events | Organization service | fraudster-handler, invitations, members, organizations, pending-organizations, vehicles |
| freightrequest.events | FreightRequest service | freight-requests, notification-dispatcher, review-receiver |
| review.events | Review service + review-receiver | review-analyzer, reviews-projection |
| support.events | Support service | support-tickets |
| notification.email | notification-dispatcher | email-sender |
| notification.telegram | notification-dispatcher | telegram-sender |

## Доменные агрегаты

### FreightRequest (`domain/freightrequest/`)
**Команды:** Update, Cancel, Reassign, Expire, MakeOffer, WithdrawOffer, SelectOffer, RejectOffer, ConfirmOffer, DeclineOffer, UnselectOffer, Complete, LeaveReview, EditReview, CancelAfterConfirmed, ReassignCarrierMember, InviteCarrier

**События:** FreightRequestCreated, Updated, Reassigned, Cancelled, Expired, Offer{Made,Withdrawn,Selected,Rejected,Confirmed,Declined,Unselected,CancelledWithRequest}, CustomerCompleted, CarrierCompleted, FreightRequestCompleted, ReviewLeft, ReviewEdited, CancelledAfterConfirmed, CarrierMemberReassigned, CarrierInvited

**Entities:** Offer, Review (в составе агрегата для исторического трекинга)

**Value Objects:** Route, CargoInfo, VehicleRequirements, Payment, Money, VatType, PaymentMethod, FreightRequestStatus, OfferStatus

### Organization (`domain/organization/`)
**Команды:** Approve, Reject, Suspend, Update, CreateInvitation, CancelInvitation, AcceptInvitation, ChangeMemberRole, BlockMember, UnblockMember, UpdateMemberInfo, RemoveMember (dev), AddMemberDirect (dev), MarkAsFraudster, UnmarkFraudster, AddVehicle, UpdateVehicle, ArchiveVehicle, VerifyVehicle, RejectVehicle

**События:** OrganizationCreated, Approved, Rejected, Suspended, Updated, Member{Added,Removed,RoleChanged,Blocked,Unblocked,InfoUpdated}, Invitation{Created,Accepted,Expired,Cancelled}, Fraudster{Marked,Unmarked}, Vehicle{Added,Updated,Verified,Rejected,Archived}

**Entities:** Member, Invitation, Vehicle

### Review (`domain/review/`)
**Команды:** Edit, RecordAnalysis, Approve, Reject, Activate, Deactivate

**События:** ReviewReceived, ReviewEdited, ReviewAnalyzed, ReviewApproved, ReviewRejected, ReviewActivated, ReviewDeactivated

### SupportTicket (`domain/support/`)
**Команды:** AddUserMessage, AddAdminMessage, Close, Reopen

**События:** TicketCreated, MessageAdded, TicketClosed, TicketReopened

**Entities:** Message

## Проекции (`infrastructure/projections/`)

| Таблица | Источник событий | Назначение |
|---|---|---|
| organizations_lookup | organization.events | Список орг |
| pending_organizations | organization.events | Очередь модерации |
| members_lookup | organization.events | Сотрудники, поиск по email |
| invitations_lookup | organization.events | Активные инвайты |
| freight_requests_lookup | freightrequest.events | Список заявок, фильтры |
| offers_lookup | freightrequest.events | Список офферов |
| freight_request_invites_log | freightrequest.events | Anti-spam для CarrierInvited |
| freight_request_views | (HTTP-side) | История просмотров |
| vehicles_lookup | organization.events | Автопарк |
| pending_vehicles | organization.events | Очередь модерации ТС |
| reviews_lookup | review.events | Список ревью |
| organization_ratings | review.events | Рейтинги орг |
| fraud_signals | review.events | Фрод-сигналы |
| support_tickets_lookup | support.events | Тикеты |
| inapp_notifications | (publish-side) | In-app уведомления |
| notification_delivery_log | (workers-side) | Лог отправок |
| email_templates_lookup | (admin-side) | Шаблоны писем |
| email_verifications | (HTTP-side + workers) | Токены верификации |
| password_resets | (HTTP-side + workers) | Токены сброса |
| session_fraud (+ behavior, geo, anomaly) | (analyzer-side) | Фрод по сессиям |
| telegram_link_codes | (HTTP + bot) | Привязка Telegram |
| freight_subscriptions_lookup | (HTTP-side) | Подписки |
| geo_countries, geo_cities | (seed) | Гео-справочник |

## Граф зависимостей событий

```
FreightRequest.ReviewLeft
    → review-receiver → Review.ReviewReceived
        → review-analyzer → Review.ReviewAnalyzed (+ ReviewApproved auto)
        → reviews-projection → reviews_lookup, fraud_signals
    → review-activator (scheduled) → Review.ReviewActivated
        → reviews-projection → organization_ratings

FreightRequest.* (любое)
    → notification-dispatcher → notification.{email,telegram} → senders
    → freight-requests worker → projections

Organization.FraudsterMarked
    → fraudster-handler → Review.Deactivate (всем ревью орг)
    → reviews-projection → organization_ratings (пересчёт)
```
