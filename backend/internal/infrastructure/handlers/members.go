package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"codeberg.org/udison/veziizi/backend/internal/domain/organization/events"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"codeberg.org/udison/veziizi/backend/internal/pkg/dbtx"
	"github.com/Masterminds/squirrel"
	"github.com/ThreeDotsLabs/watermill/message"
)

type MembersHandler struct {
	db   dbtx.TxManager
	psql squirrel.StatementBuilderType
}

func NewMembersHandler(db dbtx.TxManager) *MembersHandler {
	return &MembersHandler{
		db:   db,
		psql: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

// Handle обрабатывает watermill message. Возвращает nil для ack, error для nack.
func (h *MembersHandler) Handle(msg *message.Message) error {
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

func (h *MembersHandler) handleEvent(ctx context.Context, evt eventstore.Event) error {
	switch e := evt.(type) {
	case events.MemberAdded:
		return h.onMemberAdded(ctx, e)
	case events.MemberRoleChanged:
		return h.onMemberRoleChanged(ctx, e)
	case events.MemberBlocked:
		return h.onMemberBlocked(ctx, e)
	case events.MemberUnblocked:
		return h.onMemberUnblocked(ctx, e)
	}
	return nil
}

func (h *MembersHandler) onMemberAdded(ctx context.Context, e events.MemberAdded) error {
	query, args, err := h.psql.
		Insert("members_lookup").
		Columns("id", "organization_id", "email", "password_hash", "name", "phone", "role", "status", "created_at").
		Values(e.MemberID, e.AggregateID(), e.Email, e.PasswordHash, e.Name, e.Phone, e.Role.String(), "active", e.OccurredAt()).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build insert query: %w", err)
	}

	if _, err := h.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to insert member: %w", err)
	}

	slog.Debug("member added to lookup", slog.String("member_id", e.MemberID.String()))
	return nil
}

func (h *MembersHandler) onMemberRoleChanged(ctx context.Context, e events.MemberRoleChanged) error {
	query, args, err := h.psql.
		Update("members_lookup").
		Set("role", e.NewRole.String()).
		Where(squirrel.Eq{"id": e.MemberID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build update query: %w", err)
	}

	if _, err := h.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to update member role: %w", err)
	}

	slog.Debug("member role updated", slog.String("member_id", e.MemberID.String()))
	return nil
}

func (h *MembersHandler) onMemberBlocked(ctx context.Context, e events.MemberBlocked) error {
	query, args, err := h.psql.
		Update("members_lookup").
		Set("status", "blocked").
		Where(squirrel.Eq{"id": e.MemberID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build update query: %w", err)
	}

	if _, err := h.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to block member: %w", err)
	}

	slog.Debug("member blocked", slog.String("member_id", e.MemberID.String()))
	return nil
}

func (h *MembersHandler) onMemberUnblocked(ctx context.Context, e events.MemberUnblocked) error {
	query, args, err := h.psql.
		Update("members_lookup").
		Set("status", "active").
		Where(squirrel.Eq{"id": e.MemberID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build update query: %w", err)
	}

	if _, err := h.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to unblock member: %w", err)
	}

	slog.Debug("member unblocked", slog.String("member_id", e.MemberID.String()))
	return nil
}
