package main

import (
	// Event registration - CRITICAL for deserialization
	_ "github.com/udisondev/veziizi/backend/internal/domain/organization/events"
	_ "github.com/udisondev/veziizi/backend/internal/domain/review/events"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/handlers"
	"github.com/udisondev/veziizi/backend/internal/pkg/factory"
	"github.com/udisondev/veziizi/backend/internal/pkg/worker"
)

func main() {
	worker.Run(worker.Config{
		Name:          "fraudster-handler",
		Topic:         "organization.events",
		ConsumerGroup: "fraudster_handler",
		LogFile:       "fraudster-handler-worker.log",
		Handler: func(f *factory.Factory) message.NoPublishHandlerFunc {
			return handlers.NewFraudsterHandler(
				f.ReviewService(),
				f.ReviewsProjection(),
				f.FraudDataProjection(),
			).Handle
		},
	})
}
