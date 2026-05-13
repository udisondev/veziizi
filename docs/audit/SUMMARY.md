# Итоговый отчёт ночного аудита Veziizi

Дата: 2026-05-11 → ночь
Ветка: `audit/full-codebase-review` (от `feat/autopark`)
Базовый коммит: `b44020f` (feat: autopark)

## Что было сделано

### 1. Полная инвентаризация (docs/audit/INVENTORY.md)
Через параллельные Explore-агенты прошёлся по:
- 133 HTTP endpoint'а (11 public + 27 admin + 4 dev + 91 auth)
- 16 воркеров (14 event-driven + 2 scheduled)
- 4 агрегата (Organization, FreightRequest, Review, SupportTicket)
- ~20 проекций
- 6 pub/sub topics

### 2. Аудит по семи направлениям
- **SECURITY.md** — 3 реальных бага зафиксаны, 5 false positives отклонены
- **PERFORMANCE.md** — 1 миграция с тремя индексами, 1 N+1 (зафиксан вместе с security)
- **CONSISTENCY.md** — архитектура корректна, 3 открытых вопроса задокументированы
- **RELIABILITY.md** — главный баг (goroutine leak в trackView) уже зафиксан, наблюдения собраны
- **READABILITY.md** — четыре улучшения уже сделаны до аудита (autopark commit)

## Коммиты в ветке

| Коммит | Содержание |
|---|---|
| `2d45198` | fix(security): убрать echo ошибок, заменить fmt.Sprintf SQL-литерал, устранить N+1 в reviews |
| `<perf>` | perf: добавлены индексы под range-фильтры freight_requests_lookup |
| `<docs>` | docs: финальные отчёты CONSISTENCY/RELIABILITY/READABILITY + SUMMARY |

## Применённые фиксы

### Security
| ID | Файл | Что |
|----|------|-----|
| SEC-A1 | `subscriptions.go` (4 места) | `writeError(w, 400, err.Error())` → generic + `slog.Warn` |
| SEC-A2 | `projections/freight_requests.go` | `fmt.Sprintf("{%s}", joinInts(ids))` → нативный pgx `[]int` |
| SEC-A3 | `projections/reviews.go` | N+1 в `ListPendingModeration` → batch `WHERE review_id = ANY($1)` |

### Performance
| ID | Файл | Что |
|----|------|-----|
| PERF-A1 | `migrations/00026_missing_filter_indexes.sql` | 3 BTREE-индекса (price_amount, cargo_weight, carrier_org partial) |

### Already fixed in `feat/autopark` (commit b44020f) — упоминается для полноты
- vehicle.Get IDOR (статус модерации не утекает не-членам)
- writeVehicleDomainError / writeInviteCarrierError — не echo'им err.Error()
- CarrierInvited race → claim через TryInsert по unique-индексу + InviteID в событии
- WithLoadingType: `[]string{v}` вместо `"{"+v+"}"`
- WithVehicleCursor — мёртвый код удалён
- hasActiveVehicleByRegNumber — case-insensitive
- trackView — sync вместо unbounded goroutine
- CanManageVehicles — отдельный permission вместо CanManageMembers

## Что NOT было исправлено (open questions)

### Архитектурные / spec-уровень
- **OQ-1** (consistency): `notification-dispatcher` publishes raw watermill messages вне tx → теоретически дубль уведомлений при handler-replay. Митигируется `notification_delivery_log` UNIQUE constraint. Не блокер.
- **OQ-2** (consistency): `review-receiver` не проверял на детерминированность ID для Review. Возможен duplicate review при replay, если ID — `uuid.New()`. **Требует проверки.**
- **REL-Q1**: watermill retry backoff не верифицирован — может уйти в hot-loop при transient errors.
- **REL-Q2**: graceful shutdown воркеров — не аудитировал.

### Cosmetic / nice-to-have
- Унификация `nullableString`/`nullableFloat` helpers (дублируются в 3 файлах)
- Магические числа `limit := 50`, `<= 100` → в константы
- Watermill subscriber configuration tuning (production-зависимо)

### За рамками
- Frontend Vue/TypeScript (не было запроса детального аудита)
- Docker-compose / CI / scripts (не код)
- Внешние API integrations (Resend, Telegram) — нужны интеграционные тесты

## False positives агентов (отклонены чтением кода)

| Агент сказал | Реальность |
|---|---|
| CORS echo Origin + credentials = critical | Origin отдаётся только из allowlist — корректно |
| trackView IDOR | `memberID` берётся из сессии, не из URL — пишутся только свои данные |
| SESSION_SECRET length validation = high | Это валидация конфига, не уязвимость |
| SESSION_ADMIN_SECRET fallback = medium | Удобство для dev, `slog.Warn` SEC-006 уже логируется |
| Admin path prefix injection в middleware | Mounted через `r.Route()`, middleware видит правильный path |
| support_tickets_lookup missing indexes | Есть `idx_tickets_member/org/status` в 00013 |
| members_lookup.email без индекса | UNIQUE constraint создаёт неявный btree |

## Тесты

После каждого коммита прогонялись:
- `go build ./...` — clean
- `go vet ./...` — clean
- `go test -count=1 ./backend/internal/...` — все проходят (включая обновлённые `TestWithRouteCities/Countries_NonEmpty`)

Не запускалось:
- `go test ./backend/e2e/...` — требует Docker/Postgres, в безголовом аудите вне scope
- `task lint` — не верифицировал golangci-lint clean

## Рекомендации follow-up

1. **OQ-2** (review-receiver determinism) — проверить, как формируется Review aggregate ID при обработке `ReviewLeft`. Если `uuid.New()` — заменить на детерминированный UUIDv5 от `(freight_request_id, reviewer_org_id)`.
2. **Миграция 00026** — на production развернуть с `CREATE INDEX CONCURRENTLY` (потребует ручного развёртывания, т.к. Goose не поддерживает CONCURRENTLY в транзакции).
3. **Production observability** — graphite/grafana board под:
   - `notification_delivery_log` success rate
   - `event_store` write latency
   - `pgxpool` saturation
4. **OQ-1/REL-Q1** — настроить watermill retry backoff в `pkg/worker` если ещё не настроено.

## Финал

Все правки сделаны в ветке `audit/full-codebase-review`. PR можно создавать через `gh pr create` или вручную после `git push`.

Контекст ограничен — глубокая декомпозиция дальше не уместилась. Если потребуется ещё детализация по отдельному вопросу — открыть конкретное направление как отдельную задачу.
