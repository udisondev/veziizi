package order

import (
	"context"
	"fmt"

	"codeberg.org/udison/veziizi/backend/internal/domain/notification/rules"
	"codeberg.org/udison/veziizi/backend/internal/domain/notification/values"
	orderEvents "codeberg.org/udison/veziizi/backend/internal/domain/order/events"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
)

// OrderCancelledRule уведомляет другую сторону об отмене заказа
type OrderCancelledRule struct {
	deps rules.Dependencies
}

// NewOrderCancelledRule создает правило
func NewOrderCancelledRule(deps rules.Dependencies) *OrderCancelledRule {
	return &OrderCancelledRule{deps: deps}
}

func (r *OrderCancelledRule) EventType() string {
	return orderEvents.TypeOrderCancelled
}

func (r *OrderCancelledRule) Process(ctx context.Context, event eventstore.Event) ([]rules.NotificationRequest, error) {
	e, ok := event.(orderEvents.OrderCancelled)
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
	if e.CancelledByOrgID == order.CarrierOrgID {
		recipientMemberID = order.CustomerMemberID
		recipientOrgID = order.CustomerOrgID
	}

	return []rules.NotificationRequest{
		{
			RecipientMemberID: recipientMemberID,
			RecipientOrgID:    recipientOrgID,
			NotificationType:  values.TypeOrderCancelled,
			Title:             "Заказ отменён",
			Body:              fmt.Sprintf("Заказ #%d был отменён", order.OrderNumber),
			Link:              fmt.Sprintf("/orders/%s", order.ID),
			EntityType:        values.EntityOrder,
			EntityID:          order.ID,
		},
	}, nil
}
