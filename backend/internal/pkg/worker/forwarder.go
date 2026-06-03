package worker

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/forwarder"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/udisondev/veziizi/backend/internal/infrastructure/messaging"
	"github.com/udisondev/veziizi/backend/internal/pkg/config"
	"github.com/udisondev/veziizi/backend/internal/pkg/factory"
	"github.com/udisondev/veziizi/backend/internal/pkg/heartbeat"
	"github.com/udisondev/veziizi/backend/internal/pkg/logging"
	"github.com/udisondev/veziizi/backend/internal/pkg/metrics"
)

// forwarderName — имя forwarder-воркера для heartbeat и consumer group
// SQL-подписчика outbox-топика.
const forwarderName = "forwarder"

// outboxLag — главный health-индикатор forwarder'а: сколько сообщений лежит
// в Postgres outbox и ещё не переложено в Redis. Растущий лаг = forwarder
// не справляется или Redis недоступен; события при этом НЕ теряются.
var outboxLag = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "veziizi_forwarder_outbox_lag",
	Help: "Messages in Postgres outbox not yet forwarded to Redis Streams",
})

// watchOutboxLag периодически замеряет лаг: MAX(offset) сообщений минус
// offset_acked consumer group'ы forwarder'а.
func watchOutboxLag(ctx context.Context, pool *pgxpool.Pool, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			lag, err := measureOutboxLag(ctx, pool)
			if err != nil {
				if ctx.Err() == nil {
					slog.Error("failed to measure outbox lag", slog.String("error", err.Error()))
				}
				continue
			}
			outboxLag.Set(float64(lag))
		}
	}
}

func measureOutboxLag(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	var maxOffset, acked int64
	if err := pool.QueryRow(ctx,
		`SELECT COALESCE(MAX("offset"), 0) FROM "watermill_`+messaging.OutboxTopic+`"`,
	).Scan(&maxOffset); err != nil {
		return 0, fmt.Errorf("query max outbox offset: %w", err)
	}
	if err := pool.QueryRow(ctx,
		`SELECT COALESCE(MAX(offset_acked), 0) FROM "watermill_offsets_`+messaging.OutboxTopic+`" WHERE consumer_group = $1`,
		forwarderName,
	).Scan(&acked); err != nil {
		return 0, fmt.Errorf("query forwarder acked offset: %w", err)
	}
	if acked > maxOffset {
		return 0, nil
	}
	return maxOffset - acked, nil
}

// RunForwarder запускает единственный forwarder-воркер: подписывается на
// Postgres outbox-топик (messaging.OutboxTopic), разворачивает forwarder-envelope
// и публикует сообщение в Redis-стрим DestinationTopic.
//
// Forwarder работает в ОДНОМ инстансе — он единственный потребитель
// Postgres-очереди (SQL-подписка с одним offset'ом на группу не масштабируется,
// и второй инстанс дал бы только дубли в Redis). Его работа тривиальна
// (read → XADD) и не является bottleneck'ом.
//
// При недоступности Redis forwardMessage возвращает ошибку → nack → offset не
// двигается, сообщения копятся в outbox и не теряются. DLQ у forwarder'а нет
// намеренно: события из outbox выбрасывать нельзя, лечится восстановлением Redis.
func RunForwarder() {
	appCfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logFile, err := logging.Setup(appCfg.App.LogLevel, "forwarder-worker.log")
	if err != nil {
		slog.Error("failed to setup logger", "error", err)
		os.Exit(1)
	}
	if logFile != nil {
		defer func() {
			if err := logFile.Close(); err != nil {
				slog.Error("failed to close log file", "error", err)
			}
		}()
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	f := factory.New(appCfg)
	defer func() {
		if err := f.Close(); err != nil {
			slog.Error("failed to close factory", slog.String("error", err.Error()))
		}
	}()

	pool := f.MustPool()
	slog.Info("forwarder connected to database")

	redisClient := f.MustRedisClient()
	slog.Info("forwarder connected to redis")

	hb := heartbeat.New(pool, forwarderName, "forwarder", appCfg.Worker.HeartbeatInterval)
	if err := hb.Start(ctx); err != nil {
		slog.Error("failed to start heartbeat", "error", err)
	}
	defer hb.Stop()

	// Метрика лага outbox + /metrics endpoint. Лаг — единственный критичный
	// индикатор forwarder'а: копится → Redis/forwarder деградировал.
	go watchOutboxLag(ctx, pool, 15*time.Second)
	if appCfg.Metrics.Enabled {
		metricsSrv := metrics.NewServer(appCfg.Metrics.Addr)
		go func() {
			if err := metricsSrv.Start(); err != nil {
				slog.Error("metrics server error", slog.String("error", err.Error()))
			}
		}()
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := metricsSrv.Shutdown(shutdownCtx); err != nil {
				slog.Error("failed to shutdown metrics server", slog.String("error", err.Error()))
			}
		}()
	}

	wmLogger := watermill.NewSlogLogger(slog.Default())

	sqlSub, err := messaging.NewSQLSubscriber(pool, messaging.OutboxTopic, forwarderName, wmLogger)
	if err != nil {
		slog.Error("failed to create outbox subscriber", slog.String("error", err.Error()))
		os.Exit(1)
	}

	redisPub, err := messaging.NewRedisPublisher(redisClient, appCfg.Redis, wmLogger)
	if err != nil {
		slog.Error("failed to create redis publisher", slog.String("error", err.Error()))
		os.Exit(1)
	}

	fwd, err := forwarder.NewForwarder(sqlSub, redisPub, wmLogger, forwarder.Config{
		ForwarderTopic: messaging.OutboxTopic,
		Middlewares: []message.HandlerMiddleware{
			middleware.Recoverer,
			middleware.Retry{
				MaxRetries:      appCfg.Worker.RetryMaxRetries,
				InitialInterval: appCfg.Worker.RetryInitialInterval,
				MaxInterval:     appCfg.Worker.RetryMaxInterval,
				Multiplier:      appCfg.Worker.RetryMultiplier,
				Logger:          wmLogger,
			}.Middleware,
		},
		// Битый envelope nack'ается, а не теряется: после исчерпания Retry
		// сообщение останется первым в outbox и заблокирует поток — это
		// осознанно, такой случай означает баг публикации и требует руки.
		AckWhenCannotUnwrap: false,
		CloseTimeout:        appCfg.Worker.ShutdownTimeout,
	})
	if err != nil {
		slog.Error("failed to create forwarder", slog.String("error", err.Error()))
		os.Exit(1)
	}

	go func() {
		if err := fwd.Run(ctx); err != nil {
			slog.Error("forwarder error", slog.String("error", err.Error()))
			stop()
		}
	}()

	<-fwd.Running()
	slog.Info("forwarder started",
		slog.String("outbox_topic", messaging.OutboxTopic))

	<-ctx.Done()

	slog.Info("shutting down forwarder...")

	shutdownDone := make(chan struct{})
	go func() {
		if err := fwd.Close(); err != nil {
			slog.Error("failed to close forwarder", slog.String("error", err.Error()))
		}
		close(shutdownDone)
	}()

	select {
	case <-shutdownDone:
		slog.Info("forwarder shutdown complete")
	case <-time.After(appCfg.Worker.ShutdownTimeout):
		slog.Error("forwarder shutdown timed out, forcing exit")
		os.Exit(1)
	}
}
