package freightrequest

import (
	"context"
	"fmt"

	frEvents "codeberg.org/udison/veziizi/backend/internal/domain/freightrequest/events"
	"codeberg.org/udison/veziizi/backend/internal/domain/notification/rules"
	"codeberg.org/udison/veziizi/backend/internal/domain/notification/values"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
)

// OfferWithdrawnRule уведомляет заказчика что предложение отозвано
type OfferWithdrawnRule struct {
	deps rules.Dependencies
}

// NewOfferWithdrawnRule создает правило
func NewOfferWithdrawnRule(deps rules.Dependencies) *OfferWithdrawnRule {
	return &OfferWithdrawnRule{deps: deps}
}

func (r *OfferWithdrawnRule) EventType() string {
	return frEvents.TypeOfferWithdrawn
}

func (r *OfferWithdrawnRule) Process(ctx context.Context, event eventstore.Event) ([]rules.NotificationRequest, error) {
	e, ok := event.(frEvents.OfferWithdrawn)
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
			NotificationType:  values.TypeOfferWithdrawn,
			Title:             "Предложение отозвано",
			Body:              fmt.Sprintf("Перевозчик отозвал своё предложение по заявке #%d", fr.RequestNumber),
			Link:              fmt.Sprintf("/freight-requests/%s", fr.ID),
			EntityType:        values.EntityFreightRequest,
			EntityID:          fr.ID,
		},
	}, nil
}
