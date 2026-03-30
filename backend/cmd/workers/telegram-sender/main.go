package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	// Event registration - CRITICAL for deserialization of notification events
	_ "github.com/udisondev/veziizi/backend/internal/domain/notification/events"

	"github.com/udisondev/veziizi/backend/internal/infrastructure/handlers"
	"github.com/udisondev/veziizi/backend/internal/pkg/config"
	"github.com/udisondev/veziizi/backend/internal/pkg/factory"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-sql/v4/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/message"
)

const (
	workerName    = "telegram-sender"
	consumerGroup = "telegram_sender"
	topic         = "notification.telegram"
	logFile       = "telegram-sender-worker.log"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Проверяем наличие токена бота
	if cfg.Telegram.BotToken == "" {
		slog.Error("TELEGRAM_BOT_TOKEN is required for telegram-sender worker")
		os.Exit(1)
	}

	file, err := setupLogger(cfg.App.LogLevel)
	if err != nil {
		slog.Error("failed to setup logger", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer func() {
		if err := file.Close(); err != nil {
			slog.Error("failed to close log file", slog.String("error", err.Error()))
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create factory IoC container - only needed dependencies will be lazily initialized
	// telegram-sender doesn't need publisher (only consumes from queue)
	f := factory.New(cfg)
	defer func() {
		if err := f.Close(); err != nil {
			slog.Error("failed to close factory", slog.String("error", err.Error()))
		}
	}()

	// Get pool for subscriber (triggers lazy initialization)
	pool := f.MustPool()
	slog.Info(fmt.Sprintf("%s worker connected to database", workerName))

	wmLogger := watermill.NewSlogLogger(slog.Default())

	// Create handler - uses TelegramClient and DeliveryLogProjection from factory
	handler := handlers.NewTelegramSenderHandler(
		f.TelegramClient(),
		cfg,
		f.DeliveryLogProjection(),
	)

	// Create subscriber
	subscriber, err := sql.NewSubscriber(
		sql.BeginnerFromPgx(pool),
		sql.SubscriberConfig{
			SchemaAdapter:  sql.DefaultPostgreSQLSchema{},
			OffsetsAdapter: sql.DefaultPostgreSQLOffsetsAdapter{},
			ConsumerGroup:  consumerGroup,
		},
		wmLogger,
	)
	if err != nil {
		slog.Error("failed to create subscriber", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Initialize subscriber
	if err := subscriber.SubscribeInitialize(topic); err != nil {
		slog.Error("failed to initialize subscriber",
			slog.String("topic", topic),
			slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Create router
	router, err := message.NewRouter(message.RouterConfig{}, wmLogger)
	if err != nil {
		slog.Error("failed to create router", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Add handler
	router.AddConsumerHandler(workerName+"_handler", topic, subscriber, handler.Handle)

	go func() {
		if err := router.Run(ctx); err != nil {
			slog.Error("router error", slog.String("error", err.Error()))
			cancel()
		}
	}()

	slog.Info(fmt.Sprintf("%s worker started, listening to topic: %s", workerName, topic))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info(fmt.Sprintf("shutting down %s worker...", workerName))
	if err := router.Close(); err != nil {
		slog.Error("failed to close router", slog.String("error", err.Error()))
	}
}

func setupLogger(levelStr string) (*os.File, error) {
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
