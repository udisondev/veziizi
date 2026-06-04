package handlers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	orgValues "github.com/udisondev/veziizi/backend/internal/domain/organization/values"

	"github.com/udisondev/veziizi/backend/internal/domain/organization"
	"github.com/udisondev/veziizi/backend/internal/domain/organization/entities"
	"github.com/udisondev/veziizi/backend/internal/domain/organization/events"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/projections"
	"github.com/udisondev/veziizi/backend/internal/pkg/dbtx"
)

// VehiclesHandler обновляет vehicles_lookup и pending_vehicles по паттерну
// rebuild-from-aggregate (см. OrganizationsHandler): на любое vehicle-событие
// перечитывает агрегат организации и пишет полное состояние машины — порядок
// доставки не важен, строка всегда f(aggregate). Без этого устаревший
// VehicleAdded/VehicleUpdated (status=pending), доставленный после
// VehicleVerified, откатывал бы verified→pending и возвращал машину в очередь
// модерации.
//
// pending_vehicles — таблица присутствия (машина там ровно пока ждёт
// модерацию), version-guard к ней неприменим, поэтому конкурентные rebuild'ы
// одной машины сериализуются advisory xact-lock'ом по vehicle id: rebuild со
// старым снимком агрегата не может закоммититься поверх свежего.
type VehiclesHandler struct {
	db              dbtx.TxManager
	eventStore      eventstore.Store
	vehicles        *projections.VehiclesProjection
	pendingVehicles *projections.PendingVehiclesProjection
}

func NewVehiclesHandler(
	db dbtx.TxManager,
	eventStore eventstore.Store,
	vehicles *projections.VehiclesProjection,
	pendingVehicles *projections.PendingVehiclesProjection,
) *VehiclesHandler {
	return &VehiclesHandler{
		db:              db,
		eventStore:      eventStore,
		vehicles:        vehicles,
		pendingVehicles: pendingVehicles,
	}
}

func upsertFromVehicle(orgID uuid.UUID, v entities.Vehicle) projections.VehicleUpsertInput {
	specs := v.Specs()
	loadingTypes := make([]string, 0, len(specs.LoadingTypes))
	for _, lt := range specs.LoadingTypes {
		loadingTypes = append(loadingTypes, lt.String())
	}
	in := projections.VehicleUpsertInput{
		ID:                 v.ID(),
		OrgID:              orgID,
		RegistrationNumber: specs.RegistrationNumber,
		Brand:              specs.Brand,
		Model:              specs.Model,
		VehicleType:        specs.VehicleType.String(),
		VehicleSubType:     specs.VehicleSubType.String(),
		LoadingTypes:       loadingTypes,
		Capacity:           specs.Capacity,
		Volume:             specs.Volume,
		Length:             specs.Length,
		Width:              specs.Width,
		Height:             specs.Height,
		RequiresADR:        specs.RequiresADR,
		Thermograph:        specs.Thermograph,
		Status:             v.Status().String(),
		RejectionReason:    v.RejectionReason(),
		CreatedAt:          v.CreatedAt(),
		UpdatedAt:          v.UpdatedAt(),
	}
	if specs.Temperature != nil {
		in.HasTemperature = true
		tmin := specs.Temperature.Min
		tmax := specs.Temperature.Max
		in.TempMin = &tmin
		in.TempMax = &tmax
	}
	return in
}

func (h *VehiclesHandler) rebuild(ctx context.Context, orgID, vehicleID uuid.UUID) error {
	return h.db.InTx(ctx, func(ctx context.Context) error {
		// Лок ДО чтения агрегата: сериализованный rebuild всегда коммитит
		// состояние не старее предыдущего.
		if err := lockProjectionRow(ctx, h.db, vehicleID); err != nil {
			return err
		}

		res, err := h.eventStore.LoadWithSnapshot(ctx, orgID, events.AggregateType)
		if err != nil {
			if errors.Is(err, eventstore.ErrAggregateNotFound) {
				slog.Warn("organization not found in event store, skipping vehicle rebuild",
					slog.String("org_id", orgID.String()),
					slog.String("vehicle_id", vehicleID.String()))
				return nil
			}
			return fmt.Errorf("load organization: %w", err)
		}
		org, err := organization.NewFromStore(orgID, res.SnapshotState, res.Events)
		if err != nil {
			return fmt.Errorf("restore organization: %w", err)
		}

		v, ok := org.GetVehicle(vehicleID)
		if !ok {
			// Событие машины есть, а в агрегате её нет — битое состояние,
			// retry не поможет: логируем и ack'аем.
			slog.Error("vehicle not found in aggregate, skipping rebuild",
				slog.String("org_id", orgID.String()),
				slog.String("vehicle_id", vehicleID.String()))
			return nil
		}

		if err := h.vehicles.Upsert(ctx, upsertFromVehicle(orgID, *v)); err != nil {
			return err
		}

		if v.Status() == orgValues.VehicleStatusPending {
			specs := v.Specs()
			return h.pendingVehicles.Upsert(ctx, projections.PendingVehicle{
				ID:                 vehicleID,
				OrgID:              orgID,
				RegistrationNumber: specs.RegistrationNumber,
				Brand:              nullableStrPtr(specs.Brand),
				Model:              nullableStrPtr(specs.Model),
				VehicleType:        specs.VehicleType.String(),
				VehicleSubType:     specs.VehicleSubType.String(),
				SubmittedAt:        v.UpdatedAt(),
			})
		}
		return h.pendingVehicles.Remove(ctx, vehicleID)
	})
}

func (h *VehiclesHandler) OnAdded(ctx context.Context, e *events.VehicleAdded) error {
	return h.rebuild(ctx, e.AggregateID(), e.VehicleID)
}

func (h *VehiclesHandler) OnUpdated(ctx context.Context, e *events.VehicleUpdated) error {
	return h.rebuild(ctx, e.AggregateID(), e.VehicleID)
}

func (h *VehiclesHandler) OnVerified(ctx context.Context, e *events.VehicleVerified) error {
	return h.rebuild(ctx, e.AggregateID(), e.VehicleID)
}

func (h *VehiclesHandler) OnRejected(ctx context.Context, e *events.VehicleRejected) error {
	return h.rebuild(ctx, e.AggregateID(), e.VehicleID)
}

func (h *VehiclesHandler) OnArchived(ctx context.Context, e *events.VehicleArchived) error {
	return h.rebuild(ctx, e.AggregateID(), e.VehicleID)
}

func nullableStrPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
