package projections

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/udisondev/veziizi/backend/internal/pkg/dbtx"
)

// NotificationDedupProjection хранит UUID-ы уже отправленных уведомлений.
// При at-least-once доставке Telegram/Email handler может получить одно и
// то же сообщение повторно — таблица гарантирует, что внешний API
// (Telegram/SMTP) дёрнется ровно один раз на одно logical-сообщение.
//
// MarkSent делает INSERT ... ON CONFLICT DO NOTHING и возвращает true, если
// строка появилась (нужно отправлять), false — если уже была (skip).
type NotificationDedupProjection struct {
	db dbtx.TxManager
}

func NewNotificationDedupProjection(db dbtx.TxManager) *NotificationDedupProjection {
	return &NotificationDedupProjection{db: db}
}

// MarkSent резервирует UUID за каналом. Возвращает (true, nil) — резерв успешен,
// можно отправлять; (false, nil) — UUID уже занят, отправка должна быть
// пропущена.
func (p *NotificationDedupProjection) MarkSent(ctx context.Context, messageUUID uuid.UUID, channel string) (bool, error) {
	tag, err := p.db.Exec(ctx,
		`INSERT INTO notification_dedup (message_uuid, channel) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		messageUUID, channel,
	)
	if err != nil {
		return false, fmt.Errorf("insert notification_dedup: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}
