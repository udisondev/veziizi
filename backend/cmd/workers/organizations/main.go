package main

import (
	"github.com/ThreeDotsLabs/watermill/message"
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
		Handler: func(f *factory.Factory) message.NoPublishHandlerFunc {
			return handlers.NewOrganizationsHandler(
				f.OrganizationsProjection(),
				f.FreightRequestsProjection(),
			).Handle
		},
	})
}
