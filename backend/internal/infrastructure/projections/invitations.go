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

type InvitationsProjection struct {
	db   dbtx.TxManager
	psql squirrel.StatementBuilderType
}

func NewInvitationsProjection(db dbtx.TxManager) *InvitationsProjection {
	return &InvitationsProjection{
		db:   db,
		psql: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

type InvitationLookup struct {
	ID             uuid.UUID `db:"id"`
	OrganizationID uuid.UUID `db:"organization_id"`
	Email          string    `db:"email"`
	Role           string    `db:"role"`
	Token          string    `db:"token"`
	Status         string    `db:"status"`
	CreatedBy      uuid.UUID `db:"created_by"`
	CreatedAt      time.Time `db:"created_at"`
	ExpiresAt      time.Time `db:"expires_at"`
}

// GetByToken retrieves invitation by token
func (p *InvitationsProjection) GetByToken(ctx context.Context, token string) (*InvitationLookup, error) {
	query, args, err := p.psql.
		Select("id", "organization_id", "email", "role", "token", "status", "created_by", "created_at", "expires_at").
		From("invitations_lookup").
		Where(squirrel.Eq{"token": token}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	var inv InvitationLookup
	if err := pgxscan.Get(ctx, p.db, &inv, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get invitation by token: %w", err)
	}

	return &inv, nil
}
