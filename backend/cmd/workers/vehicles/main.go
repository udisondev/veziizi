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
		Name:          "vehicles",
		Topic:         "organization.events",
		ConsumerGroup: "vehicles_projection",
		LogFile:       "vehicles-worker.log",
		Handler: func(f *factory.Factory) message.NoPublishHandlerFunc {
			return handlers.NewVehiclesHandler(f.DB(), f.VehiclesProjection(), f.PendingVehiclesProjection()).Handle
		},
	})
}
