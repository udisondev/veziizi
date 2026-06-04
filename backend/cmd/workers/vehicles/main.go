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
		Name:          "vehicles",
		Topic:         "organization.events",
		ConsumerGroup: "vehicles_projection",
		LogFile:       "vehicles-worker.log",
		Setup: func(f *factory.Factory, ep *cqrs.EventGroupProcessor) error {
			h := handlers.NewVehiclesHandler(f.DB(), f.VehiclesProjection(), f.PendingVehiclesProjection())
			return ep.AddHandlersGroup("vehicles", handlers.VehiclesGroupHandlers(h)...)
		},
	})
}
