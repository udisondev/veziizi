//go:generate go-enum --marshal --sql --names --ptr

package values

// OrganizationStatus represents organization lifecycle status
// ENUM(pending, active, suspended, rejected)
type OrganizationStatus string

func (s OrganizationStatus) CanBeApproved() bool {
	return s == OrganizationStatusPending
}

func (s OrganizationStatus) CanBeRejected() bool {
	return s == OrganizationStatusPending
}

func (s OrganizationStatus) CanBeSuspended() bool {
	return s == OrganizationStatusActive
}
