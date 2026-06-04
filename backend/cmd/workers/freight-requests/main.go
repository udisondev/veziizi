package main

import (
	"github.com/ThreeDotsLabs/watermill/components/cqrs"

	_ "github.com/udisondev/veziizi/backend/internal/domain/freightrequest/events"
	_ "github.com/udisondev/veziizi/backend/internal/domain/organization/events"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/handlers"
	"github.com/udisondev/veziizi/backend/internal/pkg/factory"
	"github.com/udisondev/veziizi/backend/internal/pkg/worker"
)

func main() {
	worker.Run(worker.Config{
		Name:          "freight-requests",
		Topic:         "freightrequest.events",
		ConsumerGroup: "freight_requests_projection",
		LogFile:       "freight-requests-worker.log",
		Setup: func(f *factory.Factory, ep *cqrs.EventGroupProcessor) error {
			h := handlers.NewFreightRequestsHandler(f.DB(), f.EventStore(), f.FreightInvitesProjection())
			return ep.AddHandlersGroup("freight-requests", handlers.FreightRequestsGroupHandlers(h)...)
		},
	})
}
