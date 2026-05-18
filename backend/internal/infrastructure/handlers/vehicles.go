package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill/message"
	orgValues "github.com/udisondev/veziizi/backend/internal/domain/organization/values"

	"github.com/udisondev/veziizi/backend/internal/domain/organization/events"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/projections"
	"github.com/udisondev/veziizi/backend/internal/pkg/dbtx"
)

type VehiclesHandler struct {
	db              dbtx.TxManager
	vehicles        *projections.VehiclesProjection
	pendingVehicles *projections.PendingVehiclesProjection
}

func NewVehiclesHandler(
	db dbtx.TxManager,
	vehicles *projections.VehiclesProjection,
	pendingVehicles *projections.PendingVehiclesProjection,
) *VehiclesHandler {
	return &VehiclesHandler{
		db:              db,
		vehicles:        vehicles,
		pendingVehicles: pendingVehicles,
	}
}

func (h *VehiclesHandler) Handle(msg *message.Message) error {
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

func (h *VehiclesHandler) handleEvent(ctx context.Context, evt eventstore.Event) error {
	switch e := evt.(type) {
	case events.VehicleAdded:
		return h.OnAdded(ctx, &e)
	case events.VehicleUpdated:
		return h.OnUpdated(ctx, &e)
	case events.VehicleVerified:
		return h.OnVerified(ctx, &e)
	case events.VehicleRejected:
		return h.OnRejected(ctx, &e)
	case events.VehicleArchived:
		return h.OnArchived(ctx, &e)
	}
	return nil
}

func upsertFromSpecs(specs events.VehicleSpecsPayload, status string) projections.VehicleUpsertInput {
	loadingTypes := make([]string, 0, len(specs.LoadingTypes))
	for _, lt := range specs.LoadingTypes {
		loadingTypes = append(loadingTypes, lt.String())
	}
	in := projections.VehicleUpsertInput{}
	in.LoadingTypes = loadingTypes
	in.RegistrationNumber = specs.RegistrationNumber
	in.Brand = specs.Brand
	in.Model = specs.Model
	in.VehicleType = specs.VehicleType.String()
	in.VehicleSubType = specs.VehicleSubType.String()
	in.Capacity = specs.Capacity
	in.Volume = specs.Volume
	in.Length = specs.Length
	in.Width = specs.Width
	in.Height = specs.Height
	in.RequiresADR = specs.RequiresADR
	if specs.Temperature != nil {
		in.HasTemperature = true
		tmin := specs.Temperature.Min
		tmax := specs.Temperature.Max
		in.TempMin = &tmin
		in.TempMax = &tmax
	}
	in.Thermograph = specs.Thermograph
	in.Status = status
	return in
}

func (h *VehiclesHandler) OnAdded(ctx context.Context, e *events.VehicleAdded) error {
	in := upsertFromSpecs(e.Specs, orgValues.VehicleStatusPending.String())
	in.ID = e.VehicleID
	in.OrgID = e.AggregateID()
	in.CreatedAt = e.OccurredAt()
	in.UpdatedAt = e.OccurredAt()

	return h.db.InTx(ctx, func(ctx context.Context) error {
		if err := h.vehicles.Upsert(ctx, in); err != nil {
			return err
		}
		return h.pendingVehicles.Upsert(ctx, projections.PendingVehicle{
			ID:                 e.VehicleID,
			OrgID:              e.AggregateID(),
			RegistrationNumber: e.Specs.RegistrationNumber,
			Brand:              nullableStrPtr(e.Specs.Brand),
			Model:              nullableStrPtr(e.Specs.Model),
			VehicleType:        e.Specs.VehicleType.String(),
			VehicleSubType:     e.Specs.VehicleSubType.String(),
			SubmittedAt:        e.OccurredAt(),
		})
	})
}

func (h *VehiclesHandler) OnUpdated(ctx context.Context, e *events.VehicleUpdated) error {
	existing, err := h.vehicles.GetByID(ctx, e.VehicleID)
	if err != nil {
		return fmt.Errorf("load vehicle for update: %w", err)
	}
	in := upsertFromSpecs(e.Specs, orgValues.VehicleStatusPending.String())
	in.ID = e.VehicleID
	in.OrgID = e.AggregateID()
	if existing != nil {
		in.CreatedAt = existing.CreatedAt
	} else {
		in.CreatedAt = e.OccurredAt()
	}
	in.UpdatedAt = e.OccurredAt()

	return h.db.InTx(ctx, func(ctx context.Context) error {
		if err := h.vehicles.Upsert(ctx, in); err != nil {
			return err
		}
		return h.pendingVehicles.Upsert(ctx, projections.PendingVehicle{
			ID:                 e.VehicleID,
			OrgID:              e.AggregateID(),
			RegistrationNumber: e.Specs.RegistrationNumber,
			Brand:              nullableStrPtr(e.Specs.Brand),
			Model:              nullableStrPtr(e.Specs.Model),
			VehicleType:        e.Specs.VehicleType.String(),
			VehicleSubType:     e.Specs.VehicleSubType.String(),
			SubmittedAt:        e.OccurredAt(),
		})
	})
}

func (h *VehiclesHandler) OnVerified(ctx context.Context, e *events.VehicleVerified) error {
	return h.db.InTx(ctx, func(ctx context.Context) error {
		if err := h.vehicles.UpdateStatus(ctx, e.VehicleID, orgValues.VehicleStatusVerified.String(), "", e.OccurredAt()); err != nil {
			return err
		}
		return h.pendingVehicles.Remove(ctx, e.VehicleID)
	})
}

func (h *VehiclesHandler) OnRejected(ctx context.Context, e *events.VehicleRejected) error {
	return h.db.InTx(ctx, func(ctx context.Context) error {
		if err := h.vehicles.UpdateStatus(ctx, e.VehicleID, orgValues.VehicleStatusRejected.String(), e.Reason, e.OccurredAt()); err != nil {
			return err
		}
		return h.pendingVehicles.Remove(ctx, e.VehicleID)
	})
}

func (h *VehiclesHandler) OnArchived(ctx context.Context, e *events.VehicleArchived) error {
	return h.db.InTx(ctx, func(ctx context.Context) error {
		if err := h.vehicles.UpdateStatus(ctx, e.VehicleID, orgValues.VehicleStatusArchived.String(), "", e.OccurredAt()); err != nil {
			return err
		}
		return h.pendingVehicles.Remove(ctx, e.VehicleID)
	})
}

func nullableStrPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
