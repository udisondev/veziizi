package main

import (
	_ "github.com/udisondev/veziizi/backend/internal/domain/organization/events"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/handlers"
	"github.com/udisondev/veziizi/backend/internal/pkg/factory"
	"github.com/udisondev/veziizi/backend/internal/pkg/worker"
	"github.com/ThreeDotsLabs/watermill/message"
)

func main() {
	worker.Run(worker.Config{
		Name:          "pending-organizations",
		Topic:         "organization.events",
		ConsumerGroup: "pending_organizations_projection",
		LogFile:       "pending-organizations-worker.log",
		Handler: func(f *factory.Factory) message.NoPublishHandlerFunc {
			return handlers.NewPendingOrganizationsHandler(f.DB()).Handle
		},
	})
}
