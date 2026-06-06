//go:generate go-enum --marshal --sql --names --ptr

package values

// VehicleStatus represents vehicle moderation lifecycle status
// ENUM(unconfirmed, pending, verified, rejected, archived)
type VehicleStatus string

func (s VehicleStatus) IsUnconfirmed() bool { return s == VehicleStatusUnconfirmed }

func (s VehicleStatus) IsPending() bool  { return s == VehicleStatusPending }
func (s VehicleStatus) IsVerified() bool { return s == VehicleStatusVerified }
func (s VehicleStatus) IsRejected() bool { return s == VehicleStatusRejected }
func (s VehicleStatus) IsArchived() bool { return s == VehicleStatusArchived }

// CanBeModerated returns true when admin can verify/reject the vehicle.
func (s VehicleStatus) CanBeModerated() bool {
	return s == VehicleStatusPending
}

// IsActive returns true for vehicles that are visible in the carrier fleet
// (everything except archived).
func (s VehicleStatus) IsActive() bool {
	return s != VehicleStatusArchived
}
