package handlers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/udisondev/veziizi/backend/internal/domain/organization"
	"github.com/udisondev/veziizi/backend/internal/domain/organization/events"
	"github.com/udisondev/veziizi/backend/internal/domain/organization/values"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"github.com/udisondev/veziizi/backend/internal/pkg/dbtx"
)

// PendingOrganizationsHandler ведёт pending_organizations (очередь модерации
// админки) по паттерну rebuild-from-aggregate: на любое статусное событие
// перечитывает агрегат и приводит строку к f(aggregate) — организация в
// pending → строка есть, иначе удалена. Per-event модель (INSERT по Created,
// DELETE по Approved/Rejected) здесь небезопасна: DELETE, доставленный раньше
// INSERT (out-of-order при N инстансах), оставлял бы зомби в очереди навсегда.
//
// Таблица присутствия без version-колонки → конкурентные rebuild'ы одной
// организации сериализуются advisory xact-lock'ом (см. lockProjectionRow).
type PendingOrganizationsHandler struct {
	db         dbtx.TxManager
	eventStore eventstore.Store
	psql       squirrel.StatementBuilderType
}

func NewPendingOrganizationsHandler(db dbtx.TxManager, eventStore eventstore.Store) *PendingOrganizationsHandler {
	return &PendingOrganizationsHandler{
		db:         db,
		eventStore: eventStore,
		psql:       squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (h *PendingOrganizationsHandler) OnCreated(ctx context.Context, e *events.OrganizationCreated) error {
	return h.rebuild(ctx, e.AggregateID())
}

func (h *PendingOrganizationsHandler) OnApproved(ctx context.Context, e *events.OrganizationApproved) error {
	return h.rebuild(ctx, e.AggregateID())
}

func (h *PendingOrganizationsHandler) OnRejected(ctx context.Context, e *events.OrganizationRejected) error {
	return h.rebuild(ctx, e.AggregateID())
}

func (h *PendingOrganizationsHandler) rebuild(ctx context.Context, orgID uuid.UUID) error {
	return h.db.InTx(ctx, func(ctx context.Context) error {
		// Лок ДО чтения агрегата: сериализованный rebuild всегда коммитит
		// состояние не старее предыдущего.
		if err := lockProjectionRow(ctx, h.db, orgID); err != nil {
			return err
		}

		res, err := h.eventStore.LoadWithSnapshot(ctx, orgID, events.AggregateType)
		if err != nil {
			if errors.Is(err, eventstore.ErrAggregateNotFound) {
				slog.Warn("organization not found in event store, skipping pending rebuild",
					slog.String("org_id", orgID.String()))
				return nil
			}
			return fmt.Errorf("load organization: %w", err)
		}
		org, err := organization.NewFromStore(orgID, res.SnapshotState, res.Events)
		if err != nil {
			return fmt.Errorf("restore organization: %w", err)
		}

		if org.Status() != values.OrganizationStatusPending {
			return h.remove(ctx, orgID)
		}

		query, args, err := h.psql.
			Insert("pending_organizations").
			Columns("id", "name", "inn", "legal_name", "country", "email", "created_at").
			Values(orgID, org.Name(), org.INN(), org.LegalName(), org.Country().String(), org.Email(), org.CreatedAt()).
			Suffix("ON CONFLICT (id) DO NOTHING").
			ToSql()
		if err != nil {
			return fmt.Errorf("failed to build insert query: %w", err)
		}
		if _, err := h.db.Exec(ctx, query, args...); err != nil {
			return fmt.Errorf("failed to insert pending organization: %w", err)
		}

		slog.Debug("organization added to pending", slog.String("org_id", orgID.String()))
		return nil
	})
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
