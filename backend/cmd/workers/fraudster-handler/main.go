package main

import (
	"github.com/ThreeDotsLabs/watermill/components/cqrs"

	_ "github.com/udisondev/veziizi/backend/internal/domain/organization/events"
	_ "github.com/udisondev/veziizi/backend/internal/domain/review/events"
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
		Setup: func(f *factory.Factory, ep *cqrs.EventGroupProcessor) error {
			h := handlers.NewFraudsterHandler(
				f.ReviewService(),
				f.ReviewsProjection(),
				f.FraudDataProjection(),
			)
			return ep.AddHandlersGroup("fraudster-handler",
				cqrs.NewGroupEventHandler(h.OnFraudsterMarked),
				cqrs.NewGroupEventHandler(h.OnFraudsterUnmarked),
			)
		},
	})
}
