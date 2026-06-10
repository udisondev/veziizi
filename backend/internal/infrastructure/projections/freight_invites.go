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

// CarrierInviteItem — элемент входящего инвайта для перевозчика с деталями заявки.
type CarrierInviteItem struct {
	ID                   uuid.UUID `db:"id"                    json:"id"`
	FreightRequestID     uuid.UUID `db:"freight_request_id"    json:"freight_request_id"`
	RequestNumber        int64     `db:"request_number"        json:"request_number"`
	OriginAddress        *string   `db:"origin_address"        json:"origin_address,omitempty"`
	DestinationAddress   *string   `db:"destination_address"   json:"destination_address,omitempty"`
	CustomerOrgName      *string   `db:"customer_org_name"     json:"customer_org_name,omitempty"`
	FreightStatus        string    `db:"freight_status"        json:"freight_status"`
	VehicleID            uuid.UUID `db:"vehicle_id"            json:"vehicle_id"`
	VehicleRegistration  *string   `db:"vehicle_registration"  json:"vehicle_registration,omitempty"`
	VehicleBrand         *string   `db:"vehicle_brand"         json:"vehicle_brand,omitempty"`
	VehicleModel         *string   `db:"vehicle_model"         json:"vehicle_model,omitempty"`
	InvitedBy            uuid.UUID `db:"invited_by"            json:"invited_by"`
	InvitedAt            time.Time `db:"invited_at"            json:"invited_at"`
}

// InviteCursor используется для keyset pagination по (invited_at DESC, id ASC).
type InviteCursor struct {
	InvitedAt time.Time `json:"invited_at"`
	ID        uuid.UUID `json:"id"`
}

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

// ListByCarrierOrg returns incoming invitations for a carrier organization,
// joined with freight request details, ordered by invited_at DESC.
func (p *FreightInvitesProjection) ListByCarrierOrg(ctx context.Context, carrierOrgID uuid.UUID, cursor *InviteCursor, limit int) ([]CarrierInviteItem, error) {
	builder := p.psql.
		Select(
			"il.id", "il.freight_request_id", "fr.request_number",
			"fr.origin_address", "fr.destination_address", "fr.customer_org_name",
			"fr.status AS freight_status",
			"il.vehicle_id",
			"v.registration_number AS vehicle_registration",
			"v.brand AS vehicle_brand",
			"v.model AS vehicle_model",
			"il.invited_by", "il.invited_at",
		).
		From("freight_request_invites_log il").
		Join("freight_requests_lookup fr ON fr.id = il.freight_request_id").
		LeftJoin("vehicles_lookup v ON v.id = il.vehicle_id").
		Where(squirrel.Eq{"il.carrier_org_id": carrierOrgID}).
		OrderBy("il.invited_at DESC", "il.id ASC").
		Limit(uint64(limit))

	if cursor != nil {
		builder = builder.Where(
			squirrel.Or{
				squirrel.Lt{"il.invited_at": cursor.InvitedAt},
				squirrel.And{
					squirrel.Eq{"il.invited_at": cursor.InvitedAt},
					squirrel.Gt{"il.id": cursor.ID},
				},
			},
		)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list by carrier org: %w", err)
	}
	var rows []CarrierInviteItem
	if err := pgxscan.Select(ctx, p.db, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("list carrier invitations: %w", err)
	}
	return rows, nil
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
