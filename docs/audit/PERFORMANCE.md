# Performance Audit

## План

1. Аудит индексов: каждый WHERE-фильтр в проекциях должен иметь индекс
2. N+1 запросы в проекциях и хендлерах
3. Unbounded queries (List без LIMIT)
4. COUNT(*) на больших таблицах без оптимизации
5. SELECT * vs нужные колонки (аллокации)
6. Аллокации в hot-path (event-handler в воркерах)
7. Goroutine leaks / unbounded concurrency

## Находки

### Реальные баги (зафиксованы)

#### [High] PERF-A1: Missing indexes на range-фильтры `freight_requests_lookup`
**Файл:** `backend/internal/infrastructure/projections/freight_requests.go` filter options
**Запросы без индекса:**
- `WithMinPrice` / `WithMaxPrice` → `price_amount` (range) — full scan
- `WithMinWeight` / `WithMaxWeight` → `cargo_weight` (range) — full scan
- `WithFreightCarrierOrgID` → `carrier_org_id` — full scan на каждой загрузке дашборда перевозчика

**Доказательство:** В `migrations/00005_freight_requests.sql` нет индексов для этих колонок. В `00014_freight_filters.sql` есть индексы для `cargo_volume`, `payment_method/terms`, `vat_type`, но не для price/weight/carrier_org.

**Фикс:** новая миграция `00026_missing_filter_indexes.sql`:
- `idx_freight_requests_price_amount` (BTREE)
- `idx_freight_requests_cargo_weight` (BTREE)
- `idx_freight_requests_carrier_org` (partial WHERE `carrier_org_id IS NOT NULL` — экономия места, NULL не индексируем)

#### [High] PERF-A2: N+1 в `ListPendingModeration` (уже зафиксован в security commit)
**Файл:** `backend/internal/infrastructure/projections/reviews.go:99-106`
**Проблема:** для каждого review отдельный `SELECT review_fraud_signals` (51 запрос на странице из 50).
**Фикс:** batch-SELECT `WHERE review_id = ANY($1)` — см. коммит security.

### Уже корректно (не баг)

| Сценарий | Проверка |
|---|---|
| `freight_subscriptions.ListByMemberID` | Использует `getRoutePointsBatch` — нет N+1 |
| `freight_subscriptions.FindMatching` | Тот же batch-loader |
| Vehicle list / Get | Индексы `idx_vehicles_*` присутствуют в `00023_vehicles.sql` |
| Pending vehicles list | `idx_pending_vehicles_submitted` обеспечивает ORDER BY |
| Freight invites log | Уникальный (freight, carrier) + индексы по freight и carrier_org |
| Members lookup by email | UNIQUE constraint создаёт неявный btree |
| Reviews moderation list | После фикса N+1 — индекс `idx_reviews_pending_moderation` покрывает status/fraud_score |

### False positives агента

| Описание | Реальность |
|---|---|
| `support_tickets_lookup` missing indexes | Есть `idx_tickets_member/org/status/created/updated` в 00013 |
| `members_lookup.email` без индекса | UNIQUE constraint создаёт индекс автоматически |
| `reviews_lookup.reviewer_org_id + status` | Есть `idx_reviews_active_by_reviewer (reviewer_org_id, status)` в 00015 |

## Факт

| ID | Файл | Изменение |
|----|------|-----------|
| PERF-A1 | `migrations/00026_missing_filter_indexes.sql` | 3 BTREE-индекса (один partial) на price, weight, carrier_org |
| PERF-A2 | `projections/reviews.go` | См. SECURITY.md SEC-A3 — батч-SELECT вместо N+1 |

## Тесты

- `go build ./...` — clean
- `go vet ./...` — clean
- Юнит-тесты проходят

## Open questions / out-of-scope

- **CONCURRENTLY на индексах** — миграция блокирует запись на доли секунды на больших таблицах. Goose v3 не поддерживает `CREATE INDEX CONCURRENTLY` в транзакции. Для production стоит запустить миграцию вручную с `-allow-missing` или адаптировать на отдельную фазу. Сейчас оставлено в обычном виде (cost-benefit для текущего объёма данных приемлемый).
- **Heartbeat overhead воркеров** — `00022_worker_heartbeats.sql` пишет в БД каждые N секунд. Не аудитировал, выглядит безопасно.
- **Connection pool sizing** — не проверял, требует прода для замеров.
- **Watermill subscriber polling overhead** — не аудитировал.
