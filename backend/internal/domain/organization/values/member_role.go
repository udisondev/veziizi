//go:generate go-enum --marshal --sql --names --ptr

package values

// MemberRole represents member's role in organization
// ENUM(owner, administrator, employee)
type MemberRole string

// CanManageMembers returns true if role can invite/block members
func (r MemberRole) CanManageMembers() bool {
	return r == MemberRoleOwner || r == MemberRoleAdministrator
}

// CanManageOrganization returns true if role can edit organization settings
func (r MemberRole) CanManageOrganization() bool {
	return r == MemberRoleOwner || r == MemberRoleAdministrator
}

// CanBeRemoved returns true if member with this role can be removed
func (r MemberRole) CanBeRemoved() bool {
	return r != MemberRoleOwner
}

// CanManageFreightRequests returns true if role can manage freight requests (select/reject offers)
func (r MemberRole) CanManageFreightRequests() bool {
	return r == MemberRoleOwner || r == MemberRoleAdministrator
}
