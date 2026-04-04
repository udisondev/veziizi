package rules

import (
	"context"

	"github.com/google/uuid"
	"github.com/udisondev/veziizi/backend/internal/domain/notification/values"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
)

// NotificationRule определяет правило создания уведомлений для события
type NotificationRule interface {
	// EventType возвращает тип события, которое обрабатывает правило
	EventType() string

	// Process обрабатывает событие и возвращает список уведомлений для отправки
	// Возвращает пустой slice если уведомления не нужны
	Process(ctx context.Context, event eventstore.Event) ([]NotificationRequest, error)
}

// NotificationRequest запрос на создание уведомления
type NotificationRequest struct {
	RecipientMemberID uuid.UUID
	RecipientOrgID    uuid.UUID
	NotificationType  values.NotificationType
	Title             string
	Body              string
	Link              string
	EntityType        values.EntityType
	EntityID          uuid.UUID
}
