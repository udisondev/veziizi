package review

import (
	"context"
	"fmt"
	"time"

	"github.com/udisondev/veziizi/backend/internal/domain/review"
	"github.com/udisondev/veziizi/backend/internal/domain/review/events"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/messaging"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"github.com/udisondev/veziizi/backend/internal/pkg/dbtx"
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

// CreateFromFreightReviewInput contains data for creating a Review from FreightRequest.ReviewLeft
type CreateFromFreightReviewInput struct {
	ReviewID         uuid.UUID
	FreightRequestID uuid.UUID
	ReviewerOrgID    uuid.UUID
	ReviewedOrgID    uuid.UUID
	Rating           int
	Comment          string
	FreightAmount    int64
	FreightCurrency  string
	FreightCreatedAt time.Time
	CompletedAt      time.Time
}

// CreateFromFreightReview creates a new Review aggregate from FreightRequest.ReviewLeft event
func (s *Service) CreateFromFreightReview(ctx context.Context, input CreateFromFreightReviewInput) error {
	return s.db.InTx(ctx, func(ctx context.Context) error {
		// Create Review aggregate - all data comes from the event
		r := review.New(
			input.ReviewID,
			input.FreightRequestID,
			input.ReviewerOrgID,
			input.ReviewedOrgID,
			input.Rating,
			input.Comment,
			input.FreightAmount,
			input.FreightCurrency,
			input.FreightCreatedAt,
			input.CompletedAt,
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

// BatchDeactivateResult содержит результат batch деактивации
type BatchDeactivateResult struct {
	SuccessCount int
	FailedIDs    []uuid.UUID
	Errors       []error
}

// BatchDeactivate деактивирует несколько отзывов параллельно с ограниченной конкурентностью
func (s *Service) BatchDeactivate(ctx context.Context, reviewIDs []uuid.UUID, reason string) BatchDeactivateResult {
	if len(reviewIDs) == 0 {
		return BatchDeactivateResult{}
	}

	const maxConcurrency = 10

	type deactivateResult struct {
		id  uuid.UUID
		err error
	}

	results := make(chan deactivateResult, len(reviewIDs))
	sem := make(chan struct{}, maxConcurrency)

	// Запускаем горутины для параллельной деактивации
	for _, id := range reviewIDs {
		go func(reviewID uuid.UUID) {
			// Захватываем семафор
			sem <- struct{}{}
			defer func() { <-sem }()

			// Проверяем отмену контекста
			select {
			case <-ctx.Done():
				results <- deactivateResult{id: reviewID, err: ctx.Err()}
				return
			default:
			}

			err := s.Deactivate(ctx, reviewID, reason)
			results <- deactivateResult{id: reviewID, err: err}
		}(id)
	}

	// Собираем результаты
	result := BatchDeactivateResult{}
	for range reviewIDs {
		r := <-results
		if r.err != nil {
			result.FailedIDs = append(result.FailedIDs, r.id)
			result.Errors = append(result.Errors, r.err)
		} else {
			result.SuccessCount++
		}
	}

	return result
}

func (s *Service) saveAndPublish(ctx context.Context, r *review.Review) error {
	changes := r.Changes()
	if len(changes) == 0 {
		return nil
	}

	if err := s.db.InTx(ctx, func(ctx context.Context) error {
		if err := s.eventStore.Save(ctx, changes...); err != nil {
			return fmt.Errorf("save review: %w", err)
		}

		if err := s.publisher.Publish(ctx, "review.events", changes...); err != nil {
			return fmt.Errorf("publish review events: %w", err)
		}

		return nil
	}); err != nil {
		return err
	}

	r.ClearChanges()
	return nil
}
