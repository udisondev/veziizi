package main

import (
	"github.com/ThreeDotsLabs/watermill/components/cqrs"

	_ "github.com/udisondev/veziizi/backend/internal/domain/review/events"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/handlers"
	"github.com/udisondev/veziizi/backend/internal/pkg/factory"
	"github.com/udisondev/veziizi/backend/internal/pkg/worker"
)

func main() {
	worker.Run(worker.Config{
		Name:          "reviews-projection",
		Topic:         "review.events",
		ConsumerGroup: "reviews_projection",
		LogFile:       "reviews-projection-worker.log",
		Setup: func(f *factory.Factory, ep *cqrs.EventGroupProcessor) error {
			h := handlers.NewReviewsProjectionHandler(
				f.DB(),
				f.FraudDataProjection(),
				f.OrganizationRatingsProjection(),
			)
			return ep.AddHandlersGroup("reviews-projection",
				cqrs.NewGroupEventHandler(h.OnReceived),
				cqrs.NewGroupEventHandler(h.OnEdited),
				cqrs.NewGroupEventHandler(h.OnAnalyzed),
				cqrs.NewGroupEventHandler(h.OnApproved),
				cqrs.NewGroupEventHandler(h.OnRejected),
				cqrs.NewGroupEventHandler(h.OnActivated),
				cqrs.NewGroupEventHandler(h.OnDeactivated),
			)
		},
	})
}
