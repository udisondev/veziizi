package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	reviewApp "github.com/udisondev/veziizi/backend/internal/application/review"
	orgEvents "github.com/udisondev/veziizi/backend/internal/domain/organization/events"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/projections"
	"github.com/ThreeDotsLabs/watermill/message"
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

func (h *FraudsterHandler) Handle(msg *message.Message) error {
	var envelope eventstore.EventEnvelope
	if err := json.Unmarshal(msg.Payload, &envelope); err != nil {
		slog.Error("failed to unmarshal event envelope", slog.String("error", err.Error()))
		return fmt.Errorf("unmarshal event envelope: %w", err)
	}

	evt, err := envelope.UnmarshalEvent()
	if err != nil {
		slog.Error("failed to unmarshal event", slog.String("error", err.Error()))
		return fmt.Errorf("unmarshal event: %w", err)
	}

	return h.handleEvent(msg.Context(), evt)
}

func (h *FraudsterHandler) handleEvent(ctx context.Context, evt eventstore.Event) error {
	switch e := evt.(type) {
	case orgEvents.FraudsterMarked:
		return h.onFraudsterMarked(ctx, e)
	case orgEvents.FraudsterUnmarked:
		return h.onFraudsterUnmarked(ctx, e)
	}
	// Ignore other organization events
	return nil
}

func (h *FraudsterHandler) onFraudsterMarked(ctx context.Context, e orgEvents.FraudsterMarked) error {
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

	// 2. Get all active reviews by this organization
	activeReviewIDs, err := h.reviewsProjection.ListActiveReviewsByReviewer(ctx, orgID)
	if err != nil {
		slog.Error("failed to list active reviews",
			slog.String("org_id", orgID.String()),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("list active reviews: %w", err)
	}

	// 3. Batch deactivate reviews (параллельно с ограниченной конкурентностью)
	reason := fmt.Sprintf("reviewer marked as fraudster: %s", e.Reason)
	result := h.reviewService.BatchDeactivate(ctx, activeReviewIDs, reason)

	// Логируем ошибки если есть
	for i, failedID := range result.FailedIDs {
		slog.Error("failed to deactivate review",
			slog.String("review_id", failedID.String()),
			slog.String("error", result.Errors[i].Error()),
		)
	}

	slog.Info("fraudster reviews deactivated",
		slog.String("org_id", orgID.String()),
		slog.Int("total_active", len(activeReviewIDs)),
		slog.Int("deactivated", result.SuccessCount),
		slog.Int("failed", len(result.FailedIDs)),
	)

	return nil
}

func (h *FraudsterHandler) onFraudsterUnmarked(ctx context.Context, e orgEvents.FraudsterUnmarked) error {
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
