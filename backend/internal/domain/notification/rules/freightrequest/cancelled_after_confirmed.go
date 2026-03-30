package freightrequest

import (
	"context"
	"fmt"

	frEvents "github.com/udisondev/veziizi/backend/internal/domain/freightrequest/events"
	"github.com/udisondev/veziizi/backend/internal/domain/notification/rules"
	"github.com/udisondev/veziizi/backend/internal/domain/notification/values"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
)

// CancelledAfterConfirmedRule уведомляет обе стороны об отмене перевозки после подтверждения
type CancelledAfterConfirmedRule struct {
	deps rules.Dependencies
}

// NewCancelledAfterConfirmedRule создает правило
func NewCancelledAfterConfirmedRule(deps rules.Dependencies) *CancelledAfterConfirmedRule {
	return &CancelledAfterConfirmedRule{deps: deps}
}

func (r *CancelledAfterConfirmedRule) EventType() string {
	return frEvents.TypeCancelledAfterConfirmed
}

func (r *CancelledAfterConfirmedRule) Process(ctx context.Context, event eventstore.Event) ([]rules.NotificationRequest, error) {
	e, ok := event.(frEvents.CancelledAfterConfirmed)
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

	// Определяем кто отменил (по роли из события)
	cancellerText := "перевозчиком"
	if e.CancelledRole == "customer" {
		cancellerText = "заказчиком"
	}

	// Уведомляем заказчика (если не он отменил)
	if fr.CustomerMemberID != nil && e.CancelledRole != "customer" {
		notifications = append(notifications, rules.NotificationRequest{
			RecipientMemberID: *fr.CustomerMemberID,
			RecipientOrgID:    fr.CustomerOrgID,
			NotificationType:  values.TypeFreightCancelledConfirmed,
			Title:             "Перевозка отменена",
			Body:              fmt.Sprintf("Заявка #%d отменена %s", fr.RequestNumber, cancellerText),
			Link:              fmt.Sprintf("/freight-requests/%s", fr.ID),
			EntityType:        values.EntityFreightRequest,
			EntityID:          fr.ID,
		})
	}

	// Уведомляем перевозчика (если не он отменил)
	if fr.CarrierMemberID != nil && fr.CarrierOrgID != nil && e.CancelledRole != "carrier" {
		notifications = append(notifications, rules.NotificationRequest{
			RecipientMemberID: *fr.CarrierMemberID,
			RecipientOrgID:    *fr.CarrierOrgID,
			NotificationType:  values.TypeFreightCancelledConfirmed,
			Title:             "Перевозка отменена",
			Body:              fmt.Sprintf("Заявка #%d отменена %s", fr.RequestNumber, cancellerText),
			Link:              fmt.Sprintf("/freight-requests/%s", fr.ID),
			EntityType:        values.EntityFreightRequest,
			EntityID:          fr.ID,
		})
	}

	return notifications, nil
}
