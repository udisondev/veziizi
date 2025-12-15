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
	Role           string    `db:"role"`
	Status         string    `db:"status"`
	CreatedAt      time.Time `db:"created_at"`
}

// GetByEmail retrieves member by email for authentication
func (p *MembersProjection) GetByEmail(ctx context.Context, email string) (*MemberLookup, error) {
	query, args, err := p.psql.
		Select("id", "organization_id", "email", "password_hash", "name", "phone", "role", "status", "created_at").
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
