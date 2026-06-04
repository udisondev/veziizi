package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/udisondev/veziizi/backend/internal/domain/freightrequest"
	"github.com/udisondev/veziizi/backend/internal/domain/freightrequest/events"
	"github.com/udisondev/veziizi/backend/internal/domain/freightrequest/values"
	"github.com/udisondev/veziizi/backend/internal/domain/organization"
	orgEvents "github.com/udisondev/veziizi/backend/internal/domain/organization/events"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/projections"
	"github.com/udisondev/veziizi/backend/internal/pkg/dbtx"
)

// FreightRequestsHandler обновляет freight_requests_lookup и offers_lookup по
// паттерну rebuild-from-aggregate: на любое событие агрегата перечитывает его
// из event store и пишет ПОЛНОЕ состояние UPSERT'ом с version-guard'ом
// (см. versionGuardUpsertSuffix).
//
// Проекция = f(aggregate state). Это делает хендлер безопасным для
// at-least-once доставки и конкурентной обработки N инстансами:
//   - порядок событий не важен — event store всегда консистентен и упорядочен
//     по (aggregate_id, version), в отличие от самой проекции;
//   - повторная доставка идемпотентна — rebuild той же версии пишет те же данные;
//   - конкурентный устаревший rebuild отбрасывается version-guard'ом.
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

// freightRequestLookupColumns — все обновляемые колонки freight_requests_lookup
// (без id). Используются и в INSERT, и в ON CONFLICT DO UPDATE.
var freightRequestLookupColumns = []string{
	"request_number", "customer_org_id", "status", "expires_at", "created_at",
	"origin_address", "destination_address", "route", "cargo_weight", "cargo_volume",
	"price_amount", "price_currency", "vehicle_type", "vehicle_subtype",
	"payment_method", "payment_terms", "vat_type",
	"customer_org_name", "customer_org_inn", "customer_org_country", "customer_member_id",
	"route_city_ids", "route_country_ids",
	"origin_city_ids", "origin_country_ids",
	"destination_city_ids", "destination_country_ids",
	"carrier_org_id", "carrier_member_id", "confirmed_at",
	"customer_completed", "carrier_completed", "completed_at",
	"cancelled_after_confirmed_at",
	"version",
}

var offerLookupColumns = []string{
	"freight_request_id", "carrier_org_id", "carrier_member_id", "status", "created_at",
	"version",
}

// rebuild перечитывает агрегат из event store и переписывает строки проекции
// полным UPSERT'ом в одной tx. Все событийные хендлеры сводятся к нему.
func (h *FreightRequestsHandler) rebuild(ctx context.Context, id uuid.UUID) error {
	res, err := h.eventStore.LoadWithSnapshot(ctx, id, events.AggregateType)
	if err != nil {
		if errors.Is(err, eventstore.ErrAggregateNotFound) {
			// Событие опередило запись агрегата в стор — невозможно при текущей
			// схеме (outbox пишется в той же tx, что и события), но не падаем:
			// следующее событие агрегата достроит проекцию.
			slog.Warn("freight request not found in event store, skipping rebuild",
				slog.String("id", id.String()))
			return nil
		}
		return fmt.Errorf("load freight request: %w", err)
	}

	fr, err := freightrequest.NewFromStore(id, res.SnapshotState, res.Events)
	if err != nil {
		return fmt.Errorf("restore freight request: %w", err)
	}
	if fr.Version() == 0 {
		slog.Warn("freight request has no events, skipping rebuild", slog.String("id", id.String()))
		return nil
	}

	return h.db.InTx(ctx, func(ctx context.Context) error {
		if err := h.upsertLookup(ctx, fr); err != nil {
			return fmt.Errorf("upsert freight request lookup: %w", err)
		}
		if err := h.upsertOffers(ctx, fr); err != nil {
			return fmt.Errorf("upsert offers lookup: %w", err)
		}
		return nil
	})
}

func (h *FreightRequestsHandler) upsertLookup(ctx context.Context, fr *freightrequest.FreightRequest) error {
	route := fr.Route()
	originAddr, destAddr := extractRouteAddresses(route)

	routeJSON, err := json.Marshal(route)
	if err != nil {
		return fmt.Errorf("marshal route: %w", err)
	}

	payment := fr.Payment()
	var priceAmount *int64
	var priceCurrency *string
	if payment.Price != nil {
		priceAmount = &payment.Price.Amount
		curr := payment.Price.Currency.String()
		priceCurrency = &curr
	}
	var paymentMethod, paymentTerms, vatType *string
	if payment.Method != "" {
		pm := payment.Method.String()
		paymentMethod = &pm
	}
	if payment.Terms != "" {
		pt := payment.Terms.String()
		paymentTerms = &pt
	}
	if payment.VatType != "" {
		vt := payment.VatType.String()
		vatType = &vt
	}

	// Денормализация данных организации-заказчика — как и раньше, из event
	// store (организация может быть ещё не видна в organizations_lookup).
	var orgName, orgINN, orgCountry *string
	orgRes, err := h.eventStore.LoadWithSnapshot(ctx, fr.CustomerOrgID(), orgEvents.AggregateType)
	if err != nil && !errors.Is(err, eventstore.ErrAggregateNotFound) {
		return fmt.Errorf("load organization for denormalization: %w", err)
	}
	if orgRes != nil && (len(orgRes.Events) > 0 || orgRes.SnapshotState != nil) {
		org, err := organization.NewFromStore(fr.CustomerOrgID(), orgRes.SnapshotState, orgRes.Events)
		if err != nil {
			return fmt.Errorf("restore organization for denormalization: %w", err)
		}
		name := org.Name()
		inn := org.INN()
		country := org.Country().String()
		orgName = &name
		orgINN = &inn
		orgCountry = &country
	}

	cargo := fr.Cargo()
	vehicleReqs := fr.VehicleRequirements()

	query, args, err := h.psql.
		Insert("freight_requests_lookup").
		Columns(append([]string{"id"}, freightRequestLookupColumns...)...).
		Values(
			fr.ID(),
			fr.RequestNumber(), fr.CustomerOrgID(), fr.Status().String(), fr.ExpiresAt(), fr.CreatedAt(),
			originAddr, destAddr, routeJSON, cargo.Weight, cargo.Volume,
			priceAmount, priceCurrency, vehicleReqs.VehicleType.String(), vehicleReqs.VehicleSubType.String(),
			paymentMethod, paymentTerms, vatType,
			orgName, orgINN, orgCountry, fr.CustomerMemberID(),
			extractRouteCityIDs(route), extractRouteCountryIDs(route),
			extractEndpointCityIDs(route, endpointOrigin), extractEndpointCountryIDs(route, endpointOrigin),
			extractEndpointCityIDs(route, endpointDestination), extractEndpointCountryIDs(route, endpointDestination),
			fr.CarrierOrgID(), fr.CarrierMemberID(), fr.ConfirmedAt(),
			fr.CustomerCompleted(), fr.CarrierCompleted(), fr.CompletedAt(),
			fr.CancelledAfterConfirmedAt(),
			fr.Version(),
		).
		Suffix(projections.VersionGuardUpsertSuffix("freight_requests_lookup", freightRequestLookupColumns...)).
		ToSql()
	if err != nil {
		return fmt.Errorf("build upsert query: %w", err)
	}

	if _, err := h.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("exec upsert: %w", err)
	}
	return nil
}

func (h *FreightRequestsHandler) upsertOffers(ctx context.Context, fr *freightrequest.FreightRequest) error {
	offers := fr.OffersList()
	if len(offers) == 0 {
		return nil
	}

	builder := h.psql.
		Insert("offers_lookup").
		Columns(append([]string{"id"}, offerLookupColumns...)...)
	for _, o := range offers {
		var memberID *uuid.UUID
		if o.CarrierMemberID() != uuid.Nil {
			id := o.CarrierMemberID()
			memberID = &id
		}
		builder = builder.Values(
			o.ID(),
			fr.ID(), o.CarrierOrgID(), memberID, o.Status().String(), o.CreatedAt(),
			fr.Version(),
		)
	}

	query, args, err := builder.
		Suffix(projections.VersionGuardUpsertSuffix("offers_lookup", offerLookupColumns...)).
		ToSql()
	if err != nil {
		return fmt.Errorf("build upsert query: %w", err)
	}

	if _, err := h.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("exec upsert: %w", err)
	}
	return nil
}

// Все событийные хендлеры — обёртки над rebuild: конкретный тип события не
// важен, проекция всегда строится из полного состояния агрегата.

func (h *FreightRequestsHandler) OnCreated(ctx context.Context, e *events.FreightRequestCreated) error {
	return h.rebuild(ctx, e.AggregateID())
}

func (h *FreightRequestsHandler) OnUpdated(ctx context.Context, e *events.FreightRequestUpdated) error {
	return h.rebuild(ctx, e.AggregateID())
}

func (h *FreightRequestsHandler) OnReassigned(ctx context.Context, e *events.FreightRequestReassigned) error {
	return h.rebuild(ctx, e.AggregateID())
}

func (h *FreightRequestsHandler) OnCancelled(ctx context.Context, e *events.FreightRequestCancelled) error {
	return h.rebuild(ctx, e.AggregateID())
}

func (h *FreightRequestsHandler) OnExpired(ctx context.Context, e *events.FreightRequestExpired) error {
	return h.rebuild(ctx, e.AggregateID())
}

func (h *FreightRequestsHandler) OnOfferMade(ctx context.Context, e *events.OfferMade) error {
	return h.rebuild(ctx, e.AggregateID())
}

func (h *FreightRequestsHandler) OnOfferWithdrawn(ctx context.Context, e *events.OfferWithdrawn) error {
	return h.rebuild(ctx, e.AggregateID())
}

func (h *FreightRequestsHandler) OnOfferSelected(ctx context.Context, e *events.OfferSelected) error {
	return h.rebuild(ctx, e.AggregateID())
}

func (h *FreightRequestsHandler) OnOfferRejected(ctx context.Context, e *events.OfferRejected) error {
	return h.rebuild(ctx, e.AggregateID())
}

func (h *FreightRequestsHandler) OnOfferConfirmed(ctx context.Context, e *events.OfferConfirmed) error {
	return h.rebuild(ctx, e.AggregateID())
}

func (h *FreightRequestsHandler) OnOfferDeclined(ctx context.Context, e *events.OfferDeclined) error {
	return h.rebuild(ctx, e.AggregateID())
}

func (h *FreightRequestsHandler) OnOfferUnselected(ctx context.Context, e *events.OfferUnselected) error {
	return h.rebuild(ctx, e.AggregateID())
}

func (h *FreightRequestsHandler) OnOfferCancelledWithRequest(ctx context.Context, e *events.OfferCancelledWithRequest) error {
	return h.rebuild(ctx, e.AggregateID())
}

func (h *FreightRequestsHandler) OnCustomerCompleted(ctx context.Context, e *events.CustomerCompleted) error {
	return h.rebuild(ctx, e.AggregateID())
}

func (h *FreightRequestsHandler) OnCarrierCompleted(ctx context.Context, e *events.CarrierCompleted) error {
	return h.rebuild(ctx, e.AggregateID())
}

func (h *FreightRequestsHandler) OnFreightRequestCompleted(ctx context.Context, e *events.FreightRequestCompleted) error {
	return h.rebuild(ctx, e.AggregateID())
}

func (h *FreightRequestsHandler) OnCancelledAfterConfirmed(ctx context.Context, e *events.CancelledAfterConfirmed) error {
	return h.rebuild(ctx, e.AggregateID())
}

func (h *FreightRequestsHandler) OnCarrierMemberReassigned(ctx context.Context, e *events.CarrierMemberReassigned) error {
	return h.rebuild(ctx, e.AggregateID())
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

// endpointKind — какую точку маршрута денормализовать в origin_*/destination_*.
type endpointKind int

const (
	endpointOrigin endpointKind = iota
	endpointDestination
)

// endpointPoint возвращает первую (origin) или последнюю (destination) точку
// маршрута. nil — если маршрут пуст. Та же позиционная семантика, что и в
// extractRouteAddresses.
func endpointPoint(route values.Route, kind endpointKind) *values.RoutePoint {
	if len(route.Points) == 0 {
		return nil
	}
	switch kind {
	case endpointOrigin:
		return &route.Points[0]
	case endpointDestination:
		return &route.Points[len(route.Points)-1]
	}
	return nil
}

// extractEndpointCityIDs / extractEndpointCountryIDs возвращают одноэлементный
// слайс с city_id/country_id выбранной точки маршрута, либо пустой, если у
// точки нет id. Та же форма данных, что у route_city_ids/route_country_ids —
// фильтр `origin_city_ids @> ?` использует GIN-индекс по массиву.
func extractEndpointCityIDs(route values.Route, kind endpointKind) []int {
	p := endpointPoint(route, kind)
	if p == nil || p.CityID == nil {
		return nil
	}
	return []int{*p.CityID}
}

func extractEndpointCountryIDs(route values.Route, kind endpointKind) []int {
	p := endpointPoint(route, kind)
	if p == nil || p.CountryID == nil {
		return nil
	}
	return []int{*p.CountryID}
}
