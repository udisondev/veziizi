package review

import (
	"context"
	"fmt"
	"time"

	"codeberg.org/udison/veziizi/backend/internal/domain/order"
	orderEvents "codeberg.org/udison/veziizi/backend/internal/domain/order/events"
	"codeberg.org/udison/veziizi/backend/internal/domain/review"
	"codeberg.org/udison/veziizi/backend/internal/domain/review/events"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/messaging"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"codeberg.org/udison/veziizi/backend/internal/pkg/dbtx"
	"github.com/google/uuid"
)

type Service struct {
	db         dbtx.TxManager
	eventStore eventstore.Store
	publisher  *messaging.EventPublisher
}

func NewService(
	db dbtx.TxManager,
	eventStore eventstore.Store,
	publisher *messaging.EventPublisher,
) *Service {
	return &Service{
		db:         db,
		eventStore: eventStore,
		publisher:  publisher,
	}
}

// Get loads a Review aggregate by ID
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*review.Review, error) {
	evts, err := s.eventStore.Load(ctx, id, events.AggregateType)
	if err != nil {
		return nil, fmt.Errorf("load review: %w", err)
	}
	return review.NewFromEvents(id, evts), nil
}

// getOrder loads Order aggregate by ID (helper for getting order data)
func (s *Service) getOrder(ctx context.Context, id uuid.UUID) (*order.Order, error) {
	evts, err := s.eventStore.Load(ctx, id, orderEvents.AggregateType)
	if err != nil {
		return nil, fmt.Errorf("load order: %w", err)
	}
	return order.NewFromEvents(id, evts), nil
}

// CreateFromOrderReviewInput contains data for creating a Review from Order.ReviewLeft
type CreateFromOrderReviewInput struct {
	ReviewID      uuid.UUID
	OrderID       uuid.UUID
	ReviewerOrgID uuid.UUID
	Rating        int
	Comment       string
}

// CreateFromOrderReview creates a new Review aggregate from Order.ReviewLeft event
func (s *Service) CreateFromOrderReview(ctx context.Context, input CreateFromOrderReviewInput) error {
	return s.db.InTx(ctx, func(ctx context.Context) error {
		// Load Order to get metadata
		o, err := s.getOrder(ctx, input.OrderID)
		if err != nil {
			return fmt.Errorf("load order: %w", err)
		}

		// Determine reviewed org (counterparty of reviewer)
		reviewedOrgID := o.CustomerOrgID()
		if input.ReviewerOrgID == o.CustomerOrgID() {
			reviewedOrgID = o.CarrierOrgID()
		}

		// Get order amount
		var orderAmount int64
		var orderCurrency string
		if o.Payment().Price != nil {
			orderAmount = o.Payment().Price.Amount
			orderCurrency = string(o.Payment().Price.Currency)
		}

		// Create Review aggregate
		r := review.New(
			input.ReviewID,
			input.OrderID,
			input.ReviewerOrgID,
			reviewedOrgID,
			input.Rating,
			input.Comment,
			orderAmount,
			orderCurrency,
			o.CreatedAt(),
			*o.CompletedAt(),
		)

		return s.saveAndPublish(ctx, r)
	})
}

// RecordAnalysisInput contains fraud analysis results
type RecordAnalysisInput struct {
	ReviewID           uuid.UUID
	RawWeight          float64
	FraudSignals       []events.FraudSignal
	FraudScore         float64
	RequiresModeration bool
	ActivationDate     time.Time
}

// RecordAnalysis records the fraud analysis results for a review
func (s *Service) RecordAnalysis(ctx context.Context, input RecordAnalysisInput) error {
	return s.db.InTx(ctx, func(ctx context.Context) error {
		r, err := s.Get(ctx, input.ReviewID)
		if err != nil {
			return err
		}

		if err := r.RecordAnalysis(
			input.RawWeight,
			input.FraudSignals,
			input.FraudScore,
			input.RequiresModeration,
			input.ActivationDate,
		); err != nil {
			return fmt.Errorf("record analysis: %w", err)
		}

		return s.saveAndPublish(ctx, r)
	})
}

// Approve approves a review (by moderator)
func (s *Service) Approve(ctx context.Context, reviewID, moderatorID uuid.UUID, finalWeight float64, note string) error {
	return s.db.InTx(ctx, func(ctx context.Context) error {
		r, err := s.Get(ctx, reviewID)
		if err != nil {
			return err
		}

		if err := r.Approve(moderatorID, finalWeight, note); err != nil {
			return fmt.Errorf("approve review: %w", err)
		}

		return s.saveAndPublish(ctx, r)
	})
}

// Reject rejects a review (by moderator)
func (s *Service) Reject(ctx context.Context, reviewID, moderatorID uuid.UUID, reason string) error {
	return s.db.InTx(ctx, func(ctx context.Context) error {
		r, err := s.Get(ctx, reviewID)
		if err != nil {
			return err
		}

		if err := r.Reject(moderatorID, reason); err != nil {
			return fmt.Errorf("reject review: %w", err)
		}

		return s.saveAndPublish(ctx, r)
	})
}

// Activate activates a review (starts affecting rating)
func (s *Service) Activate(ctx context.Context, reviewID uuid.UUID) error {
	return s.db.InTx(ctx, func(ctx context.Context) error {
		r, err := s.Get(ctx, reviewID)
		if err != nil {
			return err
		}

		if err := r.Activate(); err != nil {
			return fmt.Errorf("activate review: %w", err)
		}

		return s.saveAndPublish(ctx, r)
	})
}

// Deactivate deactivates a review (e.g., reviewer marked as fraudster)
func (s *Service) Deactivate(ctx context.Context, reviewID uuid.UUID, reason string) error {
	return s.db.InTx(ctx, func(ctx context.Context) error {
		r, err := s.Get(ctx, reviewID)
		if err != nil {
			return err
		}

		if err := r.Deactivate(reason); err != nil {
			return fmt.Errorf("deactivate review: %w", err)
		}

		return s.saveAndPublish(ctx, r)
	})
}

func (s *Service) saveAndPublish(ctx context.Context, r *review.Review) error {
	changes := r.Changes()
	if err := s.eventStore.Save(ctx, changes...); err != nil {
		return fmt.Errorf("save review: %w", err)
	}

	for _, evt := range changes {
		if err := s.publisher.Publish(ctx, "review.events", evt); err != nil {
			return fmt.Errorf("publish event: %w", err)
		}
	}

	return nil
}
