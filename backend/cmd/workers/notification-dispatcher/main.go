package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	// Event registration - CRITICAL
	_ "github.com/udisondev/veziizi/backend/internal/domain/freightrequest/events"
	_ "github.com/udisondev/veziizi/backend/internal/domain/notification/events"
	_ "github.com/udisondev/veziizi/backend/internal/domain/organization/events"

	"github.com/udisondev/veziizi/backend/internal/infrastructure/handlers"
	"github.com/udisondev/veziizi/backend/internal/pkg/config"
	"github.com/udisondev/veziizi/backend/internal/pkg/logging"
	"github.com/udisondev/veziizi/backend/internal/pkg/factory"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-sql/v4/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/message"
)

const (
	workerName    = "notification-dispatcher"
	consumerGroup = "notification_dispatcher"
)

var topics = []string{
	"freightrequest.events",
}

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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create factory IoC container - all dependencies are lazily initialized
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

	// Create handler
	handler := handlers.NewNotificationDispatcherHandler(
		f.NotificationRulesRegistry(),
		f.NotificationService(),
		f.MustPublisher().RawPublisher(),
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

	// Initialize subscriber for all topics
	for _, topic := range topics {
		if err := subscriber.SubscribeInitialize(topic); err != nil {
			slog.Error("failed to initialize subscriber",
				slog.String("topic", topic),
				slog.String("error", err.Error()))
			os.Exit(1)
		}
	}

	// Create router
	router, err := message.NewRouter(message.RouterConfig{}, wmLogger)
	if err != nil {
		slog.Error("failed to create router", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Add handler for each topic
	for _, topic := range topics {
		handlerName := fmt.Sprintf("%s_%s_handler", workerName, topic)
		router.AddConsumerHandler(handlerName, topic, subscriber, handler.Handle)
	}

	go func() {
		if err := router.Run(ctx); err != nil {
			slog.Error("router error", slog.String("error", err.Error()))
			cancel()
		}
	}()

	slog.Info(fmt.Sprintf("%s worker started, listening to topics: %v", workerName, topics))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

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

