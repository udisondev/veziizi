package projections

import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/udisondev/veziizi/backend/internal/pkg/dbtx"
)

type FreightRequestViewsProjection struct {
	db   dbtx.TxManager
	psql squirrel.StatementBuilderType
}

func NewFreightRequestViewsProjection(db dbtx.TxManager) *FreightRequestViewsProjection {
	return &FreightRequestViewsProjection{
		db:   db,
		psql: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

// Touch records a freight request view for the given member. Idempotent: first
// view inserts a row, subsequent views bump last_viewed_at and view_count.
func (p *FreightRequestViewsProjection) Touch(ctx context.Context, memberID, freightRequestID uuid.UUID, at time.Time) error {
	query, args, err := p.psql.
		Insert("freight_request_views").
		Columns("member_id", "freight_request_id", "first_viewed_at", "last_viewed_at", "view_count").
		Values(memberID, freightRequestID, at, at, 1).
		Suffix(`ON CONFLICT (member_id, freight_request_id) DO UPDATE SET
            last_viewed_at = EXCLUDED.last_viewed_at,
            view_count = freight_request_views.view_count + 1`).
		ToSql()
	if err != nil {
		return fmt.Errorf("build touch view: %w", err)
	}
	if _, err := p.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("touch view: %w", err)
	}
	return nil
}

type ViewedFreightRequest struct {
	FreightRequestID uuid.UUID `db:"freight_request_id" json:"freight_request_id"`
	FirstViewedAt    time.Time `db:"first_viewed_at" json:"first_viewed_at"`
	LastViewedAt     time.Time `db:"last_viewed_at" json:"last_viewed_at"`
	ViewCount        int       `db:"view_count" json:"view_count"`
}

type ViewsCursor struct {
	LastViewedAt time.Time `json:"last_viewed_at"`
	FreightID    uuid.UUID `json:"freight_id"`
}

// ListViewedByMember returns freight requests viewed by the member, newest first.
// Pagination uses (last_viewed_at DESC, freight_request_id DESC) as keyset.
func (p *FreightRequestViewsProjection) ListViewedByMember(
	ctx context.Context,
	memberID uuid.UUID,
	cursor *ViewsCursor,
	limit int,
) ([]ViewedFreightRequest, error) {
	// Cap matches the handler's max user-facing limit (100) plus one extra
	// row used as a hasMore lookahead. Without the +1 slack a request for
	// ?limit=100 silently collapses to 50 and the cursor never advances.
	if limit <= 0 || limit > 101 {
		limit = 50
	}
	builder := p.psql.
		Select("freight_request_id", "first_viewed_at", "last_viewed_at", "view_count").
		From("freight_request_views").
		Where(squirrel.Eq{"member_id": memberID}).
		OrderBy("last_viewed_at DESC", "freight_request_id DESC").
		Limit(uint64(limit))

	if cursor != nil {
		builder = builder.Where(squirrel.Or{
			squirrel.Lt{"last_viewed_at": cursor.LastViewedAt},
			squirrel.And{
				squirrel.Eq{"last_viewed_at": cursor.LastViewedAt},
				squirrel.Lt{"freight_request_id": cursor.FreightID},
			},
		})
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build views list: %w", err)
	}
	rows, err := p.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query views: %w", err)
	}
	defer rows.Close()

	var result []ViewedFreightRequest
	for rows.Next() {
		var v ViewedFreightRequest
		if err := rows.Scan(&v.FreightRequestID, &v.FirstViewedAt, &v.LastViewedAt, &v.ViewCount); err != nil {
			return nil, fmt.Errorf("scan view: %w", err)
		}
		result = append(result, v)
	}
	return result, rows.Err()
}
