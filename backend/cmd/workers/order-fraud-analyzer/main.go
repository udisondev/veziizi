package main

import (
	// Event registration - CRITICAL for deserialization
	_ "codeberg.org/udison/veziizi/backend/internal/domain/order/events"

	"codeberg.org/udison/veziizi/backend/internal/infrastructure/handlers"
	"codeberg.org/udison/veziizi/backend/internal/pkg/factory"
	"codeberg.org/udison/veziizi/backend/internal/pkg/worker"
	"github.com/ThreeDotsLabs/watermill/message"
)

func main() {
	worker.Run(worker.Config{
		Name:          "order-fraud-analyzer",
		Topic:         "order.events",
		ConsumerGroup: "order_fraud_analyzer",
		LogFile:       "order-fraud-analyzer-worker.log",
		Handler: func(f *factory.Factory) message.NoPublishHandlerFunc {
			return handlers.NewOrderFraudAnalyzerHandler(
				f.OrderFraudProjection(),
				f.OrdersProjection(),
			).Handle
		},
	})
}
