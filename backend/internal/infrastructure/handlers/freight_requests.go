package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"codeberg.org/udison/veziizi/backend/internal/domain/freightrequest/events"
	"codeberg.org/udison/veziizi/backend/internal/domain/freightrequest/values"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"codeberg.org/udison/veziizi/backend/internal/pkg/dbtx"
	"github.com/Masterminds/squirrel"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
)

type FreightRequestsHandler struct {
	db   dbtx.TxManager
	psql squirrel.StatementBuilderType
}

func NewFreightRequestsHandler(db dbtx.TxManager) *FreightRequestsHandler {
	return &FreightRequestsHandler{
		db:   db,
		psql: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (h *FreightRequestsHandler) Handle(msg *message.Message) error {
	var envelope eventstore.EventEnvelope
	if err := json.Unmarshal(msg.Payload, &envelope); err != nil {
		slog.Error("failed to unmarshal event envelope", slog.String("error", err.Error()))
		return fmt.Errorf("failed to unmarshal event envelope: %w", err)
	}

	evt, err := envelope.UnmarshalEvent()
	if err != nil {
		slog.Error("failed to unmarshal event", slog.String("error", err.Error()))
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	return h.handleEvent(msg.Context(), evt)
}

func (h *FreightRequestsHandler) handleEvent(ctx context.Context, evt eventstore.Event) error {
	switch e := evt.(type) {
	case events.FreightRequestCreated:
		return h.onCreated(ctx, e)
	case events.FreightRequestUpdated:
		return h.onUpdated(ctx, e)
	case events.FreightRequestReassigned:
		return h.onReassigned(ctx, e)
	case events.FreightRequestCancelled:
		return h.onCancelled(ctx, e)
	case events.FreightRequestExpired:
		return h.onExpired(ctx, e)
	case events.OfferMade:
		return h.onOfferMade(ctx, e)
	case events.OfferWithdrawn:
		return h.updateOfferStatus(ctx, e.OfferID, values.OfferStatusWithdrawn.String())
	case events.OfferSelected:
		return h.onOfferSelected(ctx, e)
	case events.OfferRejected:
		return h.updateOfferStatus(ctx, e.OfferID, values.OfferStatusRejected.String())
	case events.OfferConfirmed:
		return h.onOfferConfirmed(ctx, e)
	case events.OfferDeclined:
		return h.onOfferDeclined(ctx, e)
	}
	return nil
}

func (h *FreightRequestsHandler) onCreated(ctx context.Context, e events.FreightRequestCreated) error {
	expiresAt := time.Unix(e.ExpiresAt, 0)

	// Extract display data
	originAddr, destAddr := extractRouteAddresses(e.Route)
	bodyTypes := extractBodyTypes(e.VehicleRequirements.BodyTypes)

	var priceAmount *int64
	var priceCurrency *string
	if e.Payment.Price != nil {
		priceAmount = &e.Payment.Price.Amount
		curr := e.Payment.Price.Currency.String()
		priceCurrency = &curr
	}

	query, args, err := h.psql.
		Insert("freight_requests_lookup").
		Columns(
			"id", "customer_org_id", "status", "expires_at", "created_at",
			"origin_address", "destination_address", "cargo_type", "cargo_weight",
			"price_amount", "price_currency", "body_types",
		).
		Values(
			e.AggregateID(), e.CustomerOrgID, values.FreightRequestStatusPublished.String(), expiresAt, e.OccurredAt(),
			originAddr, destAddr, e.Cargo.Type.String(), e.Cargo.Weight,
			priceAmount, priceCurrency, bodyTypes,
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert query: %w", err)
	}

	if _, err := h.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert freight request: %w", err)
	}

	slog.Debug("freight request created", slog.String("id", e.AggregateID().String()))
	return nil
}

func extractRouteAddresses(route values.Route) (origin, destination string) {
	if len(route.Points) == 0 {
		return "", ""
	}
	origin = route.Points[0].Address
	destination = route.Points[len(route.Points)-1].Address
	return origin, destination
}

func extractBodyTypes(types []values.BodyType) []string {
	result := make([]string, len(types))
	for i, t := range types {
		result[i] = t.String()
	}
	return result
}

func (h *FreightRequestsHandler) onUpdated(ctx context.Context, e events.FreightRequestUpdated) error {
	// Update display columns if relevant data changed
	builder := h.psql.Update("freight_requests_lookup").Where(squirrel.Eq{"id": e.AggregateID()})
	hasUpdates := false

	if e.Route != nil {
		originAddr, destAddr := extractRouteAddresses(*e.Route)
		builder = builder.Set("origin_address", originAddr).Set("destination_address", destAddr)
		hasUpdates = true
	}

	if e.Cargo != nil {
		builder = builder.Set("cargo_type", e.Cargo.Type.String()).Set("cargo_weight", e.Cargo.Weight)
		hasUpdates = true
	}

	if e.VehicleRequirements != nil {
		bodyTypes := extractBodyTypes(e.VehicleRequirements.BodyTypes)
		builder = builder.Set("body_types", bodyTypes)
		hasUpdates = true
	}

	if e.Payment != nil && e.Payment.Price != nil {
		builder = builder.Set("price_amount", e.Payment.Price.Amount).Set("price_currency", e.Payment.Price.Currency.String())
		hasUpdates = true
	}

	if !hasUpdates {
		slog.Debug("freight request updated (no display columns changed)", slog.String("id", e.AggregateID().String()))
		return nil
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build update query: %w", err)
	}

	if _, err := h.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("update freight request display data: %w", err)
	}

	slog.Debug("freight request updated", slog.String("id", e.AggregateID().String()))
	return nil
}

func (h *FreightRequestsHandler) onReassigned(ctx context.Context, e events.FreightRequestReassigned) error {
	// No filter columns changed, full data loaded from event store
	slog.Debug("freight request reassigned", slog.String("id", e.AggregateID().String()))
	return nil
}

func (h *FreightRequestsHandler) onCancelled(ctx context.Context, e events.FreightRequestCancelled) error {
	return h.updateStatus(ctx, e.AggregateID(), values.FreightRequestStatusCancelled.String())
}

func (h *FreightRequestsHandler) onExpired(ctx context.Context, e events.FreightRequestExpired) error {
	return h.updateStatus(ctx, e.AggregateID(), values.FreightRequestStatusExpired.String())
}

func (h *FreightRequestsHandler) onOfferMade(ctx context.Context, e events.OfferMade) error {
	query, args, err := h.psql.
		Insert("offers_lookup").
		Columns("id", "freight_request_id", "carrier_org_id", "status", "created_at").
		Values(e.OfferID, e.AggregateID(), e.CarrierOrgID, values.OfferStatusPending.String(), e.OccurredAt()).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert query: %w", err)
	}

	if _, err := h.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert offer: %w", err)
	}

	slog.Debug("offer made", slog.String("offer_id", e.OfferID.String()))
	return nil
}

func (h *FreightRequestsHandler) onOfferSelected(ctx context.Context, e events.OfferSelected) error {
	// Update offer status
	if err := h.updateOfferStatus(ctx, e.OfferID, values.OfferStatusSelected.String()); err != nil {
		return err
	}

	// Update freight request status
	return h.updateStatus(ctx, e.AggregateID(), values.FreightRequestStatusSelected.String())
}

func (h *FreightRequestsHandler) onOfferConfirmed(ctx context.Context, e events.OfferConfirmed) error {
	// Update offer status
	if err := h.updateOfferStatus(ctx, e.OfferID, values.OfferStatusConfirmed.String()); err != nil {
		return err
	}

	// Update freight request status
	return h.updateStatus(ctx, e.AggregateID(), values.FreightRequestStatusConfirmed.String())
}

func (h *FreightRequestsHandler) onOfferDeclined(ctx context.Context, e events.OfferDeclined) error {
	// Update offer status
	if err := h.updateOfferStatus(ctx, e.OfferID, values.OfferStatusDeclined.String()); err != nil {
		return err
	}

	// Update freight request - back to published
	return h.updateStatus(ctx, e.AggregateID(), values.FreightRequestStatusPublished.String())
}

func (h *FreightRequestsHandler) updateStatus(ctx context.Context, id uuid.UUID, status string) error {
	query, args, err := h.psql.
		Update("freight_requests_lookup").
		Set("status", status).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update query: %w", err)
	}

	if _, err := h.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("update freight request status: %w", err)
	}

	return nil
}

func (h *FreightRequestsHandler) updateOfferStatus(ctx context.Context, id uuid.UUID, status string) error {
	query, args, err := h.psql.
		Update("offers_lookup").
		Set("status", status).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update query: %w", err)
	}

	if _, err := h.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("update offer status: %w", err)
	}

	return nil
}
