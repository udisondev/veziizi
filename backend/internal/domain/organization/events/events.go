package events

import (
	"codeberg.org/udison/veziizi/backend/internal/domain/organization/values"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"github.com/google/uuid"
)

const AggregateType = "organization"

// Event type constants
const (
	TypeOrganizationCreated   = "organization.created"
	TypeOrganizationApproved  = "organization.approved"
	TypeOrganizationRejected  = "organization.rejected"
	TypeOrganizationSuspended = "organization.suspended"
	TypeOrganizationUpdated   = "organization.updated"
	TypeMemberAdded           = "member.added"
	TypeMemberRemoved         = "member.removed"
	TypeMemberRoleChanged     = "member.role_changed"
	TypeMemberBlocked         = "member.blocked"
	TypeMemberUnblocked       = "member.unblocked"
	TypeInvitationCreated     = "invitation.created"
	TypeInvitationAccepted    = "invitation.accepted"
	TypeInvitationExpired     = "invitation.expired"
	TypeInvitationCancelled   = "invitation.cancelled"
	TypeFraudsterMarked       = "fraudster.marked"
	TypeFraudsterUnmarked     = "fraudster.unmarked"
)

func init() {
	eventstore.RegisterEventType[OrganizationCreated](TypeOrganizationCreated)
	eventstore.RegisterEventType[OrganizationApproved](TypeOrganizationApproved)
	eventstore.RegisterEventType[OrganizationRejected](TypeOrganizationRejected)
	eventstore.RegisterEventType[OrganizationSuspended](TypeOrganizationSuspended)
	eventstore.RegisterEventType[OrganizationUpdated](TypeOrganizationUpdated)
	eventstore.RegisterEventType[MemberAdded](TypeMemberAdded)
	eventstore.RegisterEventType[MemberRemoved](TypeMemberRemoved)
	eventstore.RegisterEventType[MemberRoleChanged](TypeMemberRoleChanged)
	eventstore.RegisterEventType[MemberBlocked](TypeMemberBlocked)
	eventstore.RegisterEventType[MemberUnblocked](TypeMemberUnblocked)
	eventstore.RegisterEventType[InvitationCreated](TypeInvitationCreated)
	eventstore.RegisterEventType[InvitationAccepted](TypeInvitationAccepted)
	eventstore.RegisterEventType[InvitationExpired](TypeInvitationExpired)
	eventstore.RegisterEventType[InvitationCancelled](TypeInvitationCancelled)
	eventstore.RegisterEventType[FraudsterMarked](TypeFraudsterMarked)
	eventstore.RegisterEventType[FraudsterUnmarked](TypeFraudsterUnmarked)
}

// OrganizationCreated is emitted when organization is registered
type OrganizationCreated struct {
	eventstore.BaseEvent
	Name      string         `json:"name"`
	INN       string         `json:"inn"`
	LegalName string         `json:"legal_name"`
	Country   values.Country `json:"country"`
	Phone     string         `json:"phone"`
	Email     string         `json:"email"`
	Address   values.Address `json:"address"`
}

func (e OrganizationCreated) EventType() string { return TypeOrganizationCreated }

// OrganizationApproved is emitted when admin approves organization
type OrganizationApproved struct {
	eventstore.BaseEvent
	ApprovedBy uuid.UUID `json:"approved_by"` // admin ID
}

func (e OrganizationApproved) EventType() string { return TypeOrganizationApproved }

// OrganizationRejected is emitted when admin rejects organization
type OrganizationRejected struct {
	eventstore.BaseEvent
	RejectedBy uuid.UUID `json:"rejected_by"` // admin ID
	Reason     string    `json:"reason"`
}

func (e OrganizationRejected) EventType() string { return TypeOrganizationRejected }

// OrganizationSuspended is emitted when organization is suspended
type OrganizationSuspended struct {
	eventstore.BaseEvent
	SuspendedBy uuid.UUID `json:"suspended_by"` // admin ID
	Reason      string    `json:"reason"`
}

func (e OrganizationSuspended) EventType() string { return TypeOrganizationSuspended }

// OrganizationUpdated is emitted when organization info is updated
type OrganizationUpdated struct {
	eventstore.BaseEvent
	Name      *string         `json:"name,omitempty"`
	Phone     *string         `json:"phone,omitempty"`
	Email     *string         `json:"email,omitempty"`
	Address   *values.Address `json:"address,omitempty"`
	UpdatedBy uuid.UUID       `json:"updated_by"` // member ID
}

func (e OrganizationUpdated) EventType() string { return TypeOrganizationUpdated }

// MemberAdded is emitted when new member joins organization
type MemberAdded struct {
	eventstore.BaseEvent
	MemberID     uuid.UUID         `json:"member_id"`
	Email        string            `json:"email"`
	PasswordHash string            `json:"password_hash"`
	Name         string            `json:"name"`
	Phone        string            `json:"phone,omitempty"`
	Role         values.MemberRole `json:"role"`
	InvitedBy    *uuid.UUID        `json:"invited_by,omitempty"` // nil for owner
	// Registration metadata for fraud detection
	RegistrationIP          string `json:"registration_ip,omitempty"`
	RegistrationFingerprint string `json:"registration_fingerprint,omitempty"`
	RegistrationUserAgent   string `json:"registration_user_agent,omitempty"`
}

func (e MemberAdded) EventType() string { return TypeMemberAdded }

// MemberRemoved is emitted when member is removed from organization (dev only)
type MemberRemoved struct {
	eventstore.BaseEvent
	MemberID uuid.UUID `json:"member_id"`
}

func (e MemberRemoved) EventType() string { return TypeMemberRemoved }

// MemberRoleChanged is emitted when member role is changed
type MemberRoleChanged struct {
	eventstore.BaseEvent
	MemberID  uuid.UUID         `json:"member_id"`
	OldRole   values.MemberRole `json:"old_role"`
	NewRole   values.MemberRole `json:"new_role"`
	ChangedBy uuid.UUID         `json:"changed_by"` // member ID
}

func (e MemberRoleChanged) EventType() string { return TypeMemberRoleChanged }

// MemberBlocked is emitted when member is blocked
type MemberBlocked struct {
	eventstore.BaseEvent
	MemberID  uuid.UUID `json:"member_id"`
	BlockedBy uuid.UUID `json:"blocked_by"` // member ID
	Reason    string    `json:"reason,omitempty"`
}

func (e MemberBlocked) EventType() string { return TypeMemberBlocked }

// MemberUnblocked is emitted when member is unblocked
type MemberUnblocked struct {
	eventstore.BaseEvent
	MemberID    uuid.UUID `json:"member_id"`
	UnblockedBy uuid.UUID `json:"unblocked_by"` // member ID
}

func (e MemberUnblocked) EventType() string { return TypeMemberUnblocked }

// InvitationCreated is emitted when invitation is created
type InvitationCreated struct {
	eventstore.BaseEvent
	InvitationID uuid.UUID         `json:"invitation_id"`
	Email        string            `json:"email"`
	Role         values.MemberRole `json:"role"`
	Token        string            `json:"token"`
	CreatedBy    uuid.UUID         `json:"created_by"` // member ID
	ExpiresAt    int64             `json:"expires_at"` // unix timestamp
	Name         *string           `json:"name,omitempty"`  // предзаполненное ФИО
	Phone        *string           `json:"phone,omitempty"` // предзаполненный телефон
}

func (e InvitationCreated) EventType() string { return TypeInvitationCreated }

// InvitationAccepted is emitted when invitation is accepted
type InvitationAccepted struct {
	eventstore.BaseEvent
	InvitationID uuid.UUID `json:"invitation_id"`
	MemberID     uuid.UUID `json:"member_id"` // new member ID
}

func (e InvitationAccepted) EventType() string { return TypeInvitationAccepted }

// InvitationExpired is emitted when invitation expires
type InvitationExpired struct {
	eventstore.BaseEvent
	InvitationID uuid.UUID `json:"invitation_id"`
}

func (e InvitationExpired) EventType() string { return TypeInvitationExpired }

// InvitationCancelled is emitted when invitation is cancelled
type InvitationCancelled struct {
	eventstore.BaseEvent
	InvitationID uuid.UUID `json:"invitation_id"`
	CancelledBy  uuid.UUID `json:"cancelled_by"` // member ID
}

func (e InvitationCancelled) EventType() string { return TypeInvitationCancelled }

// FraudsterMarked is emitted when admin marks organization as fraudster
type FraudsterMarked struct {
	eventstore.BaseEvent
	MarkedBy    uuid.UUID `json:"marked_by"`    // admin ID
	IsConfirmed bool      `json:"is_confirmed"` // true = confirmed, false = suspected
	Reason      string    `json:"reason"`
}

func (e FraudsterMarked) EventType() string { return TypeFraudsterMarked }

// FraudsterUnmarked is emitted when admin removes fraudster status
type FraudsterUnmarked struct {
	eventstore.BaseEvent
	UnmarkedBy uuid.UUID `json:"unmarked_by"` // admin ID
	Reason     string    `json:"reason"`
}

func (e FraudsterUnmarked) EventType() string { return TypeFraudsterUnmarked }
