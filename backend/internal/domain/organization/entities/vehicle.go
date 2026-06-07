package entities

import (
	"time"

	"github.com/google/uuid"
	"github.com/udisondev/veziizi/backend/internal/domain/organization/values"
	"github.com/udisondev/veziizi/backend/internal/domain/transport"
)

// VehicleSpecs holds all moderation-sensitive attributes of a vehicle.
// Any change to these fields resets the vehicle to unconfirmed status.
type VehicleSpecs struct {
	RegistrationNumber string
	Brand              string
	Model              string
	VehicleType        transport.VehicleType
	VehicleSubType    transport.VehicleSubType
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

type Vehicle struct {
	id              uuid.UUID
	specs           VehicleSpecs
	status          values.VehicleStatus
	rejectionReason string
	createdAt       time.Time
	updatedAt       time.Time
}

func NewVehicle(id uuid.UUID, specs VehicleSpecs, createdAt time.Time) Vehicle {
	return Vehicle{
		id:        id,
		specs:     specs,
		status:    values.VehicleStatusUnconfirmed,
		createdAt: createdAt,
		updatedAt: createdAt,
	}
}

func (v Vehicle) ID() uuid.UUID                { return v.id }
func (v Vehicle) Specs() VehicleSpecs          { return v.specs }
func (v Vehicle) Status() values.VehicleStatus { return v.status }
func (v Vehicle) RejectionReason() string      { return v.rejectionReason }
func (v Vehicle) CreatedAt() time.Time         { return v.createdAt }
func (v Vehicle) UpdatedAt() time.Time         { return v.updatedAt }

func (v Vehicle) RegistrationNumber() string             { return v.specs.RegistrationNumber }
func (v Vehicle) Brand() string                          { return v.specs.Brand }
func (v Vehicle) Model() string                          { return v.specs.Model }
func (v Vehicle) VehicleType() transport.VehicleType     { return v.specs.VehicleType }
func (v Vehicle) VehicleSubType() transport.VehicleSubType {
	return v.specs.VehicleSubType
}
func (v Vehicle) LoadingTypes() []transport.LoadingType { return v.specs.LoadingTypes }
func (v Vehicle) Capacity() float64                     { return v.specs.Capacity }
func (v Vehicle) Volume() float64                       { return v.specs.Volume }
func (v Vehicle) Length() float64                       { return v.specs.Length }
func (v Vehicle) Width() float64                        { return v.specs.Width }
func (v Vehicle) Height() float64                       { return v.specs.Height }
func (v Vehicle) RequiresADR() bool                     { return v.specs.RequiresADR }
func (v Vehicle) Temperature() *transport.Temperature   { return v.specs.Temperature }
func (v Vehicle) Thermograph() bool                     { return v.specs.Thermograph }

func (v Vehicle) IsActive() bool   { return v.status.IsActive() }
func (v Vehicle) IsVerified() bool { return v.status.IsVerified() }
func (v Vehicle) IsArchived() bool { return v.status.IsArchived() }

// MarkUnconfirmed resets the vehicle to unconfirmed after a user edit:
// any spec change voids prior verification, the owner re-submits manually.
func (v *Vehicle) MarkUnconfirmed(specs VehicleSpecs, at time.Time) {
	v.specs = specs
	v.status = values.VehicleStatusUnconfirmed
	v.rejectionReason = ""
	v.updatedAt = at
}

// SubmitForVerification sends the vehicle to admin moderation.
// Status preconditions are checked by the aggregate command.
func (v *Vehicle) SubmitForVerification(at time.Time) {
	v.status = values.VehicleStatusPending
	v.rejectionReason = ""
	v.updatedAt = at
}

func (v *Vehicle) Verify(at time.Time) {
	v.status = values.VehicleStatusVerified
	v.rejectionReason = ""
	v.updatedAt = at
}

func (v *Vehicle) Reject(reason string, at time.Time) {
	v.status = values.VehicleStatusRejected
	v.rejectionReason = reason
	v.updatedAt = at
}

func (v *Vehicle) Archive(at time.Time) {
	v.status = values.VehicleStatusArchived
	v.updatedAt = at
}

// RestoreVehicle creates a Vehicle from stored data (used during event replay).
func RestoreVehicle(
	id uuid.UUID,
	specs VehicleSpecs,
	status values.VehicleStatus,
	rejectionReason string,
	createdAt, updatedAt time.Time,
) Vehicle {
	return Vehicle{
		id:              id,
		specs:           specs,
		status:          status,
		rejectionReason: rejectionReason,
		createdAt:       createdAt,
		updatedAt:       updatedAt,
	}
}
