# Security Audit

## План

1. Авторизация во всех HTTP-хендлерах (IDOR, ownership checks)
2. SQL-инъекции в проекциях, динамическая конкатенация
3. Утечка PII / статуса модерации в Get-эндпоинтах
4. Echo пользовательского ввода в 400-ответах
5. Cookie / session конфигурация
6. CORS / CSRF / security headers
7. Rate limiting / fraud detection
8. Логирование секретов / токенов

## Находки

### Реальные баги (подтверждены чтением кода и зафиксованы)

#### [Medium] SEC-A1: `err.Error()` echo в `subscriptions.go`
**Файл:** `backend/internal/interfaces/http/handlers/subscriptions.go:152,221,255,287`
**Проблема:** `Create/Update/Delete/SetActive` возвращают `writeError(w, 400, err.Error())` — внутренние ошибки от DB и валидации проекции попадают клиенту. Тот же паттерн уже фиксили в `10fc397` для `freight_request.List`.
**Фикс:** заменил на generic message + `slog.Warn` для аудита (commit ниже).

#### [Medium] SEC-A2: Фрагильный SQL array literal для `route_city_ids` / `route_country_ids`
**Файл:** `backend/internal/infrastructure/projections/freight_requests.go:226-261`
**Проблема:** `WithRouteCities` строит `fmt.Sprintf("{%s}", joinInts(cityIDs))` → передаётся параметром в `?::integer[]`. Сами `int`'ы безопасны (только цифры), но конструкция фрагильна, повторяет ровно тот паттерн, что уже фиксили в `WithLoadingType`. При пустом срезе попадаем в `{}` (теоретически безопасно, но в коде есть гард `if len(...)==0`).
**Фикс:** заменил `fmt.Sprintf("{%s}", joinInts(...))` на нативный `pgx`-маршалинг через `[]int{...}`. Удалил `joinInts` как мёртвый код.

#### [Medium] SEC-A3: N+1 при загрузке `fraud_signals` в `ListPendingModeration`
**Файл:** `backend/internal/infrastructure/projections/reviews.go:99-106`
**Проблема:** Каждый review триггерит отдельный `SELECT` к `review_fraud_signals` — на странице из 50 ревью это 51 запрос. Это И производительность, И DoS-вектор (атакующий может насыщать админ-панель).
**Фикс:** один батч-SELECT `WHERE review_id = ANY($1)` + распределение по reviews в памяти.

### Проверенные и отклонённые (false positives агентов)

| Описание агента | Реальность |
|---|---|
| CORS echo Origin + credentials | Allow-Origin отдаётся **только** для `origin in allowedOrigins` или localhost в dev — это правильный паттерн, не уязвимость |
| trackView IDOR | trackView пишет (`memberID` из сессии, `freightID`) — пользователь видит только свои просмотры, нет утечки чужих данных |
| SESSION_SECRET length validation | Не баг, валидация конфига (nice-to-have) |
| SESSION_ADMIN_SECRET fallback | Дев-удобство, уже логируется `slog.Warn` SEC-006 в `validateSecuritySettings()` |
| Admin path prefix injection | Mounted через `r.Route("/api/v1/admin")`, middleware видит полный `r.URL.Path` — проверки корректны |
| Vehicle.Get IDOR | Уже исправлено в предыдущем коммите (b44020f) |
| Inconsistency 403/404 | Дизайн-выбор, не security bug; не критично пока endpoints согласованы внутри домена |

## Факт

Применённые фиксы — см. коммит `audit-security-fixes` (этой ветки):

| ID | Файл | Изменение |
|----|------|-----------|
| SEC-A1 | `subscriptions.go` | 4× `err.Error()` → `slog.Warn` + generic 400 |
| SEC-A2 | `projections/freight_requests.go` | `fmt.Sprintf("{%s}", joinInts(ids))` → `pgx`-native `[]int{...}`; удалён `joinInts` |
| SEC-A3 | `projections/reviews.go` | `ListPendingModeration` — батч-SELECT через `ANY($1)` вместо N+1 |

## Тесты

- `go build ./...` — clean
- `go vet ./...` — clean
- `go test ./backend/internal/...` — все юнит-тесты проходят

## Что осталось / open questions

- **Rate limit на чувствительные endpoints** (MarketingConsent, прочие toggle-API). Текущий global rate limiter покрывает базовые случаи. Точечный лимит не критичен.
- **CSRF token (не только header check)** — текущий `X-Requested-With` достаточен против большинства CSRF; полный токен — улучшение, но не баг.
- **Slowloris защита** — должна жить на уровне reverse-proxy (nginx/cloudflare), не на уровне Go-сервера. Skip.
- **Member status check для `/auth/me`** — спорно: blocked member не должен ничего делать, но `/me` помогает фронту показать «вы заблокированы». Оставить как есть.
