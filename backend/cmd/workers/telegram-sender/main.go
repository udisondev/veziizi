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

// telegram-sender читает notification.telegram — payload это NotificationMessage
// из rules, не EventEnvelope. CQRS marshaler здесь не подходит, поэтому
// используем legacy Handler-путь — он всё равно получает DLQ + middleware из
// worker.Run.
func main() {
	// fail fast если бот не настроен — иначе worker всё равно ничего не отправит.
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
		Topic:         "notification.telegram",
		ConsumerGroup: "telegram_sender",
		LogFile:       "telegram-sender-worker.log",
		Handler: func(f *factory.Factory) message.NoPublishHandlerFunc {
			return handlers.NewTelegramSenderHandler(
				f.TelegramClient(),
				cfg,
				f.DeliveryLogProjection(),
			).Handle
		},
	})
}
