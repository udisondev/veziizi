package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/udisondev/veziizi/backend/internal/domain/organization/events"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"github.com/udisondev/veziizi/backend/internal/pkg/dbtx"
	"github.com/Masterminds/squirrel"
	"github.com/ThreeDotsLabs/watermill/message"
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

// Handle обрабатывает watermill message. Возвращает nil для ack, error для nack.
func (h *InvitationsHandler) Handle(msg *message.Message) error {
	var envelope eventstore.EventEnvelope
	if err := json.Unmarshal(msg.Payload, &envelope); err != nil {
		slog.Error("failed to unmarshal event envelope", slog.String("error", err.Error()))
		return fmt.Errorf("failed to unmarshal event envelope: %w", err)
	}

	evt, err := envelope.UnmarshalEvent()
	if err != nil {
		slog.Error("failed to unmarshal event", slog.String("error", err.Error()))
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	return h.handleEvent(msg.Context(), evt)
}

func (h *InvitationsHandler) handleEvent(ctx context.Context, evt eventstore.Event) error {
	switch e := evt.(type) {
	case events.InvitationCreated:
		return h.onInvitationCreated(ctx, e)
	case events.InvitationAccepted:
		return h.onInvitationAccepted(ctx, e)
	case events.InvitationExpired:
		return h.onInvitationExpired(ctx, e)
	case events.InvitationCancelled:
		return h.onInvitationCancelled(ctx, e)
	}
	return nil
}

func (h *InvitationsHandler) onInvitationCreated(ctx context.Context, e events.InvitationCreated) error {
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

func (h *InvitationsHandler) onInvitationAccepted(ctx context.Context, e events.InvitationAccepted) error {
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

func (h *InvitationsHandler) onInvitationExpired(ctx context.Context, e events.InvitationExpired) error {
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

func (h *InvitationsHandler) onInvitationCancelled(ctx context.Context, e events.InvitationCancelled) error {
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
