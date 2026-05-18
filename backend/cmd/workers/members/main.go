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
		Name:          "members",
		Topic:         "organization.events",
		ConsumerGroup: "members_projection",
		LogFile:       "members-worker.log",
		Setup: func(f *factory.Factory, ep *cqrs.EventGroupProcessor) error {
			h := handlers.NewMembersHandler(f.DB())
			return ep.AddHandlersGroup("members",
				cqrs.NewGroupEventHandler(h.OnMemberAdded),
				cqrs.NewGroupEventHandler(h.OnMemberRemoved),
				cqrs.NewGroupEventHandler(h.OnMemberRoleChanged),
				cqrs.NewGroupEventHandler(h.OnMemberBlocked),
				cqrs.NewGroupEventHandler(h.OnMemberUnblocked),
			)
		},
	})
}
