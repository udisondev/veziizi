package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	reviewApp "github.com/udisondev/veziizi/backend/internal/application/review"
	freightEvents "github.com/udisondev/veziizi/backend/internal/domain/freightrequest/events"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"github.com/ThreeDotsLabs/watermill/message"
)

// ReviewReceiverHandler listens for FreightRequest.ReviewLeft events
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
	reviewLeft, ok := evt.(freightEvents.ReviewLeft)
	if !ok {
		// Ignore other freight request events
		return nil
	}

	return h.onReviewLeft(msg.Context(), reviewLeft)
}

func (h *ReviewReceiverHandler) onReviewLeft(ctx context.Context, e freightEvents.ReviewLeft) error {
	slog.Info("processing review left event",
		slog.String("freight_request_id", e.AggregateID().String()),
		slog.String("review_id", e.ReviewID.String()),
		slog.String("reviewer_org_id", e.ReviewerOrgID.String()),
		slog.Int("rating", e.Rating),
	)

	// Create Review aggregate from FreightRequest.ReviewLeft event
	err := h.reviewService.CreateFromFreightReview(ctx, reviewApp.CreateFromFreightReviewInput{
		ReviewID:         e.ReviewID,
		FreightRequestID: e.AggregateID(),
		ReviewerOrgID:    e.ReviewerOrgID,
		ReviewedOrgID:    e.ReviewedOrgID,
		Rating:           e.Rating,
		Comment:          e.Comment,
		FreightAmount:    e.FreightAmount,
		FreightCurrency:  e.FreightCurrency,
		FreightCreatedAt: time.Unix(e.FreightCreatedAt, 0),
		CompletedAt:      time.Unix(e.CompletedAt, 0),
	})
	if err != nil {
		slog.Error("failed to create review from freight request event",
			slog.String("freight_request_id", e.AggregateID().String()),
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
