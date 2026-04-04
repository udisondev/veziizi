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
	Name           *string   `db:"name"`  // предзаполненное ФИО
	Phone          *string   `db:"phone"` // предзаполненный телефон
}

// GetByToken retrieves invitation by token
func (p *InvitationsProjection) GetByToken(ctx context.Context, token string) (*InvitationLookup, error) {
	query, args, err := p.psql.
		Select("id", "organization_id", "email", "role", "token", "status", "created_by", "created_at", "expires_at", "name", "phone").
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

// ListByOrganization возвращает список приглашений организации с опциональной фильтрацией по статусу
func (p *InvitationsProjection) ListByOrganization(ctx context.Context, orgID uuid.UUID, status *string) ([]InvitationLookup, error) {
	builder := p.psql.
		Select("id", "organization_id", "email", "role", "token", "status", "created_by", "created_at", "expires_at", "name", "phone").
		From("invitations_lookup").
		Where(squirrel.Eq{"organization_id": orgID}).
		OrderBy("created_at DESC")

	if status != nil {
		builder = builder.Where(squirrel.Eq{"status": *status})
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	var invitations []InvitationLookup
	if err := pgxscan.Select(ctx, p.db, &invitations, query, args...); err != nil {
		return nil, fmt.Errorf("failed to list invitations: %w", err)
	}

	return invitations, nil
}
