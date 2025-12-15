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
		Name:          "members",
		Topic:         "organization.events",
		ConsumerGroup: "members_projection",
		LogFile:       "members-worker.log",
		Handler: func(db dbtx.TxManager) message.NoPublishHandlerFunc {
			return handlers.NewMembersHandler(db).Handle
		},
	})
}
