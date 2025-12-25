package order

import (
	"context"
	"fmt"

	"codeberg.org/udison/veziizi/backend/internal/domain/notification/rules"
	"codeberg.org/udison/veziizi/backend/internal/domain/notification/values"
	orderEvents "codeberg.org/udison/veziizi/backend/internal/domain/order/events"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
)

// MessageSentRule уведомляет другую сторону о новом сообщении
type MessageSentRule struct {
	deps rules.Dependencies
}

// NewMessageSentRule создает правило
func NewMessageSentRule(deps rules.Dependencies) *MessageSentRule {
	return &MessageSentRule{deps: deps}
}

func (r *MessageSentRule) EventType() string {
	return orderEvents.TypeMessageSent
}

func (r *MessageSentRule) Process(ctx context.Context, event eventstore.Event) ([]rules.NotificationRequest, error) {
	e, ok := event.(orderEvents.MessageSent)
	if !ok {
		return nil, fmt.Errorf("unexpected event type: %T", event)
	}

	order, err := r.deps.Orders.GetByID(ctx, e.AggregateID())
	if err != nil {
		return nil, fmt.Errorf("get order: %w", err)
	}
	if order == nil {
		return nil, nil
	}

	// Определяем получателя (другая сторона)
	var recipientMemberID, recipientOrgID = order.CarrierMemberID, order.CarrierOrgID
	if e.SenderOrgID == order.CarrierOrgID {
		recipientMemberID = order.CustomerMemberID
		recipientOrgID = order.CustomerOrgID
	}

	return []rules.NotificationRequest{
		{
			RecipientMemberID: recipientMemberID,
			RecipientOrgID:    recipientOrgID,
			NotificationType:  values.TypeOrderMessage,
			Title:             "Новое сообщение",
			Body:              fmt.Sprintf("В заказе #%d новое сообщение", order.OrderNumber),
			Link:              fmt.Sprintf("/orders/%s", order.ID),
			EntityType:        values.EntityOrder,
			EntityID:          order.ID,
		},
	}, nil
}
