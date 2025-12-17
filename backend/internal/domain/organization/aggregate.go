package organization

import (
	"time"

	"codeberg.org/udison/veziizi/backend/internal/domain/organization/entities"
	"codeberg.org/udison/veziizi/backend/internal/domain/organization/events"
	"codeberg.org/udison/veziizi/backend/internal/domain/organization/values"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"codeberg.org/udison/veziizi/backend/internal/pkg/aggregate"
	"github.com/google/uuid"
)

type Organization struct {
	aggregate.Base

	name           string
	inn            string
	legalName      string
	country        values.Country
	phone          string
	email          string
	address   values.Address
	status    values.OrganizationStatus
	createdAt time.Time
	suspendedAt    *time.Time

	members     map[uuid.UUID]*entities.Member
	invitations map[uuid.UUID]*entities.Invitation
}

// New creates a new Organization (for registration)
func New(
	id uuid.UUID,
	name string,
	inn string,
	legalName string,
	country values.Country,
	phone string,
	email string,
	address values.Address,
	ownerID uuid.UUID,
	ownerEmail string,
	ownerPasswordHash string,
	ownerName string,
	ownerPhone string,
) *Organization {
	org := &Organization{
		Base:        aggregate.NewBase(id),
		members:     make(map[uuid.UUID]*entities.Member),
		invitations: make(map[uuid.UUID]*entities.Invitation),
	}

	// Apply OrganizationCreated
	org.Apply(events.OrganizationCreated{
		BaseEvent: eventstore.NewBaseEvent(id, events.AggregateType, org.Version()+1),
		Name:      name,
		INN:       inn,
		LegalName: legalName,
		Country:   country,
		Phone:     phone,
		Email:     email,
		Address:   address,
	})

	// Apply MemberAdded for owner
	org.Apply(events.MemberAdded{
		BaseEvent:    eventstore.NewBaseEvent(id, events.AggregateType, org.Version()+1),
		MemberID:     ownerID,
		Email:        ownerEmail,
		PasswordHash: ownerPasswordHash,
		Name:         ownerName,
		Phone:        ownerPhone,
		Role:         values.MemberRoleOwner,
		InvitedBy:    nil,
	})

	return org
}

// NewFromEvents reconstructs Organization from events
func NewFromEvents(id uuid.UUID, evts []eventstore.Event) *Organization {
	org := &Organization{
		Base:        aggregate.NewBase(id),
		members:     make(map[uuid.UUID]*entities.Member),
		invitations: make(map[uuid.UUID]*entities.Invitation),
	}

	for _, evt := range evts {
		org.apply(evt)
		org.Replay(evt)
	}

	return org
}

// Getters
func (o *Organization) Name() string                        { return o.name }
func (o *Organization) INN() string                         { return o.inn }
func (o *Organization) LegalName() string                   { return o.legalName }
func (o *Organization) Country() values.Country             { return o.country }
func (o *Organization) Phone() string                       { return o.phone }
func (o *Organization) Email() string                       { return o.email }
func (o *Organization) Address() values.Address           { return o.address }
func (o *Organization) Status() values.OrganizationStatus { return o.status }
func (o *Organization) CreatedAt() time.Time              { return o.createdAt }
func (o *Organization) SuspendedAt() *time.Time           { return o.suspendedAt }

func (o *Organization) Members() map[uuid.UUID]*entities.Member {
	return o.members
}

func (o *Organization) Invitations() map[uuid.UUID]*entities.Invitation {
	return o.invitations
}

func (o *Organization) GetMember(id uuid.UUID) (*entities.Member, bool) {
	m, ok := o.members[id]
	return m, ok
}

func (o *Organization) GetMemberByEmail(email string) (*entities.Member, bool) {
	for _, m := range o.members {
		if m.Email() == email {
			return m, true
		}
	}
	return nil, false
}

func (o *Organization) GetInvitation(id uuid.UUID) (*entities.Invitation, bool) {
	inv, ok := o.invitations[id]
	return inv, ok
}

func (o *Organization) GetInvitationByToken(token string) (*entities.Invitation, bool) {
	for _, inv := range o.invitations {
		if inv.Token() == token {
			return inv, true
		}
	}
	return nil, false
}

// Commands

func (o *Organization) Approve(adminID uuid.UUID) error {
	if !o.status.CanBeApproved() {
		return ErrOrganizationNotPending
	}

	o.Apply(events.OrganizationApproved{
		BaseEvent:  eventstore.NewBaseEvent(o.ID(), events.AggregateType, o.Version()+1),
		ApprovedBy: adminID,
	})

	return nil
}

func (o *Organization) Reject(adminID uuid.UUID, reason string) error {
	if !o.status.CanBeRejected() {
		return ErrOrganizationNotPending
	}

	o.Apply(events.OrganizationRejected{
		BaseEvent:  eventstore.NewBaseEvent(o.ID(), events.AggregateType, o.Version()+1),
		RejectedBy: adminID,
		Reason:     reason,
	})

	return nil
}

func (o *Organization) Suspend(adminID uuid.UUID, reason string) error {
	if !o.status.CanBeSuspended() {
		return ErrOrganizationNotActive
	}

	o.Apply(events.OrganizationSuspended{
		BaseEvent:   eventstore.NewBaseEvent(o.ID(), events.AggregateType, o.Version()+1),
		SuspendedBy: adminID,
		Reason:      reason,
	})

	return nil
}

func (o *Organization) Update(actorID uuid.UUID, name, phone, email *string, address *values.Address) error {
	actor, ok := o.members[actorID]
	if !ok {
		return ErrMemberNotFound
	}
	if !actor.CanManageOrganization() {
		return ErrInsufficientPermissions
	}

	o.Apply(events.OrganizationUpdated{
		BaseEvent: eventstore.NewBaseEvent(o.ID(), events.AggregateType, o.Version()+1),
		Name:      name,
		Phone:     phone,
		Email:     email,
		Address:   address,
		UpdatedBy: actorID,
	})

	return nil
}

func (o *Organization) CreateInvitation(
	actorID uuid.UUID,
	invitationID uuid.UUID,
	email string,
	role values.MemberRole,
	token string,
	expiresAt time.Time,
	name *string,
	phone *string,
) error {
	actor, ok := o.members[actorID]
	if !ok {
		return ErrMemberNotFound
	}
	if !actor.CanManageMembers() {
		return ErrInsufficientPermissions
	}

	// Check if email already exists as member
	if _, exists := o.GetMemberByEmail(email); exists {
		return ErrMemberAlreadyExists
	}

	// Check if pending invitation exists for this email
	for _, inv := range o.invitations {
		if inv.Email() == email && inv.Status() == values.InvitationStatusPending {
			return ErrEmailAlreadyInvited
		}
	}

	o.Apply(events.InvitationCreated{
		BaseEvent:    eventstore.NewBaseEvent(o.ID(), events.AggregateType, o.Version()+1),
		InvitationID: invitationID,
		Email:        email,
		Role:         role,
		Token:        token,
		CreatedBy:    actorID,
		ExpiresAt:    expiresAt.Unix(),
		Name:         name,
		Phone:        phone,
	})

	return nil
}

func (o *Organization) CancelInvitation(actorID, invitationID uuid.UUID) error {
	actor, ok := o.members[actorID]
	if !ok {
		return ErrMemberNotFound
	}
	if !actor.CanManageMembers() {
		return ErrInsufficientPermissions
	}

	inv, ok := o.invitations[invitationID]
	if !ok {
		return ErrInvitationNotFound
	}
	if !inv.CanBeCancelled() {
		return ErrInvitationCannotBeCancelled
	}

	o.Apply(events.InvitationCancelled{
		BaseEvent:    eventstore.NewBaseEvent(o.ID(), events.AggregateType, o.Version()+1),
		InvitationID: invitationID,
		CancelledBy:  actorID,
	})

	return nil
}

func (o *Organization) AcceptInvitation(
	invitationID uuid.UUID,
	memberID uuid.UUID,
	passwordHash string,
	name *string,
	phone *string,
) error {
	inv, ok := o.invitations[invitationID]
	if !ok {
		return ErrInvitationNotFound
	}
	if !inv.CanBeAccepted() {
		if inv.IsExpired() {
			return ErrInvitationExpired
		}
		return ErrInvitationAlreadyUsed
	}

	// Используем предзаполненные данные из приглашения, если они есть
	// Иначе используем данные от пользователя
	finalName := ""
	if inv.Name() != nil {
		finalName = *inv.Name()
	} else if name != nil {
		finalName = *name
	} else {
		return ErrNameRequired
	}

	finalPhone := ""
	if inv.Phone() != nil {
		finalPhone = *inv.Phone()
	} else if phone != nil {
		finalPhone = *phone
	} else {
		return ErrPhoneRequired
	}

	o.Apply(events.MemberAdded{
		BaseEvent:    eventstore.NewBaseEvent(o.ID(), events.AggregateType, o.Version()+1),
		MemberID:     memberID,
		Email:        inv.Email(),
		PasswordHash: passwordHash,
		Name:         finalName,
		Phone:        finalPhone,
		Role:         inv.Role(),
		InvitedBy:    ptr(inv.CreatedBy()),
	})

	o.Apply(events.InvitationAccepted{
		BaseEvent:    eventstore.NewBaseEvent(o.ID(), events.AggregateType, o.Version()+1),
		InvitationID: invitationID,
		MemberID:     memberID,
	})

	return nil
}

func (o *Organization) ChangeMemberRole(actorID, memberID uuid.UUID, newRole values.MemberRole) error {
	actor, ok := o.members[actorID]
	if !ok {
		return ErrMemberNotFound
	}
	if !actor.CanManageMembers() {
		return ErrInsufficientPermissions
	}

	member, ok := o.members[memberID]
	if !ok {
		return ErrMemberNotFound
	}

	if actorID == memberID {
		return ErrCannotChangeOwnRole
	}

	if member.Role() == values.MemberRoleOwner {
		return ErrMemberCannotBeRemoved
	}

	o.Apply(events.MemberRoleChanged{
		BaseEvent: eventstore.NewBaseEvent(o.ID(), events.AggregateType, o.Version()+1),
		MemberID:  memberID,
		OldRole:   member.Role(),
		NewRole:   newRole,
		ChangedBy: actorID,
	})

	return nil
}

func (o *Organization) BlockMember(actorID, memberID uuid.UUID, reason string) error {
	actor, ok := o.members[actorID]
	if !ok {
		return ErrMemberNotFound
	}
	if !actor.CanManageMembers() {
		return ErrInsufficientPermissions
	}

	member, ok := o.members[memberID]
	if !ok {
		return ErrMemberNotFound
	}

	if actorID == memberID {
		return ErrCannotBlockSelf
	}

	if member.Role() == values.MemberRoleOwner {
		return ErrMemberCannotBeRemoved
	}

	o.Apply(events.MemberBlocked{
		BaseEvent: eventstore.NewBaseEvent(o.ID(), events.AggregateType, o.Version()+1),
		MemberID:  memberID,
		BlockedBy: actorID,
		Reason:    reason,
	})

	return nil
}

func (o *Organization) UnblockMember(actorID, memberID uuid.UUID) error {
	actor, ok := o.members[actorID]
	if !ok {
		return ErrMemberNotFound
	}
	if !actor.CanManageMembers() {
		return ErrInsufficientPermissions
	}

	if _, ok := o.members[memberID]; !ok {
		return ErrMemberNotFound
	}

	o.Apply(events.MemberUnblocked{
		BaseEvent:   eventstore.NewBaseEvent(o.ID(), events.AggregateType, o.Version()+1),
		MemberID:    memberID,
		UnblockedBy: actorID,
	})

	return nil
}

// Apply applies event and records it as change
func (o *Organization) Apply(evt eventstore.Event) {
	o.apply(evt)
	o.Base.Apply(evt)
}

// apply updates state from event (used by both Apply and Replay)
func (o *Organization) apply(evt eventstore.Event) {
	switch e := evt.(type) {
	case events.OrganizationCreated:
		o.name = e.Name
		o.inn = e.INN
		o.legalName = e.LegalName
		o.country = e.Country
		o.phone = e.Phone
		o.email = e.Email
		o.address = e.Address
		o.status = values.OrganizationStatusPending
		o.createdAt = e.OccurredAt()

	case events.OrganizationApproved:
		o.status = values.OrganizationStatusActive

	case events.OrganizationRejected:
		o.status = values.OrganizationStatusRejected

	case events.OrganizationSuspended:
		o.status = values.OrganizationStatusSuspended
		now := e.OccurredAt()
		o.suspendedAt = &now

	case events.OrganizationUpdated:
		if e.Name != nil {
			o.name = *e.Name
		}
		if e.Phone != nil {
			o.phone = *e.Phone
		}
		if e.Email != nil {
			o.email = *e.Email
		}
		if e.Address != nil {
			o.address = *e.Address
		}

	case events.MemberAdded:
		member := entities.NewMember(
			e.MemberID,
			e.Email,
			e.PasswordHash,
			e.Name,
			e.Phone,
			e.Role,
		)
		o.members[e.MemberID] = &member

	case events.MemberRoleChanged:
		if m, ok := o.members[e.MemberID]; ok {
			m.ChangeRole(e.NewRole)
		}

	case events.MemberBlocked:
		if m, ok := o.members[e.MemberID]; ok {
			m.Block()
		}

	case events.MemberUnblocked:
		if m, ok := o.members[e.MemberID]; ok {
			m.Unblock()
		}

	case events.InvitationCreated:
		inv := entities.NewInvitation(
			e.InvitationID,
			e.Email,
			e.Role,
			e.Token,
			e.CreatedBy,
			time.Unix(e.ExpiresAt, 0),
			e.Name,
			e.Phone,
		)
		o.invitations[e.InvitationID] = &inv

	case events.InvitationAccepted:
		if inv, ok := o.invitations[e.InvitationID]; ok {
			inv.Accept()
		}

	case events.InvitationExpired:
		if inv, ok := o.invitations[e.InvitationID]; ok {
			inv.Expire()
		}

	case events.InvitationCancelled:
		if inv, ok := o.invitations[e.InvitationID]; ok {
			inv.Cancel()
		}
	}
}

func ptr[T any](v T) *T {
	return &v
}
