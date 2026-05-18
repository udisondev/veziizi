package main

import (
	"log/slog"
	"os"

	"github.com/ThreeDotsLabs/watermill/message"

	_ "github.com/udisondev/veziizi/backend/internal/domain/notification/events"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/handlers"
	"github.com/udisondev/veziizi/backend/internal/pkg/config"
	"github.com/udisondev/veziizi/backend/internal/pkg/factory"
	"github.com/udisondev/veziizi/backend/internal/pkg/worker"
)

// email-sender читает notification.email — payload это NotificationMessage из
// rules, не EventEnvelope. CQRS marshaler здесь не подходит, поэтому используем
// legacy Handler-путь — он всё равно получает DLQ + middleware из worker.Run.
func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}
	if !cfg.Email.Enabled {
		slog.Warn("email is disabled, worker will use noop provider")
	}

	worker.Run(worker.Config{
		Name:          "email-sender",
		Topic:         "notification.email",
		ConsumerGroup: "email_sender",
		LogFile:       "email-sender-worker.log",
		Handler: func(f *factory.Factory) message.NoPublishHandlerFunc {
			return handlers.NewEmailSenderHandler(
				f.EmailProvider(),
				cfg,
				f.DeliveryLogProjection(),
			).Handle
		},
	})
}
