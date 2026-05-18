package handlers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	reviewApp "github.com/udisondev/veziizi/backend/internal/application/review"
	reviewDomain "github.com/udisondev/veziizi/backend/internal/domain/review"
	reviewEvents "github.com/udisondev/veziizi/backend/internal/domain/review/events"
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

func (h *ReviewAnalyzerHandler) OnReviewReceived(ctx context.Context, e *reviewEvents.ReviewReceived) error {
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
		// Idempotency: another worker (or a sync test seed) has already
		// transitioned the review out of pending_analysis. Treat as success so
		// the watermill subscriber doesn't retry forever and starve the rest
		// of the topic.
		if errors.Is(err, reviewDomain.ErrReviewAlreadyAnalyzed) {
			slog.Debug("review already analyzed, skipping",
				slog.String("review_id", e.AggregateID().String()))
			return nil
		}
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
