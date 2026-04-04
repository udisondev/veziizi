package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill/message"
	reviewApp "github.com/udisondev/veziizi/backend/internal/application/review"
	reviewEvents "github.com/udisondev/veziizi/backend/internal/domain/review/events"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
)

// ReviewAnalyzerHandler listens for Review.Received events
// and performs fraud analysis and weight calculation
type ReviewAnalyzerHandler struct {
	reviewService *reviewApp.Service
	analyzer      *reviewApp.Analyzer
}

func NewReviewAnalyzerHandler(
	reviewService *reviewApp.Service,
	analyzer *reviewApp.Analyzer,
) *ReviewAnalyzerHandler {
	return &ReviewAnalyzerHandler{
		reviewService: reviewService,
		analyzer:      analyzer,
	}
}

func (h *ReviewAnalyzerHandler) Handle(msg *message.Message) error {
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

	// Only process ReviewReceived events
	reviewReceived, ok := evt.(reviewEvents.ReviewReceived)
	if !ok {
		// Ignore other review events
		return nil
	}

	return h.onReviewReceived(msg.Context(), reviewReceived)
}

func (h *ReviewAnalyzerHandler) onReviewReceived(ctx context.Context, e reviewEvents.ReviewReceived) error {
	slog.Info("analyzing review",
		slog.String("review_id", e.AggregateID().String()),
		slog.String("order_id", e.OrderID.String()),
		slog.String("reviewer_org_id", e.ReviewerOrgID.String()),
		slog.String("reviewed_org_id", e.ReviewedOrgID.String()),
		slog.Int("rating", e.Rating),
	)

	// Load review aggregate
	review, err := h.reviewService.Get(ctx, e.AggregateID())
	if err != nil {
		slog.Error("failed to load review",
			slog.String("review_id", e.AggregateID().String()),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("load review: %w", err)
	}

	// Perform fraud analysis
	result, err := h.analyzer.Analyze(ctx, review)
	if err != nil {
		slog.Error("failed to analyze review",
			slog.String("review_id", e.AggregateID().String()),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("analyze review: %w", err)
	}

	// Record analysis results
	if err := h.reviewService.RecordAnalysis(ctx, reviewApp.RecordAnalysisInput{
		ReviewID:           e.AggregateID(),
		RawWeight:          result.RawWeight,
		FraudSignals:       result.FraudSignals,
		FraudScore:         result.FraudScore,
		RequiresModeration: result.RequiresModeration,
		ActivationDate:     result.ActivationDate,
	}); err != nil {
		slog.Error("failed to record analysis",
			slog.String("review_id", e.AggregateID().String()),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("record analysis: %w", err)
	}

	slog.Info("review analysis completed",
		slog.String("review_id", e.AggregateID().String()),
		slog.Float64("raw_weight", result.RawWeight),
		slog.Float64("fraud_score", result.FraudScore),
		slog.Int("signals_count", len(result.FraudSignals)),
		slog.Bool("requires_moderation", result.RequiresModeration),
	)

	return nil
}
