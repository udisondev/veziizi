package handlers

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/udisondev/veziizi/backend/internal/domain/organization/events"
	"github.com/udisondev/veziizi/backend/internal/domain/organization/values"
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

func (h *OrganizationsHandler) OnCreated(ctx context.Context, e *events.OrganizationCreated) error {
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

func (h *OrganizationsHandler) OnApproved(ctx context.Context, e *events.OrganizationApproved) error {
	if err := h.projection.UpdateStatus(ctx, e.AggregateID(), values.OrganizationStatusActive.String()); err != nil {
		return fmt.Errorf("update organization status to active: %w", err)
	}

	slog.Debug("organization approved in lookup", slog.String("org_id", e.AggregateID().String()))
	return nil
}

func (h *OrganizationsHandler) OnRejected(ctx context.Context, e *events.OrganizationRejected) error {
	if err := h.projection.UpdateStatus(ctx, e.AggregateID(), values.OrganizationStatusRejected.String()); err != nil {
		return fmt.Errorf("update organization status to rejected: %w", err)
	}

	slog.Debug("organization rejected in lookup", slog.String("org_id", e.AggregateID().String()))
	return nil
}

func (h *OrganizationsHandler) OnSuspended(ctx context.Context, e *events.OrganizationSuspended) error {
	if err := h.projection.UpdateStatus(ctx, e.AggregateID(), values.OrganizationStatusSuspended.String()); err != nil {
		return fmt.Errorf("update organization status to suspended: %w", err)
	}

	slog.Debug("organization suspended in lookup", slog.String("org_id", e.AggregateID().String()))
	return nil
}

func (h *OrganizationsHandler) OnUpdated(ctx context.Context, e *events.OrganizationUpdated) error {
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
