package handlers

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/udisondev/veziizi/backend/internal/domain/review/events"
	"github.com/udisondev/veziizi/backend/internal/domain/review/values"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/projections"
	"github.com/udisondev/veziizi/backend/internal/pkg/dbtx"
)

const reviewsProjectionName = "reviews-projection"

// ReviewsProjectionHandler handles review events and updates lookup tables.
//
// В отличие от других проекций здесь много накопительных операций
// (IncrementPendingReviews, AddWeightedRating, AdjustReviewRatingDelta — см.
// FraudDataProjection / OrganizationRatingsProjection). При at-least-once
// доставке повторный апплай раздувает счётчики, поэтому каждый handler
// сначала резервирует event_id в projection_event_dedup внутри той же tx,
// что и сама операция. См. курс 10-at-least-once-delivery.
type ReviewsProjectionHandler struct {
	db        dbtx.TxManager
	psql      squirrel.StatementBuilderType
	fraudData *projections.FraudDataProjection
	ratings   *projections.OrganizationRatingsProjection
	dedup     *projections.ProjectionEventDedupProjection
}

func NewReviewsProjectionHandler(
	db dbtx.TxManager,
	fraudData *projections.FraudDataProjection,
	ratings *projections.OrganizationRatingsProjection,
	dedup *projections.ProjectionEventDedupProjection,
) *ReviewsProjectionHandler {
	return &ReviewsProjectionHandler{
		db:        db,
		psql:      squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
		fraudData: fraudData,
		ratings:   ratings,
		dedup:     dedup,
	}
}

// withDedup оборачивает накопительные операции проекции в общую tx с
// dedup-резервом event_id — тонкая обёртка над общим dedupGuard (см. dedup.go).
func (h *ReviewsProjectionHandler) withDedup(ctx context.Context, fn func(ctx context.Context) error) error {
	eventID, err := eventIDFromCtx(ctx)
	if err != nil {
		return err
	}
	return dedupGuard(ctx, h.db, h.dedup, reviewsProjectionName, eventID, fn)
}

func (h *ReviewsProjectionHandler) OnReceived(ctx context.Context, e *events.ReviewReceived) error {
	slog.Info("review received",
		slog.String("review_id", e.AggregateID().String()),
		slog.String("order_id", e.OrderID.String()),
		slog.Int("rating", e.Rating),
	)

	return h.withDedup(ctx, func(ctx context.Context) error {
		row := &projections.ReviewLookupRow{
			ID:                 e.AggregateID(),
			OrderID:            e.OrderID,
			ReviewerOrgID:      e.ReviewerOrgID,
			ReviewedOrgID:      e.ReviewedOrgID,
			Rating:             e.Rating,
			Comment:            e.Comment,
			OrderAmount:        e.OrderAmount,
			OrderCurrency:      e.OrderCurrency,
			OrderCreatedAt:     e.OrderCreatedAt,
			OrderCompletedAt:   e.OrderCompletedAt,
			RawWeight:          1.0,
			FinalWeight:        0.0,
			FraudScore:         0.0,
			RequiresModeration: false,
			Status:             values.StatusPendingAnalysis.String(),
			CreatedAt:          e.OccurredAt(),
		}

		if err := h.fraudData.UpsertReviewLookup(ctx, row); err != nil {
			return fmt.Errorf("upsert review lookup: %w", err)
		}
		if err := h.ratings.IncrementPendingReviews(ctx, e.ReviewedOrgID); err != nil {
			return fmt.Errorf("increment pending reviews: %w", err)
		}
		if err := h.fraudData.IncrementReviewStats(ctx, e.ReviewerOrgID, e.ReviewedOrgID, e.Rating); err != nil {
			return fmt.Errorf("update interaction stats: %w", err)
		}
		if err := h.fraudData.IncrementTotalReviewsLeft(ctx, e.ReviewerOrgID); err != nil {
			return fmt.Errorf("update reviewer reputation: %w", err)
		}
		return nil
	})
}

func (h *ReviewsProjectionHandler) OnEdited(ctx context.Context, e *events.ReviewEdited) error {
	slog.Info("review edited",
		slog.String("review_id", e.AggregateID().String()),
		slog.Int("old_rating", e.OldRating),
		slog.Int("new_rating", e.NewRating),
	)

	return h.withDedup(ctx, func(ctx context.Context) error {
		// Update rating and comment in reviews_lookup
		query := `
			UPDATE reviews_lookup SET
				rating = $2,
				comment = $3
			WHERE id = $1
		`
		res, err := h.db.Exec(ctx, query,
			e.AggregateID(), e.NewRating, e.NewComment,
		)
		if err != nil {
			return fmt.Errorf("update review lookup: %w", err)
		}
		// 0 строк = ReviewEdited обогнал ReviewReceived (out-of-order при N
		// инстансах). Молчаливый nil закоммитил бы dedup-резерв и правка
		// потерялась бы навсегда; errProjectionRowMissing → rollback → retry
		// дождётся появления строки.
		if res.RowsAffected() == 0 {
			return fmt.Errorf("reviews_lookup row %s not found yet, ReviewEdited before ReviewReceived: %w",
				e.AggregateID(), errProjectionRowMissing)
		}

		// If rating changed, adjust interaction stats
		if e.OldRating != e.NewRating {
			reviewData, err := h.getReviewData(ctx, e.AggregateID())
			if err != nil {
				return fmt.Errorf("get review data: %w", err)
			}

			ratingDelta := e.NewRating - e.OldRating
			if err := h.fraudData.AdjustReviewRatingDelta(ctx, reviewData.ReviewerOrgID, reviewData.ReviewedOrgID, ratingDelta); err != nil {
				return fmt.Errorf("adjust interaction stats rating: %w", err)
			}
		}

		return nil
	})
}

func (h *ReviewsProjectionHandler) OnAnalyzed(ctx context.Context, e *events.ReviewAnalyzed) error {
	slog.Info("review analyzed",
		slog.String("review_id", e.AggregateID().String()),
		slog.Float64("raw_weight", e.RawWeight),
		slog.Float64("fraud_score", e.FraudScore),
		slog.Bool("requires_moderation", e.RequiresModeration),
	)

	return h.withDedup(ctx, func(ctx context.Context) error {
		// Determine status after analysis.
		// If requires moderation → pending_moderation.
		// If not → keep pending_analysis; ReviewApproved event (which follows immediately) will set approved.
		status := values.StatusPendingAnalysis.String()
		if e.RequiresModeration {
			status = values.StatusPendingModeration.String()
		}

		analyzedAt := e.OccurredAt()

		// Update reviews_lookup
		query := `
			UPDATE reviews_lookup SET
				raw_weight = $2,
				fraud_score = $3,
				requires_moderation = $4,
				activation_date = $5,
				analyzed_at = $6,
				status = $7
			WHERE id = $1
		`
		res, err := h.db.Exec(ctx, query,
			e.AggregateID(), e.RawWeight, e.FraudScore, e.RequiresModeration,
			e.ActivationDate, analyzedAt, status,
		)
		if err != nil {
			return fmt.Errorf("update review lookup: %w", err)
		}
		// 0 строк = ReviewAnalyzed обогнал ReviewReceived: без проверки UPDATE
		// молча no-op'ится, fraud signals ниже вставились бы осиротевшими, а
		// dedup-резерв закоммитился бы — raw_weight/fraud_score/activation_date
		// потеряны навсегда. errProjectionRowMissing → rollback → retry.
		if res.RowsAffected() == 0 {
			return fmt.Errorf("reviews_lookup row %s not found yet, ReviewAnalyzed before ReviewReceived: %w",
				e.AggregateID(), errProjectionRowMissing)
		}

		// Insert fraud signals (batch INSERT)
		if len(e.FraudSignals) > 0 {
			signals := make([]projections.FraudSignalInput, 0, len(e.FraudSignals))
			for _, s := range e.FraudSignals {
				signals = append(signals, projections.FraudSignalInput{
					SignalType:  s.Type,
					Severity:    s.Severity,
					Description: s.Description,
					ScoreImpact: s.ScoreImpact,
					Evidence:    s.Evidence,
				})
			}
			if err := h.fraudData.InsertFraudSignalsBatch(ctx, e.AggregateID(), signals); err != nil {
				return fmt.Errorf("insert fraud signals batch: %w", err)
			}
		}

		return nil
	})
}

func (h *ReviewsProjectionHandler) OnApproved(ctx context.Context, e *events.ReviewApproved) error {
	slog.Info("review approved",
		slog.String("review_id", e.AggregateID().String()),
		slog.Float64("final_weight", e.FinalWeight),
	)

	return h.withDedup(ctx, func(ctx context.Context) error {
		moderatedAt := e.OccurredAt()

		// Update reviews_lookup
		query := `
			UPDATE reviews_lookup SET
				final_weight = $2,
				moderated_at = $3,
				moderated_by = $4,
				status = $5
			WHERE id = $1
		`
		if _, err := h.db.Exec(ctx, query,
			e.AggregateID(), e.FinalWeight, moderatedAt, e.ApprovedBy, values.StatusApproved.String(),
		); err != nil {
			return fmt.Errorf("update review lookup: %w", err)
		}

		// Get reviewed org ID to decrement pending
		reviewedOrgID, err := h.getReviewedOrgID(ctx, e.AggregateID())
		if err != nil {
			return fmt.Errorf("get reviewed org id: %w", err)
		}

		if err := h.ratings.DecrementPendingReviews(ctx, reviewedOrgID); err != nil {
			return fmt.Errorf("decrement pending reviews: %w", err)
		}

		return nil
	})
}

func (h *ReviewsProjectionHandler) OnRejected(ctx context.Context, e *events.ReviewRejected) error {
	slog.Info("review rejected",
		slog.String("review_id", e.AggregateID().String()),
		slog.String("reason", e.Reason),
	)

	return h.withDedup(ctx, func(ctx context.Context) error {
		moderatedAt := e.OccurredAt()

		// Update reviews_lookup
		query := `
			UPDATE reviews_lookup SET
				moderated_at = $2,
				moderated_by = $3,
				status = $4
			WHERE id = $1
		`
		if _, err := h.db.Exec(ctx, query,
			e.AggregateID(), moderatedAt, e.RejectedBy, values.StatusRejected.String(),
		); err != nil {
			return fmt.Errorf("update review lookup: %w", err)
		}

		// Get review data
		reviewData, err := h.getReviewData(ctx, e.AggregateID())
		if err != nil {
			return fmt.Errorf("get review data: %w", err)
		}

		// Decrement pending reviews
		if err := h.ratings.DecrementPendingReviews(ctx, reviewData.ReviewedOrgID); err != nil {
			return fmt.Errorf("decrement pending reviews: %w", err)
		}

		// Increment rejected reviews counter
		if err := h.ratings.IncrementRejectedReviews(ctx, reviewData.ReviewedOrgID); err != nil {
			return fmt.Errorf("increment rejected reviews: %w", err)
		}

		// Update reviewer reputation (атомарный INSERT/UPDATE)
		if err := h.fraudData.IncrementRejectedReviews(ctx, reviewData.ReviewerOrgID, 0.1); err != nil {
			return fmt.Errorf("update reviewer reputation: %w", err)
		}

		return nil
	})
}

func (h *ReviewsProjectionHandler) OnActivated(ctx context.Context, e *events.ReviewActivated) error {
	slog.Info("review activated",
		slog.String("review_id", e.AggregateID().String()),
		slog.Float64("final_weight", e.FinalWeight),
	)

	return h.withDedup(ctx, func(ctx context.Context) error {
		activatedAt := e.OccurredAt()

		// Update reviews_lookup
		query := `
			UPDATE reviews_lookup SET
				activated_at = $2,
				final_weight = $3,
				status = $4
			WHERE id = $1
		`
		if _, err := h.db.Exec(ctx, query,
			e.AggregateID(), activatedAt, e.FinalWeight, values.StatusActive.String(),
		); err != nil {
			return fmt.Errorf("update review lookup: %w", err)
		}

		// Get review data
		reviewData, err := h.getReviewData(ctx, e.AggregateID())
		if err != nil {
			return fmt.Errorf("get review data: %w", err)
		}

		// Add to organization ratings
		if err := h.ratings.AddWeightedRating(ctx, reviewData.ReviewedOrgID, reviewData.Rating, e.FinalWeight); err != nil {
			return fmt.Errorf("add weighted rating: %w", err)
		}

		// Update reviewer reputation (атомарный инкремент)
		if err := h.fraudData.IncrementActiveReviewsLeft(ctx, reviewData.ReviewerOrgID); err != nil {
			return fmt.Errorf("update reviewer reputation: %w", err)
		}

		return nil
	})
}

func (h *ReviewsProjectionHandler) OnDeactivated(ctx context.Context, e *events.ReviewDeactivated) error {
	slog.Info("review deactivated",
		slog.String("review_id", e.AggregateID().String()),
		slog.String("reason", e.Reason),
	)

	return h.withDedup(ctx, func(ctx context.Context) error {
		// Get review data before updating
		reviewData, err := h.getReviewData(ctx, e.AggregateID())
		if err != nil {
			return fmt.Errorf("get review data: %w", err)
		}

		// Check if was active (contributing to rating)
		wasActive := reviewData.Status == values.StatusActive.String()

		// Update reviews_lookup
		query := `
			UPDATE reviews_lookup SET
				status = $2
			WHERE id = $1
		`
		if _, err := h.db.Exec(ctx, query, e.AggregateID(), values.StatusDeactivated.String()); err != nil {
			return fmt.Errorf("update review lookup: %w", err)
		}

		// If was active, remove from organization ratings
		if wasActive {
			if err := h.ratings.RemoveWeightedRating(ctx, reviewData.ReviewedOrgID, reviewData.Rating, reviewData.FinalWeight); err != nil {
				return fmt.Errorf("remove weighted rating: %w", err)
			}
		}

		// Update reviewer reputation (атомарный инкремент)
		if err := h.fraudData.IncrementDeactivatedReviews(ctx, reviewData.ReviewerOrgID, wasActive); err != nil {
			return fmt.Errorf("update reviewer deactivation reputation: %w", err)
		}

		return nil
	})
}

type reviewDataRow struct {
	ReviewerOrgID uuid.UUID
	ReviewedOrgID uuid.UUID
	Rating        int
	FinalWeight   float64
	Status        string
}

func (h *ReviewsProjectionHandler) getReviewData(ctx context.Context, reviewID uuid.UUID) (*reviewDataRow, error) {
	query, args, err := h.psql.
		Select("reviewer_org_id", "reviewed_org_id", "rating", "final_weight", "status").
		From("reviews_lookup").
		Where(squirrel.Eq{"id": reviewID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	var row reviewDataRow
	if err := h.db.QueryRow(ctx, query, args...).Scan(
		&row.ReviewerOrgID, &row.ReviewedOrgID, &row.Rating, &row.FinalWeight, &row.Status,
	); err != nil {
		return nil, fmt.Errorf("scan row: %w", err)
	}

	return &row, nil
}

func (h *ReviewsProjectionHandler) getReviewedOrgID(ctx context.Context, reviewID uuid.UUID) (uuid.UUID, error) {
	query, args, err := h.psql.
		Select("reviewed_org_id").
		From("reviews_lookup").
		Where(squirrel.Eq{"id": reviewID}).
		ToSql()
	if err != nil {
		return uuid.Nil, fmt.Errorf("build query: %w", err)
	}

	var orgID uuid.UUID
	if err := h.db.QueryRow(ctx, query, args...).Scan(&orgID); err != nil {
		return uuid.Nil, fmt.Errorf("scan org id: %w", err)
	}

	return orgID, nil
}
