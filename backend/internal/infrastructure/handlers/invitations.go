package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/udisondev/veziizi/backend/internal/domain/organization/events"
	"github.com/udisondev/veziizi/backend/internal/pkg/dbtx"
)

type InvitationsHandler struct {
	db   dbtx.TxManager
	psql squirrel.StatementBuilderType
}

func NewInvitationsHandler(db dbtx.TxManager) *InvitationsHandler {
	return &InvitationsHandler{
		db:   db,
		psql: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (h *InvitationsHandler) OnInvitationCreated(ctx context.Context, e *events.InvitationCreated) error {
	expiresAt := time.Unix(e.ExpiresAt, 0)

	query, args, err := h.psql.
		Insert("invitations_lookup").
		Columns("id", "organization_id", "email", "role", "token", "status", "created_by", "created_at", "expires_at", "name", "phone").
		Values(e.InvitationID, e.AggregateID(), e.Email, e.Role.String(), e.Token, "pending", e.CreatedBy, e.OccurredAt(), expiresAt, e.Name, e.Phone).
		Suffix("ON CONFLICT (id) DO NOTHING").
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build insert query: %w", err)
	}

	if _, err := h.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to insert invitation: %w", err)
	}

	slog.Debug("invitation added to lookup", slog.String("invitation_id", e.InvitationID.String()))
	return nil
}

func (h *InvitationsHandler) OnInvitationAccepted(ctx context.Context, e *events.InvitationAccepted) error {
	query, args, err := h.psql.
		Update("invitations_lookup").
		Set("status", "accepted").
		Where(squirrel.Eq{"id": e.InvitationID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build update query: %w", err)
	}

	if _, err := h.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to update invitation: %w", err)
	}

	slog.Debug("invitation accepted", slog.String("invitation_id", e.InvitationID.String()))
	return nil
}

func (h *InvitationsHandler) OnInvitationExpired(ctx context.Context, e *events.InvitationExpired) error {
	query, args, err := h.psql.
		Update("invitations_lookup").
		Set("status", "expired").
		Where(squirrel.Eq{"id": e.InvitationID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build update query: %w", err)
	}

	if _, err := h.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to update invitation: %w", err)
	}

	slog.Debug("invitation expired", slog.String("invitation_id", e.InvitationID.String()))
	return nil
}

func (h *InvitationsHandler) OnInvitationCancelled(ctx context.Context, e *events.InvitationCancelled) error {
	query, args, err := h.psql.
		Update("invitations_lookup").
		Set("status", "cancelled").
		Where(squirrel.Eq{"id": e.InvitationID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build update query: %w", err)
	}

	if _, err := h.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to update invitation: %w", err)
	}

	slog.Debug("invitation cancelled", slog.String("invitation_id", e.InvitationID.String()))
	return nil
}
