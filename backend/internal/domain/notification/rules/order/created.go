package order

import (
	"context"
	"fmt"

	"codeberg.org/udison/veziizi/backend/internal/domain/notification/rules"
	"codeberg.org/udison/veziizi/backend/internal/domain/notification/values"
	orderEvents "codeberg.org/udison/veziizi/backend/internal/domain/order/events"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
)

// OrderCreatedRule уведомляет обе стороны о создании заказа
// Все данные берутся из события - не требует lookup
type OrderCreatedRule struct{}

// NewOrderCreatedRule создает правило
func NewOrderCreatedRule() *OrderCreatedRule {
	return &OrderCreatedRule{}
}

func (r *OrderCreatedRule) EventType() string {
	return orderEvents.TypeOrderCreated
}

func (r *OrderCreatedRule) Process(ctx context.Context, event eventstore.Event) ([]rules.NotificationRequest, error) {
	e, ok := event.(orderEvents.OrderCreated)
	if !ok {
		return nil, fmt.Errorf("unexpected event type: %T", event)
	}

	// Все данные есть в событии
	title := "Заказ создан"
	body := fmt.Sprintf("Создан заказ #%d", e.OrderNumber)
	link := fmt.Sprintf("/orders/%s", e.AggregateID())

	return []rules.NotificationRequest{
		// Заказчику
		{
			RecipientMemberID: e.CustomerMemberID,
			RecipientOrgID:    e.CustomerOrgID,
			NotificationType:  values.TypeOrderCreated,
			Title:             title,
			Body:              body,
			Link:              link,
			EntityType:        values.EntityOrder,
			EntityID:          e.AggregateID(),
		},
		// Перевозчику
		{
			RecipientMemberID: e.CarrierMemberID,
			RecipientOrgID:    e.CarrierOrgID,
			NotificationType:  values.TypeOrderCreated,
			Title:             title,
			Body:              body,
			Link:              link,
			EntityType:        values.EntityOrder,
			EntityID:          e.AggregateID(),
		},
	}, nil
}
