package main

import (
	_ "codeberg.org/udison/veziizi/backend/internal/domain/freightrequest/events"
	_ "codeberg.org/udison/veziizi/backend/internal/domain/organization/events"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/handlers"
	"codeberg.org/udison/veziizi/backend/internal/pkg/factory"
	"codeberg.org/udison/veziizi/backend/internal/pkg/worker"
	"github.com/ThreeDotsLabs/watermill/message"
)

func main() {
	worker.Run(worker.Config{
		Name:          "freight-requests",
		Topic:         "freightrequest.events",
		ConsumerGroup: "freight_requests_projection",
		LogFile:       "freight-requests-worker.log",
		Handler: func(f *factory.Factory) message.NoPublishHandlerFunc {
			return handlers.NewFreightRequestsHandler(f.DB(), f.EventStore()).Handle
		},
	})
}
