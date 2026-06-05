package main

import (
	"log/slog"
	"os"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"

	_ "github.com/udisondev/veziizi/backend/internal/domain/support/events"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/handlers"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/messaging"
	adminRepo "github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/admin"
	"github.com/udisondev/veziizi/backend/internal/pkg/config"
	"github.com/udisondev/veziizi/backend/internal/pkg/factory"
	"github.com/udisondev/veziizi/backend/internal/pkg/worker"
)

// support-tickets-notifier рассылает админам уведомления в Telegram о новых
// тикетах и пользовательских сообщениях. Отдельный воркер с собственной
// consumer group — чтобы сбой Telegram-отправки или БД админов не блокировал
// projection-воркер support-tickets.
//
// Воркер не запускается, если бот не настроен — в этом случае нотифицировать
// некуда, и держать пустого подписчика бессмысленно.
func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}
	if cfg.Telegram.BotToken == "" {
		slog.Info("TELEGRAM_BOT_TOKEN not set, support-tickets-notifier worker is disabled")
		return
	}

	worker.Run(worker.Config{
		Name:          "support-tickets-notifier",
		Topic:         messaging.TopicSupportEvents,
		ConsumerGroup: "support_tickets_admin_notifier",
		LogFile:       "support-tickets-notifier-worker.log",
		Setup: func(f *factory.Factory, ep *cqrs.EventGroupProcessor) error {
			h := handlers.NewSupportAdminNotifierHandler(
				f.DB(),
				f.ProjectionEventDedupProjection(),
				adminRepo.NewRepository(f.DB()),
				f.MustNotificationBus(),
			)
			return ep.AddHandlersGroup("support-tickets-notifier", handlers.SupportAdminNotifierGroupHandlers(h)...)
		},
	})
}
