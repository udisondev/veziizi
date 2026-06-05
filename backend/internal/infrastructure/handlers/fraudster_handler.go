package handlers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	reviewApp "github.com/udisondev/veziizi/backend/internal/application/review"
	orgEvents "github.com/udisondev/veziizi/backend/internal/domain/organization/events"
	reviewDomain "github.com/udisondev/veziizi/backend/internal/domain/review"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/projections"
)

// FraudsterHandler listens for FraudsterMarked/FraudsterUnmarked events
// and updates org_reviewer_reputation + deactivates reviews of fraudsters
type FraudsterHandler struct {
	reviewService     *reviewApp.Service
	reviewsProjection *projections.ReviewsProjection
	fraudProjection   *projections.FraudDataProjection
}

func NewFraudsterHandler(
	reviewService *reviewApp.Service,
	reviewsProjection *projections.ReviewsProjection,
	fraudProjection *projections.FraudDataProjection,
) *FraudsterHandler {
	return &FraudsterHandler{
		reviewService:     reviewService,
		reviewsProjection: reviewsProjection,
		fraudProjection:   fraudProjection,
	}
}

func (h *FraudsterHandler) OnFraudsterMarked(ctx context.Context, e *orgEvents.FraudsterMarked) error {
	orgID := e.AggregateID()
	slog.Info("processing FraudsterMarked",
		slog.String("org_id", orgID.String()),
		slog.Bool("is_confirmed", e.IsConfirmed),
		slog.String("reason", e.Reason),
	)

	// 1. Update org_reviewer_reputation
	if err := h.fraudProjection.MarkFraudster(ctx, orgID, e.IsConfirmed, e.MarkedBy, e.Reason); err != nil {
		slog.Error("failed to mark fraudster in projection",
			slog.String("org_id", orgID.String()),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("update fraud reputation: %w", err)
	}

	// 2. Get all active AND approved reviews by this organization
	// Approved reviews must also be deactivated — otherwise they will be activated
	// by review-activator later with their original (pre-fraudster) weight.
	reviewIDs, err := h.reviewsProjection.ListDeactivatableReviewsByReviewer(ctx, orgID)
	if err != nil {
		slog.Error("failed to list deactivatable reviews",
			slog.String("org_id", orgID.String()),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("list deactivatable reviews: %w", err)
	}

	// 3. Batch deactivate reviews (параллельно с ограниченной конкурентностью)
	reason := fmt.Sprintf("reviewer marked as fraudster: %s", e.Reason)
	result := h.reviewService.BatchDeactivate(ctx, reviewIDs, reason)

	// Терминальный статус — не failure, а идемпотентный повтор: отзыв уже
	// деактивирован (повторная at-least-once доставка события либо
	// конкурентный инстанс успел раньше). Домен защищает от двойного
	// занижения рейтинга, хендлеру остаётся Ack.
	var realFailures int
	for i, failedID := range result.FailedIDs {
		if errors.Is(result.Errors[i], reviewDomain.ErrReviewTerminalStatus) {
			slog.Debug("review already deactivated, skipping",
				slog.String("review_id", failedID.String()))
			continue
		}
		realFailures++
		slog.Error("failed to deactivate review",
			slog.String("review_id", failedID.String()),
			slog.String("error", result.Errors[i].Error()),
		)
	}

	slog.Info("fraudster reviews deactivated",
		slog.String("org_id", orgID.String()),
		slog.Int("total_found", len(reviewIDs)),
		slog.Int("deactivated", result.SuccessCount),
		slog.Int("failed", realFailures),
	)

	if realFailures > 0 {
		return fmt.Errorf("failed to deactivate %d of %d reviews for fraudster %s",
			realFailures, len(reviewIDs), orgID)
	}

	return nil
}

func (h *FraudsterHandler) OnFraudsterUnmarked(ctx context.Context, e *orgEvents.FraudsterUnmarked) error {
	orgID := e.AggregateID()
	slog.Info("processing FraudsterUnmarked",
		slog.String("org_id", orgID.String()),
		slog.String("reason", e.Reason),
	)

	// Clear fraudster flags in org_reviewer_reputation
	// Note: we don't reactivate previously deactivated reviews - that would require manual review
	if err := h.fraudProjection.UnmarkFraudster(ctx, orgID); err != nil {
		slog.Error("failed to unmark fraudster in projection",
			slog.String("org_id", orgID.String()),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("update fraud reputation: %w", err)
	}

	slog.Info("fraudster unmarked successfully", slog.String("org_id", orgID.String()))
	return nil
}
