package worker

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"codeberg.org/udison/veziizi/backend/internal/pkg/config"
	"codeberg.org/udison/veziizi/backend/internal/pkg/dbtx"
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
	Handler       func(db dbtx.TxManager) message.NoPublishHandlerFunc
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
	defer logFile.Close()

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

	txManager := dbtx.NewTxExecutor(pool)
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

	handler := cfg.Handler(txManager)
	router.AddConsumerHandler(cfg.Name+"_handler", cfg.Topic, subscriber, handler)

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
