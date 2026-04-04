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
		Name:          "invitations",
		Topic:         "organization.events",
		ConsumerGroup: "invitations_projection",
		LogFile:       "invitations-worker.log",
		Handler: func(f *factory.Factory) message.NoPublishHandlerFunc {
			return handlers.NewInvitationsHandler(f.DB()).Handle
		},
	})
}
