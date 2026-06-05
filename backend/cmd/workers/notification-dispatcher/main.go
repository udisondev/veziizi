package main

import (
	"github.com/ThreeDotsLabs/watermill/message"

	_ "github.com/udisondev/veziizi/backend/internal/domain/freightrequest/events"
	_ "github.com/udisondev/veziizi/backend/internal/domain/notification/events"
	_ "github.com/udisondev/veziizi/backend/internal/domain/organization/events"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/handlers"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/messaging"
	"github.com/udisondev/veziizi/backend/internal/pkg/factory"
	"github.com/udisondev/veziizi/backend/internal/pkg/worker"
)

// notification-dispatcher делает rule-based dispatch через NotificationRulesRegistry,
// а не type switch по событию. Поэтому остаётся на legacy Handler-пути: rules
// сами выбирают, какие событиям требуются нотификации, и публикуют команды
// через NotificationBus.
func main() {
	worker.Run(worker.Config{
		Name:          "notification-dispatcher",
		Topic:         messaging.TopicFreightRequestEvents,
		ConsumerGroup: "notification_dispatcher",
		LogFile:       "notification-dispatcher-worker.log",
		Handler: func(f *factory.Factory) message.NoPublishHandlerFunc {
			return handlers.NewNotificationDispatcherHandler(
				f.DB(),
				f.ProjectionEventDedupProjection(),
				f.NotificationRulesRegistry(),
				f.NotificationService(),
				f.MustNotificationBus(),
			).Handle
		},
	})
}
