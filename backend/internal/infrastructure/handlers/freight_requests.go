package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/udisondev/veziizi/backend/internal/domain/freightrequest/events"
	"github.com/udisondev/veziizi/backend/internal/domain/freightrequest/values"
	"github.com/udisondev/veziizi/backend/internal/domain/organization"
	orgEvents "github.com/udisondev/veziizi/backend/internal/domain/organization/events"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/projections"
	"github.com/udisondev/veziizi/backend/internal/pkg/dbtx"
)

type FreightRequestsHandler struct {
	db         dbtx.TxManager
	eventStore eventstore.Store
	invites    *projections.FreightInvitesProjection
	psql       squirrel.StatementBuilderType
}

func NewFreightRequestsHandler(db dbtx.TxManager, eventStore eventstore.Store, invites *projections.FreightInvitesProjection) *FreightRequestsHandler {
	return &FreightRequestsHandler{
		db:         db,
		eventStore: eventStore,
		invites:    invites,
		psql:       squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
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
		return h.OnCreated(ctx, &e)
	case events.FreightRequestUpdated:
		return h.OnUpdated(ctx, &e)
	case events.FreightRequestReassigned:
		return h.OnReassigned(ctx, &e)
	case events.FreightRequestCancelled:
		return h.OnCancelled(ctx, &e)
	case events.FreightRequestExpired:
		return h.OnExpired(ctx, &e)
	case events.OfferMade:
		return h.OnOfferMade(ctx, &e)
	case events.OfferWithdrawn:
		return h.OnOfferWithdrawn(ctx, &e)
	case events.OfferSelected:
		return h.OnOfferSelected(ctx, &e)
	case events.OfferRejected:
		return h.OnOfferRejected(ctx, &e)
	case events.OfferConfirmed:
		return h.OnOfferConfirmed(ctx, &e)
	case events.OfferDeclined:
		return h.OnOfferDeclined(ctx, &e)
	case events.OfferUnselected:
		return h.OnOfferUnselected(ctx, &e)
	case events.OfferCancelledWithRequest:
		return h.OnOfferCancelledWithRequest(ctx, &e)
	case events.CustomerCompleted:
		return h.OnCustomerCompleted(ctx, &e)
	case events.CarrierCompleted:
		return h.OnCarrierCompleted(ctx, &e)
	case events.FreightRequestCompleted:
		return h.OnFreightRequestCompleted(ctx, &e)
	case events.ReviewLeft:
		return h.OnReviewLeft(ctx, &e)
	case events.CancelledAfterConfirmed:
		return h.OnCancelledAfterConfirmed(ctx, &e)
	case events.CarrierMemberReassigned:
		return h.OnCarrierMemberReassigned(ctx, &e)
	case events.CarrierInvited:
		return h.OnCarrierInvited(ctx, &e)
	}
	return nil
}

// OnOfferWithdrawn / OnOfferRejected / OnOfferCancelledWithRequest — обёртки
// для CQRS-регистрации. Бизнес-логика — простой UPDATE статуса оффера.
func (h *FreightRequestsHandler) OnOfferWithdrawn(ctx context.Context, e *events.OfferWithdrawn) error {
	return h.updateOfferStatus(ctx, e.OfferID, values.OfferStatusWithdrawn.String())
}

func (h *FreightRequestsHandler) OnOfferRejected(ctx context.Context, e *events.OfferRejected) error {
	return h.updateOfferStatus(ctx, e.OfferID, values.OfferStatusRejected.String())
}

func (h *FreightRequestsHandler) OnOfferCancelledWithRequest(ctx context.Context, e *events.OfferCancelledWithRequest) error {
	return h.updateOfferStatus(ctx, e.OfferID, values.OfferStatusRejected.String())
}

// OnReviewLeft — no-op: создание Review aggregate делает review-receiver на том
// же топике с собственным consumer group. Тут только debug-лог для трейса.
func (h *FreightRequestsHandler) OnReviewLeft(_ context.Context, e *events.ReviewLeft) error {
	slog.Debug("review left",
		slog.String("id", e.AggregateID().String()),
		slog.String("review_id", e.ReviewID.String()),
	)
	return nil
}

func (h *FreightRequestsHandler) OnCarrierInvited(ctx context.Context, e *events.CarrierInvited) error {
	// Use the InviteID baked into the event so projection replays keep a
	// stable row identity (the service has already inserted with this ID inside
	// the event-store transaction; this insert is a no-op via ON CONFLICT, kept
	// for replay/backfill scenarios where the projection is reset).
	return h.invites.Insert(ctx, projections.FreightInviteLogItem{
		ID:               e.InviteID,
		FreightRequestID: e.AggregateID(),
		CarrierOrgID:     e.CarrierOrgID,
		VehicleID:        e.VehicleID,
		InvitedBy:        e.InvitedBy,
		InvitedAt:        e.OccurredAt(),
	})
}

func (h *FreightRequestsHandler) OnCreated(ctx context.Context, e *events.FreightRequestCreated) error {
	expiresAt := time.Unix(e.ExpiresAt, 0)

	// Extract display data
	originAddr, destAddr := extractRouteAddresses(e.Route)

	// Serialize route to JSON
	routeJSON, err := json.Marshal(e.Route)
	if err != nil {
		return fmt.Errorf("marshal route: %w", err)
	}

	var priceAmount *int64
	var priceCurrency *string
	if e.Payment.Price != nil {
		priceAmount = &e.Payment.Price.Amount
		curr := e.Payment.Price.Currency.String()
		priceCurrency = &curr
	}

	// Load organization data for denormalization
	var orgName, orgINN, orgCountry *string
	orgEvts, err := h.eventStore.Load(ctx, e.CustomerOrgID, orgEvents.AggregateType)
	if err != nil {
		return fmt.Errorf("load organization for denormalization: %w", err)
	}
	if len(orgEvts) > 0 {
		org := organization.NewFromEvents(e.CustomerOrgID, orgEvts)
		name := org.Name()
		inn := org.INN()
		country := org.Country().String()
		orgName = &name
		orgINN = &inn
		orgCountry = &country
	}

	// Extract city and country IDs from route for filtering
	routeCityIDs := extractRouteCityIDs(e.Route)
	routeCountryIDs := extractRouteCountryIDs(e.Route)

	// Extract payment info
	var paymentMethod, paymentTerms, vatType *string
	if e.Payment.Method != "" {
		pm := e.Payment.Method.String()
		paymentMethod = &pm
	}
	if e.Payment.Terms != "" {
		pt := e.Payment.Terms.String()
		paymentTerms = &pt
	}
	if e.Payment.VatType != "" {
		vt := e.Payment.VatType.String()
		vatType = &vt
	}

	query, args, err := h.psql.
		Insert("freight_requests_lookup").
		Columns(
			"id", "request_number", "customer_org_id", "status", "expires_at", "created_at",
			"origin_address", "destination_address", "route", "cargo_weight", "cargo_volume",
			"price_amount", "price_currency", "vehicle_type", "vehicle_subtype",
			"payment_method", "payment_terms", "vat_type",
			"customer_org_name", "customer_org_inn", "customer_org_country", "customer_member_id",
			"route_city_ids", "route_country_ids",
		).
		Values(
			e.AggregateID(), e.RequestNumber, e.CustomerOrgID, values.FreightRequestStatusPublished.String(), expiresAt, e.OccurredAt(),
			originAddr, destAddr, routeJSON, e.Cargo.Weight, e.Cargo.Volume,
			priceAmount, priceCurrency, e.VehicleRequirements.VehicleType.String(), e.VehicleRequirements.VehicleSubType.String(),
			paymentMethod, paymentTerms, vatType,
			orgName, orgINN, orgCountry, e.CustomerMemberID,
			routeCityIDs, routeCountryIDs,
		).
		Suffix("ON CONFLICT (id) DO NOTHING").
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert query: %w", err)
	}

	if _, err := h.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert freight request: %w", err)
	}

	slog.Debug("freight request created", slog.String("id", e.AggregateID().String()), slog.Int64("request_number", e.RequestNumber))
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

func extractRouteCityIDs(route values.Route) []int {
	var ids []int
	for _, p := range route.Points {
		if p.CityID != nil {
			ids = append(ids, *p.CityID)
		}
	}
	return ids
}

func extractRouteCountryIDs(route values.Route) []int {
	var ids []int
	for _, p := range route.Points {
		if p.CountryID != nil {
			ids = append(ids, *p.CountryID)
		}
	}
	return ids
}

func (h *FreightRequestsHandler) OnUpdated(ctx context.Context, e *events.FreightRequestUpdated) error {
	// Update display columns if relevant data changed
	builder := h.psql.Update("freight_requests_lookup").Where(squirrel.Eq{"id": e.AggregateID()})
	hasUpdates := false

	if e.Route != nil {
		originAddr, destAddr := extractRouteAddresses(*e.Route)
		routeJSON, err := json.Marshal(e.Route)
		if err != nil {
			return fmt.Errorf("marshal route: %w", err)
		}
		routeCityIDs := extractRouteCityIDs(*e.Route)
		routeCountryIDs := extractRouteCountryIDs(*e.Route)
		builder = builder.
			Set("origin_address", originAddr).
			Set("destination_address", destAddr).
			Set("route", routeJSON).
			Set("route_city_ids", routeCityIDs).
			Set("route_country_ids", routeCountryIDs)
		hasUpdates = true
	}

	if e.Cargo != nil {
		builder = builder.Set("cargo_weight", e.Cargo.Weight).Set("cargo_volume", e.Cargo.Volume)
		hasUpdates = true
	}

	if e.VehicleRequirements != nil {
		builder = builder.
			Set("vehicle_type", e.VehicleRequirements.VehicleType.String()).
			Set("vehicle_subtype", e.VehicleRequirements.VehicleSubType.String())
		hasUpdates = true
	}

	if e.Payment != nil {
		if e.Payment.Price != nil {
			builder = builder.Set("price_amount", e.Payment.Price.Amount).Set("price_currency", e.Payment.Price.Currency.String())
			hasUpdates = true
		}
		if e.Payment.Method != "" {
			builder = builder.Set("payment_method", e.Payment.Method.String())
			hasUpdates = true
		}
		if e.Payment.Terms != "" {
			builder = builder.Set("payment_terms", e.Payment.Terms.String())
			hasUpdates = true
		}
		if e.Payment.VatType != "" {
			builder = builder.Set("vat_type", e.Payment.VatType.String())
			hasUpdates = true
		}
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

func (h *FreightRequestsHandler) OnReassigned(ctx context.Context, e *events.FreightRequestReassigned) error {
	query, args, err := h.psql.
		Update("freight_requests_lookup").
		Set("customer_member_id", e.NewMemberID).
		Where(squirrel.Eq{"id": e.AggregateID()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build reassign query: %w", err)
	}

	if _, err := h.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("update customer_member_id: %w", err)
	}

	slog.Debug("freight request reassigned",
		slog.String("id", e.AggregateID().String()),
		slog.String("new_member_id", e.NewMemberID.String()))
	return nil
}

func (h *FreightRequestsHandler) OnCancelled(ctx context.Context, e *events.FreightRequestCancelled) error {
	return h.updateStatus(ctx, e.AggregateID(), values.FreightRequestStatusCancelled.String())
}

func (h *FreightRequestsHandler) OnExpired(ctx context.Context, e *events.FreightRequestExpired) error {
	return h.updateStatus(ctx, e.AggregateID(), values.FreightRequestStatusExpired.String())
}

func (h *FreightRequestsHandler) OnOfferMade(ctx context.Context, e *events.OfferMade) error {
	query, args, err := h.psql.
		Insert("offers_lookup").
		Columns("id", "freight_request_id", "carrier_org_id", "carrier_member_id", "status", "created_at").
		Values(e.OfferID, e.AggregateID(), e.CarrierOrgID, e.CarrierMemberID, values.OfferStatusPending.String(), e.OccurredAt()).
		Suffix("ON CONFLICT (id) DO NOTHING").
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

func (h *FreightRequestsHandler) OnOfferSelected(ctx context.Context, e *events.OfferSelected) error {
	// Update offer status
	if err := h.updateOfferStatus(ctx, e.OfferID, values.OfferStatusSelected.String()); err != nil {
		return err
	}

	// Update freight request status
	return h.updateStatus(ctx, e.AggregateID(), values.FreightRequestStatusSelected.String())
}

func (h *FreightRequestsHandler) OnOfferConfirmed(ctx context.Context, e *events.OfferConfirmed) error {
	// Update offer status
	if err := h.updateOfferStatus(ctx, e.OfferID, values.OfferStatusConfirmed.String()); err != nil {
		return err
	}

	// Get carrier info from offer
	var carrierOrgID uuid.UUID
	var carrierMemberID *uuid.UUID
	query, args, err := h.psql.
		Select("carrier_org_id", "carrier_member_id").
		From("offers_lookup").
		Where(squirrel.Eq{"id": e.OfferID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build select query: %w", err)
	}

	if err := h.db.QueryRow(ctx, query, args...).Scan(&carrierOrgID, &carrierMemberID); err != nil {
		return fmt.Errorf("get carrier info from offer: %w", err)
	}

	// Update freight request with carrier info and confirmed status
	confirmedAt := e.OccurredAt()
	query, args, err = h.psql.
		Update("freight_requests_lookup").
		Set("status", values.FreightRequestStatusConfirmed.String()).
		Set("carrier_org_id", carrierOrgID).
		Set("carrier_member_id", carrierMemberID).
		Set("confirmed_at", confirmedAt).
		Where(squirrel.Eq{"id": e.AggregateID()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update query: %w", err)
	}

	if _, err := h.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("update freight request with carrier info: %w", err)
	}

	slog.Debug("offer confirmed",
		slog.String("freight_request_id", e.AggregateID().String()),
		slog.String("offer_id", e.OfferID.String()),
		slog.String("carrier_org_id", carrierOrgID.String()))
	return nil
}

func (h *FreightRequestsHandler) OnOfferDeclined(ctx context.Context, e *events.OfferDeclined) error {
	// Update offer status
	if err := h.updateOfferStatus(ctx, e.OfferID, values.OfferStatusDeclined.String()); err != nil {
		return err
	}

	// Update freight request - back to published
	return h.updateStatus(ctx, e.AggregateID(), values.FreightRequestStatusPublished.String())
}

func (h *FreightRequestsHandler) OnOfferUnselected(ctx context.Context, e *events.OfferUnselected) error {
	// Update offer status - back to pending
	if err := h.updateOfferStatus(ctx, e.OfferID, values.OfferStatusPending.String()); err != nil {
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

func (h *FreightRequestsHandler) OnCustomerCompleted(ctx context.Context, e *events.CustomerCompleted) error {
	query, args, err := h.psql.
		Update("freight_requests_lookup").
		Set("customer_completed", true).
		Set("status", values.FreightRequestStatusPartiallyCompleted.String()).
		Where(squirrel.Eq{"id": e.AggregateID()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update query: %w", err)
	}

	if _, err := h.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("update customer completed: %w", err)
	}

	slog.Debug("customer completed freight request",
		slog.String("id", e.AggregateID().String()),
		slog.String("completed_by", e.CompletedBy.String()))
	return nil
}

func (h *FreightRequestsHandler) OnCarrierCompleted(ctx context.Context, e *events.CarrierCompleted) error {
	query, args, err := h.psql.
		Update("freight_requests_lookup").
		Set("carrier_completed", true).
		Set("status", values.FreightRequestStatusPartiallyCompleted.String()).
		Where(squirrel.Eq{"id": e.AggregateID()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update query: %w", err)
	}

	if _, err := h.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("update carrier completed: %w", err)
	}

	slog.Debug("carrier completed freight request",
		slog.String("id", e.AggregateID().String()),
		slog.String("completed_by", e.CompletedBy.String()))
	return nil
}

func (h *FreightRequestsHandler) OnFreightRequestCompleted(ctx context.Context, e *events.FreightRequestCompleted) error {
	completedAt := e.OccurredAt()

	query, args, err := h.psql.
		Update("freight_requests_lookup").
		Set("status", values.FreightRequestStatusCompleted.String()).
		Set("completed_at", completedAt).
		Where(squirrel.Eq{"id": e.AggregateID()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update query: %w", err)
	}

	if _, err := h.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("update freight request completed: %w", err)
	}

	slog.Debug("freight request fully completed",
		slog.String("id", e.AggregateID().String()),
		slog.Time("completed_at", completedAt))
	return nil
}

func (h *FreightRequestsHandler) OnCancelledAfterConfirmed(ctx context.Context, e *events.CancelledAfterConfirmed) error {
	cancelledAt := e.OccurredAt()

	query, args, err := h.psql.
		Update("freight_requests_lookup").
		Set("status", values.FreightRequestStatusCancelledAfterConfirmed.String()).
		Set("cancelled_after_confirmed_at", cancelledAt).
		Where(squirrel.Eq{"id": e.AggregateID()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update query: %w", err)
	}

	if _, err := h.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("update cancelled after confirmed: %w", err)
	}

	slog.Debug("freight request cancelled after confirmed",
		slog.String("id", e.AggregateID().String()),
		slog.String("cancelled_by", e.CancelledBy.String()))
	return nil
}

func (h *FreightRequestsHandler) OnCarrierMemberReassigned(ctx context.Context, e *events.CarrierMemberReassigned) error {
	query, args, err := h.psql.
		Update("freight_requests_lookup").
		Set("carrier_member_id", e.NewMemberID).
		Where(squirrel.Eq{"id": e.AggregateID()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build reassign carrier query: %w", err)
	}

	if _, err := h.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("update carrier_member_id: %w", err)
	}

	slog.Debug("carrier member reassigned",
		slog.String("id", e.AggregateID().String()),
		slog.String("new_member_id", e.NewMemberID.String()))
	return nil
}
