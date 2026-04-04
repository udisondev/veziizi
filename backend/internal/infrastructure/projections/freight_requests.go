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
// Сортировка: (status = 'published') DESC, request_number DESC
type FreightRequestCursor struct {
	IsPublished   bool  `json:"p"` // status == 'published'
	RequestNumber int64 `json:"n"` // request_number
}

// WithCursor добавляет условие keyset pagination.
// Возвращает записи "после" cursor в порядке сортировки.
func WithCursor(cursor FreightRequestCursor) FilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		if cursor.IsPublished {
			// Cursor на published записи.
			// Следующие записи:
			//   1. published с меньшим request_number, ИЛИ
			//   2. не-published (любой request_number)
			return b.Where(
				squirrel.Or{
					squirrel.And{
						squirrel.Eq{"status": "published"},
						squirrel.Lt{"request_number": cursor.RequestNumber},
					},
					squirrel.NotEq{"status": "published"},
				},
			)
		}
		// Cursor на не-published записи.
		// Следующие записи: не-published с меньшим request_number.
		return b.Where(
			squirrel.And{
				squirrel.NotEq{"status": "published"},
				squirrel.Lt{"request_number": cursor.RequestNumber},
			},
		)
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

func WithVehicleType(vt string) FilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		if vt == "" {
			return b
		}
		return b.Where(squirrel.Eq{"vehicle_type": vt})
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

// joinStrings joins strings with comma for PostgreSQL array literal
func joinStrings(s []string) string {
	if len(s) == 0 {
		return ""
	}
	result := s[0]
	for i := 1; i < len(s); i++ {
		result += "," + s[i]
	}
	return result
}

// joinInts joins integers with comma for PostgreSQL array literal
func joinInts(nums []int) string {
	if len(nums) == 0 {
		return ""
	}
	result := fmt.Sprintf("%d", nums[0])
	for i := 1; i < len(nums); i++ {
		result += fmt.Sprintf(",%d", nums[i])
	}
	return result
}

func WithRouteCities(cityIDs []int) FilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		if len(cityIDs) == 0 {
			return b
		}
		// route_city_ids is an array, use @> operator to check that ALL specified cities are in route
		return b.Where("route_city_ids @> ?::integer[]", fmt.Sprintf("{%s}", joinInts(cityIDs)))
	}
}

func WithRouteCountries(countryIDs []int) FilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		if len(countryIDs) == 0 {
			return b
		}
		// route_country_ids is an array, use @> operator to check that ALL specified countries are in route
		return b.Where("route_country_ids @> ?::integer[]", fmt.Sprintf("{%s}", joinInts(countryIDs)))
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
		OrderBy("(status = 'published') DESC", "request_number DESC")

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

// OfferWithFreightData represents offer with joined freight request data for "My Offers" page
type OfferWithFreightData struct {
	ID                 uuid.UUID `json:"id"`
	FreightRequestID   uuid.UUID `json:"freight_request_id"`
	CarrierOrgID       uuid.UUID `json:"carrier_org_id"`
	Status             string    `json:"status"`
	CreatedAt          time.Time `json:"created_at"`
	OriginAddress      *string   `json:"origin_address,omitempty"`
	DestinationAddress *string   `json:"destination_address,omitempty"`
	CargoWeight        *float64  `json:"cargo_weight,omitempty"`
	PriceAmount        *int64    `json:"price_amount,omitempty"`
	PriceCurrency      *string   `json:"price_currency,omitempty"`
}

func (p *FreightRequestsProjection) ListOffersWithFreightData(ctx context.Context, opts ...OfferFilterOption) ([]OfferWithFreightData, error) {
	builder := p.psql.
		Select(
			"o.id", "o.freight_request_id", "o.carrier_org_id", "o.status", "o.created_at",
			"fr.origin_address", "fr.destination_address", "fr.cargo_weight",
			"fr.price_amount", "fr.price_currency",
		).
		From("offers_lookup o").
		LeftJoin("freight_requests_lookup fr ON fr.id = o.freight_request_id").
		OrderBy("o.created_at DESC")

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
			&item.ID,
			&item.FreightRequestID,
			&item.CarrierOrgID,
			&item.Status,
			&item.CreatedAt,
			&item.OriginAddress,
			&item.DestinationAddress,
			&item.CargoWeight,
			&item.PriceAmount,
			&item.PriceCurrency,
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
func (p *FreightRequestsProjection) UpdateCustomerOrgName(ctx context.Context, orgID uuid.UUID, name string) error {
	query, args, err := p.psql.
		Update("freight_requests_lookup").
		Set("customer_org_name", name).
		Where(squirrel.Eq{"customer_org_id": orgID}).
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
