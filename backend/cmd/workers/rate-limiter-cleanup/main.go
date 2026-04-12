package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/udisondev/veziizi/backend/internal/pkg/config"
	"github.com/udisondev/veziizi/backend/internal/pkg/factory"
	"github.com/udisondev/veziizi/backend/internal/pkg/worker"
)

func main() {
	worker.RunScheduled(worker.ScheduledConfig{
		Name:    "rate-limiter-cleanup",
		LogFile: "rate-limiter-cleanup-worker.log",
		IntervalFunc: func(cfg *config.Config) time.Duration {
			return cfg.Worker.RateLimiterCleanupInterval
		},
		Handler: newCleanupHandler,
	})
}

func newCleanupHandler(f *factory.Factory) func(ctx context.Context) error {
	sessionFraudProjection := f.SessionFraudProjection()

	return func(ctx context.Context) error {
		if err := sessionFraudProjection.CleanupOldRateLimits(ctx); err != nil {
			slog.Error("failed to cleanup old rate limits",
				slog.String("error", err.Error()))
			return err
		}

		slog.Debug("rate limiter cleanup completed")
		return nil
	}
}
