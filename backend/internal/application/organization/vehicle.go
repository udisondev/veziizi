package organization

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/udisondev/veziizi/backend/internal/domain/organization"
	"github.com/udisondev/veziizi/backend/internal/domain/organization/entities"
	"github.com/udisondev/veziizi/backend/internal/domain/transport"
)

// VehicleSpecsInput is a transport-agnostic DTO accepted by the service from HTTP layer.
type VehicleSpecsInput struct {
	RegistrationNumber string
	Brand              string
	Model              string
	VehicleType        transport.VehicleType
	VehicleSubType     transport.VehicleSubType
	LoadingTypes       []transport.LoadingType
	Capacity           float64
	Volume             float64
	Length             float64
	Width              float64
	Height             float64
	RequiresADR        bool
	Temperature        *transport.Temperature
	Thermograph        bool
}

func (v VehicleSpecsInput) Validate() error {
	if v.RegistrationNumber == "" {
		return fmt.Errorf("registration_number is required")
	}
	if len(v.RegistrationNumber) > 32 {
		return fmt.Errorf("registration_number is too long (max 32)")
	}
	if v.VehicleType == "" || !v.VehicleType.IsValid() {
		return fmt.Errorf("invalid vehicle_type: %s", v.VehicleType)
	}
	if v.VehicleSubType == "" || !v.VehicleSubType.IsValid() {
		return fmt.Errorf("invalid vehicle_subtype: %s", v.VehicleSubType)
	}
	if !transport.IsValidSubtypeForType(v.VehicleType, v.VehicleSubType) {
		return fmt.Errorf("vehicle_subtype %s is not valid for vehicle_type %s", v.VehicleSubType, v.VehicleType)
	}
	for _, lt := range v.LoadingTypes {
		if !lt.IsValid() {
			return fmt.Errorf("invalid loading_type: %s", lt)
		}
	}
	if v.Temperature != nil {
		if err := v.Temperature.Validate(); err != nil {
			return fmt.Errorf("temperature: %w", err)
		}
	}
	return nil
}

func (v VehicleSpecsInput) toEntity() entities.VehicleSpecs {
	return entities.VehicleSpecs{
		RegistrationNumber: v.RegistrationNumber,
		Brand:              v.Brand,
		Model:              v.Model,
		VehicleType:        v.VehicleType,
		VehicleSubType:     v.VehicleSubType,
		LoadingTypes:       v.LoadingTypes,
		Capacity:           v.Capacity,
		Volume:             v.Volume,
		Length:             v.Length,
		Width:              v.Width,
		Height:             v.Height,
		RequiresADR:        v.RequiresADR,
		Temperature:        v.Temperature,
		Thermograph:        v.Thermograph,
	}
}

type AddVehicleInput struct {
	OrganizationID uuid.UUID
	ActorID        uuid.UUID
	Specs          VehicleSpecsInput
}

func (s *Service) AddVehicle(ctx context.Context, input AddVehicleInput) (uuid.UUID, error) {
	if err := input.Specs.Validate(); err != nil {
		return uuid.Nil, err
	}

	var vehicleID uuid.UUID
	err := s.db.InTx(ctx, func(ctx context.Context) error {
		org, err := s.Get(ctx, input.OrganizationID)
		if err != nil {
			return err
		}
		vehicleID = uuid.New()
		if err := org.AddVehicle(input.ActorID, vehicleID, input.Specs.toEntity()); err != nil {
			return err
		}
		return s.saveAndPublish(ctx, org)
	})
	if err != nil {
		return uuid.Nil, err
	}
	return vehicleID, nil
}

type UpdateVehicleInput struct {
	OrganizationID uuid.UUID
	ActorID        uuid.UUID
	VehicleID      uuid.UUID
	Specs          VehicleSpecsInput
}

func (s *Service) UpdateVehicle(ctx context.Context, input UpdateVehicleInput) error {
	if err := input.Specs.Validate(); err != nil {
		return err
	}

	return s.db.InTx(ctx, func(ctx context.Context) error {
		org, err := s.Get(ctx, input.OrganizationID)
		if err != nil {
			return err
		}
		if err := org.UpdateVehicle(input.ActorID, input.VehicleID, input.Specs.toEntity()); err != nil {
			return err
		}
		return s.saveAndPublish(ctx, org)
	})
}

type ArchiveVehicleInput struct {
	OrganizationID uuid.UUID
	ActorID        uuid.UUID
	VehicleID      uuid.UUID
}

func (s *Service) ArchiveVehicle(ctx context.Context, input ArchiveVehicleInput) error {
	return s.db.InTx(ctx, func(ctx context.Context) error {
		org, err := s.Get(ctx, input.OrganizationID)
		if err != nil {
			return err
		}
		if err := org.ArchiveVehicle(input.ActorID, input.VehicleID); err != nil {
			return err
		}
		return s.saveAndPublish(ctx, org)
	})
}

type SubmitVehicleInput struct {
	OrganizationID uuid.UUID
	ActorID        uuid.UUID
	VehicleID      uuid.UUID
}

// SubmitVehicle отправляет транспорт на модерацию по запросу владельца.
func (s *Service) SubmitVehicle(ctx context.Context, input SubmitVehicleInput) error {
	return s.db.InTx(ctx, func(ctx context.Context) error {
		org, err := s.Get(ctx, input.OrganizationID)
		if err != nil {
			return err
		}
		if err := org.SubmitVehicleForVerification(input.ActorID, input.VehicleID); err != nil {
			return err
		}
		return s.saveAndPublish(ctx, org)
	})
}

// VehicleAggregateOps хелперы для admin service: они достают агрегат и применяют команду.
// Не пересоздаём admin-логику отдельно: admin service использует Organization service
// для управления автопарком.

type VerifyVehicleInput struct {
	OrganizationID uuid.UUID
	VehicleID      uuid.UUID
	AdminID        uuid.UUID
}

func (s *Service) VerifyVehicle(ctx context.Context, input VerifyVehicleInput) error {
	return s.db.InTx(ctx, func(ctx context.Context) error {
		org, err := s.Get(ctx, input.OrganizationID)
		if err != nil {
			return err
		}
		if err := org.VerifyVehicle(input.AdminID, input.VehicleID); err != nil {
			return err
		}
		return s.saveAndPublish(ctx, org)
	})
}

type RejectVehicleInput struct {
	OrganizationID uuid.UUID
	VehicleID      uuid.UUID
	AdminID        uuid.UUID
	Reason         string
}

func (s *Service) RejectVehicle(ctx context.Context, input RejectVehicleInput) error {
	return s.db.InTx(ctx, func(ctx context.Context) error {
		org, err := s.Get(ctx, input.OrganizationID)
		if err != nil {
			return err
		}
		if err := org.RejectVehicle(input.AdminID, input.VehicleID, input.Reason); err != nil {
			return err
		}
		return s.saveAndPublish(ctx, org)
	})
}

// VehicleInfo is a lightweight aggregated view returned by Service.GetVehicle.
// Combines aggregate-side spec with projection-side status snapshot.
type VehicleInfo struct {
	OrgID   uuid.UUID
	Vehicle entities.Vehicle
}

// GetVehicleFromAggregate loads vehicle directly from the org aggregate.
// Use it for moderation flows where we need authoritative state (no projection lag).
func (s *Service) GetVehicleFromAggregate(ctx context.Context, orgID, vehicleID uuid.UUID) (*entities.Vehicle, *organization.Organization, error) {
	org, err := s.Get(ctx, orgID)
	if err != nil {
		return nil, nil, err
	}
	vehicle, ok := org.GetVehicle(vehicleID)
	if !ok {
		return nil, org, organization.ErrVehicleNotFound
	}
	return vehicle, org, nil
}
