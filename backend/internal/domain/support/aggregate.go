package support

import (
	"strings"
	"time"

	"github.com/udisondev/veziizi/backend/internal/domain/support/entities"
	"github.com/udisondev/veziizi/backend/internal/domain/support/events"
	"github.com/udisondev/veziizi/backend/internal/domain/support/values"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"github.com/udisondev/veziizi/backend/internal/pkg/aggregate"
	"github.com/google/uuid"
)

const (
	MaxSubjectLength  = 255
	MaxMessageLength  = 10000
)

// Ticket represents a support ticket aggregate
type Ticket struct {
	aggregate.Base

	ticketNumber int64
	memberID     uuid.UUID // Creator of the ticket
	orgID        uuid.UUID // Organization of the creator
	subject      string
	status       values.TicketStatus

	messages      map[uuid.UUID]*entities.Message
	messagesCache []*entities.Message

	createdAt time.Time
	updatedAt time.Time
	closedAt  *time.Time
}

// New creates a new Ticket with initial message
func New(
	id uuid.UUID,
	ticketNumber int64,
	memberID uuid.UUID,
	orgID uuid.UUID,
	subject string,
	initialMessage string,
) (*Ticket, error) {
	subject = strings.TrimSpace(subject)
	initialMessage = strings.TrimSpace(initialMessage)

	if subject == "" {
		return nil, ErrEmptySubject
	}
	if len(subject) > MaxSubjectLength {
		return nil, ErrSubjectTooLong
	}
	if initialMessage == "" {
		return nil, ErrEmptyMessage
	}
	if len(initialMessage) > MaxMessageLength {
		return nil, ErrMessageTooLong
	}

	t := &Ticket{
		Base:     aggregate.NewBase(id),
		messages: make(map[uuid.UUID]*entities.Message),
	}

	t.Apply(events.TicketCreated{
		BaseEvent:        eventstore.NewBaseEvent(id, events.AggregateType, t.Version()+1),
		TicketNumber:     ticketNumber,
		MemberID:         memberID,
		OrgID:            orgID,
		Subject:          subject,
		InitialMessage:   initialMessage,
		InitialMessageID: uuid.New(),
	})

	return t, nil
}

// NewFromEvents reconstructs Ticket from events
func NewFromEvents(id uuid.UUID, evts []eventstore.Event) *Ticket {
	t := &Ticket{
		Base:     aggregate.NewBase(id),
		messages: make(map[uuid.UUID]*entities.Message),
	}

	for _, evt := range evts {
		t.apply(evt)
		t.Replay(evt)
	}

	return t
}

// Getters
func (t *Ticket) TicketNumber() int64            { return t.ticketNumber }
func (t *Ticket) MemberID() uuid.UUID            { return t.memberID }
func (t *Ticket) OrgID() uuid.UUID               { return t.orgID }
func (t *Ticket) Subject() string                { return t.subject }
func (t *Ticket) Status() values.TicketStatus    { return t.status }
func (t *Ticket) MessagesList() []*entities.Message {
	if t.messagesCache == nil {
		t.messagesCache = make([]*entities.Message, 0, len(t.messages))
		for _, m := range t.messages {
			t.messagesCache = append(t.messagesCache, m)
		}
	}
	return t.messagesCache
}
func (t *Ticket) CreatedAt() time.Time           { return t.createdAt }
func (t *Ticket) UpdatedAt() time.Time           { return t.updatedAt }
func (t *Ticket) ClosedAt() *time.Time           { return t.closedAt }

// CanUserAccess checks if a member can access this ticket
func (t *Ticket) CanUserAccess(memberID uuid.UUID) bool {
	return t.memberID == memberID
}

// CanAddMessage checks if messages can be added to this ticket
func (t *Ticket) CanAddMessage() bool {
	return !t.status.IsClosed()
}

// AddUserMessage adds a message from the ticket owner
func (t *Ticket) AddUserMessage(memberID uuid.UUID, content string) error {
	if !t.CanUserAccess(memberID) {
		return ErrNotTicketOwner
	}
	return t.addMessage(entities.SenderTypeUser, memberID, content)
}

// AddAdminMessage adds a message from an admin
func (t *Ticket) AddAdminMessage(adminID uuid.UUID, content string) error {
	return t.addMessage(entities.SenderTypeAdmin, adminID, content)
}

func (t *Ticket) addMessage(senderType entities.SenderType, senderID uuid.UUID, content string) error {
	if t.status.IsClosed() {
		return ErrTicketClosed
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return ErrEmptyMessage
	}
	if len(content) > MaxMessageLength {
		return ErrMessageTooLong
	}

	t.Apply(events.MessageAdded{
		BaseEvent:  eventstore.NewBaseEvent(t.ID(), events.AggregateType, t.Version()+1),
		MessageID:  uuid.New(),
		SenderType: senderType,
		SenderID:   senderID,
		Content:    content,
	})

	return nil
}

// Close closes the ticket (admin only)
func (t *Ticket) Close(adminID uuid.UUID, resolution string) error {
	if t.status.IsClosed() {
		return ErrTicketClosed
	}

	t.Apply(events.TicketClosed{
		BaseEvent:       eventstore.NewBaseEvent(t.ID(), events.AggregateType, t.Version()+1),
		ClosedByAdminID: adminID,
		Resolution:      strings.TrimSpace(resolution),
	})

	return nil
}

// Reopen reopens a closed ticket (user only)
func (t *Ticket) Reopen(memberID uuid.UUID) error {
	if !t.CanUserAccess(memberID) {
		return ErrNotTicketOwner
	}
	if !t.status.IsClosed() {
		return ErrTicketNotClosed
	}

	t.Apply(events.TicketReopened{
		BaseEvent:          eventstore.NewBaseEvent(t.ID(), events.AggregateType, t.Version()+1),
		ReopenedByMemberID: memberID,
	})

	return nil
}

// Apply applies event and records it as change
func (t *Ticket) Apply(evt eventstore.Event) {
	t.apply(evt)
	t.Base.Apply(evt)
}

// apply updates state from event (used by both Apply and Replay)
func (t *Ticket) apply(evt eventstore.Event) {
	t.messagesCache = nil

	switch e := evt.(type) {
	case events.TicketCreated:
		t.ticketNumber = e.TicketNumber
		t.memberID = e.MemberID
		t.orgID = e.OrgID
		t.subject = e.Subject
		t.status = values.TicketStatusOpen
		t.createdAt = e.OccurredAt()
		t.updatedAt = e.OccurredAt()

		// Add initial message
		msg := entities.NewMessage(
			e.InitialMessageID,
			entities.SenderTypeUser,
			e.MemberID,
			e.InitialMessage,
			e.OccurredAt(),
		)
		t.messages[msg.ID()] = &msg

	case events.MessageAdded:
		msg := entities.NewMessage(
			e.MessageID,
			e.SenderType,
			e.SenderID,
			e.Content,
			e.OccurredAt(),
		)
		t.messages[e.MessageID] = &msg
		t.updatedAt = e.OccurredAt()

		// Update status based on sender
		if e.SenderType == entities.SenderTypeAdmin {
			t.status = values.TicketStatusAnswered
		} else {
			t.status = values.TicketStatusAwaitingReply
		}

	case events.TicketClosed:
		t.status = values.TicketStatusClosed
		now := e.OccurredAt()
		t.closedAt = &now
		t.updatedAt = e.OccurredAt()

	case events.TicketReopened:
		t.status = values.TicketStatusAwaitingReply
		t.closedAt = nil
		t.updatedAt = e.OccurredAt()
	}
}

// =====================================
// Snapshot support for efficient loading
// =====================================

// TicketSnapshot represents serializable state of Ticket aggregate
type TicketSnapshot struct {
	ID           uuid.UUID                    `json:"id"`
	Version      int64                        `json:"version"`
	TicketNumber int64                        `json:"ticket_number"`
	MemberID     uuid.UUID                    `json:"member_id"`
	OrgID        uuid.UUID                    `json:"org_id"`
	Subject      string                       `json:"subject"`
	Status       values.TicketStatus          `json:"status"`
	Messages     map[uuid.UUID]MessageSnapshot `json:"messages"`
	CreatedAt    time.Time                    `json:"created_at"`
	UpdatedAt    time.Time                    `json:"updated_at"`
	ClosedAt     *time.Time                   `json:"closed_at,omitempty"`
}

// MessageSnapshot represents serializable state of Message entity
type MessageSnapshot struct {
	ID         uuid.UUID            `json:"id"`
	SenderType entities.SenderType  `json:"sender_type"`
	SenderID   uuid.UUID            `json:"sender_id"`
	Content    string               `json:"content"`
	CreatedAt  time.Time            `json:"created_at"`
}

// State returns current aggregate state for snapshot storage.
// Implements aggregate.Snapshotable interface.
func (t *Ticket) State() any {
	messages := make(map[uuid.UUID]MessageSnapshot, len(t.messages))
	for id, m := range t.messages {
		messages[id] = MessageSnapshot{
			ID:         m.ID(),
			SenderType: m.SenderType(),
			SenderID:   m.SenderID(),
			Content:    m.Content(),
			CreatedAt:  m.CreatedAt(),
		}
	}

	return TicketSnapshot{
		ID:           t.ID(),
		Version:      t.Version(),
		TicketNumber: t.ticketNumber,
		MemberID:     t.memberID,
		OrgID:        t.orgID,
		Subject:      t.subject,
		Status:       t.status,
		Messages:     messages,
		CreatedAt:    t.createdAt,
		UpdatedAt:    t.updatedAt,
		ClosedAt:     t.closedAt,
	}
}

// FromSnapshot restores aggregate from snapshot state.
// Implements aggregate.Snapshotable interface.
func (t *Ticket) FromSnapshot(state any) error {
	snap, ok := state.(TicketSnapshot)
	if !ok {
		return ErrInvalidSnapshotType
	}

	t.Base.SetID(snap.ID)
	t.Base.SetVersion(snap.Version)

	t.ticketNumber = snap.TicketNumber
	t.memberID = snap.MemberID
	t.orgID = snap.OrgID
	t.subject = snap.Subject
	t.status = snap.Status
	t.createdAt = snap.CreatedAt
	t.updatedAt = snap.UpdatedAt
	t.closedAt = snap.ClosedAt

	t.messages = make(map[uuid.UUID]*entities.Message, len(snap.Messages))
	for id, ms := range snap.Messages {
		msg := entities.NewMessage(
			ms.ID,
			ms.SenderType,
			ms.SenderID,
			ms.Content,
			ms.CreatedAt,
		)
		t.messages[id] = &msg
	}

	return nil
}

// NewFromSnapshot creates Ticket from snapshot state.
func NewFromSnapshot(id uuid.UUID, state any) (*Ticket, error) {
	t := &Ticket{
		Base:     aggregate.NewBase(id),
		messages: make(map[uuid.UUID]*entities.Message),
	}

	if err := t.FromSnapshot(state); err != nil {
		return nil, err
	}

	return t, nil
}
