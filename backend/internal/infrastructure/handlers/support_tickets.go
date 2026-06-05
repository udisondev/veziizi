package handlers

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Masterminds/squirrel"

	"github.com/udisondev/veziizi/backend/internal/domain/support/entities"
	"github.com/udisondev/veziizi/backend/internal/domain/support/events"
	"github.com/udisondev/veziizi/backend/internal/domain/support/values"
	"github.com/udisondev/veziizi/backend/internal/pkg/dbtx"
)

// SupportTicketsHandler — projection: лента support_tickets_lookup.
// Используется воркером support-tickets-projection с собственной consumer group;
// admin-нотификации живут в отдельном воркере с другой group, чтобы их сбой
// не блокировал обновление проекции.
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

func (h *SupportTicketsHandler) OnTicketCreated(ctx context.Context, e *events.TicketCreated) error {
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

// Статусные апдейты идут через versionGuardedUpdate: устаревшее событие
// (out-of-order при N инстансах) не перетирает свежий статус, событие раньше
// Created уходит в retry до появления строки.

func (h *SupportTicketsHandler) OnMessageAdded(ctx context.Context, e *events.MessageAdded) error {
	var newStatus string
	if e.SenderType == entities.SenderTypeAdmin {
		newStatus = values.TicketStatusAnswered.String()
	} else {
		newStatus = values.TicketStatusAwaitingReply.String()
	}

	if err := versionGuardedUpdate(ctx, h.db, h.psql, "support_tickets_lookup", "version", e.AggregateID(), e.Version(), map[string]any{
		"status":     newStatus,
		"updated_at": e.OccurredAt(),
	}); err != nil {
		return fmt.Errorf("update ticket status: %w", err)
	}

	slog.Debug("support ticket message added",
		slog.String("ticket_id", e.AggregateID().String()),
		slog.String("sender_type", string(e.SenderType)))
	return nil
}

func (h *SupportTicketsHandler) OnTicketClosed(ctx context.Context, e *events.TicketClosed) error {
	if err := versionGuardedUpdate(ctx, h.db, h.psql, "support_tickets_lookup", "version", e.AggregateID(), e.Version(), map[string]any{
		"status":     values.TicketStatusClosed.String(),
		"updated_at": e.OccurredAt(),
		"closed_at":  e.OccurredAt(),
	}); err != nil {
		return fmt.Errorf("update ticket status: %w", err)
	}

	slog.Info("support ticket closed",
		slog.String("ticket_id", e.AggregateID().String()),
		slog.String("admin_id", e.ClosedByAdminID.String()))
	return nil
}

func (h *SupportTicketsHandler) OnTicketReopened(ctx context.Context, e *events.TicketReopened) error {
	if err := versionGuardedUpdate(ctx, h.db, h.psql, "support_tickets_lookup", "version", e.AggregateID(), e.Version(), map[string]any{
		"status":     values.TicketStatusAwaitingReply.String(),
		"updated_at": e.OccurredAt(),
		"closed_at":  nil,
	}); err != nil {
		return fmt.Errorf("update ticket status: %w", err)
	}

	slog.Info("support ticket reopened",
		slog.String("ticket_id", e.AggregateID().String()),
		slog.String("member_id", e.ReopenedByMemberID.String()))
	return nil
}
