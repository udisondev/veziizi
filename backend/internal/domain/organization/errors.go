package organization

import "errors"

var (
	ErrOrganizationNotFound     = errors.New("organization not found")
	ErrOrganizationNotActive    = errors.New("organization is not active")
	ErrOrganizationNotPending   = errors.New("organization is not pending")
	ErrMemberNotFound           = errors.New("member not found")
	ErrMemberNotActive          = errors.New("member is not active")
	ErrMemberAlreadyExists      = errors.New("member with this email already exists")
	ErrMemberCannotBeRemoved    = errors.New("owner cannot be removed")
	ErrInvitationNotFound       = errors.New("invitation not found")
	ErrInvitationExpired        = errors.New("invitation has expired")
	ErrInvitationAlreadyUsed    = errors.New("invitation already used")
	ErrInsufficientPermissions  = errors.New("insufficient permissions")
	ErrCannotChangeOwnRole      = errors.New("cannot change own role")
	ErrCannotBlockSelf          = errors.New("cannot block yourself")
	ErrEmailAlreadyInvited      = errors.New("email already has pending invitation")
)
