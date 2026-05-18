package main

import (
	"github.com/ThreeDotsLabs/watermill/components/cqrs"

	_ "github.com/udisondev/veziizi/backend/internal/domain/freightrequest/events"
	_ "github.com/udisondev/veziizi/backend/internal/domain/organization/events"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/handlers"
	"github.com/udisondev/veziizi/backend/internal/pkg/factory"
	"github.com/udisondev/veziizi/backend/internal/pkg/worker"
)

func main() {
	worker.Run(worker.Config{
		Name:          "freight-requests",
		Topic:         "freightrequest.events",
		ConsumerGroup: "freight_requests_projection",
		LogFile:       "freight-requests-worker.log",
		Setup: func(f *factory.Factory, ep *cqrs.EventGroupProcessor) error {
			h := handlers.NewFreightRequestsHandler(f.DB(), f.EventStore(), f.FreightInvitesProjection())
			return ep.AddHandlersGroup("freight-requests",
				cqrs.NewGroupEventHandler(h.OnCreated),
				cqrs.NewGroupEventHandler(h.OnUpdated),
				cqrs.NewGroupEventHandler(h.OnReassigned),
				cqrs.NewGroupEventHandler(h.OnCancelled),
				cqrs.NewGroupEventHandler(h.OnExpired),
				cqrs.NewGroupEventHandler(h.OnOfferMade),
				cqrs.NewGroupEventHandler(h.OnOfferWithdrawn),
				cqrs.NewGroupEventHandler(h.OnOfferSelected),
				cqrs.NewGroupEventHandler(h.OnOfferRejected),
				cqrs.NewGroupEventHandler(h.OnOfferConfirmed),
				cqrs.NewGroupEventHandler(h.OnOfferDeclined),
				cqrs.NewGroupEventHandler(h.OnOfferUnselected),
				cqrs.NewGroupEventHandler(h.OnOfferCancelledWithRequest),
				cqrs.NewGroupEventHandler(h.OnCustomerCompleted),
				cqrs.NewGroupEventHandler(h.OnCarrierCompleted),
				cqrs.NewGroupEventHandler(h.OnFreightRequestCompleted),
				cqrs.NewGroupEventHandler(h.OnReviewLeft),
				cqrs.NewGroupEventHandler(h.OnCancelledAfterConfirmed),
				cqrs.NewGroupEventHandler(h.OnCarrierMemberReassigned),
				cqrs.NewGroupEventHandler(h.OnCarrierInvited),
			)
		},
	})
}
