package heartbeat

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultInterval = 30 * time.Second

// Recorder периодически пишет heartbeat в БД для мониторинга воркера.
type Recorder struct {
	pool       *pgxpool.Pool
	name       string
	workerType string
	interval   time.Duration
	cancel     context.CancelFunc
}

// New создаёт Recorder и регистрирует воркер в БД.
func New(pool *pgxpool.Pool, name, workerType string) *Recorder {
	return &Recorder{
		pool:       pool,
		name:       name,
		workerType: workerType,
		interval:   defaultInterval,
	}
}

// Start регистрирует воркер и запускает фоновую горутину heartbeat.
func (r *Recorder) Start(ctx context.Context) error {
	if err := r.register(ctx); err != nil {
		return fmt.Errorf("register worker heartbeat: %w", err)
	}

	hbCtx, cancel := context.WithCancel(ctx)
	r.cancel = cancel

	go r.loop(hbCtx)
	return nil
}

// Stop помечает воркер как stopped.
func (r *Recorder) Stop() {
	if r.cancel != nil {
		r.cancel()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.pool.Exec(ctx,
		`UPDATE worker_heartbeats SET status = 'stopped', last_heartbeat = NOW() WHERE name = $1`,
		r.name,
	)
	if err != nil {
		slog.Error("failed to mark worker as stopped", "worker", r.name, "error", err)
	}
}

func (r *Recorder) register(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO worker_heartbeats (name, worker_type, started_at, last_heartbeat, status)
		VALUES ($1, $2, NOW(), NOW(), 'running')
		ON CONFLICT (name) DO UPDATE
		SET started_at = NOW(), last_heartbeat = NOW(), status = 'running', worker_type = $2
	`, r.name, r.workerType)
	return err
}

func (r *Recorder) loop(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := r.beat(ctx); err != nil {
				slog.Error("heartbeat failed", "worker", r.name, "error", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (r *Recorder) beat(ctx context.Context) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE worker_heartbeats SET last_heartbeat = NOW() WHERE name = $1`,
		r.name,
	)
	return err
}
