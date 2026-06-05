package handlers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/udisondev/veziizi/backend/internal/domain/organization"
	"github.com/udisondev/veziizi/backend/internal/domain/organization/events"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/projections"
)

// OrganizationsHandler обновляет organizations_lookup по паттерну
// rebuild-from-aggregate (см. FreightRequestsHandler): на любое событие
// перечитывает агрегат из event store и пишет полное состояние с
// version-guard'ом — безопасно для at-least-once и конкурентных инстансов.
type OrganizationsHandler struct {
	eventStore                eventstore.Store
	projection                *projections.OrganizationsProjection
	freightRequestsProjection *projections.FreightRequestsProjection
}

func NewOrganizationsHandler(
	eventStore eventstore.Store,
	projection *projections.OrganizationsProjection,
	freightRequestsProjection *projections.FreightRequestsProjection,
) *OrganizationsHandler {
	return &OrganizationsHandler{
		eventStore:                eventStore,
		projection:                projection,
		freightRequestsProjection: freightRequestsProjection,
	}
}

func (h *OrganizationsHandler) rebuild(ctx context.Context, id uuid.UUID) error {
	res, err := h.eventStore.LoadWithSnapshot(ctx, id, events.AggregateType)
	if err != nil {
		if errors.Is(err, eventstore.ErrAggregateNotFound) {
			slog.Warn("organization not found in event store, skipping rebuild",
				slog.String("org_id", id.String()))
			return nil
		}
		return fmt.Errorf("load organization: %w", err)
	}

	org, err := organization.NewFromStore(id, res.SnapshotState, res.Events)
	if err != nil {
		return fmt.Errorf("restore organization: %w", err)
	}
	if org.Version() == 0 {
		slog.Warn("organization has no events, skipping rebuild", slog.String("org_id", id.String()))
		return nil
	}

	if err := h.projection.Upsert(ctx, projections.OrganizationLookup{
		ID:        id,
		Name:      org.Name(),
		LegalName: org.LegalName(),
		INN:       org.INN(),
		Status:    org.Status().String(),
		CreatedAt: org.CreatedAt(),
		Version:   org.Version(),
	}); err != nil {
		return fmt.Errorf("upsert organization: %w", err)
	}

	slog.Debug("organization lookup rebuilt",
		slog.String("org_id", id.String()),
		slog.Int64("version", org.Version()))
	return nil
}

func (h *OrganizationsHandler) OnCreated(ctx context.Context, e *events.OrganizationCreated) error {
	return h.rebuild(ctx, e.AggregateID())
}

func (h *OrganizationsHandler) OnApproved(ctx context.Context, e *events.OrganizationApproved) error {
	return h.rebuild(ctx, e.AggregateID())
}

func (h *OrganizationsHandler) OnRejected(ctx context.Context, e *events.OrganizationRejected) error {
	return h.rebuild(ctx, e.AggregateID())
}

func (h *OrganizationsHandler) OnSuspended(ctx context.Context, e *events.OrganizationSuspended) error {
	return h.rebuild(ctx, e.AggregateID())
}

func (h *OrganizationsHandler) OnUpdated(ctx context.Context, e *events.OrganizationUpdated) error {
	if err := h.rebuild(ctx, e.AggregateID()); err != nil {
		return err
	}

	// Денормализованное имя заказчика в freight_requests_lookup — eventual
	// consistency: UPDATE по customer_org_id затрагивает только существующие на
	// этот момент строки. Заявка, созданная конкурентно (другой воркер, другой
	// топик), возьмёт актуальное имя сама при своём rebuild'е — синхронизировать
	// два независимых consumer group'а здесь нечем и незачем (косметика).
	if e.Name != nil && h.freightRequestsProjection != nil {
		if err := h.freightRequestsProjection.UpdateCustomerOrgName(ctx, e.AggregateID(), *e.Name); err != nil {
			return fmt.Errorf("update denormalized org name in freight requests: %w", err)
		}
	}
	return nil
}
