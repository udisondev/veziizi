package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	orderApp "codeberg.org/udison/veziizi/backend/internal/application/order"
	frEvents "codeberg.org/udison/veziizi/backend/internal/domain/freightrequest/events"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/projections"
	"github.com/ThreeDotsLabs/watermill/message"
)

type OrderCreatorHandler struct {
	orderService   *orderApp.Service
	frProjection   *projections.FreightRequestsProjection
}

func NewOrderCreatorHandler(
	orderService *orderApp.Service,
	frProjection *projections.FreightRequestsProjection,
) *OrderCreatorHandler {
	return &OrderCreatorHandler{
		orderService:   orderService,
		frProjection:   frProjection,
	}
}

func (h *OrderCreatorHandler) Handle(msg *message.Message) error {
	var envelope eventstore.EventEnvelope
	if err := json.Unmarshal(msg.Payload, &envelope); err != nil {
		slog.Error("failed to unmarshal event envelope", slog.String("error", err.Error()))
		return fmt.Errorf("unmarshal event envelope: %w", err)
	}

	evt, err := envelope.UnmarshalEvent()
	if err != nil {
		slog.Error("failed to unmarshal event", slog.String("error", err.Error()))
		return fmt.Errorf("unmarshal event: %w", err)
	}

	return h.handleEvent(msg.Context(), evt)
}

func (h *OrderCreatorHandler) handleEvent(ctx context.Context, evt eventstore.Event) error {
	switch e := evt.(type) {
	case frEvents.OfferConfirmed:
		return h.onOfferConfirmed(ctx, e)
	}
	return nil
}

func (h *OrderCreatorHandler) onOfferConfirmed(ctx context.Context, e frEvents.OfferConfirmed) error {
	orderID, err := h.orderService.CreateFromConfirmedOffer(ctx, orderApp.CreateFromOfferInput{
		FreightRequestID: e.AggregateID(),
		OfferID:          e.OfferID,
	})
	if err != nil {
		slog.Error("failed to create order from confirmed offer",
			slog.String("freight_request_id", e.AggregateID().String()),
			slog.String("offer_id", e.OfferID.String()),
			slog.String("error", err.Error()))
		return fmt.Errorf("create order from confirmed offer: %w", err)
	}

	// Update freight_requests_lookup with order_id
	if err := h.frProjection.UpdateOrderID(ctx, e.AggregateID(), orderID); err != nil {
		slog.Error("failed to update freight request order_id",
			slog.String("freight_request_id", e.AggregateID().String()),
			slog.String("order_id", orderID.String()),
			slog.String("error", err.Error()))
		return fmt.Errorf("update freight request order_id: %w", err)
	}

	slog.Info("order created from confirmed offer",
		slog.String("order_id", orderID.String()),
		slog.String("freight_request_id", e.AggregateID().String()),
		slog.String("offer_id", e.OfferID.String()))

	return nil
}
