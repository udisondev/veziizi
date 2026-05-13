# Readability / Idiomaticity Audit

## План

1. Go-стиль: имена, error handling, generics, slog
2. Дубли логики
3. Велосипеды вместо stdlib/popular libs
4. Магические числа, длинные функции
5. Dead code
6. Комментарии: WHY vs WHAT
7. Тестируемость
8. Naming consistency

## Находки

### Реальные баги / qualities (зафиксованы ранее в этой ветке)

| ID | Файл | Изменение |
|----|------|-----------|
| READ-A1 | `projections/freight_requests.go` | Удалён `joinInts` — заменён на нативный pgx-marshalling |
| READ-A2 | `projections/vehicles.go` | Удалён `WithVehicleCursor` (dead code) |
| READ-A3 | `domain/freightrequest/values/vehicle_*` | Перемещены в отдельный пакет `domain/transport` + transport_aliases.go для backward-compat |
| READ-A4 | `entities/member.go` + `values/member_role.go` | Добавлен `CanManageVehicles` — раньше использовали `CanManageMembers` для vehicle commands (семантическая каша) |

### Наблюдения (good practices)

- `slog` используется повсеместно, не передаётся как dependency
- `errors.Is/As` — корректно
- `fmt.Errorf("...: %w", err)` — corrected wrapping
- DDD-разделение: domain/entities/values/events/errors
- Factory pattern для lazy-init dependencies (sync.Once) — чисто
- `TxExecutor` с savepoint-вложенностью — нестандартный, но грамотный
- Generic enum'ы через go-enum — `//go:generate` comments

### Open questions / стилистические улучшения (не блокирующие)

#### READ-Q1: `nullableFloat`/`nullableString` дублируется
Есть в `projections/notification_delivery_log.go`, `projections/vehicles.go`, `tools/seed-geo/main.go`.
Можно вынести в `pkg/dbtx` или `pkg/util`. Не баг — минор.

#### READ-Q2: `interface{}` vs `any`
Проверил — в новом коде используется `any`. Может быть legacy-использования `interface{}` в старых файлах. Не критично.

#### READ-Q3: Длинные функции
`FreightRequest.aggregate.go:apply` — switch с >20 case'ами. Это норма для event-sourcing aggregate, не баг.

#### READ-Q4: Магические числа
`maxInvitesPerFreight = 20` — константа есть, хорошо. Но в `projections/vehicles.go` `limit := 50` и `n <= 100` — magic в коде. Можно вынести.

## Факт

Все qualities, найденные при основном аудите, уже зафиксованы в:
- Коммит autopark feature: семантика CanManageVehicles, dead code cleanup
- Коммит security fixes: joinInts удалён

Дополнительных рефакторингов сейчас не делаю — за рамками "фиксить баги".

## Out-of-scope

- Стандартизация `nullableX` хелперов (минор-рефакторинг)
- Магические числа → константы (косметика)
- Frontend стиль (Vue/TypeScript)
