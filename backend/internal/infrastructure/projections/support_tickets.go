package projections

import (
	"context"
	"fmt"
	"time"

	"github.com/udisondev/veziizi/backend/internal/pkg/dbtx"
	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
)

type SupportTicketsProjection struct {
	db   dbtx.TxManager
	psql squirrel.StatementBuilderType
}

func NewSupportTicketsProjection(db dbtx.TxManager) *SupportTicketsProjection {
	return &SupportTicketsProjection{
		db:   db,
		psql: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

// TicketListItem represents minimal ticket data for listing
type TicketListItem struct {
	ID           uuid.UUID  `json:"id"`
	TicketNumber int64      `json:"ticket_number"`
	MemberID     uuid.UUID  `json:"member_id"`
	OrgID        uuid.UUID  `json:"org_id"`
	Subject      string     `json:"subject"`
	Status       string     `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	ClosedAt     *time.Time `json:"closed_at,omitempty"`
}

type TicketFilterOption func(squirrel.SelectBuilder) squirrel.SelectBuilder

func TicketWithMemberID(id uuid.UUID) TicketFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.Eq{"member_id": id})
	}
}

func TicketWithOrgID(id uuid.UUID) TicketFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.Eq{"org_id": id})
	}
}

func TicketWithStatus(status string) TicketFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.Eq{"status": status})
	}
}

func TicketWithOpenStatus() TicketFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.NotEq{"status": "closed"})
	}
}

func TicketWithLimit(limit int) TicketFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Limit(uint64(limit))
	}
}

func TicketWithOffset(offset int) TicketFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Offset(uint64(offset))
	}
}

func TicketWithNumber(num int64) TicketFilterOption {
	return func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.Eq{"ticket_number": num})
	}
}

func (p *SupportTicketsProjection) GetByID(ctx context.Context, id uuid.UUID) (*TicketListItem, error) {
	query, args, err := p.psql.
		Select("id", "ticket_number", "member_id", "org_id", "subject", "status", "created_at", "updated_at", "closed_at").
		From("support_tickets_lookup").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select query: %w", err)
	}

	var item TicketListItem
	if err := p.db.QueryRow(ctx, query, args...).Scan(
		&item.ID,
		&item.TicketNumber,
		&item.MemberID,
		&item.OrgID,
		&item.Subject,
		&item.Status,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.ClosedAt,
	); err != nil {
		return nil, fmt.Errorf("get ticket: %w", err)
	}

	return &item, nil
}

func (p *SupportTicketsProjection) List(ctx context.Context, opts ...TicketFilterOption) ([]TicketListItem, error) {
	builder := p.psql.
		Select("id", "ticket_number", "member_id", "org_id", "subject", "status", "created_at", "updated_at", "closed_at").
		From("support_tickets_lookup").
		OrderBy("updated_at DESC")

	for _, opt := range opts {
		builder = opt(builder)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select query: %w", err)
	}

	rows, err := p.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query tickets: %w", err)
	}
	defer rows.Close()

	result := make([]TicketListItem, 0)
	for rows.Next() {
		var item TicketListItem
		if err := rows.Scan(
			&item.ID,
			&item.TicketNumber,
			&item.MemberID,
			&item.OrgID,
			&item.Subject,
			&item.Status,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.ClosedAt,
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

func (p *SupportTicketsProjection) Count(ctx context.Context, opts ...TicketFilterOption) (int, error) {
	builder := p.psql.
		Select("COUNT(*)").
		From("support_tickets_lookup")

	for _, opt := range opts {
		builder = opt(builder)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return 0, fmt.Errorf("build count query: %w", err)
	}

	var count int
	if err := p.db.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count tickets: %w", err)
	}

	return count, nil
}

// ListForAdmin returns all tickets with pagination for admin panel
func (p *SupportTicketsProjection) ListForAdmin(ctx context.Context, status string, limit, offset int) ([]TicketListItem, int, error) {
	opts := make([]TicketFilterOption, 0)
	if status != "" && status != "all" {
		opts = append(opts, TicketWithStatus(status))
	}

	// Get total count first
	total, err := p.Count(ctx, opts...)
	if err != nil {
		return nil, 0, err
	}

	// Add pagination
	if limit > 0 {
		opts = append(opts, TicketWithLimit(limit))
	}
	if offset > 0 {
		opts = append(opts, TicketWithOffset(offset))
	}

	items, err := p.List(ctx, opts...)
	if err != nil {
		return nil, 0, err
	}

	return items, total, nil
}
