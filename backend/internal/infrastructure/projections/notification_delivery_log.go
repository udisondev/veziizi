package projections

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/udisondev/veziizi/backend/internal/pkg/dbtx"
)

// NotificationDeliveryLogProjection логирует доставку уведомлений
type NotificationDeliveryLogProjection struct {
	db   dbtx.TxManager
	psql squirrel.StatementBuilderType
}

// NewNotificationDeliveryLogProjection создает новую projection
func NewNotificationDeliveryLogProjection(db dbtx.TxManager) *NotificationDeliveryLogProjection {
	return &NotificationDeliveryLogProjection{
		db:   db,
		psql: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

// DeliveryLogInput входные данные для записи лога
type DeliveryLogInput struct {
	MemberID         uuid.UUID
	NotificationType string
	Channel          string
	Status           string
	ErrorMessage     string
}

// LogDelivery записывает лог доставки
func (p *NotificationDeliveryLogProjection) LogDelivery(ctx context.Context, input DeliveryLogInput) error {
	query, args, err := p.psql.
		Insert("notification_delivery_log").
		Columns(
			"member_id",
			"notification_type",
			"channel",
			"status",
			"error_message",
		).
		Values(
			input.MemberID,
			input.NotificationType,
			input.Channel,
			input.Status,
			nullableString(input.ErrorMessage),
		).
		ToSql()

	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	if _, err := p.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert delivery log: %w", err)
	}

	return nil
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
