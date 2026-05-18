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
		Name:          "organizations",
		Topic:         "organization.events",
		ConsumerGroup: "organizations_projection",
		LogFile:       "organizations-worker.log",
		Setup: func(f *factory.Factory, ep *cqrs.EventGroupProcessor) error {
			h := handlers.NewOrganizationsHandler(
				f.OrganizationsProjection(),
				f.FreightRequestsProjection(),
			)
			return ep.AddHandlersGroup("organizations",
				cqrs.NewGroupEventHandler(h.OnCreated),
				cqrs.NewGroupEventHandler(h.OnApproved),
				cqrs.NewGroupEventHandler(h.OnRejected),
				cqrs.NewGroupEventHandler(h.OnSuspended),
				cqrs.NewGroupEventHandler(h.OnUpdated),
			)
		},
	})
}
