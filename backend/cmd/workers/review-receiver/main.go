package main

import (
	// Event registration - CRITICAL for deserialization
	_ "github.com/udisondev/veziizi/backend/internal/domain/freightrequest/events"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/handlers"
	"github.com/udisondev/veziizi/backend/internal/pkg/factory"
	"github.com/udisondev/veziizi/backend/internal/pkg/worker"
)

func main() {
	worker.Run(worker.Config{
		Name:          "review-receiver",
		Topic:         "freightrequest.events",
		ConsumerGroup: "review_receiver",
		LogFile:       "review-receiver-worker.log",
		Handler: func(f *factory.Factory) message.NoPublishHandlerFunc {
			return handlers.NewReviewReceiverHandler(f.ReviewService()).Handle
		},
	})
}
