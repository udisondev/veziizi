package organization

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/udisondev/veziizi/backend/internal/domain/organization/entities"
	"github.com/udisondev/veziizi/backend/internal/domain/organization/events"
	"github.com/udisondev/veziizi/backend/internal/domain/organization/values"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"github.com/udisondev/veziizi/backend/internal/pkg/aggregate"
)

type Organization struct {
	aggregate.Base

	name        string
	inn         string
	legalName   string
	country     values.Country
	phone       string
	email       string
	address     values.Address
	status      values.OrganizationStatus
	createdAt   time.Time
	suspendedAt *time.Time

	// Fraudster status
	isConfirmedFraudster bool
	isSuspectedFraudster bool
	fraudsterMarkedAt    *time.Time
	fraudsterMarkedBy    *uuid.UUID
	fraudsterReason      string

	members     map[uuid.UUID]*entities.Member
	invitations map[uuid.UUID]*entities.Invitation

	membersCache     []entities.Member
	invitationsCache []entities.Invitation
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
	// Registration metadata for fraud detection
	registrationIP string,
	registrationFingerprint string,
	registrationUserAgent string,
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
		BaseEvent:               eventstore.NewBaseEvent(id, events.AggregateType, org.Version()+1),
		MemberID:                ownerID,
		Email:                   ownerEmail,
		PasswordHash:            ownerPasswordHash,
		Name:                    ownerName,
		Phone:                   ownerPhone,
		Role:                    values.MemberRoleOwner,
		InvitedBy:               nil,
		RegistrationIP:          registrationIP,
		RegistrationFingerprint: registrationFingerprint,
		RegistrationUserAgent:   registrationUserAgent,
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
func (o *Organization) Name() string                      { return o.name }
func (o *Organization) INN() string                       { return o.inn }
func (o *Organization) LegalName() string                 { return o.legalName }
func (o *Organization) Country() values.Country           { return o.country }
func (o *Organization) Phone() string                     { return o.phone }
func (o *Organization) Email() string                     { return o.email }
func (o *Organization) Address() values.Address           { return o.address }
func (o *Organization) Status() values.OrganizationStatus { return o.status }
func (o *Organization) CreatedAt() time.Time              { return o.createdAt }
func (o *Organization) SuspendedAt() *time.Time           { return o.suspendedAt }

// Fraudster getters
func (o *Organization) IsConfirmedFraudster() bool    { return o.isConfirmedFraudster }
func (o *Organization) IsSuspectedFraudster() bool    { return o.isSuspectedFraudster }
func (o *Organization) IsFraudster() bool             { return o.isConfirmedFraudster || o.isSuspectedFraudster }
func (o *Organization) FraudsterMarkedAt() *time.Time { return o.fraudsterMarkedAt }
func (o *Organization) FraudsterMarkedBy() *uuid.UUID { return o.fraudsterMarkedBy }
func (o *Organization) FraudsterReason() string       { return o.fraudsterReason }

func (o *Organization) MembersList() []entities.Member {
	if o.membersCache == nil {
		o.membersCache = make([]entities.Member, 0, len(o.members))
		for _, m := range o.members {
			o.membersCache = append(o.membersCache, *m)
		}
	}
	return o.membersCache
}

func (o *Organization) InvitationsList() []entities.Invitation {
	if o.invitationsCache == nil {
		o.invitationsCache = make([]entities.Invitation, 0, len(o.invitations))
		for _, inv := range o.invitations {
			o.invitationsCache = append(o.invitationsCache, *inv)
		}
	}
	return o.invitationsCache
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
		return fmt.Errorf("approve organization %s: %w", o.ID(), ErrOrganizationNotPending)
	}

	o.Apply(events.OrganizationApproved{
		BaseEvent:  eventstore.NewBaseEvent(o.ID(), events.AggregateType, o.Version()+1),
		ApprovedBy: adminID,
	})

	return nil
}

func (o *Organization) Reject(adminID uuid.UUID, reason string) error {
	if !o.status.CanBeRejected() {
		return fmt.Errorf("reject organization %s: %w", o.ID(), ErrOrganizationNotPending)
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
		return fmt.Errorf("suspend organization %s: %w", o.ID(), ErrOrganizationNotActive)
	}

	o.Apply(events.OrganizationSuspended{
		BaseEvent:   eventstore.NewBaseEvent(o.ID(), events.AggregateType, o.Version()+1),
		SuspendedBy: adminID,
		Reason:      reason,
	})

	return nil
}

func (o *Organization) Update(actorID uuid.UUID, name, phone, email *string, address *values.Address) error {
	if o.status != values.OrganizationStatusActive {
		return fmt.Errorf("update organization %s: %w", o.ID(), ErrOrganizationNotActive)
	}
	actor, ok := o.members[actorID]
	if !ok {
		return fmt.Errorf("update organization %s by member %s: %w", o.ID(), actorID, ErrMemberNotFound)
	}
	if !actor.CanManageOrganization() {
		return fmt.Errorf("update organization %s by member %s: %w", o.ID(), actorID, ErrInsufficientPermissions)
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
	if o.status != values.OrganizationStatusActive {
		return fmt.Errorf("create invitation in org %s: %w", o.ID(), ErrOrganizationNotActive)
	}
	actor, ok := o.members[actorID]
	if !ok {
		return fmt.Errorf("create invitation in org %s by member %s: %w", o.ID(), actorID, ErrMemberNotFound)
	}
	if !actor.CanManageMembers() {
		return fmt.Errorf("create invitation in org %s by member %s: %w", o.ID(), actorID, ErrInsufficientPermissions)
	}

	// Check if email already exists as member
	if _, exists := o.GetMemberByEmail(email); exists {
		return fmt.Errorf("create invitation in org %s for email %s: %w", o.ID(), email, ErrMemberAlreadyExists)
	}

	// Check if pending invitation exists for this email
	for _, inv := range o.invitations {
		if inv.Email() == email && inv.Status() == values.InvitationStatusPending {
			return fmt.Errorf("create invitation in org %s for email %s: %w", o.ID(), email, ErrEmailAlreadyInvited)
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
		return fmt.Errorf("cancel invitation in org %s by member %s: %w", o.ID(), actorID, ErrMemberNotFound)
	}
	if !actor.CanManageMembers() {
		return fmt.Errorf("cancel invitation in org %s by member %s: %w", o.ID(), actorID, ErrInsufficientPermissions)
	}

	inv, ok := o.invitations[invitationID]
	if !ok {
		return fmt.Errorf("cancel invitation %s in org %s: %w", invitationID, o.ID(), ErrInvitationNotFound)
	}
	if !inv.CanBeCancelled() {
		return fmt.Errorf("cancel invitation %s in org %s: %w", invitationID, o.ID(), ErrInvitationCannotBeCancelled)
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
	registrationIP string,
	registrationFingerprint string,
	registrationUserAgent string,
) error {
	if o.status != values.OrganizationStatusActive {
		return fmt.Errorf("accept invitation in org %s: %w", o.ID(), ErrOrganizationNotActive)
	}

	inv, ok := o.invitations[invitationID]
	if !ok {
		return ErrInvitationNotFound
	}
	if !inv.CanBeAccepted() {
		if inv.IsExpired() {
			return ErrInvitationExpired
		}
		if inv.IsCancelled() {
			return ErrInvitationCancelled
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
		BaseEvent:               eventstore.NewBaseEvent(o.ID(), events.AggregateType, o.Version()+1),
		MemberID:                memberID,
		Email:                   inv.Email(),
		PasswordHash:            passwordHash,
		Name:                    finalName,
		Phone:                   finalPhone,
		Role:                    inv.Role(),
		InvitedBy:               ptr(inv.CreatedBy()),
		RegistrationIP:          registrationIP,
		RegistrationFingerprint: registrationFingerprint,
		RegistrationUserAgent:   registrationUserAgent,
	})

	o.Apply(events.InvitationAccepted{
		BaseEvent:    eventstore.NewBaseEvent(o.ID(), events.AggregateType, o.Version()+1),
		InvitationID: invitationID,
		MemberID:     memberID,
	})

	return nil
}

func (o *Organization) ChangeMemberRole(actorID, memberID uuid.UUID, newRole values.MemberRole) error {
	if o.status != values.OrganizationStatusActive {
		return fmt.Errorf("change member role in org %s: %w", o.ID(), ErrOrganizationNotActive)
	}
	actor, ok := o.members[actorID]
	if !ok {
		return fmt.Errorf("change role in org %s by member %s: %w", o.ID(), actorID, ErrMemberNotFound)
	}
	if !actor.CanManageMembers() {
		return fmt.Errorf("change role in org %s by member %s: %w", o.ID(), actorID, ErrInsufficientPermissions)
	}

	member, ok := o.members[memberID]
	if !ok {
		return fmt.Errorf("change role for member %s in org %s: %w", memberID, o.ID(), ErrMemberNotFound)
	}

	if actorID == memberID {
		return fmt.Errorf("change role in org %s: %w", o.ID(), ErrCannotChangeOwnRole)
	}

	if member.Role() == values.MemberRoleOwner {
		return fmt.Errorf("change role for member %s in org %s: %w", memberID, o.ID(), ErrMemberCannotBeRemoved)
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
	if o.status != values.OrganizationStatusActive {
		return fmt.Errorf("block member in org %s: %w", o.ID(), ErrOrganizationNotActive)
	}
	actor, ok := o.members[actorID]
	if !ok {
		return fmt.Errorf("block member in org %s by member %s: %w", o.ID(), actorID, ErrMemberNotFound)
	}
	if !actor.CanManageMembers() {
		return fmt.Errorf("block member in org %s by member %s: %w", o.ID(), actorID, ErrInsufficientPermissions)
	}

	member, ok := o.members[memberID]
	if !ok {
		return fmt.Errorf("block member %s in org %s: %w", memberID, o.ID(), ErrMemberNotFound)
	}

	if actorID == memberID {
		return fmt.Errorf("block member in org %s: %w", o.ID(), ErrCannotBlockSelf)
	}

	if member.Role() == values.MemberRoleOwner {
		return fmt.Errorf("block member %s in org %s: %w", memberID, o.ID(), ErrMemberCannotBeRemoved)
	}

	if !member.IsActive() {
		return nil // already blocked, no-op
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
	if o.status != values.OrganizationStatusActive {
		return fmt.Errorf("unblock member in org %s: %w", o.ID(), ErrOrganizationNotActive)
	}
	actor, ok := o.members[actorID]
	if !ok {
		return fmt.Errorf("unblock member in org %s by member %s: %w", o.ID(), actorID, ErrMemberNotFound)
	}
	if !actor.CanManageMembers() {
		return fmt.Errorf("unblock member in org %s by member %s: %w", o.ID(), actorID, ErrInsufficientPermissions)
	}

	member, ok := o.members[memberID]
	if !ok {
		return fmt.Errorf("unblock member %s in org %s: %w", memberID, o.ID(), ErrMemberNotFound)
	}

	if member.IsActive() {
		return nil // already active, no-op
	}

	o.Apply(events.MemberUnblocked{
		BaseEvent:   eventstore.NewBaseEvent(o.ID(), events.AggregateType, o.Version()+1),
		MemberID:    memberID,
		UnblockedBy: actorID,
	})

	return nil
}

// UpdateMemberInfo updates member's name, email and phone (partial update)
// Rules:
// - Owner/admin can edit any non-owner member
// - Owner can edit only themselves among owners
// - Cannot edit blocked members
// - nil values mean "don't change"
func (o *Organization) UpdateMemberInfo(actorID, memberID uuid.UUID, name, email, phone *string) error {
	if o.status != values.OrganizationStatusActive {
		return fmt.Errorf("update member info in org %s: %w", o.ID(), ErrOrganizationNotActive)
	}

	actor, ok := o.members[actorID]
	if !ok {
		return ErrMemberNotFound
	}

	member, ok := o.members[memberID]
	if !ok {
		return ErrMemberNotFound
	}

	// Cannot edit blocked members
	if !member.IsActive() {
		return ErrMemberNotActive
	}

	// Owner can only be edited by themselves
	if member.Role() == values.MemberRoleOwner {
		if actorID != memberID {
			return ErrCannotEditOwner
		}
	} else {
		// For non-owner members, actor must have CanManageMembers permission
		// OR be editing themselves
		if actorID != memberID && !actor.CanManageMembers() {
			return ErrInsufficientPermissions
		}
	}

	// Resolve final values - use current if nil
	newName := member.Name()
	if name != nil {
		newName = *name
	}
	newEmail := member.Email()
	if email != nil {
		newEmail = *email
		// Проверяем уникальность email внутри организации
		if existingMember, exists := o.GetMemberByEmail(newEmail); exists && existingMember.ID() != memberID {
			return fmt.Errorf("update member info in org %s: %w", o.ID(), ErrMemberAlreadyExists)
		}
	}
	newPhone := member.Phone()
	if phone != nil {
		newPhone = *phone
	}

	// Don't apply if nothing changed
	if member.Name() == newName && member.Email() == newEmail && member.Phone() == newPhone {
		return nil
	}

	o.Apply(events.MemberInfoUpdated{
		BaseEvent: eventstore.NewBaseEvent(o.ID(), events.AggregateType, o.Version()+1),
		MemberID:  memberID,
		OldName:   member.Name(),
		NewName:   newName,
		OldEmail:  member.Email(),
		NewEmail:  newEmail,
		OldPhone:  member.Phone(),
		NewPhone:  newPhone,
		UpdatedBy: actorID,
	})

	return nil
}

// RemoveMember removes member from organization (dev only, no permission checks)
func (o *Organization) RemoveMember(memberID uuid.UUID) error {
	member, ok := o.members[memberID]
	if !ok {
		return ErrMemberNotFound
	}

	if member.Role() == values.MemberRoleOwner {
		return ErrMemberCannotBeRemoved
	}

	o.Apply(events.MemberRemoved{
		BaseEvent: eventstore.NewBaseEvent(o.ID(), events.AggregateType, o.Version()+1),
		MemberID:  memberID,
	})

	return nil
}

// AddMemberDirect adds member directly without invitation (for seeding/dev)
func (o *Organization) AddMemberDirect(memberID uuid.UUID, email, passwordHash, name, phone string, role values.MemberRole) error {
	if _, exists := o.GetMemberByEmail(email); exists {
		return ErrMemberAlreadyExists
	}

	o.Apply(events.MemberAdded{
		BaseEvent:    eventstore.NewBaseEvent(o.ID(), events.AggregateType, o.Version()+1),
		MemberID:     memberID,
		Email:        email,
		PasswordHash: passwordHash,
		Name:         name,
		Phone:        phone,
		Role:         role,
		InvitedBy:    nil,
	})

	return nil
}

// MarkAsFraudster marks organization as fraudster (admin action)
func (o *Organization) MarkAsFraudster(adminID uuid.UUID, isConfirmed bool, reason string) error {
	if o.IsFraudster() {
		return ErrAlreadyFraudster
	}

	o.Apply(events.FraudsterMarked{
		BaseEvent:   eventstore.NewBaseEvent(o.ID(), events.AggregateType, o.Version()+1),
		MarkedBy:    adminID,
		IsConfirmed: isConfirmed,
		Reason:      reason,
	})

	return nil
}

// UnmarkFraudster removes fraudster status (admin action)
func (o *Organization) UnmarkFraudster(adminID uuid.UUID, reason string) error {
	if !o.IsFraudster() {
		return ErrNotFraudster
	}

	o.Apply(events.FraudsterUnmarked{
		BaseEvent:  eventstore.NewBaseEvent(o.ID(), events.AggregateType, o.Version()+1),
		UnmarkedBy: adminID,
		Reason:     reason,
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
	o.membersCache = nil
	o.invitationsCache = nil

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
			e.Name,
			e.Phone,
			e.Role,
			e.OccurredAt(),
		)
		o.members[e.MemberID] = &member

	case events.MemberRemoved:
		delete(o.members, e.MemberID)

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

	case events.MemberInfoUpdated:
		if m, ok := o.members[e.MemberID]; ok {
			m.UpdateInfo(e.NewName, e.NewEmail, e.NewPhone)
		}

	case events.InvitationCreated:
		inv := entities.NewInvitation(
			e.InvitationID,
			e.Email,
			e.Role,
			e.Token,
			e.CreatedBy,
			e.OccurredAt(),
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

	case events.FraudsterMarked:
		if e.IsConfirmed {
			o.isConfirmedFraudster = true
			o.isSuspectedFraudster = false
		} else {
			o.isSuspectedFraudster = true
			o.isConfirmedFraudster = false
		}
		now := e.OccurredAt()
		o.fraudsterMarkedAt = &now
		o.fraudsterMarkedBy = &e.MarkedBy
		o.fraudsterReason = e.Reason

	case events.FraudsterUnmarked:
		o.isConfirmedFraudster = false
		o.isSuspectedFraudster = false
		o.fraudsterMarkedAt = nil
		o.fraudsterMarkedBy = nil
		o.fraudsterReason = ""
	}
}

func ptr[T any](v T) *T {
	return &v
}

// =====================================
// Snapshot support for efficient loading
// =====================================

// OrganizationSnapshot represents serializable state of Organization aggregate
type OrganizationSnapshot struct {
	ID                   uuid.UUID                        `json:"id"`
	Version              int64                            `json:"version"`
	Name                 string                           `json:"name"`
	INN                  string                           `json:"inn"`
	LegalName            string                           `json:"legal_name"`
	Country              values.Country                   `json:"country"`
	Phone                string                           `json:"phone"`
	Email                string                           `json:"email"`
	Address              values.Address                   `json:"address"`
	Status               values.OrganizationStatus        `json:"status"`
	CreatedAt            time.Time                        `json:"created_at"`
	SuspendedAt          *time.Time                       `json:"suspended_at,omitempty"`
	IsConfirmedFraudster bool                             `json:"is_confirmed_fraudster"`
	IsSuspectedFraudster bool                             `json:"is_suspected_fraudster"`
	FraudsterMarkedAt    *time.Time                       `json:"fraudster_marked_at,omitempty"`
	FraudsterMarkedBy    *uuid.UUID                       `json:"fraudster_marked_by,omitempty"`
	FraudsterReason      string                           `json:"fraudster_reason,omitempty"`
	Members              map[uuid.UUID]MemberSnapshot     `json:"members"`
	Invitations          map[uuid.UUID]InvitationSnapshot `json:"invitations"`
}

// MemberSnapshot represents serializable state of Member entity
type MemberSnapshot struct {
	ID         uuid.UUID           `json:"id"`
	Email      string              `json:"email"`
	Name       string              `json:"name"`
	Phone      string              `json:"phone"`
	TelegramID *int64              `json:"telegram_id,omitempty"`
	Role       values.MemberRole   `json:"role"`
	Status     values.MemberStatus `json:"status"`
	CreatedAt  time.Time           `json:"created_at"`
}

// InvitationSnapshot represents serializable state of Invitation entity
type InvitationSnapshot struct {
	ID        uuid.UUID               `json:"id"`
	Email     string                  `json:"email"`
	Role      values.MemberRole       `json:"role"`
	Token     string                  `json:"token"`
	Status    values.InvitationStatus `json:"status"`
	CreatedBy uuid.UUID               `json:"created_by"`
	CreatedAt time.Time               `json:"created_at"`
	ExpiresAt time.Time               `json:"expires_at"`
	Name      *string                 `json:"name,omitempty"`
	Phone     *string                 `json:"phone,omitempty"`
}

// State returns current aggregate state for snapshot storage.
// Implements aggregate.Snapshotable interface.
func (o *Organization) State() any {
	members := make(map[uuid.UUID]MemberSnapshot, len(o.members))
	for id, m := range o.members {
		members[id] = MemberSnapshot{
			ID:         m.ID(),
			Email:      m.Email(),
			Name:       m.Name(),
			Phone:      m.Phone(),
			TelegramID: m.TelegramID(),
			Role:       m.Role(),
			Status:     m.Status(),
			CreatedAt:  m.CreatedAt(),
		}
	}

	invitations := make(map[uuid.UUID]InvitationSnapshot, len(o.invitations))
	for id, inv := range o.invitations {
		invitations[id] = InvitationSnapshot{
			ID:        inv.ID(),
			Email:     inv.Email(),
			Role:      inv.Role(),
			Token:     inv.Token(),
			Status:    inv.Status(),
			CreatedBy: inv.CreatedBy(),
			CreatedAt: inv.CreatedAt(),
			ExpiresAt: inv.ExpiresAt(),
			Name:      inv.Name(),
			Phone:     inv.Phone(),
		}
	}

	return OrganizationSnapshot{
		ID:                   o.ID(),
		Version:              o.Version(),
		Name:                 o.name,
		INN:                  o.inn,
		LegalName:            o.legalName,
		Country:              o.country,
		Phone:                o.phone,
		Email:                o.email,
		Address:              o.address,
		Status:               o.status,
		CreatedAt:            o.createdAt,
		SuspendedAt:          o.suspendedAt,
		IsConfirmedFraudster: o.isConfirmedFraudster,
		IsSuspectedFraudster: o.isSuspectedFraudster,
		FraudsterMarkedAt:    o.fraudsterMarkedAt,
		FraudsterMarkedBy:    o.fraudsterMarkedBy,
		FraudsterReason:      o.fraudsterReason,
		Members:              members,
		Invitations:          invitations,
	}
}

// FromSnapshot restores aggregate from snapshot state.
// Implements aggregate.Snapshotable interface.
func (o *Organization) FromSnapshot(state any) error {
	snap, ok := state.(OrganizationSnapshot)
	if !ok {
		return fmt.Errorf("invalid snapshot type: expected OrganizationSnapshot, got %T", state)
	}

	// Restore base aggregate state
	o.Base.SetID(snap.ID)
	o.Base.SetVersion(snap.Version)

	o.name = snap.Name
	o.inn = snap.INN
	o.legalName = snap.LegalName
	o.country = snap.Country
	o.phone = snap.Phone
	o.email = snap.Email
	o.address = snap.Address
	o.status = snap.Status
	o.createdAt = snap.CreatedAt
	o.suspendedAt = snap.SuspendedAt
	o.isConfirmedFraudster = snap.IsConfirmedFraudster
	o.isSuspectedFraudster = snap.IsSuspectedFraudster
	o.fraudsterMarkedAt = snap.FraudsterMarkedAt
	o.fraudsterMarkedBy = snap.FraudsterMarkedBy
	o.fraudsterReason = snap.FraudsterReason

	o.members = make(map[uuid.UUID]*entities.Member, len(snap.Members))
	for id, ms := range snap.Members {
		m := entities.RestoreMember(
			ms.ID,
			ms.Email,
			ms.Name,
			ms.Phone,
			ms.TelegramID,
			ms.Role,
			ms.Status,
			ms.CreatedAt,
		)
		o.members[id] = &m
	}

	o.invitations = make(map[uuid.UUID]*entities.Invitation, len(snap.Invitations))
	for id, is := range snap.Invitations {
		inv := entities.RestoreInvitation(
			is.ID,
			is.Email,
			is.Role,
			is.Token,
			is.Status,
			is.CreatedBy,
			is.CreatedAt,
			is.ExpiresAt,
			is.Name,
			is.Phone,
		)
		o.invitations[id] = &inv
	}

	return nil
}

// NewFromSnapshot creates Organization from snapshot state.
// Used by event store when loading aggregate with snapshot.
func NewFromSnapshot(id uuid.UUID, state any) (*Organization, error) {
	org := &Organization{
		Base:        aggregate.NewBase(id),
		members:     make(map[uuid.UUID]*entities.Member),
		invitations: make(map[uuid.UUID]*entities.Invitation),
	}

	if err := org.FromSnapshot(state); err != nil {
		return nil, fmt.Errorf("restore from snapshot: %w", err)
	}

	return org, nil
}
