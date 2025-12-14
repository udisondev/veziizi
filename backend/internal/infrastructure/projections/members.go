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

func (p *MembersProjection) Handle(ctx context.Context, evt eventstore.Event) error {
	switch e := evt.(type) {
	case events.MemberAdded:
		return p.onMemberAdded(ctx, e)
	case events.MemberRoleChanged:
		return p.onMemberRoleChanged(ctx, e)
	case events.MemberBlocked:
		return p.onMemberBlocked(ctx, e)
	case events.MemberUnblocked:
		return p.onMemberUnblocked(ctx, e)
	}
	return nil
}

func (p *MembersProjection) onMemberAdded(ctx context.Context, e events.MemberAdded) error {
	query, args, err := p.psql.
		Insert("members_lookup").
		Columns("id", "organization_id", "email", "password_hash", "name", "phone", "role", "status", "created_at").
		Values(e.MemberID, e.AggregateID(), e.Email, e.PasswordHash, e.Name, e.Phone, e.Role.String(), "active", e.OccurredAt()).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build insert query: %w", err)
	}

	if _, err := p.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to insert member: %w", err)
	}

	slog.Debug("member added to projection", slog.String("member_id", e.MemberID.String()))
	return nil
}

func (p *MembersProjection) onMemberRoleChanged(ctx context.Context, e events.MemberRoleChanged) error {
	query, args, err := p.psql.
		Update("members_lookup").
		Set("role", e.NewRole.String()).
		Where(squirrel.Eq{"id": e.MemberID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build update query: %w", err)
	}

	if _, err := p.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to update member role: %w", err)
	}

	slog.Debug("member role updated", slog.String("member_id", e.MemberID.String()))
	return nil
}

func (p *MembersProjection) onMemberBlocked(ctx context.Context, e events.MemberBlocked) error {
	query, args, err := p.psql.
		Update("members_lookup").
		Set("status", "blocked").
		Where(squirrel.Eq{"id": e.MemberID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build update query: %w", err)
	}

	if _, err := p.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to block member: %w", err)
	}

	slog.Debug("member blocked", slog.String("member_id", e.MemberID.String()))
	return nil
}

func (p *MembersProjection) onMemberUnblocked(ctx context.Context, e events.MemberUnblocked) error {
	query, args, err := p.psql.
		Update("members_lookup").
		Set("status", "active").
		Where(squirrel.Eq{"id": e.MemberID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build update query: %w", err)
	}

	if _, err := p.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to unblock member: %w", err)
	}

	slog.Debug("member unblocked", slog.String("member_id", e.MemberID.String()))
	return nil
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
	row := p.db.QueryRow(ctx, query, args...)
	if err := row.Scan(&m.ID, &m.OrganizationID, &m.Email, &m.PasswordHash, &m.Name, &m.Phone, &m.Role, &m.Status, &m.CreatedAt); err != nil {
		return nil, fmt.Errorf("failed to scan member: %w", err)
	}

	return &m, nil
}

type MemberLookup struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Email          string
	PasswordHash   string
	Name           string
	Phone          *string
	Role           string
	Status         string
	CreatedAt      time.Time
}
