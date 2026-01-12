package entities

import (
	"time"

	"codeberg.org/udison/veziizi/backend/internal/domain/organization/values"
	"github.com/google/uuid"
)

type Member struct {
	id         uuid.UUID
	email      string
	name       string
	phone      string
	telegramID *int64
	role       values.MemberRole
	status     values.MemberStatus
	createdAt  time.Time
	// NOTE: passwordHash removed from domain entity (SEC-007)
	// Password hash stored only in members_lookup projection
}

func NewMember(
	id uuid.UUID,
	email string,
	name string,
	phone string,
	role values.MemberRole,
) Member {
	return Member{
		id:        id,
		email:     email,
		name:      name,
		phone:     phone,
		role:      role,
		status:    values.MemberStatusActive,
		createdAt: time.Now().UTC(),
	}
}

func (m Member) ID() uuid.UUID   { return m.id }
func (m Member) Email() string   { return m.email }
func (m Member) Name() string    { return m.name }
func (m Member) Phone() string              { return m.phone }
func (m Member) TelegramID() *int64         { return m.telegramID }
func (m Member) Role() values.MemberRole    { return m.role }
func (m Member) Status() values.MemberStatus { return m.status }
func (m Member) CreatedAt() time.Time       { return m.createdAt }

func (m Member) IsActive() bool {
	return m.status.IsActive()
}

func (m Member) CanManageMembers() bool {
	return m.IsActive() && m.role.CanManageMembers()
}

func (m Member) CanManageOrganization() bool {
	return m.IsActive() && m.role.CanManageOrganization()
}

func (m Member) CanBeRemoved() bool {
	return m.role.CanBeRemoved()
}

func (m *Member) ChangeRole(newRole values.MemberRole) {
	m.role = newRole
}

func (m *Member) Block() {
	m.status = values.MemberStatusBlocked
}

func (m *Member) Unblock() {
	m.status = values.MemberStatusActive
}

func (m *Member) SetTelegramID(telegramID int64) {
	m.telegramID = &telegramID
}

func (m *Member) UpdateInfo(name, email, phone string) {
	m.name = name
	m.email = email
	m.phone = phone
}

// RestoreMember creates Member from stored data (for event replay)
func RestoreMember(
	id uuid.UUID,
	email string,
	name string,
	phone string,
	telegramID *int64,
	role values.MemberRole,
	status values.MemberStatus,
	createdAt time.Time,
) Member {
	return Member{
		id:         id,
		email:      email,
		name:       name,
		phone:      phone,
		telegramID: telegramID,
		role:       role,
		status:     status,
		createdAt:  createdAt,
	}
}
