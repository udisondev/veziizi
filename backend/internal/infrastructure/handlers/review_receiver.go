package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	reviewApp "codeberg.org/udison/veziizi/backend/internal/application/review"
	orderEvents "codeberg.org/udison/veziizi/backend/internal/domain/order/events"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"github.com/ThreeDotsLabs/watermill/message"
)

// ReviewReceiverHandler listens for Order.ReviewLeft events
// and creates Review aggregates for fraud analysis pipeline
type ReviewReceiverHandler struct {
	reviewService *reviewApp.Service
}

func NewReviewReceiverHandler(reviewService *reviewApp.Service) *ReviewReceiverHandler {
	return &ReviewReceiverHandler{
		reviewService: reviewService,
	}
}

func (h *ReviewReceiverHandler) Handle(msg *message.Message) error {
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

	// Only process ReviewLeft events
	reviewLeft, ok := evt.(orderEvents.ReviewLeft)
	if !ok {
		// Ignore other order events
		return nil
	}

	return h.onReviewLeft(msg.Context(), reviewLeft)
}

func (h *ReviewReceiverHandler) onReviewLeft(ctx context.Context, e orderEvents.ReviewLeft) error {
	slog.Info("processing review left event",
		slog.String("order_id", e.AggregateID().String()),
		slog.String("review_id", e.ReviewID.String()),
		slog.String("reviewer_org_id", e.ReviewerOrgID.String()),
		slog.Int("rating", e.Rating),
	)

	// Create Review aggregate from Order.ReviewLeft event
	err := h.reviewService.CreateFromOrderReview(ctx, reviewApp.CreateFromOrderReviewInput{
		ReviewID:      e.ReviewID,
		OrderID:       e.AggregateID(),
		ReviewerOrgID: e.ReviewerOrgID,
		Rating:        e.Rating,
		Comment:       e.Comment,
	})
	if err != nil {
		slog.Error("failed to create review from order event",
			slog.String("order_id", e.AggregateID().String()),
			slog.String("review_id", e.ReviewID.String()),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("create review: %w", err)
	}

	slog.Info("review created successfully",
		slog.String("review_id", e.ReviewID.String()),
	)

	return nil
}
