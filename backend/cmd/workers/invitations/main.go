package main

import (
	"github.com/ThreeDotsLabs/watermill/components/cqrs"

	_ "github.com/udisondev/veziizi/backend/internal/domain/organization/events"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/handlers"
	"github.com/udisondev/veziizi/backend/internal/pkg/factory"
	"github.com/udisondev/veziizi/backend/internal/pkg/worker"
)

func main() {
	worker.Run(worker.Config{
		Name:          "invitations",
		Topic:         "organization.events",
		ConsumerGroup: "invitations_projection",
		LogFile:       "invitations-worker.log",
		Setup: func(f *factory.Factory, ep *cqrs.EventGroupProcessor) error {
			h := handlers.NewInvitationsHandler(f.DB())
			return ep.AddHandlersGroup("invitations", handlers.InvitationsGroupHandlers(h)...)
		},
	})
}
