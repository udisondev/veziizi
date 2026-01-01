package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	// IMPORTANT: Register event types for deserialization
	_ "codeberg.org/udison/veziizi/backend/internal/domain/support/events"

	"codeberg.org/udison/veziizi/backend/internal/infrastructure/handlers"
	adminRepo "codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/admin"
	"codeberg.org/udison/veziizi/backend/internal/pkg/config"
	"codeberg.org/udison/veziizi/backend/internal/pkg/factory"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-sql/v4/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/message"
)

const (
	workerName    = "support-tickets"
	topic         = "support.events"
	consumerGroup = "support_tickets"
	logFile       = "support-tickets-worker.log"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logF, err := setupLogger(cfg.App.LogLevel)
	if err != nil {
		slog.Error("failed to setup logger", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer func() {
		if err := logF.Close(); err != nil {
			slog.Error("failed to close log file", slog.String("error", err.Error()))
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

// Unused but kept for potential future use
var _ = json.Marshal
