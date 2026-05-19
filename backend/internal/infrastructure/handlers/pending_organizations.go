package handlers

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/udisondev/veziizi/backend/internal/domain/organization/events"
	"github.com/udisondev/veziizi/backend/internal/pkg/dbtx"
)

type PendingOrganizationsHandler struct {
	db   dbtx.TxManager
	psql squirrel.StatementBuilderType
}

func NewPendingOrganizationsHandler(db dbtx.TxManager) *PendingOrganizationsHandler {
	return &PendingOrganizationsHandler{
		db:   db,
		psql: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

// OnApproved и OnRejected делают одно и то же — удаляют запись из pending.
// Разные точки входа нужны CQRS-фабрике: GroupEventHandler по типу события.
func (h *PendingOrganizationsHandler) OnApproved(ctx context.Context, e *events.OrganizationApproved) error {
	return h.remove(ctx, e.AggregateID())
}

func (h *PendingOrganizationsHandler) OnRejected(ctx context.Context, e *events.OrganizationRejected) error {
	return h.remove(ctx, e.AggregateID())
}

func (h *PendingOrganizationsHandler) OnCreated(ctx context.Context, e *events.OrganizationCreated) error {
	query, args, err := h.psql.
		Insert("pending_organizations").
		Columns("id", "name", "inn", "legal_name", "country", "email", "created_at").
		Values(e.AggregateID(), e.Name, e.INN, e.LegalName, e.Country.String(), e.Email, e.OccurredAt()).
		Suffix("ON CONFLICT (id) DO NOTHING").
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build insert query: %w", err)
	}

	if _, err := h.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to insert pending organization: %w", err)
	}

	slog.Debug("organization added to pending", slog.String("org_id", e.AggregateID().String()))
	return nil
}

func (h *PendingOrganizationsHandler) remove(ctx context.Context, orgID uuid.UUID) error {
	query, args, err := h.psql.
		Delete("pending_organizations").
		Where(squirrel.Eq{"id": orgID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build delete query: %w", err)
	}

	if _, err := h.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to delete pending organization: %w", err)
	}

	slog.Debug("organization removed from pending", slog.String("org_id", orgID.String()))
	return nil
}
