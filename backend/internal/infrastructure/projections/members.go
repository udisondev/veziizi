package projections

import (
	"context"
	"fmt"
	"time"

	"codeberg.org/udison/veziizi/backend/internal/pkg/dbtx"
	"github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/google/uuid"
)

type MembersProjection struct {
	db   dbtx.TxManager
	psql squirrel.StatementBuilderType
}

func NewMembersProjection(db dbtx.TxManager) *MembersProjection {
	return &MembersProjection{
		db:   db,
		psql: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

type MemberLookup struct {
	ID             uuid.UUID `db:"id"`
	OrganizationID uuid.UUID `db:"organization_id"`
	Email          string    `db:"email"`
	PasswordHash   string    `db:"password_hash"`
	Name           string    `db:"name"`
	Phone          *string   `db:"phone"`
	TelegramID     *int64    `db:"telegram_id"`
	Role           string    `db:"role"`
	Status         string    `db:"status"`
	CreatedAt      time.Time `db:"created_at"`
}

// GetByEmail retrieves member by email for authentication
func (p *MembersProjection) GetByEmail(ctx context.Context, email string) (*MemberLookup, error) {
	query, args, err := p.psql.
		Select("id", "organization_id", "email", "password_hash", "name", "phone", "telegram_id", "role", "status", "created_at").
		From("members_lookup").
		Where(squirrel.Eq{"email": email}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	var m MemberLookup
	if err := pgxscan.Get(ctx, p.db, &m, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get member by email: %w", err)
	}

	return &m, nil
}

// GetByID retrieves member by ID
func (p *MembersProjection) GetByID(ctx context.Context, id uuid.UUID) (*MemberLookup, error) {
	query, args, err := p.psql.
		Select("id", "organization_id", "email", "password_hash", "name", "phone", "telegram_id", "role", "status", "created_at").
		From("members_lookup").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	var m MemberLookup
	if err := pgxscan.Get(ctx, p.db, &m, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get member by id: %w", err)
	}

	return &m, nil
}

// DevMemberItem represents minimal member data for dev switcher (no password_hash)
type DevMemberItem struct {
	ID             uuid.UUID `db:"id" json:"id"`
	OrganizationID uuid.UUID `db:"organization_id" json:"organization_id"`
	Email          string    `db:"email" json:"email"`
	Name           string    `db:"name" json:"name"`
	Role           string    `db:"role" json:"role"`
	Status         string    `db:"status" json:"status"`
}

// ListAll returns all members for dev user switcher (dev mode only)
func (p *MembersProjection) ListAll(ctx context.Context, search string, limit int) ([]DevMemberItem, error) {
	builder := p.psql.
		Select("id", "organization_id", "email", "name", "role", "status").
		From("members_lookup").
		OrderBy("created_at DESC")

	if search != "" {
		builder = builder.Where(
			squirrel.Or{
				squirrel.ILike{"email": "%" + search + "%"},
				squirrel.ILike{"name": "%" + search + "%"},
			},
		)
	}

	if limit > 0 {
		builder = builder.Limit(uint64(limit))
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build list query: %w", err)
	}

	var members []DevMemberItem
	if err := pgxscan.Select(ctx, p.db, &members, query, args...); err != nil {
		return nil, fmt.Errorf("failed to list members: %w", err)
	}

	return members, nil
}

// GetNames возвращает имена членов по их ID
func (p *MembersProjection) GetNames(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error) {
	if len(ids) == 0 {
		return make(map[uuid.UUID]string), nil
	}

	query, args, err := p.psql.
		Select("id", "name").
		From("members_lookup").
		Where(squirrel.Eq{"id": ids}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	type idName struct {
		ID   uuid.UUID `db:"id"`
		Name string    `db:"name"`
	}

	var rows []idName
	if err := pgxscan.Select(ctx, p.db, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get member names: %w", err)
	}

	result := make(map[uuid.UUID]string, len(rows))
	for _, row := range rows {
		result[row.ID] = row.Name
	}

	return result, nil
}
