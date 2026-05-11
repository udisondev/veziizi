# Полный аудит кодовой базы Veziizi

Дата начала: 2026-05-11
Ветка: `audit/full-codebase-review` (от `feat/autopark`)

## Зона ответственности

Аудит всех компонентов backend (Go) и точечно frontend (Vue) на:
1. Безопасность (auth/авторизация, IDOR, injection, утечки данных, секреты)
2. Производительность (индексы, N+1, ненужные запросы, аллокации)
3. Согласованность данных (транзакции, event→projection sync, race conditions)
4. Надёжность (retry, обработка ошибок)
5. Читаемость / идиоматичность Go
6. Консистентность стиля и подходов
7. Отказоустойчивость (partial failures, restart-safety, идемпотентность)

## Структура отчётов

```
docs/audit/
├── PLAN.md            # этот файл
├── INVENTORY.md       # полная инвентаризация (endpoints, workers, aggregates, projections)
├── SECURITY.md        # план + факт по безопасности
├── PERFORMANCE.md     # план + факт по производительности
├── CONSISTENCY.md     # план + факт по согласованности данных
├── RELIABILITY.md     # план + факт по надёжности и отказоустойчивости
├── READABILITY.md     # план + факт по читаемости / идиоматичности
└── SUMMARY.md         # итоговый отчёт
```

Каждый отчёт содержит:
- **План** — что и где проверять
- **Находки** — список с уровнем (Critical/High/Medium/Low), кратким описанием, файлами:строками
- **Факт** — что исправлено, ссылка на коммит, какие риски остались
- **Тесты** — что проверено (go build / go vet / go test)

## Принципы работы

- Фиксить только подтверждённые баги, не рефакторить ради рефакторинга
- Не выходить за рамки проекта
- Сохранять обратную совместимость API
- Каждый коммит — атомарный, с понятным сообщением, проходит go build + go vet + go test
- Если что-то спорное — оставлять в отчёте как "Open question", не фиксить

## Этапы

| # | Этап | Статус |
|---|------|--------|
| 1 | Создание ветки + структура docs/audit | done |
| 2 | Инвентаризация всех компонентов (parallel via Explore) | done |
| 3 | Security audit + фиксы | in progress |
| 4 | Performance audit + фиксы | pending |
| 5 | Consistency audit + фиксы | pending |
| 6 | Reliability audit + фиксы | pending |
| 7 | Readability/idiomaticity audit + фиксы | pending |
| 8 | Финальный SUMMARY + push | pending |

## Контекст ограничений

- Все события передаются через watermill-sql (at-least-once)
- Event Store — PostgreSQL, optimistic locking на (aggregate_id, version)
- Проекции eventually-consistent
- 16 воркеров: 14 event-driven + 2 scheduled
- 133 HTTP endpoint'а (11 public + 27 admin + 4 dev + 91 auth)
- Доменные агрегаты: Organization, FreightRequest, Review, Support (notification — без агрегата)

## Что вне scope

- Frontend (Vue) — только пробежать глазами на грубые ошибки
- Конфигурация инфраструктуры (docker-compose, Procfile.dev)
- Документация (CLAUDE.md, ROADMAP.md)
- Скрипты сборки (Taskfile.yml)
- Tools (backend/cmd/tools/*) — упомянуть проблемы, не фиксить
