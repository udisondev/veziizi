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
		Name:          "pending-organizations",
		Topic:         "organization.events",
		ConsumerGroup: "pending_organizations_projection",
		LogFile:       "pending-organizations-worker.log",
		Setup: func(f *factory.Factory, ep *cqrs.EventGroupProcessor) error {
			h := handlers.NewPendingOrganizationsHandler(f.DB())
			return ep.AddHandlersGroup("pending-organizations",
				cqrs.NewGroupEventHandler(h.OnCreated),
				cqrs.NewGroupEventHandler(h.OnApproved),
				cqrs.NewGroupEventHandler(h.OnRejected),
			)
		},
	})
}
