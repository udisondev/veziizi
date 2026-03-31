package rules

import (
	"context"

	frValues "github.com/udisondev/veziizi/backend/internal/domain/freightrequest/values"
	"github.com/google/uuid"
)

// Dependencies зависимости для правил уведомлений
// Использует интерфейсы для IoC - не прямые зависимости от projections
type Dependencies struct {
	FreightRequests FreightRequestGetter
	Members         MemberGetter
}

// FreightRequestGetter интерфейс для получения данных о заявках
type FreightRequestGetter interface {
	GetByID(ctx context.Context, id uuid.UUID) (*FreightRequestInfo, error)
	GetOfferByID(ctx context.Context, id uuid.UUID) (*OfferInfo, error)
}

// MemberGetter интерфейс для получения данных о членах
type MemberGetter interface {
	GetByID(ctx context.Context, id uuid.UUID) (*MemberInfo, error)
}

// SubscriptionMatcher находит подписки, соответствующие заявке (opt-in модель)
type SubscriptionMatcher interface {
	FindMatchingSubscriptions(ctx context.Context, data frValues.FreightRequestMatchData, excludeMemberID uuid.UUID) ([]frValues.MatchedSubscription, error)
}

// FreightRequestInfo минимальные данные заявки для уведомлений
type FreightRequestInfo struct {
	ID               uuid.UUID
	RequestNumber    int64
	CustomerMemberID *uuid.UUID
	CustomerOrgID    uuid.UUID
	// Carrier fields (populated after offer confirmed)
	CarrierMemberID *uuid.UUID
	CarrierOrgID    *uuid.UUID
}

// OfferInfo минимальные данные оффера для уведомлений
type OfferInfo struct {
	ID               uuid.UUID
	FreightRequestID uuid.UUID
	CarrierMemberID  *uuid.UUID
	CarrierOrgID     uuid.UUID
}

// MemberInfo минимальные данные члена для уведомлений
type MemberInfo struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
}
