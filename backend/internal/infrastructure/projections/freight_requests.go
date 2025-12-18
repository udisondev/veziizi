package projections

import (
	"context"
	"fmt"
	"time"

	"codeberg.org/udison/veziizi/backend/internal/pkg/dbtx"
	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
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
	ID                 uuid.UUID  `json:"id"`
	RequestNumber      int64      `json:"request_number"`
	CustomerOrgID      uuid.UUID  `json:"customer_org_id"`
	Status             string     `json:"status"`
	ExpiresAt          time.Time  `json:"expires_at"`
	CreatedAt          time.Time  `json:"created_at"`
	OriginAddress      *string    `json:"origin_address,omitempty"`
	DestinationAddress *string    `json:"destination_address,omitempty"`
	CargoType          *string    `json:"cargo_type,omitempty"`
	CargoWeight        *float64   `json:"cargo_weight,omitempty"`
	PriceAmount        *int64     `json:"price_amount,omitempty"`
	PriceCurrency      *string    `json:"price_currency,omitempty"`
	BodyTypes          []string   `json:"body_types,omitempty"`
	CustomerOrgName    *string    `json:"customer_org_name,omitempty"`
	CustomerOrgINN     *string    `json:"customer_org_inn,omitempty"`
	CustomerOrgCountry *string    `json:"customer_org_country,omitempty"`
	CustomerMemberID   *uuid.UUID `json:"customer_member_id,omitempty"`
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

func WithCustomerMemberID(id uuid.UUID) FilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.Eq{"customer_member_id": id})
	}
}

func WithOrgNameLike(name string) FilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.ILike{"customer_org_name": "%" + name + "%"})
	}
}

func WithOrgINN(inn string) FilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.ILike{"customer_org_inn": "%" + inn + "%"})
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

func (p *FreightRequestsProjection) GetByID(ctx context.Context, id uuid.UUID) (*FreightRequestListItem, error) {
	query, args, err := p.psql.
		Select(
			"id", "request_number", "customer_org_id", "status", "expires_at", "created_at",
			"origin_address", "destination_address", "cargo_type", "cargo_weight",
			"price_amount", "price_currency", "body_types",
			"customer_org_name", "customer_org_inn", "customer_org_country", "customer_member_id",
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
		&item.CargoType,
		&item.CargoWeight,
		&item.PriceAmount,
		&item.PriceCurrency,
		&item.BodyTypes,
		&item.CustomerOrgName,
		&item.CustomerOrgINN,
		&item.CustomerOrgCountry,
		&item.CustomerMemberID,
	); err != nil {
		return nil, fmt.Errorf("get freight request: %w", err)
	}

	return &item, nil
}

func (p *FreightRequestsProjection) List(ctx context.Context, opts ...FilterOption) ([]FreightRequestListItem, error) {
	builder := p.psql.
		Select(
			"id", "request_number", "customer_org_id", "status", "expires_at", "created_at",
			"origin_address", "destination_address", "cargo_type", "cargo_weight",
			"price_amount", "price_currency", "body_types",
			"customer_org_name", "customer_org_inn", "customer_org_country", "customer_member_id",
		).
		From("freight_requests_lookup").
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
			&item.CargoType,
			&item.CargoWeight,
			&item.PriceAmount,
			&item.PriceCurrency,
			&item.BodyTypes,
			&item.CustomerOrgName,
			&item.CustomerOrgINN,
			&item.CustomerOrgCountry,
			&item.CustomerMemberID,
		); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		result = append(result, item)
	}

	return result, nil
}

// Offer filter options

// OfferListItem represents minimal data for listing
// Full data is loaded from FreightRequest aggregate when needed
type OfferListItem struct {
	ID               uuid.UUID `json:"id"`
	FreightRequestID uuid.UUID `json:"freight_request_id"`
	CarrierOrgID     uuid.UUID `json:"carrier_org_id"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
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
		Select("id", "freight_request_id", "carrier_org_id", "status", "created_at").
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
		&item.Status,
		&item.CreatedAt,
	); err != nil {
		return nil, fmt.Errorf("get offer: %w", err)
	}

	return &item, nil
}

func (p *FreightRequestsProjection) ListOffers(ctx context.Context, opts ...OfferFilterOption) ([]OfferListItem, error) {
	builder := p.psql.
		Select("id", "freight_request_id", "carrier_org_id", "status", "created_at").
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
			&item.Status,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		result = append(result, item)
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

	return result, nil
}
