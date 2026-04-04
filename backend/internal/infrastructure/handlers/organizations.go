package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/udisondev/veziizi/backend/internal/domain/organization/events"
	"github.com/udisondev/veziizi/backend/internal/domain/organization/values"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/projections"
)

type OrganizationsHandler struct {
	projection                *projections.OrganizationsProjection
	freightRequestsProjection *projections.FreightRequestsProjection
}

func NewOrganizationsHandler(
	projection *projections.OrganizationsProjection,
	freightRequestsProjection *projections.FreightRequestsProjection,
) *OrganizationsHandler {
	return &OrganizationsHandler{
		projection:                projection,
		freightRequestsProjection: freightRequestsProjection,
	}
}

func (h *OrganizationsHandler) Handle(msg *message.Message) error {
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

func (h *OrganizationsHandler) handleEvent(ctx context.Context, evt eventstore.Event) error {
	switch e := evt.(type) {
	case events.OrganizationCreated:
		return h.onCreated(ctx, e)
	case events.OrganizationApproved:
		return h.onApproved(ctx, e)
	case events.OrganizationRejected:
		return h.onRejected(ctx, e)
	case events.OrganizationSuspended:
		return h.onSuspended(ctx, e)
	case events.OrganizationUpdated:
		return h.onUpdated(ctx, e)
	}
	return nil
}

func (h *OrganizationsHandler) onCreated(ctx context.Context, e events.OrganizationCreated) error {
	org := projections.OrganizationLookup{
		ID:        e.AggregateID(),
		Name:      e.Name,
		LegalName: e.LegalName,
		INN:       e.INN,
		Status:    values.OrganizationStatusPending.String(),
		CreatedAt: e.OccurredAt(),
	}

	if err := h.projection.Upsert(ctx, org); err != nil {
		return fmt.Errorf("insert organization: %w", err)
	}

	slog.Debug("organization created in lookup",
		slog.String("org_id", e.AggregateID().String()),
		slog.String("name", e.Name))
	return nil
}

func (h *OrganizationsHandler) onApproved(ctx context.Context, e events.OrganizationApproved) error {
	if err := h.projection.UpdateStatus(ctx, e.AggregateID(), values.OrganizationStatusActive.String()); err != nil {
		return fmt.Errorf("update organization status to active: %w", err)
	}

	slog.Debug("organization approved in lookup", slog.String("org_id", e.AggregateID().String()))
	return nil
}

func (h *OrganizationsHandler) onRejected(ctx context.Context, e events.OrganizationRejected) error {
	if err := h.projection.UpdateStatus(ctx, e.AggregateID(), values.OrganizationStatusRejected.String()); err != nil {
		return fmt.Errorf("update organization status to rejected: %w", err)
	}

	slog.Debug("organization rejected in lookup", slog.String("org_id", e.AggregateID().String()))
	return nil
}

func (h *OrganizationsHandler) onSuspended(ctx context.Context, e events.OrganizationSuspended) error {
	if err := h.projection.UpdateStatus(ctx, e.AggregateID(), values.OrganizationStatusSuspended.String()); err != nil {
		return fmt.Errorf("update organization status to suspended: %w", err)
	}

	slog.Debug("organization suspended in lookup", slog.String("org_id", e.AggregateID().String()))
	return nil
}

func (h *OrganizationsHandler) onUpdated(ctx context.Context, e events.OrganizationUpdated) error {
	if e.Name != nil {
		// Обновляем в organizations_lookup
		if err := h.projection.UpdateName(ctx, e.AggregateID(), *e.Name); err != nil {
			return fmt.Errorf("update organization name: %w", err)
		}

		// Обновляем денормализованное имя в freight_requests_lookup
		if h.freightRequestsProjection != nil {
			if err := h.freightRequestsProjection.UpdateCustomerOrgName(ctx, e.AggregateID(), *e.Name); err != nil {
				return fmt.Errorf("update denormalized org name in freight requests: %w", err)
			}
		}

		slog.Debug("organization name updated in lookups",
			slog.String("org_id", e.AggregateID().String()),
			slog.String("name", *e.Name))
	}
	return nil
}
