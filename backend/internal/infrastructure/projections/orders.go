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
	OrderNumber      int64     `json:"order_number"`
	FreightRequestID uuid.UUID `json:"freight_request_id"`
	CustomerOrgID    uuid.UUID `json:"customer_org_id"`
	CarrierOrgID     uuid.UUID `json:"carrier_org_id"`
	CustomerMemberID uuid.UUID `json:"customer_member_id"`
	CarrierMemberID  uuid.UUID `json:"carrier_member_id"`
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

func OrderWithNumber(num int64) OrderFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.Eq{"order_number": num})
	}
}

func OrderWithCustomerMemberID(id uuid.UUID) OrderFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.Eq{"customer_member_id": id})
	}
}

func OrderWithCarrierMemberID(id uuid.UUID) OrderFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.Eq{"carrier_member_id": id})
	}
}

// OrderWithMemberID фильтрует заказы где пользователь = customer OR carrier
func OrderWithMemberID(id uuid.UUID) OrderFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.Or{
			squirrel.Eq{"customer_member_id": id},
			squirrel.Eq{"carrier_member_id": id},
		})
	}
}

// OrderWithOrgID фильтрует заказы где организация = customer OR carrier
func OrderWithOrgID(id uuid.UUID) OrderFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.Or{
			squirrel.Eq{"customer_org_id": id},
			squirrel.Eq{"carrier_org_id": id},
		})
	}
}

func (p *OrdersProjection) GetByID(ctx context.Context, id uuid.UUID) (*OrderListItem, error) {
	query, args, err := p.psql.
		Select("id", "order_number", "freight_request_id", "customer_org_id", "carrier_org_id", "customer_member_id", "carrier_member_id", "status", "created_at").
		From("orders_lookup").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select query: %w", err)
	}

	var item OrderListItem
	if err := p.db.QueryRow(ctx, query, args...).Scan(
		&item.ID,
		&item.OrderNumber,
		&item.FreightRequestID,
		&item.CustomerOrgID,
		&item.CarrierOrgID,
		&item.CustomerMemberID,
		&item.CarrierMemberID,
		&item.Status,
		&item.CreatedAt,
	); err != nil {
		return nil, fmt.Errorf("get order: %w", err)
	}

	return &item, nil
}

func (p *OrdersProjection) List(ctx context.Context, opts ...OrderFilterOption) ([]OrderListItem, error) {
	builder := p.psql.
		Select("id", "order_number", "freight_request_id", "customer_org_id", "carrier_org_id", "customer_member_id", "carrier_member_id", "status", "created_at").
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

	result := make([]OrderListItem, 0)
	for rows.Next() {
		var item OrderListItem
		if err := rows.Scan(
			&item.ID,
			&item.OrderNumber,
			&item.FreightRequestID,
			&item.CustomerOrgID,
			&item.CarrierOrgID,
			&item.CustomerMemberID,
			&item.CarrierMemberID,
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

// HaveSharedOrder проверяет есть ли общий заказ между двумя организациями
func (p *OrdersProjection) HaveSharedOrder(ctx context.Context, orgID1, orgID2 uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM orders_lookup
			WHERE (customer_org_id = $1 AND carrier_org_id = $2)
			   OR (customer_org_id = $2 AND carrier_org_id = $1)
		)
	`

	var exists bool
	if err := p.db.QueryRow(ctx, query, orgID1, orgID2).Scan(&exists); err != nil {
		return false, fmt.Errorf("check shared order: %w", err)
	}

	return exists, nil
}
