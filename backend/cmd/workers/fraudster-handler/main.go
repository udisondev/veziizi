package main

import (
	// Event registration - CRITICAL for deserialization
	_ "codeberg.org/udison/veziizi/backend/internal/domain/organization/events"
	_ "codeberg.org/udison/veziizi/backend/internal/domain/review/events"

	"codeberg.org/udison/veziizi/backend/internal/infrastructure/handlers"
	"codeberg.org/udison/veziizi/backend/internal/pkg/factory"
	"codeberg.org/udison/veziizi/backend/internal/pkg/worker"
	"github.com/ThreeDotsLabs/watermill/message"
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
