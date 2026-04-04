package organization

import "errors"

var (
	ErrOrganizationNotFound        = errors.New("organization not found")
	ErrOrganizationNotActive       = errors.New("organization is not active")
	ErrOrganizationNotPending      = errors.New("organization is not pending")
	ErrMemberNotFound              = errors.New("member not found")
	ErrMemberNotActive             = errors.New("member is not active")
	ErrMemberAlreadyExists         = errors.New("member with this email already exists")
	ErrMemberCannotBeRemoved       = errors.New("owner cannot be removed")
	ErrInvitationNotFound          = errors.New("invitation not found")
	ErrInvitationExpired           = errors.New("invitation has expired")
	ErrInvitationAlreadyUsed       = errors.New("invitation already used")
	ErrInvitationCancelled         = errors.New("invitation has been cancelled")
	ErrInvitationCannotBeCancelled = errors.New("invitation cannot be cancelled")
	ErrInsufficientPermissions     = errors.New("insufficient permissions")
	ErrCannotChangeOwnRole         = errors.New("cannot change own role")
	ErrCannotBlockSelf             = errors.New("cannot block yourself")
	ErrCannotEditOwner             = errors.New("only owner can edit their own info")
	ErrEmailAlreadyInvited         = errors.New("email already has pending invitation")
	ErrNameRequired                = errors.New("name is required")
	ErrPhoneRequired               = errors.New("phone is required")
	ErrAlreadyFraudster            = errors.New("organization is already marked as fraudster")
	ErrNotFraudster                = errors.New("organization is not marked as fraudster")
	ErrDisposableEmail             = errors.New("disposable email addresses are not allowed")
	ErrRegistrationVelocity        = errors.New("too many registration attempts")
)
