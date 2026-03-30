package entities

import (
	"time"

	"github.com/udisondev/veziizi/backend/internal/domain/organization/values"
	"github.com/google/uuid"
)

type Invitation struct {
	id        uuid.UUID
	email     string
	role      values.MemberRole
	token     string
	status    values.InvitationStatus
	createdBy uuid.UUID
	createdAt time.Time
	expiresAt time.Time
	name      *string // предзаполненное ФИО (опционально)
	phone     *string // предзаполненный телефон (опционально)
}

func NewInvitation(
	id uuid.UUID,
	email string,
	role values.MemberRole,
	token string,
	createdBy uuid.UUID,
	expiresAt time.Time,
	name *string,
	phone *string,
) Invitation {
	return Invitation{
		id:        id,
		email:     email,
		role:      role,
		token:     token,
		status:    values.InvitationStatusPending,
		createdBy: createdBy,
		createdAt: time.Now().UTC(),
		expiresAt: expiresAt,
		name:      name,
		phone:     phone,
	}
}

func (i Invitation) ID() uuid.UUID                   { return i.id }
func (i Invitation) Email() string                   { return i.email }
func (i Invitation) Role() values.MemberRole         { return i.role }
func (i Invitation) Token() string                   { return i.token }
func (i Invitation) Status() values.InvitationStatus { return i.status }
func (i Invitation) CreatedBy() uuid.UUID            { return i.createdBy }
func (i Invitation) CreatedAt() time.Time            { return i.createdAt }
func (i Invitation) ExpiresAt() time.Time            { return i.expiresAt }
func (i Invitation) Name() *string                   { return i.name }
func (i Invitation) Phone() *string                  { return i.phone }

func (i Invitation) IsExpired() bool {
	return time.Now().UTC().After(i.expiresAt)
}

func (i Invitation) CanBeAccepted() bool {
	return i.status.CanBeAccepted() && !i.IsExpired()
}

func (i Invitation) CanBeCancelled() bool {
	return i.status.CanBeCancelled()
}

func (i *Invitation) Accept() {
	i.status = values.InvitationStatusAccepted
}

func (i *Invitation) Expire() {
	i.status = values.InvitationStatusExpired
}

func (i *Invitation) Cancel() {
	i.status = values.InvitationStatusCancelled
}

// RestoreInvitation creates Invitation from stored data (for event replay)
func RestoreInvitation(
	id uuid.UUID,
	email string,
	role values.MemberRole,
	token string,
	status values.InvitationStatus,
	createdBy uuid.UUID,
	createdAt time.Time,
	expiresAt time.Time,
	name *string,
	phone *string,
) Invitation {
	return Invitation{
		id:        id,
		email:     email,
		role:      role,
		token:     token,
		status:    status,
		createdBy: createdBy,
		createdAt: createdAt,
		expiresAt: expiresAt,
		name:      name,
		phone:     phone,
	}
}
