package projections

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/udisondev/veziizi/backend/internal/pkg/dbtx"
)

// ProjectionEventDedupProjection защищает накопительные операции проекций от
// двойной обработки при at-least-once доставке. Курс 10-at-least-once-delivery:
// для не-идемпотентных handler'ов нужен dedup на уровне приложения.
//
// projection_name — логическое имя проекции (например "reviews-projection"),
// event_id — EventEnvelope.ID из metadata сообщения. Композитный ключ позволяет
// разным проекциям независимо дедуплицировать одно и то же событие.
type ProjectionEventDedupProjection struct {
	db dbtx.TxManager
}

func NewProjectionEventDedupProjection(db dbtx.TxManager) *ProjectionEventDedupProjection {
	return &ProjectionEventDedupProjection{db: db}
}

// Begin резервирует event_id за проекцией. Возвращает (true, nil) — резерв
// успешен, handler должен выполнить операцию; (false, nil) — событие уже
// обработано этой проекцией, handler должен сделать ack без побочных эффектов.
//
// Вызывать ВНУТРИ той же tx, что и накопительная операция: иначе при failure
// между Begin и UPDATE счётчика останется фантомная dedup-строка без
// соответствующего обновления, и повторная доставка молча пропустит запись.
func (p *ProjectionEventDedupProjection) Begin(ctx context.Context, projectionName string, eventID uuid.UUID) (bool, error) {
	tag, err := p.db.Exec(ctx,
		`INSERT INTO projection_event_dedup (projection_name, event_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		projectionName, eventID,
	)
	if err != nil {
		return false, fmt.Errorf("insert projection_event_dedup: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}
