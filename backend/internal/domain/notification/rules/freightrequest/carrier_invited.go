package freightrequest

import (
	"context"
	"fmt"

	frEvents "github.com/udisondev/veziizi/backend/internal/domain/freightrequest/events"
	"github.com/udisondev/veziizi/backend/internal/domain/notification/rules"
	"github.com/udisondev/veziizi/backend/internal/domain/notification/values"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
)

// CarrierInvitedRule fans out a notification to every active member of the
// carrier organization when the customer pings them about a freight request.
type CarrierInvitedRule struct {
	deps rules.Dependencies
}

func NewCarrierInvitedRule(deps rules.Dependencies) *CarrierInvitedRule {
	return &CarrierInvitedRule{deps: deps}
}

func (r *CarrierInvitedRule) EventType() string {
	return frEvents.TypeCarrierInvited
}

func (r *CarrierInvitedRule) Process(ctx context.Context, event eventstore.Event) ([]rules.NotificationRequest, error) {
	e, ok := event.(frEvents.CarrierInvited)
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

	memberIDs, err := r.deps.Members.ListActiveByOrg(ctx, e.CarrierOrgID)
	if err != nil {
		return nil, fmt.Errorf("list carrier members: %w", err)
	}
	if len(memberIDs) == 0 {
		return nil, nil
	}

	requests := make([]rules.NotificationRequest, 0, len(memberIDs))
	for _, mid := range memberIDs {
		requests = append(requests, rules.NotificationRequest{
			RecipientMemberID: mid,
			RecipientOrgID:    e.CarrierOrgID,
			NotificationType:  values.TypeFreightInvitation,
			Title:             "Вам предложили заявку",
			Body:              fmt.Sprintf("Заказчик предложил вам заявку #%d", fr.RequestNumber),
			Link:              fmt.Sprintf("/freight-requests/%s", fr.ID),
			EntityType:        values.EntityFreightRequest,
			EntityID:          fr.ID,
		})
	}
	return requests, nil
}
