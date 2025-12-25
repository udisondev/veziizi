package order

import (
	"context"
	"fmt"

	"codeberg.org/udison/veziizi/backend/internal/domain/notification/rules"
	"codeberg.org/udison/veziizi/backend/internal/domain/notification/values"
	orderEvents "codeberg.org/udison/veziizi/backend/internal/domain/order/events"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
)

// OrderCompletedRule уведомляет обе стороны о завершении заказа
type OrderCompletedRule struct {
	deps rules.Dependencies
}

// NewOrderCompletedRule создает правило
func NewOrderCompletedRule(deps rules.Dependencies) *OrderCompletedRule {
	return &OrderCompletedRule{deps: deps}
}

func (r *OrderCompletedRule) EventType() string {
	return orderEvents.TypeOrderCompleted
}

func (r *OrderCompletedRule) Process(ctx context.Context, event eventstore.Event) ([]rules.NotificationRequest, error) {
	e, ok := event.(orderEvents.OrderCompleted)
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

	title := "Заказ завершён"
	body := fmt.Sprintf("Заказ #%d успешно завершён", order.OrderNumber)
	link := fmt.Sprintf("/orders/%s", order.ID)

	return []rules.NotificationRequest{
		// Заказчику
		{
			RecipientMemberID: order.CustomerMemberID,
			RecipientOrgID:    order.CustomerOrgID,
			NotificationType:  values.TypeOrderCompleted,
			Title:             title,
			Body:              body,
			Link:              link,
			EntityType:        values.EntityOrder,
			EntityID:          order.ID,
		},
		// Перевозчику
		{
			RecipientMemberID: order.CarrierMemberID,
			RecipientOrgID:    order.CarrierOrgID,
			NotificationType:  values.TypeOrderCompleted,
			Title:             title,
			Body:              body,
			Link:              link,
			EntityType:        values.EntityOrder,
			EntityID:          order.ID,
		},
	}, nil
}
