package main

import (
	"github.com/ThreeDotsLabs/watermill/components/cqrs"

	_ "github.com/udisondev/veziizi/backend/internal/domain/freightrequest/events"
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
		Setup: func(f *factory.Factory, ep *cqrs.EventGroupProcessor) error {
			h := handlers.NewReviewReceiverHandler(f.ReviewService())
			return ep.AddHandlersGroup("review-receiver", handlers.ReviewReceiverGroupHandlers(h)...)
		},
	})
}
