package main

import (
	"github.com/ThreeDotsLabs/watermill/components/cqrs"

	_ "github.com/udisondev/veziizi/backend/internal/domain/support/events"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/handlers"
	"github.com/udisondev/veziizi/backend/internal/pkg/factory"
	"github.com/udisondev/veziizi/backend/internal/pkg/worker"
)

// support-tickets — projection-воркер, обновляет support_tickets_lookup.
// Admin-нотификации обрабатываются отдельным воркером support-tickets-notifier
// со своей consumer group: их сбой не блокирует обновление проекции.
func main() {
	worker.Run(worker.Config{
		Name:          "support-tickets",
		Topic:         "support.events",
		ConsumerGroup: "support_tickets_projection",
		LogFile:       "support-tickets-worker.log",
		Setup: func(f *factory.Factory, ep *cqrs.EventGroupProcessor) error {
			h := handlers.NewSupportTicketsHandler(f.DB())
			return ep.AddHandlersGroup("support-tickets", handlers.SupportTicketsGroupHandlers(h)...)
		},
	})
}
