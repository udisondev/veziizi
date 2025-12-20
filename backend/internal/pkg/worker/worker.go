package worker

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"codeberg.org/udison/veziizi/backend/internal/infrastructure/messaging"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/filestorage"
	"codeberg.org/udison/veziizi/backend/internal/pkg/config"
	"codeberg.org/udison/veziizi/backend/internal/pkg/dbtx"
	"codeberg.org/udison/veziizi/backend/internal/pkg/factory"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-sql/v4/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	Name          string
	Topic         string
	ConsumerGroup string
	LogFile       string

	// Handler receives Factory and returns message handler
	Handler func(f *factory.Factory) message.NoPublishHandlerFunc
}

func Run(cfg Config) {
	appCfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load cconfig", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logFile, err := setupLogger(appCfg.App.LogLevel, cfg.LogFile)
	if err != nil {
		slog.Error("failed to setup logger", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer func() {
		if err := logFile.Close(); err != nil {
			slog.Error("failed to close log file", slog.String("error", err.Error()))
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := pgxpool.New(ctx, appCfg.Database.URL)
	if err != nil {
		slog.Error("failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		slog.Error("failed to ping database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	slog.Info(fmt.Sprintf("%s worker connected to database", cfg.Name))

	wmLogger := watermill.NewSlogLogger(slog.Default())

	// Create base dependencies
	txManager := dbtx.NewTxExecutor(pool)
	es := eventstore.NewPostgresStore(txManager)
	fs := filestorage.NewPostgresStorage(txManager)

	publisher, err := messaging.NewEventPublisher(pool, wmLogger)
	if err != nil {
		slog.Error("failed to create event publisher", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer func() {
		if err := publisher.Close(); err != nil {
			slog.Error("failed to close publisher", slog.String("error", err.Error()))
		}
	}()

	// Create factory
	f := factory.New(txManager, es, publisher, fs)

	subscriber, err := sql.NewSubscriber(
		sql.BeginnerFromPgx(pool),
		sql.SubscriberConfig{
			SchemaAdapter:  sql.DefaultPostgreSQLSchema{},
			OffsetsAdapter: sql.DefaultPostgreSQLOffsetsAdapter{},
			ConsumerGroup:  cfg.ConsumerGroup,
		},
		wmLogger,
	)
	if err != nil {
		slog.Error("failed to create subscriber", slog.String("error", err.Error()))
		os.Exit(1)
	}

	if err := subscriber.SubscribeInitialize(cfg.Topic); err != nil {
		slog.Error("failed to initialize subscriber", slog.String("error", err.Error()))
		os.Exit(1)
	}

	router, err := message.NewRouter(message.RouterConfig{}, wmLogger)
	if err != nil {
		slog.Error("failed to create router", slog.String("error", err.Error()))
		os.Exit(1)
	}

	router.AddConsumerHandler(cfg.Name+"_handler", cfg.Topic, subscriber, cfg.Handler(f))

	go func() {
		if err := router.Run(ctx); err != nil {
			slog.Error("router error", slog.String("error", err.Error()))
			cancel()
		}
	}()

	slog.Info(fmt.Sprintf("%s worker started", cfg.Name))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info(fmt.Sprintf("shutting down %s worker...", cfg.Name))
	if err := router.Close(); err != nil {
		slog.Error("failed to close router", slog.String("error", err.Error()))
	}
}

// ScheduledConfig configures a scheduled (cron-like) worker
type ScheduledConfig struct {
	Name     string
	Interval time.Duration
	LogFile  string

	// Handler receives Factory and returns a function to execute on each tick
	Handler func(f *factory.Factory) func(ctx context.Context) error
}

// RunScheduled runs a scheduled worker that executes at regular intervals
func RunScheduled(cfg ScheduledConfig) {
	appCfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logFile, err := setupLogger(appCfg.App.LogLevel, cfg.LogFile)
	if err != nil {
		slog.Error("failed to setup logger", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer func() {
		if err := logFile.Close(); err != nil {
			slog.Error("failed to close log file", slog.String("error", err.Error()))
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := pgxpool.New(ctx, appCfg.Database.URL)
	if err != nil {
		slog.Error("failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		slog.Error("failed to ping database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	slog.Info(fmt.Sprintf("%s scheduled worker connected to database", cfg.Name))

	wmLogger := watermill.NewSlogLogger(slog.Default())

	// Create base dependencies
	txManager := dbtx.NewTxExecutor(pool)
	es := eventstore.NewPostgresStore(txManager)
	fs := filestorage.NewPostgresStorage(txManager)

	publisher, err := messaging.NewEventPublisher(pool, wmLogger)
	if err != nil {
		slog.Error("failed to create event publisher", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer func() {
		if err := publisher.Close(); err != nil {
			slog.Error("failed to close publisher", slog.String("error", err.Error()))
		}
	}()

	// Create factory
	f := factory.New(txManager, es, publisher, fs)

	// Get handler
	handler := cfg.Handler(f)

	// Start ticker
	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	slog.Info(fmt.Sprintf("%s scheduled worker started (interval: %s)", cfg.Name, cfg.Interval))

	// Run immediately on start
	go func() {
		if err := handler(ctx); err != nil {
			slog.Error("handler error on startup", slog.String("error", err.Error()))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-ticker.C:
			if err := handler(ctx); err != nil {
				slog.Error("handler error", slog.String("error", err.Error()))
			}
		case <-quit:
			slog.Info(fmt.Sprintf("shutting down %s scheduled worker...", cfg.Name))
			return
		case <-ctx.Done():
			return
		}
	}
}

func setupLogger(levelStr, logFile string) (*os.File, error) {
	var level slog.Level
	if err := level.UnmarshalText([]byte(levelStr)); err != nil {
		level = slog.LevelInfo
	}

	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	handler := slog.NewJSONHandler(file, &slog.HandlerOptions{
		Level: level,
	})
	slog.SetDefault(slog.New(handler))

	return file, nil
}
