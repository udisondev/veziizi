package projections

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"codeberg.org/udison/veziizi/backend/internal/domain/organization/events"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"codeberg.org/udison/veziizi/backend/internal/pkg/dbtx"
	"github.com/Masterminds/squirrel"
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

func (p *InvitationsProjection) Handle(ctx context.Context, evt eventstore.Event) error {
	switch e := evt.(type) {
	case events.InvitationCreated:
		return p.onInvitationCreated(ctx, e)
	case events.InvitationAccepted:
		return p.onInvitationAccepted(ctx, e)
	case events.InvitationExpired:
		return p.onInvitationExpired(ctx, e)
	}
	return nil
}

func (p *InvitationsProjection) onInvitationCreated(ctx context.Context, e events.InvitationCreated) error {
	expiresAt := time.Unix(e.ExpiresAt, 0)

	query, args, err := p.psql.
		Insert("invitations_lookup").
		Columns("id", "organization_id", "email", "role", "token", "status", "created_by", "created_at", "expires_at").
		Values(e.InvitationID, e.AggregateID(), e.Email, e.Role.String(), e.Token, "pending", e.CreatedBy, e.OccurredAt(), expiresAt).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build insert query: %w", err)
	}

	if _, err := p.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to insert invitation: %w", err)
	}

	slog.Debug("invitation added to projection", slog.String("invitation_id", e.InvitationID.String()))
	return nil
}

func (p *InvitationsProjection) onInvitationAccepted(ctx context.Context, e events.InvitationAccepted) error {
	query, args, err := p.psql.
		Update("invitations_lookup").
		Set("status", "accepted").
		Where(squirrel.Eq{"id": e.InvitationID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build update query: %w", err)
	}

	if _, err := p.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to update invitation: %w", err)
	}

	slog.Debug("invitation accepted", slog.String("invitation_id", e.InvitationID.String()))
	return nil
}

func (p *InvitationsProjection) onInvitationExpired(ctx context.Context, e events.InvitationExpired) error {
	query, args, err := p.psql.
		Update("invitations_lookup").
		Set("status", "expired").
		Where(squirrel.Eq{"id": e.InvitationID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build update query: %w", err)
	}

	if _, err := p.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to update invitation: %w", err)
	}

	slog.Debug("invitation expired", slog.String("invitation_id", e.InvitationID.String()))
	return nil
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
	row := p.db.QueryRow(ctx, query, args...)
	if err := row.Scan(&inv.ID, &inv.OrganizationID, &inv.Email, &inv.Role, &inv.Token, &inv.Status, &inv.CreatedBy, &inv.CreatedAt, &inv.ExpiresAt); err != nil {
		return nil, fmt.Errorf("failed to scan invitation: %w", err)
	}

	return &inv, nil
}

type InvitationLookup struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Email          string
	Role           string
	Token          string
	Status         string
	CreatedBy      uuid.UUID
	CreatedAt      time.Time
	ExpiresAt      time.Time
}
