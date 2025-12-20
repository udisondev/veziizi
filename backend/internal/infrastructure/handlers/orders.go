package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"codeberg.org/udison/veziizi/backend/internal/domain/order/events"
	"codeberg.org/udison/veziizi/backend/internal/domain/order/values"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"codeberg.org/udison/veziizi/backend/internal/pkg/dbtx"
	"github.com/Masterminds/squirrel"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
)

type OrdersHandler struct {
	db   dbtx.TxManager
	psql squirrel.StatementBuilderType
}

func NewOrdersHandler(db dbtx.TxManager) *OrdersHandler {
	return &OrdersHandler{
		db:   db,
		psql: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (h *OrdersHandler) Handle(msg *message.Message) error {
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

func (h *OrdersHandler) handleEvent(ctx context.Context, evt eventstore.Event) error {
	switch e := evt.(type) {
	case events.OrderCreated:
		return h.onCreated(ctx, e)
	case events.OrderCancelled:
		return h.onCancelled(ctx, e)
	case events.CustomerCompleted:
		return h.onCustomerCompleted(ctx, e)
	case events.CarrierCompleted:
		return h.onCarrierCompleted(ctx, e)
	case events.OrderCompleted:
		return h.onCompleted(ctx, e)
	case events.MessageSent:
		// Messages are part of the aggregate, no separate lookup needed
		slog.Debug("message sent", slog.String("order_id", e.AggregateID().String()))
		return nil
	case events.DocumentAttached:
		slog.Debug("document attached", slog.String("order_id", e.AggregateID().String()))
		return nil
	case events.DocumentRemoved:
		slog.Debug("document removed", slog.String("order_id", e.AggregateID().String()))
		return nil
	case events.ReviewLeft:
		// ReviewLeft теперь обрабатывается в review-receiver worker
		slog.Debug("review left (handled by review-receiver)", slog.String("order_id", e.AggregateID().String()))
		return nil
	}
	return nil
}

func (h *OrdersHandler) onCreated(ctx context.Context, e events.OrderCreated) error {
	query, args, err := h.psql.
		Insert("orders_lookup").
		Columns("id", "order_number", "freight_request_id", "customer_org_id", "carrier_org_id", "customer_member_id", "carrier_member_id", "status", "created_at").
		Values(e.AggregateID(), e.OrderNumber, e.FreightRequestID, e.CustomerOrgID, e.CarrierOrgID, e.CustomerMemberID, e.CarrierMemberID, values.OrderStatusActive.String(), e.OccurredAt()).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert query: %w", err)
	}

	if _, err := h.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert order: %w", err)
	}

	slog.Debug("order created", slog.String("id", e.AggregateID().String()), slog.Int64("order_number", e.OrderNumber))
	return nil
}

func (h *OrdersHandler) onCancelled(ctx context.Context, e events.OrderCancelled) error {
	// Get customer_org_id to determine who cancelled
	customerOrgID, err := h.getCustomerOrgID(ctx, e.AggregateID())
	if err != nil {
		return err
	}

	var status string
	if customerOrgID == e.CancelledByOrgID {
		status = values.OrderStatusCancelledByCustomer.String()
	} else {
		status = values.OrderStatusCancelledByCarrier.String()
	}

	return h.updateStatus(ctx, e.AggregateID(), status)
}

func (h *OrdersHandler) onCustomerCompleted(ctx context.Context, e events.CustomerCompleted) error {
	return h.updateStatus(ctx, e.AggregateID(), values.OrderStatusCustomerCompleted.String())
}

func (h *OrdersHandler) onCarrierCompleted(ctx context.Context, e events.CarrierCompleted) error {
	return h.updateStatus(ctx, e.AggregateID(), values.OrderStatusCarrierCompleted.String())
}

func (h *OrdersHandler) onCompleted(ctx context.Context, e events.OrderCompleted) error {
	return h.updateStatus(ctx, e.AggregateID(), values.OrderStatusCompleted.String())
}

func (h *OrdersHandler) getCustomerOrgID(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
	query, args, err := h.psql.
		Select("customer_org_id").
		From("orders_lookup").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return uuid.Nil, fmt.Errorf("build select query: %w", err)
	}

	var customerOrgID uuid.UUID
	if err := h.db.QueryRow(ctx, query, args...).Scan(&customerOrgID); err != nil {
		return uuid.Nil, fmt.Errorf("get customer org id: %w", err)
	}

	return customerOrgID, nil
}

func (h *OrdersHandler) updateStatus(ctx context.Context, id uuid.UUID, status string) error {
	query, args, err := h.psql.
		Update("orders_lookup").
		Set("status", status).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update query: %w", err)
	}

	if _, err := h.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("update order status: %w", err)
	}

	return nil
}

