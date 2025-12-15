package main

import (
	_ "codeberg.org/udison/veziizi/backend/internal/domain/freightrequest/events"
	_ "codeberg.org/udison/veziizi/backend/internal/domain/order/events"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/handlers"
	"codeberg.org/udison/veziizi/backend/internal/pkg/factory"
	"codeberg.org/udison/veziizi/backend/internal/pkg/worker"
	"github.com/ThreeDotsLabs/watermill/message"
)

func main() {
	worker.Run(worker.Config{
		Name:          "order-creator",
		Topic:         "freightrequest.events",
		ConsumerGroup: "order_creator",
		LogFile:       "order-creator-worker.log",
		Handler: func(f *factory.Factory) message.NoPublishHandlerFunc {
			return handlers.NewOrderCreatorHandler(f.OrderService()).Handle
		},
	})
}
