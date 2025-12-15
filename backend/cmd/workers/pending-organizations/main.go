package main

import (
	_ "codeberg.org/udison/veziizi/backend/internal/domain/organization/events"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/handlers"
	"codeberg.org/udison/veziizi/backend/internal/pkg/dbtx"
	"codeberg.org/udison/veziizi/backend/internal/pkg/worker"
	"github.com/ThreeDotsLabs/watermill/message"
)

func main() {
	worker.Run(worker.Config{
		Name:          "pending-organizations",
		Topic:         "organization.events",
		ConsumerGroup: "pending_organizations_projection",
		LogFile:       "pending-organizations-worker.log",
		Handler: func(db dbtx.TxManager) message.NoPublishHandlerFunc {
			return handlers.NewPendingOrganizationsHandler(db).Handle
		},
	})
}
