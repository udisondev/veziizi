package rules

import (
	"context"

	frValues "github.com/udisondev/veziizi/backend/internal/domain/freightrequest/values"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/projections"
	"github.com/google/uuid"
)

// SubscriptionMatcher находит подписки, соответствующие заявке (opt-in модель)
type SubscriptionMatcher interface {
	FindMatchingSubscriptions(ctx context.Context, data frValues.FreightRequestMatchData, excludeMemberID uuid.UUID) ([]frValues.MatchedSubscription, error)
}

// FreightSubscriptionsAdapter адаптирует FreightSubscriptionsProjection (opt-in модель)
type FreightSubscriptionsAdapter struct {
	projection *projections.FreightSubscriptionsProjection
}

// NewFreightSubscriptionsAdapter создает адаптер
func NewFreightSubscriptionsAdapter(projection *projections.FreightSubscriptionsProjection) *FreightSubscriptionsAdapter {
	return &FreightSubscriptionsAdapter{projection: projection}
}

func (a *FreightSubscriptionsAdapter) FindMatchingSubscriptions(ctx context.Context, data frValues.FreightRequestMatchData, excludeMemberID uuid.UUID) ([]frValues.MatchedSubscription, error) {
	return a.projection.FindMatchingSubscriptions(ctx, data, excludeMemberID)
}

// FreightRequestsAdapter адаптирует FreightRequestsProjection к FreightRequestGetter
type FreightRequestsAdapter struct {
	projection *projections.FreightRequestsProjection
}

// NewFreightRequestsAdapter создает адаптер
func NewFreightRequestsAdapter(projection *projections.FreightRequestsProjection) *FreightRequestsAdapter {
	return &FreightRequestsAdapter{projection: projection}
}

func (a *FreightRequestsAdapter) GetByID(ctx context.Context, id uuid.UUID) (*FreightRequestInfo, error) {
	item, err := a.projection.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, nil
	}
	return &FreightRequestInfo{
		ID:               item.ID,
		RequestNumber:    item.RequestNumber,
		CustomerMemberID: item.CustomerMemberID,
		CustomerOrgID:    item.CustomerOrgID,
		CarrierMemberID:  item.CarrierMemberID,
		CarrierOrgID:     item.CarrierOrgID,
	}, nil
}

func (a *FreightRequestsAdapter) GetOfferByID(ctx context.Context, id uuid.UUID) (*OfferInfo, error) {
	item, err := a.projection.GetOfferByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, nil
	}
	return &OfferInfo{
		ID:               item.ID,
		FreightRequestID: item.FreightRequestID,
		CarrierMemberID:  item.CarrierMemberID,
		CarrierOrgID:     item.CarrierOrgID,
	}, nil
}

// MembersAdapter адаптирует MembersProjection к MemberGetter
type MembersAdapter struct {
	projection *projections.MembersProjection
}

// NewMembersAdapter создает адаптер
func NewMembersAdapter(projection *projections.MembersProjection) *MembersAdapter {
	return &MembersAdapter{projection: projection}
}

func (a *MembersAdapter) GetByID(ctx context.Context, id uuid.UUID) (*MemberInfo, error) {
	item, err := a.projection.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, nil
	}
	return &MemberInfo{
		ID:             item.ID,
		OrganizationID: item.OrganizationID,
	}, nil
}
