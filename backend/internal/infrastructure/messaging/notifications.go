package messaging

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
)

// TelegramNotification — команда «отправить уведомление в Telegram». Реальной
// доставкой занимается telegram-sender worker. Это не доменное событие
// (eventstore.Event): нет aggregate id, нет версии — это команда на канал.
type TelegramNotification struct {
	MemberID uuid.UUID `json:"member_id"`
	ChatID   int64     `json:"chat_id"`
	Title    string    `json:"title"`
	Body     string    `json:"body"`
	Link     string    `json:"link,omitempty"`
}

// EmailNotification — команда «отправить уведомление на e-mail». Получатель —
// email-sender worker.
type EmailNotification struct {
	MemberID         uuid.UUID `json:"member_id"`
	Email            string    `json:"email"`
	NotificationType string    `json:"notification_type"`
	Title            string    `json:"title"`
	Body             string    `json:"body"`
	Link             string    `json:"link,omitempty"`
}

// notificationTopics — маппинг тип команды → watermill topic. Имена топиков
// заданы исторически (notification.telegram / notification.email) и сохраняются
// для совместимости с существующими сообщениями в очереди.
var notificationTopics = map[string]string{
	"TelegramNotification": "notification.telegram",
	"EmailNotification":    "notification.email",
}

// NotificationBus — CQRS event bus для команд на отправку уведомлений.
// Использует JSONMarshaler (notifications не EventEnvelope). Publisher —
// forwarder-обёрнутый tx-aware outbox-publisher (EventPublisher.ForwarderPublisher):
// команда пишется в Postgres outbox (атомарно с tx, если она есть в ctx) и
// доставляется forwarder'ом в Redis-стрим notification.telegram / .email.
type NotificationBus struct {
	bus *cqrs.EventBus
}

func NewNotificationBus(publisher message.Publisher, logger watermill.LoggerAdapter) (*NotificationBus, error) {
	marshaler := cqrs.JSONMarshaler{
		GenerateName: cqrs.StructName,
	}

	bus, err := cqrs.NewEventBusWithConfig(publisher, cqrs.EventBusConfig{
		Marshaler: marshaler,
		GeneratePublishTopic: func(p cqrs.GenerateEventPublishTopicParams) (string, error) {
			topic, ok := notificationTopics[p.EventName]
			if !ok {
				return "", fmt.Errorf("messaging: no topic mapping for notification %q", p.EventName)
			}
			return topic, nil
		},
		Logger: logger,
	})
	if err != nil {
		return nil, fmt.Errorf("create notification bus: %w", err)
	}
	return &NotificationBus{bus: bus}, nil
}

// Publish отправляет команду (TelegramNotification / EmailNotification) в канал.
// Топик определяется автоматически по типу команды.
func (b *NotificationBus) Publish(ctx context.Context, cmd any) error {
	return b.bus.Publish(ctx, cmd)
}

// NotificationMarshaler возвращает marshaler, который subscriber'ы используют
// для распаковки команд. Это тот же JSONMarshaler что и в Publish-стороне —
// нужно держать в одном месте, чтобы encode/decode были симметричны.
func NotificationMarshaler() cqrs.CommandEventMarshaler {
	return cqrs.JSONMarshaler{GenerateName: cqrs.StructName}
}
