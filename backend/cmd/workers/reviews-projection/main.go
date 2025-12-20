package main

import (
	_ "codeberg.org/udison/veziizi/backend/internal/domain/review/events"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/handlers"
	"codeberg.org/udison/veziizi/backend/internal/pkg/factory"
	"codeberg.org/udison/veziizi/backend/internal/pkg/worker"
	"github.com/ThreeDotsLabs/watermill/message"
)

func main() {
	worker.Run(worker.Config{
		Name:          "reviews-projection",
		Topic:         "review.events",
		ConsumerGroup: "reviews_projection",
		LogFile:       "reviews-projection-worker.log",
		Handler: func(f *factory.Factory) message.NoPublishHandlerFunc {
			return handlers.NewReviewsProjectionHandler(
				f.DB(),
				f.FraudDataProjection(),
				f.OrganizationRatingsProjection(),
			).Handle
		},
	})
}
