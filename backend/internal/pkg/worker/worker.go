package worker

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/udisondev/veziizi/backend/internal/pkg/config"
	"github.com/udisondev/veziizi/backend/internal/pkg/factory"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-sql/v4/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/message"
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

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Create factory - all dependencies are lazily initialized
	f := factory.New(appCfg)
	defer func() {
		if err := f.Close(); err != nil {
			slog.Error("failed to close factory", slog.String("error", err.Error()))
		}
	}()

	// Get pool for subscriber (triggers lazy initialization)
	pool := f.MustPool()
	slog.Info(fmt.Sprintf("%s worker connected to database", cfg.Name))

	wmLogger := watermill.NewSlogLogger(slog.Default())

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
			stop()
		}
	}()

	slog.Info(fmt.Sprintf("%s worker started", cfg.Name))

	<-ctx.Done()

	slog.Info(fmt.Sprintf("shutting down %s worker...", cfg.Name))

	// Graceful shutdown с таймаутом
	shutdownDone := make(chan struct{})
	go func() {
		if err := router.Close(); err != nil {
			slog.Error("failed to close router", slog.String("error", err.Error()))
		}
		close(shutdownDone)
	}()

	select {
	case <-shutdownDone:
		slog.Info(fmt.Sprintf("%s worker shutdown complete", cfg.Name))
	case <-time.After(30 * time.Second):
		slog.Error(fmt.Sprintf("%s worker shutdown timed out", cfg.Name))
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

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Create factory - all dependencies are lazily initialized
	f := factory.New(appCfg)
	defer func() {
		if err := f.Close(); err != nil {
			slog.Error("failed to close factory", slog.String("error", err.Error()))
		}
	}()

	// Trigger pool initialization and log connection
	_ = f.MustPool()
	slog.Info(fmt.Sprintf("%s scheduled worker connected to database", cfg.Name))

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

	for {
		select {
		case <-ticker.C:
			if err := handler(ctx); err != nil {
				slog.Error("handler error", slog.String("error", err.Error()))
			}
		case <-ctx.Done():
			slog.Info(fmt.Sprintf("shutting down %s scheduled worker...", cfg.Name))
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
