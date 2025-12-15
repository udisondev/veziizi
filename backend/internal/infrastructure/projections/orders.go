package projections

import (
	"context"
	"fmt"
	"time"

	"codeberg.org/udison/veziizi/backend/internal/pkg/dbtx"
	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
)

type OrdersProjection struct {
	db   dbtx.TxManager
	psql squirrel.StatementBuilderType
}

func NewOrdersProjection(db dbtx.TxManager) *OrdersProjection {
	return &OrdersProjection{
		db:   db,
		psql: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

// OrderListItem represents minimal order data for listing
// Full order data is loaded from event store when needed
type OrderListItem struct {
	ID               uuid.UUID `json:"id"`
	FreightRequestID uuid.UUID `json:"freight_request_id"`
	CustomerOrgID    uuid.UUID `json:"customer_org_id"`
	CarrierOrgID     uuid.UUID `json:"carrier_org_id"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
}

type OrderFilterOption func(squirrel.SelectBuilder) squirrel.SelectBuilder

func OrderWithCustomerOrgID(id uuid.UUID) OrderFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.Eq{"customer_org_id": id})
	}
}

func OrderWithCarrierOrgID(id uuid.UUID) OrderFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.Eq{"carrier_org_id": id})
	}
}

func OrderWithFreightRequestID(id uuid.UUID) OrderFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.Eq{"freight_request_id": id})
	}
}

func OrderWithStatus(status string) OrderFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.Eq{"status": status})
	}
}

func OrderWithLimit(limit int) OrderFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Limit(uint64(limit))
	}
}

func OrderWithOffset(offset int) OrderFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Offset(uint64(offset))
	}
}

func (p *OrdersProjection) GetByID(ctx context.Context, id uuid.UUID) (*OrderListItem, error) {
	query, args, err := p.psql.
		Select("id", "freight_request_id", "customer_org_id", "carrier_org_id", "status", "created_at").
		From("orders_lookup").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select query: %w", err)
	}

	var item OrderListItem
	if err := p.db.QueryRow(ctx, query, args...).Scan(
		&item.ID,
		&item.FreightRequestID,
		&item.CustomerOrgID,
		&item.CarrierOrgID,
		&item.Status,
		&item.CreatedAt,
	); err != nil {
		return nil, fmt.Errorf("get order: %w", err)
	}

	return &item, nil
}

func (p *OrdersProjection) List(ctx context.Context, opts ...OrderFilterOption) ([]OrderListItem, error) {
	builder := p.psql.
		Select("id", "freight_request_id", "customer_org_id", "carrier_org_id", "status", "created_at").
		From("orders_lookup").
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
		return nil, fmt.Errorf("query orders: %w", err)
	}
	defer rows.Close()

	var result []OrderListItem
	for rows.Next() {
		var item OrderListItem
		if err := rows.Scan(
			&item.ID,
			&item.FreightRequestID,
			&item.CustomerOrgID,
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

func (p *OrdersProjection) Count(ctx context.Context, opts ...OrderFilterOption) (int, error) {
	builder := p.psql.
		Select("COUNT(*)").
		From("orders_lookup")

	for _, opt := range opts {
		builder = opt(builder)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return 0, fmt.Errorf("build count query: %w", err)
	}

	var count int
	if err := p.db.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count orders: %w", err)
	}

	return count, nil
}
