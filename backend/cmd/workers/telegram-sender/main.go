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
	if cfg.Telegram.BotToken == "" {
		slog.Error("TELEGRAM_BOT_TOKEN is required for telegram-sender worker")
		os.Exit(1)
	}

	worker.Run(worker.Config{
		Name:          "telegram-sender",
		Topic:         messaging.TopicNotificationTelegram,
		ConsumerGroup: "telegram_sender",
		LogFile:       "telegram-sender-worker.log",
		Marshaler:     messaging.NotificationMarshaler(),
		Setup: func(f *factory.Factory, ep *cqrs.EventGroupProcessor) error {
			h := handlers.NewTelegramSenderHandler(
				f.TelegramClient(),
				cfg,
				f.DeliveryLogProjection(),
				f.NotificationDedupProjection(),
			)
			return ep.AddHandlersGroup("telegram-sender", handlers.TelegramSenderGroupHandlers(h)...)
		},
	})
}
