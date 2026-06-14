package projections

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/udisondev/veziizi/backend/internal/pkg/dbtx"
)

type FreightRequestsProjection struct {
	db   dbtx.TxManager
	psql squirrel.StatementBuilderType
}

func NewFreightRequestsProjection(db dbtx.TxManager) *FreightRequestsProjection {
	return &FreightRequestsProjection{
		db:   db,
		psql: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

// FreightRequestListItem represents data for listing
// Includes display fields for UI, full data from event store when needed
type FreightRequestListItem struct {
	ID                 uuid.UUID       `json:"id"`
	RequestNumber      int64           `json:"request_number"`
	CustomerOrgID      uuid.UUID       `json:"customer_org_id"`
	Status             string          `json:"status"`
	ExpiresAt          time.Time       `json:"expires_at"`
	CreatedAt          time.Time       `json:"created_at"`
	OriginAddress      *string         `json:"origin_address,omitempty"`
	DestinationAddress *string         `json:"destination_address,omitempty"`
	Route              json.RawMessage `json:"route,omitempty"`
	CargoWeight        *float64        `json:"cargo_weight,omitempty"`
	PriceAmount        *int64          `json:"price_amount,omitempty"`
	PriceCurrency      *string         `json:"price_currency,omitempty"`
	VehicleType        *string         `json:"vehicle_type,omitempty"`
	VehicleSubType     *string         `json:"vehicle_subtype,omitempty"`
	CustomerOrgName    *string         `json:"customer_org_name,omitempty"`
	CustomerOrgINN     *string         `json:"customer_org_inn,omitempty"`
	CustomerOrgCountry *string         `json:"customer_org_country,omitempty"`
	CustomerMemberID   *uuid.UUID      `json:"customer_member_id,omitempty"`
	// Carrier fields (populated after offer confirmed)
	CarrierOrgID    *uuid.UUID `json:"carrier_org_id,omitempty"`
	CarrierMemberID *uuid.UUID `json:"carrier_member_id,omitempty"`
	ConfirmedAt     *time.Time `json:"confirmed_at,omitempty"`
}

type FilterOption func(squirrel.SelectBuilder) squirrel.SelectBuilder

func WithCustomerOrgID(id uuid.UUID) FilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.Eq{"customer_org_id": id})
	}
}

func WithFreightCarrierOrgID(id uuid.UUID) FilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.Eq{"carrier_org_id": id})
	}
}

func WithStatus(status string) FilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.Eq{"status": status})
	}
}

func WithStatuses(statuses []string) FilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		if len(statuses) == 0 {
			return b
		}
		return b.Where(squirrel.Eq{"status": statuses})
	}
}

// WithVisibleToOrg — видимостный invariant для списка заявок.
// Заявка видна организации, если она:
//   - её собственная (customer_org_id), в любом статусе;
//   - её собственная как перевозчика (carrier_org_id), в любом статусе;
//   - чужая, но опубликована (status='published').
//
// Применяется БЕЗУСЛОВНО для авторизованных запросов — чтобы приватные
// стадии чужих заявок (selected/confirmed/...) не утекали ни в одной ветке
// маршрутизации фильтров. Пользовательские фильтры customer_org_id/
// carrier_org_id/member_id/statuses работают поверх и могут только
// сужать выдачу.
func WithVisibleToOrg(orgID uuid.UUID) FilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.Or{
			squirrel.Eq{"customer_org_id": orgID},
			squirrel.Eq{"carrier_org_id": orgID},
			squirrel.Eq{"status": "published"},
		})
	}
}

func WithLimit(limit int) FilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Limit(uint64(limit))
	}
}

func WithOffset(offset int) FilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Offset(uint64(offset))
	}
}

// FreightRequestCursor для keyset pagination.
// Сортировка: request_number DESC (request_number — UNIQUE, обеспечивает стабильный порядок).
type FreightRequestCursor struct {
	RequestNumber int64 `json:"n"`
}

// WithCursor добавляет условие keyset pagination — записи "после" cursor.
func WithCursor(cursor FreightRequestCursor) FilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.Lt{"request_number": cursor.RequestNumber})
	}
}

func WithCustomerMemberID(id uuid.UUID) FilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.Eq{"customer_member_id": id})
	}
}

func WithOrgNameLike(name string) FilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		// SEC-014: Экранируем спецсимволы ILIKE
		return b.Where(squirrel.ILike{"customer_org_name": WrapLikePattern(name)})
	}
}

func WithOrgINN(inn string) FilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		// SEC-014: Экранируем спецсимволы ILIKE
		return b.Where(squirrel.ILike{"customer_org_inn": WrapLikePattern(inn)})
	}
}

func WithOrgCountry(country string) FilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.Eq{"customer_org_country": country})
	}
}

func WithRequestNumber(num int64) FilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.Eq{"request_number": num})
	}
}

// Extended filter options for subscription-like filtering

func WithMinWeight(weight float64) FilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.GtOrEq{"cargo_weight": weight})
	}
}

func WithMaxWeight(weight float64) FilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.LtOrEq{"cargo_weight": weight})
	}
}

func WithMinPrice(price int64) FilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.GtOrEq{"price_amount": price})
	}
}

func WithMaxPrice(price int64) FilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.LtOrEq{"price_amount": price})
	}
}

func WithVehicleTypes(types []string) FilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		if len(types) == 0 {
			return b
		}
		return b.Where(squirrel.Eq{"vehicle_type": types})
	}
}

func WithVehicleSubTypes(subtypes []string) FilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		if len(subtypes) == 0 {
			return b
		}
		return b.Where(squirrel.Eq{"vehicle_subtype": subtypes})
	}
}

func WithMinVolume(volume float64) FilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.GtOrEq{"cargo_volume": volume})
	}
}

func WithMaxVolume(volume float64) FilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.LtOrEq{"cargo_volume": volume})
	}
}

func WithPaymentMethods(methods []string) FilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		if len(methods) == 0 {
			return b
		}
		return b.Where(squirrel.Eq{"payment_method": methods})
	}
}

func WithPaymentTerms(terms []string) FilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		if len(terms) == 0 {
			return b
		}
		return b.Where(squirrel.Eq{"payment_terms": terms})
	}
}

func WithVatTypes(types []string) FilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		if len(types) == 0 {
			return b
		}
		return b.Where(squirrel.Eq{"vat_type": types})
	}
}

func WithHasPendingOffers() FilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where("EXISTS (SELECT 1 FROM offers_lookup o WHERE o.freight_request_id = freight_requests_lookup.id AND o.status = 'pending')")
	}
}

func WithNotExpired() FilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where("expires_at > NOW()")
	}
}

func WithRouteCities(cityIDs []int) FilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		if len(cityIDs) == 0 {
			return b
		}
		// pgx нативно маршалит []int → integer[]: ни fmt.Sprintf, ни ручной
		// joinInts здесь не нужны. Раньше строили "{1,2,3}" вручную — тот же
		// фрагильный паттерн, что фиксили в WithLoadingType.
		return b.Where("route_city_ids @> ?", cityIDs)
	}
}

func WithRouteCountries(countryIDs []int) FilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		if len(countryIDs) == 0 {
			return b
		}
		return b.Where("route_country_ids @> ?", countryIDs)
	}
}

func WithOriginCities(cityIDs []int) FilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		if len(cityIDs) == 0 {
			return b
		}
		return b.Where("origin_city_ids @> ?", cityIDs)
	}
}

func WithOriginCountries(countryIDs []int) FilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		if len(countryIDs) == 0 {
			return b
		}
		return b.Where("origin_country_ids @> ?", countryIDs)
	}
}

func WithDestinationCities(cityIDs []int) FilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		if len(cityIDs) == 0 {
			return b
		}
		return b.Where("destination_city_ids @> ?", cityIDs)
	}
}

func WithDestinationCountries(countryIDs []int) FilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		if len(countryIDs) == 0 {
			return b
		}
		return b.Where("destination_country_ids @> ?", countryIDs)
	}
}

func (p *FreightRequestsProjection) GetByID(ctx context.Context, id uuid.UUID) (*FreightRequestListItem, error) {
	query, args, err := p.psql.
		Select(
			"id", "request_number", "customer_org_id", "status", "expires_at", "created_at",
			"origin_address", "destination_address", "route", "cargo_weight",
			"price_amount", "price_currency", "vehicle_type", "vehicle_subtype",
			"customer_org_name", "customer_org_inn", "customer_org_country", "customer_member_id",
			"carrier_org_id", "carrier_member_id", "confirmed_at",
		).
		From("freight_requests_lookup").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select query: %w", err)
	}

	var item FreightRequestListItem
	if err := p.db.QueryRow(ctx, query, args...).Scan(
		&item.ID,
		&item.RequestNumber,
		&item.CustomerOrgID,
		&item.Status,
		&item.ExpiresAt,
		&item.CreatedAt,
		&item.OriginAddress,
		&item.DestinationAddress,
		&item.Route,
		&item.CargoWeight,
		&item.PriceAmount,
		&item.PriceCurrency,
		&item.VehicleType,
		&item.VehicleSubType,
		&item.CustomerOrgName,
		&item.CustomerOrgINN,
		&item.CustomerOrgCountry,
		&item.CustomerMemberID,
		&item.CarrierOrgID,
		&item.CarrierMemberID,
		&item.ConfirmedAt,
	); err != nil {
		return nil, fmt.Errorf("get freight request: %w", err)
	}

	return &item, nil
}

func (p *FreightRequestsProjection) List(ctx context.Context, opts ...FilterOption) ([]FreightRequestListItem, error) {
	builder := p.psql.
		Select(
			"id", "request_number", "customer_org_id", "status", "expires_at", "created_at",
			"origin_address", "destination_address", "route", "cargo_weight",
			"price_amount", "price_currency", "vehicle_type", "vehicle_subtype",
			"customer_org_name", "customer_org_inn", "customer_org_country", "customer_member_id",
			"carrier_org_id", "carrier_member_id", "confirmed_at",
		).
		From("freight_requests_lookup").
		OrderBy("request_number DESC")

	for _, opt := range opts {
		builder = opt(builder)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select query: %w", err)
	}

	rows, err := p.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query freight requests: %w", err)
	}
	defer rows.Close()

	var result []FreightRequestListItem
	for rows.Next() {
		var item FreightRequestListItem
		if err := rows.Scan(
			&item.ID,
			&item.RequestNumber,
			&item.CustomerOrgID,
			&item.Status,
			&item.ExpiresAt,
			&item.CreatedAt,
			&item.OriginAddress,
			&item.DestinationAddress,
			&item.Route,
			&item.CargoWeight,
			&item.PriceAmount,
			&item.PriceCurrency,
			&item.VehicleType,
			&item.VehicleSubType,
			&item.CustomerOrgName,
			&item.CustomerOrgINN,
			&item.CustomerOrgCountry,
			&item.CustomerMemberID,
			&item.CarrierOrgID,
			&item.CarrierMemberID,
			&item.ConfirmedAt,
		); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return result, nil
}

// Count возвращает количество заявок, удовлетворяющих фильтрам.
// Опции WithLimit / WithCursor / WithOffset не имеют смысла для COUNT —
// вызывающий код их не передаёт.
func (p *FreightRequestsProjection) Count(ctx context.Context, opts ...FilterOption) (int, error) {
	builder := p.psql.Select("COUNT(*)").From("freight_requests_lookup")

	for _, opt := range opts {
		builder = opt(builder)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return 0, fmt.Errorf("build count query: %w", err)
	}

	var count int
	if err := p.db.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("query freight requests count: %w", err)
	}
	return count, nil
}

// Offer filter options

// OfferListItem represents minimal data for listing
// Full data is loaded from FreightRequest aggregate when needed
type OfferListItem struct {
	ID               uuid.UUID  `json:"id"`
	FreightRequestID uuid.UUID  `json:"freight_request_id"`
	CarrierOrgID     uuid.UUID  `json:"carrier_org_id"`
	CarrierMemberID  *uuid.UUID `json:"carrier_member_id,omitempty"`
	Status           string     `json:"status"`
	CreatedAt        time.Time  `json:"created_at"`
}

type OfferFilterOption func(squirrel.SelectBuilder) squirrel.SelectBuilder

func WithFreightRequestID(id uuid.UUID) OfferFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.Eq{"freight_request_id": id})
	}
}

func WithCarrierOrgID(id uuid.UUID) OfferFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.Eq{"carrier_org_id": id})
	}
}

func WithOfferStatus(status string) OfferFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.Eq{"status": status})
	}
}

func WithOfferLimit(limit int) OfferFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Limit(uint64(limit))
	}
}

func WithOfferOffset(offset int) OfferFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Offset(uint64(offset))
	}
}

// Filter options with table alias for JOIN queries
func WithCarrierOrgIDAlias(id uuid.UUID) OfferFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.Eq{"o.carrier_org_id": id})
	}
}

func WithOfferStatusAlias(status string) OfferFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.Eq{"o.status": status})
	}
}

func WithCarrierMemberIDAlias(id uuid.UUID) OfferFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.Eq{"o.carrier_member_id": id})
	}
}

// Фильтры по данным заявки (используют псевдоним fr.*)

func WithOfferMinPriceAlias(price int64) OfferFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.GtOrEq{"fr.price_amount": price})
	}
}

func WithOfferMaxPriceAlias(price int64) OfferFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.LtOrEq{"fr.price_amount": price})
	}
}

func WithOfferPaymentMethodsAlias(methods []string) OfferFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		if len(methods) == 0 {
			return b
		}
		return b.Where(squirrel.Eq{"fr.payment_method": methods})
	}
}

func WithOfferVatTypesAlias(vatTypes []string) OfferFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		if len(vatTypes) == 0 {
			return b
		}
		return b.Where(squirrel.Eq{"fr.vat_type": vatTypes})
	}
}

// WithOutgoingOfferSort задаёт сортировку для ListOffersWithFreightData.
func WithOutgoingOfferSort(sortBy string) OfferFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		switch sortBy {
		case "price_desc":
			return b.OrderBy("fr.price_amount DESC NULLS LAST", "o.created_at DESC", "o.id DESC")
		case "price_asc":
			return b.OrderBy("fr.price_amount ASC NULLS FIRST", "o.created_at DESC", "o.id DESC")
		default:
			return b.OrderBy("o.created_at DESC", "o.id DESC")
		}
	}
}

// WithOfferStatusesAlias поддерживает plural-формат statuses (comma-separated → IN).
func WithOfferStatusesAlias(statuses []string) OfferFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		if len(statuses) == 0 {
			return b
		}
		return b.Where(squirrel.Eq{"o.status": statuses})
	}
}

func (p *FreightRequestsProjection) GetOfferByID(ctx context.Context, id uuid.UUID) (*OfferListItem, error) {
	query, args, err := p.psql.
		Select("id", "freight_request_id", "carrier_org_id", "carrier_member_id", "status", "created_at").
		From("offers_lookup").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select query: %w", err)
	}

	var item OfferListItem
	if err := p.db.QueryRow(ctx, query, args...).Scan(
		&item.ID,
		&item.FreightRequestID,
		&item.CarrierOrgID,
		&item.CarrierMemberID,
		&item.Status,
		&item.CreatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get offer: %w", err)
	}

	return &item, nil
}

func (p *FreightRequestsProjection) ListOffers(ctx context.Context, opts ...OfferFilterOption) ([]OfferListItem, error) {
	builder := p.psql.
		Select("id", "freight_request_id", "carrier_org_id", "carrier_member_id", "status", "created_at").
		From("offers_lookup").
		OrderBy("created_at DESC")

	for _, opt := range opts {
		builder = opt(builder)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select query: %w", err)
	}

	rows, err := p.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query offers: %w", err)
	}
	defer rows.Close()

	var result []OfferListItem
	for rows.Next() {
		var item OfferListItem
		if err := rows.Scan(
			&item.ID,
			&item.FreightRequestID,
			&item.CarrierOrgID,
			&item.CarrierMemberID,
			&item.Status,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return result, nil
}

// OfferWithFreightData represents offer with joined freight request data for "My Offers" page.
type OfferWithFreightData struct {
	ID                   uuid.UUID `json:"id"`
	FreightRequestID     uuid.UUID `json:"freight_request_id"`
	FreightRequestNumber *int64    `json:"freight_request_number,omitempty"`
	CarrierOrgID         uuid.UUID `json:"carrier_org_id"`
	CustomerOrgName      *string   `json:"customer_org_name,omitempty"`
	Status               string    `json:"status"`
	CreatedAt            time.Time `json:"created_at"`
	OriginAddress        *string   `json:"origin_address,omitempty"`
	DestinationAddress   *string   `json:"destination_address,omitempty"`
	CargoWeight          *float64  `json:"cargo_weight,omitempty"`
	PriceAmount          *int64    `json:"price_amount,omitempty"`
	PriceCurrency        *string   `json:"price_currency,omitempty"`
	PaymentMethod        *string   `json:"payment_method,omitempty"`
	VatType              *string   `json:"vat_type,omitempty"`
}

// OutgoingOfferCursor используется для keyset pagination исходящих офферов.
// SortBy должен совпадать с sort_by текущего запроса — курсор кодирует первичный ключ сортировки.
// ID — PK строки, стабилен между запросами, используется как финальный tie-breaker.
type OutgoingOfferCursor struct {
	SortBy      string    `json:"sort_by"`
	CreatedAt   time.Time `json:"created_at"`
	PriceAmount *int64    `json:"price_amount,omitempty"`
	ID          uuid.UUID `json:"id"`
}

// WithOutgoingOfferCursor строит keyset-условие в соответствии с первичным ключом сортировки.
// ID используется как финальный tie-breaker во всех ветках.
func WithOutgoingOfferCursor(cursor OutgoingOfferCursor) OfferFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		id := squirrel.Lt{"o.id": cursor.ID}
		atTie := squirrel.And{squirrel.Eq{"o.created_at": cursor.CreatedAt}, id}
		switch cursor.SortBy {
		case "price_desc":
			if cursor.PriceAmount != nil {
				priceTie := squirrel.And{squirrel.Eq{"fr.price_amount": *cursor.PriceAmount}, squirrel.Lt{"o.created_at": cursor.CreatedAt}}
				priceFull := squirrel.And{squirrel.Eq{"fr.price_amount": *cursor.PriceAmount}, squirrel.Eq{"o.created_at": cursor.CreatedAt}, id}
				return b.Where(squirrel.Or{
					squirrel.Lt{"fr.price_amount": *cursor.PriceAmount},
					priceTie,
					priceFull,
					squirrel.Expr("fr.price_amount IS NULL"),
				})
			}
			return b.Where(squirrel.And{squirrel.Expr("fr.price_amount IS NULL"),
				squirrel.Or{squirrel.Lt{"o.created_at": cursor.CreatedAt}, atTie}})
		case "price_asc":
			if cursor.PriceAmount != nil {
				priceTie := squirrel.And{squirrel.Eq{"fr.price_amount": *cursor.PriceAmount}, squirrel.Lt{"o.created_at": cursor.CreatedAt}}
				priceFull := squirrel.And{squirrel.Eq{"fr.price_amount": *cursor.PriceAmount}, squirrel.Eq{"o.created_at": cursor.CreatedAt}, id}
				return b.Where(squirrel.Or{
					squirrel.Gt{"fr.price_amount": *cursor.PriceAmount},
					priceTie,
					priceFull,
				})
			}
			return b.Where(squirrel.Or{
				squirrel.Expr("fr.price_amount IS NOT NULL"),
				squirrel.And{squirrel.Expr("fr.price_amount IS NULL"),
					squirrel.Or{squirrel.Lt{"o.created_at": cursor.CreatedAt}, atTie}},
			})
		default: // created_at_desc
			return b.Where(squirrel.Or{squirrel.Lt{"o.created_at": cursor.CreatedAt}, atTie})
		}
	}
}

func (p *FreightRequestsProjection) ListOffersWithFreightData(ctx context.Context, opts ...OfferFilterOption) ([]OfferWithFreightData, error) {
	builder := p.psql.
		Select(
			"o.id", "o.freight_request_id", "fr.request_number",
			"o.carrier_org_id", "fr.customer_org_name",
			"o.status", "o.created_at",
			"fr.origin_address", "fr.destination_address", "fr.cargo_weight",
			"fr.price_amount", "fr.price_currency",
			"fr.payment_method", "fr.vat_type",
		).
		From("offers_lookup o").
		LeftJoin("freight_requests_lookup fr ON fr.id = o.freight_request_id")
	// Сортировка задаётся через WithOutgoingOfferSort (по умолчанию created_at DESC).

	for _, opt := range opts {
		builder = opt(builder)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select query: %w", err)
	}

	rows, err := p.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query offers with freight data: %w", err)
	}
	defer rows.Close()

	var result []OfferWithFreightData
	for rows.Next() {
		var item OfferWithFreightData
		if err := rows.Scan(
			&item.ID, &item.FreightRequestID, &item.FreightRequestNumber,
			&item.CarrierOrgID, &item.CustomerOrgName,
			&item.Status, &item.CreatedAt,
			&item.OriginAddress, &item.DestinationAddress, &item.CargoWeight,
			&item.PriceAmount, &item.PriceCurrency,
			&item.PaymentMethod, &item.VatType,
		); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return result, nil
}

// UpdateCustomerOrgName обновляет имя организации-заказчика во всех заявках этой организации.
// Используется для поддержания денормализованных данных в актуальном состоянии.
// IS DISTINCT FROM не трогает строки, где имя уже актуально: OrganizationUpdated
// прилетает на любое изменение профиля, а не только имени — без guard'а каждый
// апдейт организации переписывал бы все её заявки впустую.
func (p *FreightRequestsProjection) UpdateCustomerOrgName(ctx context.Context, orgID uuid.UUID, name string) error {
	query, args, err := p.psql.
		Update("freight_requests_lookup").
		Set("customer_org_name", name).
		Where(squirrel.Eq{"customer_org_id": orgID}).
		Where(squirrel.Expr("customer_org_name IS DISTINCT FROM ?", name)).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update query: %w", err)
	}

	if _, err := p.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("update customer org name: %w", err)
	}

	return nil
}

// HaveSharedConfirmedFreight проверяет есть ли подтверждённая перевозка между двумя организациями
// (одна как заказчик, другая как перевозчик)
func (p *FreightRequestsProjection) HaveSharedConfirmedFreight(ctx context.Context, orgID1, orgID2 uuid.UUID) (bool, error) {
	// Проверяем есть ли freight request где одна организация - заказчик, другая - перевозчик
	// и статус confirmed или выше (partially_completed, completed)
	query, args, err := p.psql.
		Select("1").
		From("freight_requests_lookup").
		Where(squirrel.Or{
			squirrel.And{
				squirrel.Eq{"customer_org_id": orgID1},
				squirrel.Eq{"carrier_org_id": orgID2},
			},
			squirrel.And{
				squirrel.Eq{"customer_org_id": orgID2},
				squirrel.Eq{"carrier_org_id": orgID1},
			},
		}).
		Where(squirrel.Eq{"status": []string{"confirmed", "partially_completed", "completed"}}).
		Limit(1).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build query: %w", err)
	}

	var exists int
	err = p.db.QueryRow(ctx, query, args...).Scan(&exists)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check shared freight: %w", err)
	}

	return true, nil
}

// OrgStats — публичная агрегированная статистика организации.
// Симметрично с GetRating: эти счётчики и так показываются на публичном профиле.
type OrgStats struct {
	TotalFreightRequests  int `json:"total_freight_requests"`
	ActiveFreightRequests int `json:"active_freight_requests"`
	CompletedDeals        int `json:"completed_deals"`
	TotalOffersMade       int `json:"total_offers_made"`
	SuccessfulOffers      int `json:"successful_offers"`
}

// GetOrgStats возвращает публичную статистику организации.
func (p *FreightRequestsProjection) GetOrgStats(ctx context.Context, orgID uuid.UUID) (*OrgStats, error) {
	query := `
		SELECT
			(SELECT COUNT(*) FROM freight_requests_lookup WHERE customer_org_id = $1),
			(SELECT COUNT(*) FROM freight_requests_lookup WHERE customer_org_id = $1 AND status IN ('published', 'selected', 'confirmed')),
			(SELECT COUNT(*) FROM freight_requests_lookup WHERE (customer_org_id = $1 OR carrier_org_id = $1) AND status = 'completed'),
			(SELECT COUNT(*) FROM offers_lookup WHERE carrier_org_id = $1),
			(SELECT COUNT(*) FROM offers_lookup WHERE carrier_org_id = $1 AND status = 'confirmed')
	`

	var stats OrgStats
	if err := p.db.QueryRow(ctx, query, orgID).Scan(
		&stats.TotalFreightRequests,
		&stats.ActiveFreightRequests,
		&stats.CompletedDeals,
		&stats.TotalOffersMade,
		&stats.SuccessfulOffers,
	); err != nil {
		return nil, fmt.Errorf("get org stats: %w", err)
	}

	return &stats, nil
}

// DashboardStats — операционные показатели в моменте: пайплайн, входящие офферы, рынок.
// Доступны только членам организации (см. handler).
type DashboardStats struct {
	AsCustomerPublished         int `json:"as_customer_published"`
	AsCustomerSelected          int `json:"as_customer_selected"`
	AsCustomerConfirmed         int `json:"as_customer_confirmed"`
	AsCarrierConfirmed          int `json:"as_carrier_confirmed"`
	AsCarrierPartiallyCompleted int `json:"as_carrier_partially_completed"`
	PendingOffersCount          int `json:"pending_offers_count"`
	PendingOffersToday          int `json:"pending_offers_today"`
	MarketPublishedToday        int `json:"market_published_today"`
}

// GetDashboardStats возвращает моментальные операционные показатели организации.
func (p *FreightRequestsProjection) GetDashboardStats(ctx context.Context, orgID uuid.UUID) (*DashboardStats, error) {
	query := `
		SELECT
			(SELECT COUNT(*) FROM freight_requests_lookup WHERE customer_org_id = $1 AND status = 'published'),
			(SELECT COUNT(*) FROM freight_requests_lookup WHERE customer_org_id = $1 AND status = 'selected'),
			(SELECT COUNT(*) FROM freight_requests_lookup WHERE customer_org_id = $1 AND status = 'confirmed'),
			(SELECT COUNT(*) FROM freight_requests_lookup WHERE carrier_org_id = $1 AND status = 'confirmed'),
			(SELECT COUNT(*) FROM freight_requests_lookup WHERE carrier_org_id = $1 AND status = 'partially_completed'),
			(SELECT COUNT(DISTINCT fr.id) FROM offers_lookup o JOIN freight_requests_lookup fr ON fr.id = o.freight_request_id WHERE fr.customer_org_id = $1 AND o.status = 'pending'),
			(SELECT COUNT(DISTINCT fr.id) FROM offers_lookup o JOIN freight_requests_lookup fr ON fr.id = o.freight_request_id WHERE fr.customer_org_id = $1 AND o.status = 'pending' AND o.created_at >= NOW() - INTERVAL '24 hours'),
			(SELECT COUNT(*) FROM freight_requests_lookup WHERE status = 'published' AND customer_org_id != $1 AND created_at >= NOW() - INTERVAL '24 hours')
	`

	var stats DashboardStats
	if err := p.db.QueryRow(ctx, query, orgID).Scan(
		&stats.AsCustomerPublished,
		&stats.AsCustomerSelected,
		&stats.AsCustomerConfirmed,
		&stats.AsCarrierConfirmed,
		&stats.AsCarrierPartiallyCompleted,
		&stats.PendingOffersCount,
		&stats.PendingOffersToday,
		&stats.MarketPublishedToday,
	); err != nil {
		return nil, fmt.Errorf("get dashboard stats: %w", err)
	}

	return &stats, nil
}

// PendingOfferItem — заявка с pending-офферами (по одной строке на заявку, последний оффер)
type PendingOfferItem struct {
	ID                 uuid.UUID  `json:"id"`
	FreightRequestID   uuid.UUID  `json:"freight_request_id"`
	RequestNumber      int        `json:"request_number"`
	CarrierOrgID       uuid.UUID  `json:"carrier_org_id"`
	CarrierOrgName     string     `json:"carrier_org_name"`
	OriginAddress      string     `json:"origin_address"`
	DestinationAddress string     `json:"destination_address"`
	PriceAmount        *int64     `json:"price_amount,omitempty"`
	PriceCurrency      *string    `json:"price_currency,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	OffersCount        int        `json:"offers_count"`
}

// GetPendingOffersOnMyRequests возвращает по одной строке на заявку (последний оффер), с общим числом офферов.
// customerMemberID — опциональный фильтр по создателю заявки; нулевой UUID означает «без фильтра».
func (p *FreightRequestsProjection) GetPendingOffersOnMyRequests(ctx context.Context, customerOrgID uuid.UUID, customerMemberID uuid.UUID, limit int) ([]PendingOfferItem, error) {
	memberFilter := ""
	args := []any{customerOrgID, limit}
	if customerMemberID != uuid.Nil {
		memberFilter = " AND fr.customer_member_id = $3"
		args = []any{customerOrgID, limit, customerMemberID}
	}

	query := `
		SELECT
			sub.id, sub.freight_request_id, sub.request_number,
			sub.carrier_org_id, sub.carrier_org_name,
			sub.origin_address, sub.destination_address,
			sub.price_amount, sub.price_currency,
			sub.created_at,
			sub.offers_count
		FROM (
			SELECT DISTINCT ON (o.freight_request_id)
				o.id, o.freight_request_id, fr.request_number,
				o.carrier_org_id, COALESCE(org.name, '') AS carrier_org_name,
				fr.origin_address, fr.destination_address,
				fr.price_amount, fr.price_currency,
				o.created_at,
				COUNT(*) OVER (PARTITION BY o.freight_request_id) AS offers_count
			FROM offers_lookup o
			JOIN freight_requests_lookup fr ON fr.id = o.freight_request_id
			LEFT JOIN organizations_lookup org ON org.id = o.carrier_org_id
			WHERE fr.customer_org_id = $1 AND o.status = 'pending'` + memberFilter + `
			ORDER BY o.freight_request_id, o.created_at DESC
		) sub
		ORDER BY sub.created_at DESC
		LIMIT $2
	`

	rows, err := p.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query pending offers on my requests: %w", err)
	}
	defer rows.Close()

	var result []PendingOfferItem
	for rows.Next() {
		var item PendingOfferItem
		if err := rows.Scan(
			&item.ID,
			&item.FreightRequestID,
			&item.RequestNumber,
			&item.CarrierOrgID,
			&item.CarrierOrgName,
			&item.OriginAddress,
			&item.DestinationAddress,
			&item.PriceAmount,
			&item.PriceCurrency,
			&item.CreatedAt,
			&item.OffersCount,
		); err != nil {
			return nil, fmt.Errorf("scan pending offer: %w", err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return result, nil
}

// ─── Incoming offers (офферы на заявки текущей организации) ──────────────────

// IncomingOfferListItem — оффер, полученный текущей организацией как заказчиком.
type IncomingOfferListItem struct {
	ID                   uuid.UUID  `json:"id"`
	FreightRequestID     uuid.UUID  `json:"freight_request_id"`
	FreightRequestNumber *int64     `json:"freight_request_number,omitempty"`
	CarrierOrgID         uuid.UUID  `json:"carrier_org_id"`
	CarrierOrgName       *string    `json:"carrier_org_name,omitempty"`
	CarrierMemberID      *uuid.UUID `json:"carrier_member_id,omitempty"`
	CarrierMemberName    *string    `json:"carrier_member_name,omitempty"`
	Status               string     `json:"status"`
	CreatedAt            time.Time  `json:"created_at"`
	OriginAddress        *string    `json:"origin_address,omitempty"`
	DestinationAddress   *string    `json:"destination_address,omitempty"`
	PriceAmount          *int64     `json:"price_amount,omitempty"`
}

// IncomingOfferCursor используется для keyset pagination входящих офферов.
// SortBy должен совпадать с sort_by текущего запроса.
// ID — PK строки, стабилен между запросами, используется как финальный tie-breaker.
type IncomingOfferCursor struct {
	SortBy      string    `json:"sort_by"`
	CreatedAt   time.Time `json:"created_at"`
	PriceAmount *int64    `json:"price_amount,omitempty"`
	ID          uuid.UUID `json:"id"`
}

// IncomingOfferFilterOption — опция фильтрации входящих офферов.
type IncomingOfferFilterOption func(squirrel.SelectBuilder) squirrel.SelectBuilder

func WithIncomingStatus(status string) IncomingOfferFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.Eq{"o.status": status})
	}
}

func WithIncomingStatuses(statuses []string) IncomingOfferFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		if len(statuses) == 0 {
			return b
		}
		return b.Where(squirrel.Eq{"o.status": statuses})
	}
}

func WithIncomingCarrierMemberID(id uuid.UUID) IncomingOfferFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.Eq{"o.carrier_member_id": id})
	}
}

func WithIncomingCustomerMemberID(id uuid.UUID) IncomingOfferFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.Eq{"fr.customer_member_id": id})
	}
}

func WithIncomingCarrierOrgName(name string) IncomingOfferFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.ILike{"org.name": WrapLikePattern(name)})
	}
}

func WithIncomingFreightRequestNumber(n int64) IncomingOfferFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.Eq{"fr.request_number": n})
	}
}

// WithIncomingCursor строит keyset-условие с учётом первичного ключа сортировки.
// ID используется как финальный tie-breaker во всех ветках.
func WithIncomingCursor(cursor IncomingOfferCursor) IncomingOfferFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		id := squirrel.Lt{"o.id": cursor.ID}
		atTie := squirrel.And{squirrel.Eq{"o.created_at": cursor.CreatedAt}, id}
		switch cursor.SortBy {
		case "price_desc":
			if cursor.PriceAmount != nil {
				priceTie := squirrel.And{squirrel.Eq{"fr.price_amount": *cursor.PriceAmount}, squirrel.Lt{"o.created_at": cursor.CreatedAt}}
				priceFull := squirrel.And{squirrel.Eq{"fr.price_amount": *cursor.PriceAmount}, squirrel.Eq{"o.created_at": cursor.CreatedAt}, id}
				return b.Where(squirrel.Or{
					squirrel.Lt{"fr.price_amount": *cursor.PriceAmount},
					priceTie,
					priceFull,
					squirrel.Expr("fr.price_amount IS NULL"),
				})
			}
			return b.Where(squirrel.And{squirrel.Expr("fr.price_amount IS NULL"),
				squirrel.Or{squirrel.Lt{"o.created_at": cursor.CreatedAt}, atTie}})
		case "price_asc":
			if cursor.PriceAmount != nil {
				priceTie := squirrel.And{squirrel.Eq{"fr.price_amount": *cursor.PriceAmount}, squirrel.Lt{"o.created_at": cursor.CreatedAt}}
				priceFull := squirrel.And{squirrel.Eq{"fr.price_amount": *cursor.PriceAmount}, squirrel.Eq{"o.created_at": cursor.CreatedAt}, id}
				return b.Where(squirrel.Or{
					squirrel.Gt{"fr.price_amount": *cursor.PriceAmount},
					priceTie,
					priceFull,
				})
			}
			return b.Where(squirrel.Or{
				squirrel.Expr("fr.price_amount IS NOT NULL"),
				squirrel.And{squirrel.Expr("fr.price_amount IS NULL"),
					squirrel.Or{squirrel.Lt{"o.created_at": cursor.CreatedAt}, atTie}},
			})
		default: // created_at_desc
			return b.Where(squirrel.Or{squirrel.Lt{"o.created_at": cursor.CreatedAt}, atTie})
		}
	}
}

func WithIncomingMinPrice(price int64) IncomingOfferFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.GtOrEq{"fr.price_amount": price})
	}
}

func WithIncomingMaxPrice(price int64) IncomingOfferFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.LtOrEq{"fr.price_amount": price})
	}
}

func WithIncomingPaymentMethods(methods []string) IncomingOfferFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		if len(methods) == 0 {
			return b
		}
		return b.Where(squirrel.Eq{"fr.payment_method": methods})
	}
}

func WithIncomingVatTypes(vatTypes []string) IncomingOfferFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		if len(vatTypes) == 0 {
			return b
		}
		return b.Where(squirrel.Eq{"fr.vat_type": vatTypes})
	}
}

// WithIncomingSort задаёт сортировку для ListIncomingOffers.
func WithIncomingSort(sortBy string) IncomingOfferFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		switch sortBy {
		case "price_desc":
			return b.OrderBy("fr.price_amount DESC NULLS LAST", "o.created_at DESC", "o.id DESC")
		case "price_asc":
			return b.OrderBy("fr.price_amount ASC NULLS FIRST", "o.created_at DESC", "o.id DESC")
		default:
			return b.OrderBy("o.created_at DESC", "o.id DESC")
		}
	}
}

func WithIncomingLimit(limit int) IncomingOfferFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Limit(uint64(limit))
	}
}

// ListIncomingOffers возвращает офферы, поданные на заявки организации-заказчика.
func (p *FreightRequestsProjection) ListIncomingOffers(
	ctx context.Context,
	customerOrgID uuid.UUID,
	opts ...IncomingOfferFilterOption,
) ([]IncomingOfferListItem, error) {
	builder := p.psql.
		Select(
			"o.id", "o.freight_request_id", "fr.request_number",
			"o.carrier_org_id", "org.name AS carrier_org_name",
			"o.carrier_member_id", "m.name AS carrier_member_name",
			"o.status", "o.created_at",
			"fr.origin_address", "fr.destination_address",
			"fr.price_amount",
		).
		From("offers_lookup o").
		Join("freight_requests_lookup fr ON fr.id = o.freight_request_id").
		LeftJoin("organizations_lookup org ON org.id = o.carrier_org_id").
		LeftJoin("members_lookup m ON m.id = o.carrier_member_id").
		Where(squirrel.Eq{"fr.customer_org_id": customerOrgID})
	// Сортировка задаётся через WithIncomingSort (по умолчанию created_at DESC).

	for _, opt := range opts {
		builder = opt(builder)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build incoming offers query: %w", err)
	}

	rows, err := p.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query incoming offers: %w", err)
	}
	defer rows.Close()

	var result []IncomingOfferListItem
	for rows.Next() {
		var item IncomingOfferListItem
		if err := rows.Scan(
			&item.ID, &item.FreightRequestID, &item.FreightRequestNumber,
			&item.CarrierOrgID, &item.CarrierOrgName,
			&item.CarrierMemberID, &item.CarrierMemberName,
			&item.Status, &item.CreatedAt,
			&item.OriginAddress, &item.DestinationAddress,
			&item.PriceAmount,
		); err != nil {
			return nil, fmt.Errorf("scan incoming offer: %w", err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return result, nil
}
