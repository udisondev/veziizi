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
		Name:          "review-receiver",
		Topic:         "order.events",
		ConsumerGroup: "review_receiver",
		LogFile:       "review-receiver-worker.log",
		Handler: func(f *factory.Factory) message.NoPublishHandlerFunc {
			return handlers.NewReviewReceiverHandler(f.ReviewService()).Handle
		},
	})
}
