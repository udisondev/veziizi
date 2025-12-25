package freightrequest

import (
	"context"
	"fmt"

	frEvents "codeberg.org/udison/veziizi/backend/internal/domain/freightrequest/events"
	"codeberg.org/udison/veziizi/backend/internal/domain/notification/rules"
	"codeberg.org/udison/veziizi/backend/internal/domain/notification/values"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
)

// OfferDeclinedRule уведомляет заказчика что перевозчик отклонил выбор
type OfferDeclinedRule struct {
	deps rules.Dependencies
}

// NewOfferDeclinedRule создает правило
func NewOfferDeclinedRule(deps rules.Dependencies) *OfferDeclinedRule {
	return &OfferDeclinedRule{deps: deps}
}

func (r *OfferDeclinedRule) EventType() string {
	return frEvents.TypeOfferDeclined
}

func (r *OfferDeclinedRule) Process(ctx context.Context, event eventstore.Event) ([]rules.NotificationRequest, error) {
	e, ok := event.(frEvents.OfferDeclined)
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
			NotificationType:  values.TypeOfferDeclined,
			Title:             "Предложение отклонено",
			Body:              fmt.Sprintf("Перевозчик отклонил выбранное предложение по заявке #%d", fr.RequestNumber),
			Link:              fmt.Sprintf("/freight-requests/%s", fr.ID),
			EntityType:        values.EntityFreightRequest,
			EntityID:          fr.ID,
		},
	}, nil
}
