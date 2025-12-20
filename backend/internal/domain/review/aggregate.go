package review

import (
	"errors"
	"time"

	"codeberg.org/udison/veziizi/backend/internal/domain/review/events"
	"codeberg.org/udison/veziizi/backend/internal/domain/review/values"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"codeberg.org/udison/veziizi/backend/internal/pkg/aggregate"
	"github.com/google/uuid"
)

// Errors
var (
	ErrReviewAlreadyAnalyzed   = errors.New("review already analyzed")
	ErrReviewNotPendingMod     = errors.New("review is not pending moderation")
	ErrReviewNotApproved       = errors.New("review is not approved")
	ErrReviewAlreadyActive     = errors.New("review is already active")
	ErrReviewTerminalStatus    = errors.New("review is in terminal status")
	ErrActivationDateNotPassed = errors.New("activation date has not passed yet")
)

// Review aggregate represents a review with fraud detection and weighted rating
type Review struct {
	aggregate.Base

	orderID          uuid.UUID
	reviewerOrgID    uuid.UUID
	reviewedOrgID    uuid.UUID
	rating           int
	comment          string
	orderAmount      int64
	orderCurrency    string
	orderCreatedAt   time.Time
	orderCompletedAt time.Time

	// Weight calculation
	rawWeight   float64
	finalWeight float64

	// Fraud detection
	fraudSignals []events.FraudSignal
	fraudScore   float64

	// Status
	status             values.ReviewStatus
	requiresModeration bool

	// Timing
	activationDate *time.Time
	createdAt      time.Time
	analyzedAt     *time.Time
	moderatedAt    *time.Time
	moderatedBy    *uuid.UUID
	activatedAt    *time.Time
}

// New creates a new Review from Order.ReviewLeft event data
func New(
	id uuid.UUID,
	orderID uuid.UUID,
	reviewerOrgID uuid.UUID,
	reviewedOrgID uuid.UUID,
	rating int,
	comment string,
	orderAmount int64,
	orderCurrency string,
	orderCreatedAt time.Time,
	orderCompletedAt time.Time,
) *Review {
	r := &Review{
		Base: aggregate.NewBase(id),
	}

	r.Apply(events.ReviewReceived{
		BaseEvent:        eventstore.NewBaseEvent(id, events.AggregateType, r.Version()+1),
		OrderID:          orderID,
		ReviewerOrgID:    reviewerOrgID,
		ReviewedOrgID:    reviewedOrgID,
		Rating:           rating,
		Comment:          comment,
		OrderAmount:      orderAmount,
		OrderCurrency:    orderCurrency,
		OrderCreatedAt:   orderCreatedAt,
		OrderCompletedAt: orderCompletedAt,
	})

	return r
}

// NewFromEvents reconstructs Review from events
func NewFromEvents(id uuid.UUID, evts []eventstore.Event) *Review {
	r := &Review{
		Base: aggregate.NewBase(id),
	}

	for _, evt := range evts {
		r.apply(evt)
		r.Replay(evt)
	}

	return r
}

// Getters
func (r *Review) OrderID() uuid.UUID           { return r.orderID }
func (r *Review) ReviewerOrgID() uuid.UUID     { return r.reviewerOrgID }
func (r *Review) ReviewedOrgID() uuid.UUID     { return r.reviewedOrgID }
func (r *Review) Rating() int                  { return r.rating }
func (r *Review) Comment() string              { return r.comment }
func (r *Review) OrderAmount() int64           { return r.orderAmount }
func (r *Review) OrderCurrency() string        { return r.orderCurrency }
func (r *Review) OrderCreatedAt() time.Time    { return r.orderCreatedAt }
func (r *Review) OrderCompletedAt() time.Time  { return r.orderCompletedAt }
func (r *Review) RawWeight() float64           { return r.rawWeight }
func (r *Review) FinalWeight() float64         { return r.finalWeight }
func (r *Review) FraudSignals() []events.FraudSignal { return r.fraudSignals }
func (r *Review) FraudScore() float64          { return r.fraudScore }
func (r *Review) Status() values.ReviewStatus  { return r.status }
func (r *Review) RequiresModeration() bool     { return r.requiresModeration }
func (r *Review) ActivationDate() *time.Time   { return r.activationDate }
func (r *Review) CreatedAt() time.Time         { return r.createdAt }
func (r *Review) AnalyzedAt() *time.Time       { return r.analyzedAt }
func (r *Review) ModeratedAt() *time.Time      { return r.moderatedAt }
func (r *Review) ModeratedBy() *uuid.UUID      { return r.moderatedBy }
func (r *Review) ActivatedAt() *time.Time      { return r.activatedAt }

// Commands

// RecordAnalysis records the fraud analysis results
func (r *Review) RecordAnalysis(
	rawWeight float64,
	fraudSignals []events.FraudSignal,
	fraudScore float64,
	requiresModeration bool,
	activationDate time.Time,
) error {
	if r.status != values.StatusPendingAnalysis {
		return ErrReviewAlreadyAnalyzed
	}

	r.Apply(events.ReviewAnalyzed{
		BaseEvent:          eventstore.NewBaseEvent(r.ID(), events.AggregateType, r.Version()+1),
		RawWeight:          rawWeight,
		FraudSignals:       fraudSignals,
		FraudScore:         fraudScore,
		RequiresModeration: requiresModeration,
		ActivationDate:     activationDate,
	})

	// Auto-approve if no moderation required
	if !requiresModeration {
		r.Apply(events.ReviewApproved{
			BaseEvent:   eventstore.NewBaseEvent(r.ID(), events.AggregateType, r.Version()+1),
			ApprovedBy:  nil, // auto-approved
			FinalWeight: rawWeight,
			Note:        "auto-approved: no fraud signals detected",
		})
	}

	return nil
}

// Approve approves the review (by moderator)
func (r *Review) Approve(moderatorID uuid.UUID, finalWeight float64, note string) error {
	if r.status != values.StatusPendingModeration {
		return ErrReviewNotPendingMod
	}

	r.Apply(events.ReviewApproved{
		BaseEvent:   eventstore.NewBaseEvent(r.ID(), events.AggregateType, r.Version()+1),
		ApprovedBy:  &moderatorID,
		FinalWeight: finalWeight,
		Note:        note,
	})

	return nil
}

// Reject rejects the review (by moderator)
func (r *Review) Reject(moderatorID uuid.UUID, reason string) error {
	if r.status != values.StatusPendingModeration {
		return ErrReviewNotPendingMod
	}

	r.Apply(events.ReviewRejected{
		BaseEvent:  eventstore.NewBaseEvent(r.ID(), events.AggregateType, r.Version()+1),
		RejectedBy: moderatorID,
		Reason:     reason,
	})

	return nil
}

// Activate activates the review (starts affecting rating)
func (r *Review) Activate() error {
	if r.status.IsTerminal() {
		return ErrReviewTerminalStatus
	}
	if r.status != values.StatusApproved {
		return ErrReviewNotApproved
	}
	if r.activationDate != nil && time.Now().Before(*r.activationDate) {
		return ErrActivationDateNotPassed
	}

	r.Apply(events.ReviewActivated{
		BaseEvent:   eventstore.NewBaseEvent(r.ID(), events.AggregateType, r.Version()+1),
		FinalWeight: r.finalWeight,
	})

	return nil
}

// Deactivate deactivates the review (e.g., reviewer marked as fraudster)
func (r *Review) Deactivate(reason string) error {
	if r.status.IsTerminal() {
		return ErrReviewTerminalStatus
	}

	r.Apply(events.ReviewDeactivated{
		BaseEvent: eventstore.NewBaseEvent(r.ID(), events.AggregateType, r.Version()+1),
		Reason:    reason,
	})

	return nil
}

// Apply applies event and records it as change
func (r *Review) Apply(evt eventstore.Event) {
	r.apply(evt)
	r.Base.Apply(evt)
}

// apply updates state from event (used by both Apply and Replay)
func (r *Review) apply(evt eventstore.Event) {
	switch e := evt.(type) {
	case events.ReviewReceived:
		r.orderID = e.OrderID
		r.reviewerOrgID = e.ReviewerOrgID
		r.reviewedOrgID = e.ReviewedOrgID
		r.rating = e.Rating
		r.comment = e.Comment
		r.orderAmount = e.OrderAmount
		r.orderCurrency = e.OrderCurrency
		r.orderCreatedAt = e.OrderCreatedAt
		r.orderCompletedAt = e.OrderCompletedAt
		r.status = values.StatusPendingAnalysis
		r.createdAt = e.OccurredAt()

	case events.ReviewAnalyzed:
		r.rawWeight = e.RawWeight
		r.fraudSignals = e.FraudSignals
		r.fraudScore = e.FraudScore
		r.requiresModeration = e.RequiresModeration
		r.activationDate = &e.ActivationDate
		now := e.OccurredAt()
		r.analyzedAt = &now
		if e.RequiresModeration {
			r.status = values.StatusPendingModeration
		}
		// Note: if not requiring moderation, ReviewApproved will follow immediately

	case events.ReviewApproved:
		r.finalWeight = e.FinalWeight
		now := e.OccurredAt()
		r.moderatedAt = &now
		r.moderatedBy = e.ApprovedBy
		r.status = values.StatusApproved

	case events.ReviewRejected:
		now := e.OccurredAt()
		r.moderatedAt = &now
		r.moderatedBy = &e.RejectedBy
		r.status = values.StatusRejected

	case events.ReviewActivated:
		r.finalWeight = e.FinalWeight
		now := e.OccurredAt()
		r.activatedAt = &now
		r.status = values.StatusActive

	case events.ReviewDeactivated:
		r.status = values.StatusDeactivated
	}
}
