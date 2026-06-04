package handlers

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/udisondev/veziizi/backend/internal/infrastructure/projections"
	"github.com/udisondev/veziizi/backend/internal/pkg/dbtx"
)

// dedupGuard оборачивает обработку события в одну tx с dedup-резервом
// (projectionName, eventID) в projection_event_dedup. Если событие уже
// обрабатывалось (повторная at-least-once доставка) — fn не вызывается,
// возвращаем nil (молчаливый Ack).
//
// dedup.Begin делает INSERT ... ON CONFLICT DO NOTHING внутри той же tx, что и
// fn — это даёт atomic «либо весь набор операций применился и dedup-строка
// есть, либо ничего не применилось и dedup-строки нет». Конкурентная обработка
// одного события двумя инстансами тоже безопасна: второй insert упрётся в
// блокировку/конфликт и получит first=false.
//
// eventID — EventEnvelope.ID, стабильный через retry'и и переезд forwarder→Redis.
// CQRS-хендлеры достают его через eventIDFromCtx, legacy-хендлеры — из envelope.
//
// КОНТРАКТ fn (нарушение = тихая потеря данных):
//   - fn ОБЯЗАН пропагировать ЛЮБУЮ ошибку наверх. Проглоченная ошибка
//     (slog + return nil) = закоммиченный dedup-резерв = подавленный
//     redelivery = эффект события потерян НАВСЕГДА. Rollback, наоборот,
//     снимает резерв, и Retry middleware доставит событие повторно.
//   - Намеренный пропуск события (return nil без выполнения работы) допустим
//     ТОЛЬКО до первого побочного эффекта и только по детерминированному
//     условию (устаревшая версия, неподходящий тип события) — не по ошибке.
//   - Запросы, которые могут вернуть «строки ещё нет» (событие обогнало
//     Created), обязаны превращать это в ошибку (errProjectionRowMissing,
//     проверка RowsAffected), а не молча no-op'иться.
//
// Регрессии контракта ловит e2e: backend/e2e/tests/notification_pipeline_test.go.
func dedupGuard(
	ctx context.Context,
	db dbtx.TxManager,
	dedup *projections.ProjectionEventDedupProjection,
	projectionName string,
	eventID uuid.UUID,
	fn func(ctx context.Context) error,
) error {
	return db.InTx(ctx, func(ctx context.Context) error {
		first, err := dedup.Begin(ctx, projectionName, eventID)
		if err != nil {
			return fmt.Errorf("dedup begin: %w", err)
		}
		if !first {
			slog.Debug("event already processed, skipping",
				slog.String("projection", projectionName),
				slog.String("event_id", eventID.String()))
			return nil
		}
		return fn(ctx)
	})
}
