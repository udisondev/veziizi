package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"codeberg.org/udison/veziizi/backend/internal/domain/support/entities"
	"codeberg.org/udison/veziizi/backend/internal/domain/support/events"
	"codeberg.org/udison/veziizi/backend/internal/domain/support/values"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"codeberg.org/udison/veziizi/backend/internal/pkg/dbtx"
	"github.com/Masterminds/squirrel"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
)

type SupportTicketsHandler struct {
	db   dbtx.TxManager
	psql squirrel.StatementBuilderType
}

func NewSupportTicketsHandler(db dbtx.TxManager) *SupportTicketsHandler {
	return &SupportTicketsHandler{
		db:   db,
		psql: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (h *SupportTicketsHandler) Handle(msg *message.Message) error {
	var envelope eventstore.EventEnvelope
	if err := json.Unmarshal(msg.Payload, &envelope); err != nil {
		slog.Error("failed to unmarshal event envelope", slog.String("error", err.Error()))
		return fmt.Errorf("unmarshal event envelope: %w", err)
	}

	evt, err := envelope.UnmarshalEvent()
	if err != nil {
		slog.Error("failed to unmarshal event", slog.String("error", err.Error()))
		return fmt.Errorf("unmarshal event: %w", err)
	}

	return h.handleEvent(msg.Context(), evt)
}

func (h *SupportTicketsHandler) handleEvent(ctx context.Context, evt eventstore.Event) error {
	switch e := evt.(type) {
	case events.TicketCreated:
		return h.onCreated(ctx, e)
	case events.MessageAdded:
		return h.onMessageAdded(ctx, e)
	case events.TicketClosed:
		return h.onClosed(ctx, e)
	case events.TicketReopened:
		return h.onReopened(ctx, e)
	}
	return nil
}

func (h *SupportTicketsHandler) onCreated(ctx context.Context, e events.TicketCreated) error {
	query, args, err := h.psql.
		Insert("support_tickets_lookup").
		Columns("id", "ticket_number", "member_id", "org_id", "subject", "status", "created_at", "updated_at").
		Values(e.AggregateID(), e.TicketNumber, e.MemberID, e.OrgID, e.Subject, values.TicketStatusOpen.String(), e.OccurredAt(), e.OccurredAt()).
		Suffix("ON CONFLICT (id) DO NOTHING").
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert query: %w", err)
	}

	if _, err := h.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert ticket: %w", err)
	}

	slog.Info("support ticket created",
		slog.String("id", e.AggregateID().String()),
		slog.Int64("ticket_number", e.TicketNumber),
		slog.String("subject", e.Subject))
	return nil
}

func (h *SupportTicketsHandler) onMessageAdded(ctx context.Context, e events.MessageAdded) error {
	// Determine new status based on sender type
	var newStatus string
	if e.SenderType == entities.SenderTypeAdmin {
		newStatus = values.TicketStatusAnswered.String()
	} else {
		newStatus = values.TicketStatusAwaitingReply.String()
	}

	query, args, err := h.psql.
		Update("support_tickets_lookup").
		Set("status", newStatus).
		Set("updated_at", e.OccurredAt()).
		Where(squirrel.Eq{"id": e.AggregateID()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update query: %w", err)
	}

	if _, err := h.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("update ticket status: %w", err)
	}

	slog.Debug("support ticket message added",
		slog.String("ticket_id", e.AggregateID().String()),
		slog.String("sender_type", string(e.SenderType)))
	return nil
}

func (h *SupportTicketsHandler) onClosed(ctx context.Context, e events.TicketClosed) error {
	query, args, err := h.psql.
		Update("support_tickets_lookup").
		Set("status", values.TicketStatusClosed.String()).
		Set("updated_at", e.OccurredAt()).
		Set("closed_at", e.OccurredAt()).
		Where(squirrel.Eq{"id": e.AggregateID()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update query: %w", err)
	}

	if _, err := h.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("update ticket status: %w", err)
	}

	slog.Info("support ticket closed",
		slog.String("ticket_id", e.AggregateID().String()),
		slog.String("admin_id", e.ClosedByAdminID.String()))
	return nil
}

func (h *SupportTicketsHandler) onReopened(ctx context.Context, e events.TicketReopened) error {
	query, args, err := h.psql.
		Update("support_tickets_lookup").
		Set("status", values.TicketStatusAwaitingReply.String()).
		Set("updated_at", e.OccurredAt()).
		Set("closed_at", nil).
		Where(squirrel.Eq{"id": e.AggregateID()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update query: %w", err)
	}

	if _, err := h.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("update ticket status: %w", err)
	}

	slog.Info("support ticket reopened",
		slog.String("ticket_id", e.AggregateID().String()),
		slog.String("member_id", e.ReopenedByMemberID.String()))
	return nil
}

// Helper to get ticket by ID from lookup
func (h *SupportTicketsHandler) getTicket(ctx context.Context, id uuid.UUID) (memberID uuid.UUID, orgID uuid.UUID, err error) {
	query, args, err := h.psql.
		Select("member_id", "org_id").
		From("support_tickets_lookup").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("build select query: %w", err)
	}

	if err := h.db.QueryRow(ctx, query, args...).Scan(&memberID, &orgID); err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("get ticket: %w", err)
	}

	return memberID, orgID, nil
}
