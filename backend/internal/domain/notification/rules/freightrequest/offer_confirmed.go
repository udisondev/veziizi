package freightrequest

import (
	"context"
	"fmt"

	frEvents "github.com/udisondev/veziizi/backend/internal/domain/freightrequest/events"
	"github.com/udisondev/veziizi/backend/internal/domain/notification/rules"
	"github.com/udisondev/veziizi/backend/internal/domain/notification/values"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
)

// OfferConfirmedRule уведомляет заказчика что предложение подтверждено
type OfferConfirmedRule struct {
	deps rules.Dependencies
}

// NewOfferConfirmedRule создает правило
func NewOfferConfirmedRule(deps rules.Dependencies) *OfferConfirmedRule {
	return &OfferConfirmedRule{deps: deps}
}

func (r *OfferConfirmedRule) EventType() string {
	return frEvents.TypeOfferConfirmed
}

func (r *OfferConfirmedRule) Process(ctx context.Context, event eventstore.Event) ([]rules.NotificationRequest, error) {
	e, ok := event.(frEvents.OfferConfirmed)
	if !ok {
		return nil, fmt.Errorf("unexpected event type: %T", event)
	}

	fr, err := r.deps.FreightRequests.GetByID(ctx, e.AggregateID())
	if err != nil {
		return nil, fmt.Errorf("get freight request: %w", err)
	}
	if fr == nil || fr.CustomerMemberID == nil {
		return nil, nil
	}

	return []rules.NotificationRequest{
		{
			RecipientMemberID: *fr.CustomerMemberID,
			RecipientOrgID:    fr.CustomerOrgID,
			NotificationType:  values.TypeOfferConfirmed,
			Title:             "Предложение подтверждено",
			Body:              fmt.Sprintf("По заявке #%d создан заказ", fr.RequestNumber),
			Link:              fmt.Sprintf("/freight-requests/%s", fr.ID),
			EntityType:        values.EntityFreightRequest,
			EntityID:          fr.ID,
		},
	}, nil
}
