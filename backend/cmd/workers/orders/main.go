package main

import (
	_ "codeberg.org/udison/veziizi/backend/internal/domain/order/events"
	_ "codeberg.org/udison/veziizi/backend/internal/domain/organization/events"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/handlers"
	"codeberg.org/udison/veziizi/backend/internal/pkg/factory"
	"codeberg.org/udison/veziizi/backend/internal/pkg/worker"
	"github.com/ThreeDotsLabs/watermill/message"
)

func main() {
	worker.Run(worker.Config{
		Name:          "orders",
		Topic:         "order.events",
		ConsumerGroup: "orders_projection",
		LogFile:       "orders-worker.log",
		Handler: func(f *factory.Factory) message.NoPublishHandlerFunc {
			return handlers.NewOrdersHandler(
				f.DB(),
				f.OrganizationService(),
				f.OrganizationRatingsProjection(),
			).Handle
		},
	})
}
