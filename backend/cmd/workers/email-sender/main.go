package main

import (
	"log/slog"
	"os"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"

	"github.com/udisondev/veziizi/backend/internal/infrastructure/handlers"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/messaging"
	"github.com/udisondev/veziizi/backend/internal/pkg/config"
	"github.com/udisondev/veziizi/backend/internal/pkg/factory"
	"github.com/udisondev/veziizi/backend/internal/pkg/worker"
)

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
		Topic:         messaging.TopicNotificationEmail,
		ConsumerGroup: "email_sender",
		LogFile:       "email-sender-worker.log",
		Marshaler:     messaging.NotificationMarshaler(),
		Setup: func(f *factory.Factory, ep *cqrs.EventGroupProcessor) error {
			h := handlers.NewEmailSenderHandler(
				f.EmailProvider(),
				cfg,
				f.DeliveryLogProjection(),
				f.NotificationDedupProjection(),
			)
			return ep.AddHandlersGroup("email-sender", handlers.EmailSenderGroupHandlers(h)...)
		},
	})
}
