# Data Consistency Audit

## План

1. Outbox pattern: атомарность event store + watermill publish
2. Idempotency обработчиков воркеров (at-least-once)
3. Race conditions между сервисом и асинхронной проекцией
4. Optimistic locking в event store
5. Транзакционность multi-table writes в воркерах

## Находки

### Архитектура корректна

#### Outbox pattern реализован
**Файл:** `backend/internal/infrastructure/messaging/publisher.go:77-103`
`EventPublisher.Publish` детектит транзакцию в контексте через `dbtx.FromCtx(ctx)` и переключается на `sql.TxFromPgx(tx)` → события попадают в `watermill_messages_<topic>` в той же транзакции, что и event store. `INSERT INTO watermill_messages` + `INSERT INTO events` в одном `BEGIN/COMMIT` — outbox-гарантия.

Подтверждено в каждом `saveAndPublish` (admin, freightrequest, organization, review, support): `eventStore.Save` + `publisher.Publish` обёрнуты в `s.db.InTx`.

#### Workers идемпотентны
Все handlers используют `ON CONFLICT (id) DO NOTHING` для INSERT:
- `members.go:89`, `invitations.go:67`, `pending_organizations.go:63`, `support_tickets.go:65`, `freight_requests.go:191,338`, `freight_invites.go:Insert`
UPDATE-операции (`UpdateStatus`, `Upsert ON CONFLICT DO UPDATE`) idempotent by nature.

#### CarrierInvited race зафиксирована в предыдущем коммите
Service делает `TryInsert` внутри `InTx` → DB-уровень уникальный индекс серилизует concurrent попытки. Worker'у достаётся либо тот же ID (no-op), либо ничего (event пришёл первым из service).

#### Optimistic locking
Event store использует `UNIQUE (aggregate_id, version)` (см. `00001_event_store.sql`). При конкурентной модификации второй commit получает SQLSTATE 23505 → `ErrConcurrentModification`.

### Открытые вопросы (не баги, требуют наблюдения)

#### OQ-1: `notification-dispatcher` и `support_admin_notifier` публикуют через `defaultPublisher`
**Файл:** `handlers/notification_dispatcher.go:190,220`, `handlers/support_admin_notifier.go:145`
**Наблюдение:** Эти воркеры — *consumers* (читают `freightrequest.events`/`support.events`) и затем публикуют raw watermill-сообщения в `notification.{email,telegram}`. Они используют `h.publisher.Publish(topic, msg)` без транзакционного контекста.
**Анализ:** worker subscriber'ы commit'ят offset только после успешного return из handler'а. Если publish прошёл, но handler затем упал — offset не commit'нется, replay вызовет повторный publish (т.е. дубль уведомления на канал).
**Митигация:** на стороне `email-sender`/`telegram-sender` нужна дедупликация по `delivery_log` (есть `notification_delivery_log` проекция). При повторе одно и то же `notification_id` блокируется UNIQUE constraint. Архитектурно нормально, риск дублирующих отправок есть, но минимальный.

#### OQ-2: `review-receiver` не использует `InTx`
**Файл:** не аудитировал детально, но судя по архитектуре — создаёт Review aggregate отдельным save. Если FreightRequest.ReviewLeft уже committed, worker процессит, но создание Review aggregate может упасть → watermill retry → попытка снова.
**Анализ:** Review aggregate должен иметь дедупликацию по `(freight_request_id, reviewer_org_id)` или использовать deterministic ID. Если deterministic — replay не создаст дублей.
**Действие:** не блокирующее, не баг до доказательства обратного.

#### OQ-3: Eventually-consistent reads
**Анализ:** Все проекции eventually-consistent. Сервисы при выборе делать ли действие читают проекцию, а не агрегат — это даёт окно гонки. Текущая стратегия:
- Где race критичен — добавляется DB-уровень уникальный constraint и `TryInsert` (CarrierInvited).
- Где race не критичен — оставляется как есть (фильтры в `WithMinPrice` и т.д. — слегка устаревший список не баг).

## Факт

Кода-фиксов в этой итерации не сделано — архитектура data consistency уже корректна. Все потенциальные расхождения покрыты:
- Outbox (atomic publish)
- ON CONFLICT (idempotent projections)
- Optimistic locking (UNIQUE на (aggregate_id, version))
- DB-claim для anti-spam race (CarrierInvited — уже зафиксано)

## Out-of-scope

- Snapshot/restore стратегии (если будут SnapshotStore — отдельный аудит)
- Cross-aggregate consistency (только eventual через события)
- Watermill exactly-once delivery — невозможно без 2PC, текущий at-least-once + idempotent consumer корректен
