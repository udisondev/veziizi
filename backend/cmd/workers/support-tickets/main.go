package main

import (
	"log/slog"

	"github.com/ThreeDotsLabs/watermill/message"

	_ "github.com/udisondev/veziizi/backend/internal/domain/support/events"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/handlers"
	adminRepo "github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/admin"
	"github.com/udisondev/veziizi/backend/internal/pkg/factory"
	"github.com/udisondev/veziizi/backend/internal/pkg/worker"
)

// support-tickets — composite handler: сначала projection-обновление (его сбой
// возвращает err → retry), затем best-effort admin-нотификация (её сбой не
// должен мешать ack'у, иначе ticket застрянет в очереди при недоступном Telegram).
//
// CQRS не используем: ProjectionHandler делает type switch внутри, а
// SupportAdminNotifierHandler — это side effect на тот же набор событий.
// Перевод обоих на CQRS дал бы 2× регистраций per event type без выигрыша.
func main() {
	worker.Run(worker.Config{
		Name:          "support-tickets",
		Topic:         "support.events",
		ConsumerGroup: "support_tickets",
		LogFile:       "support-tickets-worker.log",
		Handler: func(f *factory.Factory) message.NoPublishHandlerFunc {
			projectionHandler := handlers.NewSupportTicketsHandler(f.DB())

			// Admin notifier подключается только если бот настроен.
			// Иначе composite вырождается в одиночный projection-handler.
			var adminNotifier *handlers.SupportAdminNotifierHandler
			if f.Config().Telegram.BotToken != "" {
				adminNotifier = handlers.NewSupportAdminNotifierHandler(
					adminRepo.NewRepository(f.DB()),
					f.MustPublisher().RawPublisher(),
				)
				slog.Info("admin telegram notifications enabled")
			} else {
				slog.Info("admin telegram notifications disabled (no bot token)")
			}

			return func(msg *message.Message) error {
				// Клонируем payload до Handle, потому что первый хендлер может
				// модифицировать buffer'ы (UnmarshalEvent внутри держит ссылки).
				payload := make([]byte, len(msg.Payload))
				copy(payload, msg.Payload)

				if err := projectionHandler.Handle(msg); err != nil {
					return err
				}

				if adminNotifier != nil {
					notif := message.NewMessage(msg.UUID, payload)
					notif.SetContext(msg.Context())
					if err := adminNotifier.Handle(notif); err != nil {
						slog.Warn("admin notification failed, continuing",
							slog.String("error", err.Error()))
					}
				}
				return nil
			}
		},
	})
}
