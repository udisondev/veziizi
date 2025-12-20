package events

import (
	"time"

	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"github.com/google/uuid"
)

const AggregateType = "review"

// Event type constants
const (
	TypeReviewReceived    = "review.received"
	TypeReviewAnalyzed    = "review.analyzed"
	TypeReviewApproved    = "review.approved"
	TypeReviewRejected    = "review.rejected"
	TypeReviewActivated   = "review.activated"
	TypeReviewDeactivated = "review.deactivated"
)

func init() {
	eventstore.RegisterEventType[ReviewReceived](TypeReviewReceived)
	eventstore.RegisterEventType[ReviewAnalyzed](TypeReviewAnalyzed)
	eventstore.RegisterEventType[ReviewApproved](TypeReviewApproved)
	eventstore.RegisterEventType[ReviewRejected](TypeReviewRejected)
	eventstore.RegisterEventType[ReviewActivated](TypeReviewActivated)
	eventstore.RegisterEventType[ReviewDeactivated](TypeReviewDeactivated)
}

// FraudSignal represents a detected anomaly in the review
type FraudSignal struct {
	Type        string  `json:"type"`
	Severity    string  `json:"severity"`
	Description string  `json:"description"`
	ScoreImpact float64 `json:"score_impact"`
	Evidence    string  `json:"evidence,omitempty"`
}

// ReviewReceived is emitted when a review is created from Order.ReviewLeft
type ReviewReceived struct {
	eventstore.BaseEvent
	OrderID          uuid.UUID `json:"order_id"`
	ReviewerOrgID    uuid.UUID `json:"reviewer_org_id"`
	ReviewedOrgID    uuid.UUID `json:"reviewed_org_id"`
	Rating           int       `json:"rating"`
	Comment          string    `json:"comment,omitempty"`
	OrderAmount      int64     `json:"order_amount"`
	OrderCurrency    string    `json:"order_currency"`
	OrderCreatedAt   time.Time `json:"order_created_at"`
	OrderCompletedAt time.Time `json:"order_completed_at"`
}

func (e ReviewReceived) EventType() string { return TypeReviewReceived }

// ReviewAnalyzed is emitted after fraud analysis and weight calculation
type ReviewAnalyzed struct {
	eventstore.BaseEvent
	RawWeight          float64       `json:"raw_weight"`
	FraudSignals       []FraudSignal `json:"fraud_signals,omitempty"`
	FraudScore         float64       `json:"fraud_score"`
	RequiresModeration bool          `json:"requires_moderation"`
	ActivationDate     time.Time     `json:"activation_date"`
}

func (e ReviewAnalyzed) EventType() string { return TypeReviewAnalyzed }

// ReviewApproved is emitted when moderator approves the review
// or when review is auto-approved (no fraud signals)
type ReviewApproved struct {
	eventstore.BaseEvent
	ApprovedBy  *uuid.UUID `json:"approved_by,omitempty"` // nil for auto-approval
	FinalWeight float64    `json:"final_weight"`
	Note        string     `json:"note,omitempty"`
}

func (e ReviewApproved) EventType() string { return TypeReviewApproved }

// ReviewRejected is emitted when moderator rejects the review
type ReviewRejected struct {
	eventstore.BaseEvent
	RejectedBy uuid.UUID `json:"rejected_by"`
	Reason     string    `json:"reason"`
}

func (e ReviewRejected) EventType() string { return TypeReviewRejected }

// ReviewActivated is emitted when the review starts affecting the rating
// (after activation_date has passed)
type ReviewActivated struct {
	eventstore.BaseEvent
	FinalWeight float64 `json:"final_weight"`
}

func (e ReviewActivated) EventType() string { return TypeReviewActivated }

// ReviewDeactivated is emitted when the review is invalidated
// (e.g., reviewer marked as fraudster)
type ReviewDeactivated struct {
	eventstore.BaseEvent
	Reason string `json:"reason"`
}

func (e ReviewDeactivated) EventType() string { return TypeReviewDeactivated }
