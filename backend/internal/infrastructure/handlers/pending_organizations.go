package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/Masterminds/squirrel"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/udisondev/veziizi/backend/internal/domain/organization/events"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
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

// Handle обрабатывает watermill message. Возвращает nil для ack, error для nack.
func (h *PendingOrganizationsHandler) Handle(msg *message.Message) error {
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

func (h *PendingOrganizationsHandler) handleEvent(ctx context.Context, evt eventstore.Event) error {
	switch e := evt.(type) {
	case events.OrganizationCreated:
		return h.onCreated(ctx, e)
	case events.OrganizationApproved:
		return h.onRemoved(ctx, e.AggregateID())
	case events.OrganizationRejected:
		return h.onRemoved(ctx, e.AggregateID())
	}
	return nil
}

func (h *PendingOrganizationsHandler) onCreated(ctx context.Context, e events.OrganizationCreated) error {
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

func (h *PendingOrganizationsHandler) onRemoved(ctx context.Context, orgID uuid.UUID) error {
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
