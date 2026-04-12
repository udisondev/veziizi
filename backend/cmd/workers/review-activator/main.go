package main

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/udisondev/veziizi/backend/internal/domain/review"
	_ "github.com/udisondev/veziizi/backend/internal/domain/review/events"
	"github.com/udisondev/veziizi/backend/internal/pkg/config"
	"github.com/udisondev/veziizi/backend/internal/pkg/factory"
	"github.com/udisondev/veziizi/backend/internal/pkg/worker"
)

func main() {
	worker.RunScheduled(worker.ScheduledConfig{
		Name:    "review-activator",
		LogFile: "review-activator-worker.log",
		IntervalFunc: func(cfg *config.Config) time.Duration {
			return cfg.Worker.ReviewActivatorInterval
		},
		Handler: newActivatorHandler,
	})
}

func newActivatorHandler(f *factory.Factory) func(ctx context.Context) error {
	reviewService := f.ReviewService()
	reviewsProjection := f.ReviewsProjection()
	batchSize := f.Config().Worker.ReviewActivatorBatchSize

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
			select {
			case <-ctx.Done():
				slog.Info("activation cancelled",
					slog.Int("processed", activated+failed),
					slog.Int("remaining", len(ids)-activated-failed))
				return ctx.Err()
			default:
			}

			if err := reviewService.Activate(ctx, id); err != nil {
				if errors.Is(err, review.ErrActivationDateNotPassed) ||
					errors.Is(err, review.ErrReviewNotApproved) ||
					errors.Is(err, review.ErrReviewAlreadyActive) {
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
