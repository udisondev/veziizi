package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	// IMPORTANT: Register event types for deserialization
	_ "github.com/udisondev/veziizi/backend/internal/domain/support/events"

	"github.com/udisondev/veziizi/backend/internal/infrastructure/handlers"
	adminRepo "github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/admin"
	"github.com/udisondev/veziizi/backend/internal/pkg/config"
	"github.com/udisondev/veziizi/backend/internal/pkg/logging"
	"github.com/udisondev/veziizi/backend/internal/pkg/factory"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-sql/v4/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/message"
)

const (
	workerName    = "support-tickets"
	topic         = "support.events"
	consumerGroup = "support_tickets"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", slog.String("error", err.Error()))
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

	// Create factory - all dependencies are lazily initialized
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

	// Create handlers
	projectionHandler := handlers.NewSupportTicketsHandler(f.DB())

	// Create admin notifier only if Telegram bot is configured
	var adminNotifier *handlers.SupportAdminNotifierHandler
	if cfg.Telegram.BotToken != "" {
		adminRepository := adminRepo.NewRepository(f.DB())
		adminNotifier = handlers.NewSupportAdminNotifierHandler(
			adminRepository,
			f.MustPublisher().RawPublisher(),
		)
		slog.Info("admin telegram notifications enabled")
	} else {
		slog.Info("admin telegram notifications disabled (no bot token)")
	}

	// Composite handler that calls both handlers
	compositeHandler := func(msg *message.Message) error {
		// Clone payload for second handler
		payload := make([]byte, len(msg.Payload))
		copy(payload, msg.Payload)

		// First: update projection
		if err := projectionHandler.Handle(msg); err != nil {
			return err // Fail the message to retry
		}

		// Second: send admin notifications (don't fail on errors)
		if adminNotifier != nil {
			notifMsg := message.NewMessage(msg.UUID, payload)
			notifMsg.SetContext(msg.Context())
			if err := adminNotifier.Handle(notifMsg); err != nil {
				slog.Warn("admin notification failed, continuing",
					slog.String("error", err.Error()))
			}
		}

		return nil
	}

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

	if err := subscriber.SubscribeInitialize(topic); err != nil {
		slog.Error("failed to initialize subscriber", slog.String("error", err.Error()))
		os.Exit(1)
	}

	router, err := message.NewRouter(message.RouterConfig{}, wmLogger)
	if err != nil {
		slog.Error("failed to create router", slog.String("error", err.Error()))
		os.Exit(1)
	}

	router.AddNoPublisherHandler(workerName+"_handler", topic, subscriber, compositeHandler)

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

