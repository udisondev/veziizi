package main

import (
	_ "github.com/udisondev/veziizi/backend/internal/domain/review/events"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/handlers"
	"github.com/udisondev/veziizi/backend/internal/pkg/factory"
	"github.com/udisondev/veziizi/backend/internal/pkg/worker"
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
