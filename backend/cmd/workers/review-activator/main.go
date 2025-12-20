package main

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"codeberg.org/udison/veziizi/backend/internal/domain/review"
	_ "codeberg.org/udison/veziizi/backend/internal/domain/review/events"
	"codeberg.org/udison/veziizi/backend/internal/pkg/factory"
	"codeberg.org/udison/veziizi/backend/internal/pkg/worker"
)

const (
	batchSize = 100
	interval  = 1 * time.Minute
)

func main() {
	worker.RunScheduled(worker.ScheduledConfig{
		Name:     "review-activator",
		Interval: interval,
		LogFile:  "review-activator-worker.log",
		Handler:  newActivatorHandler,
	})
}

func newActivatorHandler(f *factory.Factory) func(ctx context.Context) error {
	reviewService := f.ReviewService()
	reviewsProjection := f.ReviewsProjection()

	return func(ctx context.Context) error {
		ids, err := reviewsProjection.ListReviewsForActivation(ctx, batchSize)
		if err != nil {
			return err
		}

		if len(ids) == 0 {
			return nil
		}

		slog.Info("activating reviews", slog.Int("count", len(ids)))

		var activated, failed int
		for _, id := range ids {
			if err := reviewService.Activate(ctx, id); err != nil {
				if errors.Is(err, review.ErrActivationDateNotPassed) ||
					errors.Is(err, review.ErrReviewNotApproved) ||
					errors.Is(err, review.ErrReviewAlreadyActive) {
					// Skip reviews that are not ready or already active
					slog.Debug("skipping review",
						slog.String("review_id", id.String()),
						slog.String("reason", err.Error()))
					continue
				}
				slog.Error("failed to activate review",
					slog.String("review_id", id.String()),
					slog.String("error", err.Error()))
				failed++
				continue
			}
			activated++
		}

		if activated > 0 || failed > 0 {
			slog.Info("activation batch completed",
				slog.Int("activated", activated),
				slog.Int("failed", failed))
		}

		return nil
	}
}
