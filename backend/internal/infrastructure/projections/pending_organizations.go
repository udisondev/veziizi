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

type PendingOrganizationsProjection struct {
	db   dbtx.TxManager
	psql squirrel.StatementBuilderType
}

func NewPendingOrganizationsProjection(db dbtx.TxManager) *PendingOrganizationsProjection {
	return &PendingOrganizationsProjection{
		db:   db,
		psql: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

type PendingOrganization struct {
	ID        uuid.UUID `db:"id"`
	Name      string    `db:"name"`
	INN       string    `db:"inn"`
	LegalName string    `db:"legal_name"`
	Country   string    `db:"country"`
	Email     string    `db:"email"`
	CreatedAt time.Time `db:"created_at"`
}

// List returns all pending organizations ordered by creation date (newest first)
func (p *PendingOrganizationsProjection) List(ctx context.Context) ([]PendingOrganization, error) {
	query, args, err := p.psql.
		Select("id", "name", "inn", "legal_name", "country", "email", "created_at").
		From("pending_organizations").
		OrderBy("created_at DESC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	var orgs []PendingOrganization
	if err := pgxscan.Select(ctx, p.db, &orgs, query, args...); err != nil {
		return nil, fmt.Errorf("failed to list pending organizations: %w", err)
	}

	return orgs, nil
}
