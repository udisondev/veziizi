package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/udisondev/veziizi/backend/internal/pkg/config"
	"github.com/udisondev/veziizi/backend/internal/pkg/factory"
	"github.com/udisondev/veziizi/backend/internal/pkg/worker"
)

// dedup-cleanup периодически чистит dedup-таблицы (projection_event_dedup,
// notification_dedup): строки нужны только на окно повторной доставки
// (retry + XAUTOCLAIM), дальше они лишь раздувают таблицы и индексы, которые
// сидят в hot path каждого обрабатываемого события.
func main() {
	worker.RunScheduled(worker.ScheduledConfig{
		Name:    "dedup-cleanup",
		LogFile: "dedup-cleanup-worker.log",
		IntervalFunc: func(cfg *config.Config) time.Duration {
			return cfg.Worker.DedupCleanupInterval
		},
		Handler: newCleanupHandler,
	})
}

func newCleanupHandler(f *factory.Factory) func(ctx context.Context) error {
	projectionDedup := f.ProjectionEventDedupProjection()
	notificationDedup := f.NotificationDedupProjection()
	retention := f.Config().Worker.DedupRetention

	return func(ctx context.Context) error {
		cutoff := time.Now().Add(-retention)

		projDeleted, err := projectionDedup.DeleteOlderThan(ctx, cutoff)
		if err != nil {
			slog.Error("failed to cleanup projection_event_dedup", slog.String("error", err.Error()))
			return err
		}

		notifDeleted, err := notificationDedup.DeleteOlderThan(ctx, cutoff)
		if err != nil {
			slog.Error("failed to cleanup notification_dedup", slog.String("error", err.Error()))
			return err
		}

		slog.Debug("dedup cleanup completed",
			slog.Int64("projection_event_dedup_deleted", projDeleted),
			slog.Int64("notification_dedup_deleted", notifDeleted))
		return nil
	}
}
