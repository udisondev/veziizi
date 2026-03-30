package freightrequest

import (
	"context"
	"fmt"

	frEvents "github.com/udisondev/veziizi/backend/internal/domain/freightrequest/events"
	"github.com/udisondev/veziizi/backend/internal/domain/notification/rules"
	"github.com/udisondev/veziizi/backend/internal/domain/notification/values"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
)

// OfferRejectedRule уведомляет перевозчика что его предложение отклонено
type OfferRejectedRule struct {
	deps rules.Dependencies
}

// NewOfferRejectedRule создает правило
func NewOfferRejectedRule(deps rules.Dependencies) *OfferRejectedRule {
	return &OfferRejectedRule{deps: deps}
}

func (r *OfferRejectedRule) EventType() string {
	return frEvents.TypeOfferRejected
}

func (r *OfferRejectedRule) Process(ctx context.Context, event eventstore.Event) ([]rules.NotificationRequest, error) {
	e, ok := event.(frEvents.OfferRejected)
	if !ok {
		return nil, fmt.Errorf("unexpected event type: %T", event)
	}

	// Получаем offer для carrier_member_id
	offer, err := r.deps.FreightRequests.GetOfferByID(ctx, e.OfferID)
	if err != nil {
		return nil, fmt.Errorf("get offer: %w", err)
	}
	if offer == nil || offer.CarrierMemberID == nil {
		return nil, nil
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
			NotificationType:  values.TypeOfferRejected,
			Title:             "Предложение отклонено",
			Body:              fmt.Sprintf("Ваше предложение по заявке #%d отклонено", fr.RequestNumber),
			Link:              fmt.Sprintf("/freight-requests/%s", fr.ID),
			EntityType:        values.EntityFreightRequest,
			EntityID:          fr.ID,
		},
	}, nil
}
