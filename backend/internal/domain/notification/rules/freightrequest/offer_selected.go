package freightrequest

import (
	"context"
	"fmt"

	frEvents "github.com/udisondev/veziizi/backend/internal/domain/freightrequest/events"
	"github.com/udisondev/veziizi/backend/internal/domain/notification/rules"
	"github.com/udisondev/veziizi/backend/internal/domain/notification/values"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
)

// OfferSelectedRule уведомляет перевозчика что его предложение выбрано
type OfferSelectedRule struct {
	deps rules.Dependencies
}

// NewOfferSelectedRule создает правило
func NewOfferSelectedRule(deps rules.Dependencies) *OfferSelectedRule {
	return &OfferSelectedRule{deps: deps}
}

func (r *OfferSelectedRule) EventType() string {
	return frEvents.TypeOfferSelected
}

func (r *OfferSelectedRule) Process(ctx context.Context, event eventstore.Event) ([]rules.NotificationRequest, error) {
	e, ok := event.(frEvents.OfferSelected)
	if !ok {
		return nil, fmt.Errorf("unexpected event type: %T", event)
	}

	// Получаем offer для carrier_member_id
	offer, err := r.deps.FreightRequests.GetOfferByID(ctx, e.OfferID)
	if err != nil {
		return nil, fmt.Errorf("get offer: %w", err)
	}
	if offer == nil || offer.CarrierMemberID == nil {
		return nil, nil // Projection ещё не обновлена или нет получателя
	}

	// Получаем заявку для номера
	fr, err := r.deps.FreightRequests.GetByID(ctx, e.AggregateID())
	if err != nil || fr == nil {
		return nil, nil
	}

	return []rules.NotificationRequest{
		{
			RecipientMemberID: *offer.CarrierMemberID,
			RecipientOrgID:    offer.CarrierOrgID,
			NotificationType:  values.TypeOfferSelected,
			Title:             "Предложение выбрано",
			Body:              fmt.Sprintf("Ваше предложение по заявке #%d выбрано", fr.RequestNumber),
			Link:              fmt.Sprintf("/freight-requests/%s", fr.ID),
			EntityType:        values.EntityFreightRequest,
			EntityID:          fr.ID,
		},
	}, nil
}
