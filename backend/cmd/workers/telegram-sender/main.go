package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	// Event registration - CRITICAL for deserialization of notification events
	_ "github.com/udisondev/veziizi/backend/internal/domain/notification/events"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-sql/v4/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/handlers"
	"github.com/udisondev/veziizi/backend/internal/pkg/config"
	"github.com/udisondev/veziizi/backend/internal/pkg/factory"
	"github.com/udisondev/veziizi/backend/internal/pkg/logging"
)

const (
	workerName    = "telegram-sender"
	consumerGroup = "telegram_sender"
	topic         = "notification.telegram"
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

	logFile, err := logging.Setup(cfg.App.LogLevel, cfg.App.LogFile)
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
			stop()
		}
	}()

	slog.Info(fmt.Sprintf("%s worker started, listening to topic: %s", workerName, topic))

	<-ctx.Done()

	slog.Info(fmt.Sprintf("shutting down %s worker...", workerName))

	shutdownDone := make(chan struct{})
	go func() {
		if err := router.Close(); err != nil {
			slog.Error("failed to close router", "error", err)
		}
		close(shutdownDone)
	}()

	select {
	case <-shutdownDone:
		slog.Info(fmt.Sprintf("%s worker stopped gracefully", workerName))
	case <-time.After(30 * time.Second):
		slog.Error(fmt.Sprintf("%s worker shutdown timed out", workerName))
	}
}
