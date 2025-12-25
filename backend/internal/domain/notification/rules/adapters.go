package rules

import (
	"context"

	"codeberg.org/udison/veziizi/backend/internal/infrastructure/projections"
	"github.com/google/uuid"
)

// SubscribedMembersResolver находит подписчиков для уведомлений о заявках
type SubscribedMembersResolver interface {
	GetSubscribedMembers(ctx context.Context, filter SubscriptionFilter, excludeMemberID uuid.UUID) ([]SubscriberResult, error)
}

// SubscriptionFilter фильтры для matching подписок
type SubscriptionFilter struct {
	OriginCountryID      *int
	DestinationCountryID *int
	CargoType            string
	CargoWeight          float64
	BodyTypes            []string
}

// SubscriberResult результат поиска подписчика
type SubscriberResult struct {
	MemberID       uuid.UUID
	OrganizationID uuid.UUID
}

// SubscriptionsAdapter адаптирует FreightRequestSubscriptionsProjection
type SubscriptionsAdapter struct {
	projection *projections.FreightRequestSubscriptionsProjection
}

// NewSubscriptionsAdapter создает адаптер
func NewSubscriptionsAdapter(projection *projections.FreightRequestSubscriptionsProjection) *SubscriptionsAdapter {
	return &SubscriptionsAdapter{projection: projection}
}

func (a *SubscriptionsAdapter) GetSubscribedMembers(ctx context.Context, filter SubscriptionFilter, excludeMemberID uuid.UUID) ([]SubscriberResult, error) {
	// Конвертируем типы для projection
	projFilter := projections.SubscriptionFilter{
		OriginCountryID:      filter.OriginCountryID,
		DestinationCountryID: filter.DestinationCountryID,
		CargoType:            filter.CargoType,
		CargoWeight:          filter.CargoWeight,
		BodyTypes:            filter.BodyTypes,
	}

	subs, err := a.projection.GetSubscribedMembers(ctx, projFilter, excludeMemberID)
	if err != nil {
		return nil, err
	}

	result := make([]SubscriberResult, len(subs))
	for i, sub := range subs {
		result[i] = SubscriberResult{
			MemberID:       sub.MemberID,
			OrganizationID: sub.OrganizationID,
		}
	}
	return result, nil
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

// OrdersAdapter адаптирует OrdersProjection к OrderGetter
type OrdersAdapter struct {
	projection *projections.OrdersProjection
}

// NewOrdersAdapter создает адаптер
func NewOrdersAdapter(projection *projections.OrdersProjection) *OrdersAdapter {
	return &OrdersAdapter{projection: projection}
}

func (a *OrdersAdapter) GetByID(ctx context.Context, id uuid.UUID) (*OrderInfo, error) {
	item, err := a.projection.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, nil
	}
	return &OrderInfo{
		ID:               item.ID,
		OrderNumber:      item.OrderNumber,
		CustomerMemberID: item.CustomerMemberID,
		CustomerOrgID:    item.CustomerOrgID,
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
