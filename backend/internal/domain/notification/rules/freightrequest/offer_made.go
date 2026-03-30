package freightrequest

import (
	"context"
	"fmt"

	frEvents "github.com/udisondev/veziizi/backend/internal/domain/freightrequest/events"
	"github.com/udisondev/veziizi/backend/internal/domain/notification/rules"
	"github.com/udisondev/veziizi/backend/internal/domain/notification/values"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
)

// OfferMadeRule уведомляет заказчика о новом предложении
type OfferMadeRule struct {
	deps rules.Dependencies
}

// NewOfferMadeRule создает правило
func NewOfferMadeRule(deps rules.Dependencies) *OfferMadeRule {
	return &OfferMadeRule{deps: deps}
}

func (r *OfferMadeRule) EventType() string {
	return frEvents.TypeOfferMade
}

func (r *OfferMadeRule) Process(ctx context.Context, event eventstore.Event) ([]rules.NotificationRequest, error) {
	e, ok := event.(frEvents.OfferMade)
	if !ok {
		return nil, fmt.Errorf("unexpected event type: %T", event)
	}

	// Получаем заявку для уведомления заказчика
	fr, err := r.deps.FreightRequests.GetByID(ctx, e.AggregateID())
	if err != nil {
		return nil, fmt.Errorf("get freight request: %w", err)
	}
	if fr == nil || fr.CustomerMemberID == nil {
		return nil, nil // Пропускаем, нет получателя
	}

	return []rules.NotificationRequest{
		{
			RecipientMemberID: *fr.CustomerMemberID,
			RecipientOrgID:    fr.CustomerOrgID,
			NotificationType:  values.TypeNewOffer,
			Title:             "Новое предложение",
			Body:              fmt.Sprintf("На заявку #%d поступило новое предложение", fr.RequestNumber),
			Link:              fmt.Sprintf("/freight-requests/%s", fr.ID),
			EntityType:        values.EntityFreightRequest,
			EntityID:          fr.ID,
		},
	}, nil
}
