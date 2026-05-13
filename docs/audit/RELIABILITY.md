# Reliability & Fault Tolerance Audit

## План

1. Retry стратегии в воркерах (watermill behavior)
2. Обработка ошибок: errors.Is/As, fmt.Errorf wrap
3. Partial failure recovery (publisher upstream/downstream)
4. Crash safety: idempotency на restart
5. Graceful shutdown
6. Goroutine leaks
7. Context propagation (request-scoped vs background)

## Находки

### Реальные баги (зафиксованы ранее)

#### [Medium] REL-A1: Unbounded goroutines в trackView — уже зафиксован
**Файл:** `handlers/freight_request_views.go:23` (предыдущая версия)
**Проблема:** `go func() { ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second); ... }()` — на каждый GET freight-request порождалась goroutine, отвязанная от request context. При нагрузке — leak.
**Фикс:** в коммите `b44020f` переведено в синхронный вызов с `r.Context()`.

### Хорошие практики (наблюдения)

- Все воркеры импортируют domain/events через blank-import → eventstore.UnmarshalEvent работает корректно после restart.
- Heartbeats: `00022_worker_heartbeats.sql` + `pkg/heartbeat` — воркеры регистрируют liveness, можно мониторить.
- `review-analyzer` (handler) уже обрабатывает `ErrReviewAlreadyAnalyzed` как success → idempotent при replay (см. `handlers/review_analyzer.go:onReviewReceived`).
- Все scheduled-воркеры (`review-activator`, `rate-limiter-cleanup`) — небольшие батчи, безопасны при перезапуске (берут "следующие N" по таймштампу).

### Open questions / risks без фиксов

#### REL-Q1: Watermill retry без backoff?
Не аудитировал конфиг subscriber'а в `pkg/worker`. По умолчанию watermill-sql retry на ошибку handler'а немедленный → может уйти в hot-loop. Если работает — наблюдать в production.

#### REL-Q2: Graceful shutdown воркеров
`worker.Run` принимает `Config` → должен корректно дожидаться завершения текущего handler перед exit. Не аудитировал.

#### REL-Q3: Email/Telegram external API failures
- `email-sender`: если Resend API недоступен — пишет в `notification_delivery_log` со статусом failed. ОК.
- `telegram-sender`: аналогично через `DeliveryLogProjection`. ОК.
- Retry стратегия для transient errors — следует проверить.

#### REL-Q4: Pool exhaustion
`pgxpool.Pool` имеет конфигурируемый размер. Если все workers + API толкают tx одновременно — может насыщаться. Конфигурируется через `DATABASE_*` env. Не аудитировал.

## Факт

Реальные баги — все уже зафиксованы в предыдущих коммитах этой ветки. Архитектура отказоустойчивости адекватна:
- Outbox pattern → consistency at publish
- Idempotent handlers → safe replays
- Heartbeats → внешний monitoring
- DeliveryLog → audit отправок

## Out-of-scope

- Конфиг watermill backoff/retry — требует production-замера
- DB pool sizing
- Внешние API timeouts (Telegram, Email)
