package projections

import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/google/uuid"
	"github.com/udisondev/veziizi/backend/internal/pkg/dbtx"
)

type FreightInvitesProjection struct {
	db   dbtx.TxManager
	psql squirrel.StatementBuilderType
}

func NewFreightInvitesProjection(db dbtx.TxManager) *FreightInvitesProjection {
	return &FreightInvitesProjection{
		db:   db,
		psql: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

type FreightInviteLogItem struct {
	ID               uuid.UUID `db:"id" json:"id"`
	FreightRequestID uuid.UUID `db:"freight_request_id" json:"freight_request_id"`
	CarrierOrgID     uuid.UUID `db:"carrier_org_id" json:"carrier_org_id"`
	VehicleID        uuid.UUID `db:"vehicle_id" json:"vehicle_id"`
	InvitedBy        uuid.UUID `db:"invited_by" json:"invited_by"`
	InvitedAt        time.Time `db:"invited_at" json:"invited_at"`
}

// Insert appends a new invite log entry. ON CONFLICT keeps the first record so
// the worker is idempotent: replays don't create duplicates and the read-side
// "already invited" check stays consistent with what the user sees.
func (p *FreightInvitesProjection) Insert(ctx context.Context, item FreightInviteLogItem) error {
	_, err := p.TryInsert(ctx, item)
	return err
}

// TryInsert performs the same idempotent insert as Insert but reports whether
// the row was actually created (true) or skipped due to the unique-constraint
// conflict on (freight_request_id, carrier_org_id) (false). The service uses
// this to atomically "claim" an invite slot inside the event-store transaction
// — the DB-level unique constraint serializes concurrent attempts and removes
// the read-then-write race the previous Exists() check had against the
// eventually-consistent projection.
func (p *FreightInvitesProjection) TryInsert(ctx context.Context, item FreightInviteLogItem) (bool, error) {
	query, args, err := p.psql.
		Insert("freight_request_invites_log").
		Columns("id", "freight_request_id", "carrier_org_id", "vehicle_id", "invited_by", "invited_at").
		Values(item.ID, item.FreightRequestID, item.CarrierOrgID, item.VehicleID, item.InvitedBy, item.InvitedAt).
		Suffix("ON CONFLICT (freight_request_id, carrier_org_id) DO NOTHING").
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build insert: %w", err)
	}
	tag, err := p.db.Exec(ctx, query, args...)
	if err != nil {
		return false, fmt.Errorf("insert invite: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

// LockFreight берёт транзакционный advisory-lock на freight_request_id, чтобы
// сериализовать параллельные попытки пригласить перевозчиков. Без него два
// конкурирующих InviteCarrier с *разными* carrier_org_id оба прочитают
// count<limit (уникальный индекс на (freight, carrier) ловит только дубль
// пары) и оба вставят строку, превысив maxInvitesPerFreight. Замок снимается
// при коммите/откате — обязан вызываться внутри InTx до CountByFreight.
func (p *FreightInvitesProjection) LockFreight(ctx context.Context, freightRequestID uuid.UUID) error {
	if _, err := p.db.Exec(ctx,
		"SELECT pg_advisory_xact_lock(hashtext('freight_invite_count'), hashtext($1::text))",
		freightRequestID.String()); err != nil {
		return fmt.Errorf("lock freight invites: %w", err)
	}
	return nil
}

func (p *FreightInvitesProjection) CountByFreight(ctx context.Context, freightRequestID uuid.UUID) (int, error) {
	query, args, err := p.psql.
		Select("COUNT(*)").
		From("freight_request_invites_log").
		Where(squirrel.Eq{"freight_request_id": freightRequestID}).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build count query: %w", err)
	}
	var count int
	if err := p.db.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count invites: %w", err)
	}
	return count, nil
}

// ListByFreight returns who the customer has already pinged on this freight request.
func (p *FreightInvitesProjection) ListByFreight(ctx context.Context, freightRequestID uuid.UUID) ([]FreightInviteLogItem, error) {
	query, args, err := p.psql.
		Select("id", "freight_request_id", "carrier_org_id", "vehicle_id", "invited_by", "invited_at").
		From("freight_request_invites_log").
		Where(squirrel.Eq{"freight_request_id": freightRequestID}).
		OrderBy("invited_at DESC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list query: %w", err)
	}
	var rows []FreightInviteLogItem
	if err := pgxscan.Select(ctx, p.db, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("list invites: %w", err)
	}
	return rows, nil
}
