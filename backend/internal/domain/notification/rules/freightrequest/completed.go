package freightrequest

import (
	"context"
	"fmt"

	frEvents "github.com/udisondev/veziizi/backend/internal/domain/freightrequest/events"
	"github.com/udisondev/veziizi/backend/internal/domain/notification/rules"
	"github.com/udisondev/veziizi/backend/internal/domain/notification/values"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
)

// FreightRequestCompletedRule уведомляет обе стороны что перевозка завершена
type FreightRequestCompletedRule struct {
	deps rules.Dependencies
}

// NewFreightRequestCompletedRule создает правило
func NewFreightRequestCompletedRule(deps rules.Dependencies) *FreightRequestCompletedRule {
	return &FreightRequestCompletedRule{deps: deps}
}

func (r *FreightRequestCompletedRule) EventType() string {
	return frEvents.TypeFreightRequestCompleted
}

func (r *FreightRequestCompletedRule) Process(ctx context.Context, event eventstore.Event) ([]rules.NotificationRequest, error) {
	e, ok := event.(frEvents.FreightRequestCompleted)
	if !ok {
		return nil, fmt.Errorf("unexpected event type: %T", event)
	}

	fr, err := r.deps.FreightRequests.GetByID(ctx, e.AggregateID())
	if err != nil {
		return nil, fmt.Errorf("get freight request: %w", err)
	}
	if fr == nil {
		return nil, nil
	}

	var notifications []rules.NotificationRequest

	// Уведомляем заказчика
	if fr.CustomerMemberID != nil {
		notifications = append(notifications, rules.NotificationRequest{
			RecipientMemberID: *fr.CustomerMemberID,
			RecipientOrgID:    fr.CustomerOrgID,
			NotificationType:  values.TypeFreightCompleted,
			Title:             "Перевозка завершена",
			Body:              fmt.Sprintf("Заявка #%d успешно завершена", fr.RequestNumber),
			Link:              fmt.Sprintf("/freight-requests/%s", fr.ID),
			EntityType:        values.EntityFreightRequest,
			EntityID:          fr.ID,
		})
	}

	// Уведомляем перевозчика
	if fr.CarrierMemberID != nil {
		notifications = append(notifications, rules.NotificationRequest{
			RecipientMemberID: *fr.CarrierMemberID,
			RecipientOrgID:    *fr.CarrierOrgID,
			NotificationType:  values.TypeFreightCompleted,
			Title:             "Перевозка завершена",
			Body:              fmt.Sprintf("Заявка #%d успешно завершена", fr.RequestNumber),
			Link:              fmt.Sprintf("/freight-requests/%s", fr.ID),
			EntityType:        values.EntityFreightRequest,
			EntityID:          fr.ID,
		})
	}

	return notifications, nil
}
