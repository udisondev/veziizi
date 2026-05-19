package projections

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/udisondev/veziizi/backend/internal/pkg/dbtx"
)

// NotificationDedupProjection хранит UUID-ы уже отправленных уведомлений.
// При at-least-once доставке Telegram/Email handler может получить одно и
// то же сообщение повторно (рестарт воркера между API-вызовом и Ack) —
// таблица гарантирует, что внешний API дёрнется не более одного раза на
// одно logical-сообщение в нормальных условиях.
//
// Контракт: IsSent проверяется ДО внешнего вызова, MarkSent делается ПОСЛЕ
// успешного вызова. Если sender упадёт между API call и MarkSent — на
// повторе IsSent вернёт false и сообщение уйдёт второй раз; это окно
// двойной доставки принято как меньшее зло против потери уведомления при
// транзиентной ошибке внешнего API + retry middleware.
type NotificationDedupProjection struct {
	db dbtx.TxManager
}

func NewNotificationDedupProjection(db dbtx.TxManager) *NotificationDedupProjection {
	return &NotificationDedupProjection{db: db}
}

// IsSent возвращает true, если для (messageUUID, channel) уже фиксировался
// успешный send. Вызывается до внешнего API, чтобы пропустить повторную
// доставку того же watermill-сообщения.
func (p *NotificationDedupProjection) IsSent(ctx context.Context, messageUUID uuid.UUID, channel string) (bool, error) {
	var exists bool
	err := p.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM notification_dedup WHERE message_uuid = $1 AND channel = $2)`,
		messageUUID, channel,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("query notification_dedup: %w", err)
	}
	return exists, nil
}

// MarkSent фиксирует факт успешной отправки. Вызывается ПОСЛЕ внешнего API.
// INSERT ... ON CONFLICT DO NOTHING — на конкурентный insert не падаем.
func (p *NotificationDedupProjection) MarkSent(ctx context.Context, messageUUID uuid.UUID, channel string) error {
	if _, err := p.db.Exec(ctx,
		`INSERT INTO notification_dedup (message_uuid, channel) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		messageUUID, channel,
	); err != nil {
		return fmt.Errorf("insert notification_dedup: %w", err)
	}
	return nil
}
