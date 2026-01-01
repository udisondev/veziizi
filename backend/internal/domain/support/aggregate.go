package support

import (
	"strings"
	"time"

	"codeberg.org/udison/veziizi/backend/internal/domain/support/entities"
	"codeberg.org/udison/veziizi/backend/internal/domain/support/events"
	"codeberg.org/udison/veziizi/backend/internal/domain/support/values"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"codeberg.org/udison/veziizi/backend/internal/pkg/aggregate"
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

	messages map[uuid.UUID]*entities.Message

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
		BaseEvent:      eventstore.NewBaseEvent(id, events.AggregateType, t.Version()+1),
		TicketNumber:   ticketNumber,
		MemberID:       memberID,
		OrgID:          orgID,
		Subject:        subject,
		InitialMessage: initialMessage,
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
func (t *Ticket) Messages() map[uuid.UUID]*entities.Message { return t.messages }
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
			uuid.New(),
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
