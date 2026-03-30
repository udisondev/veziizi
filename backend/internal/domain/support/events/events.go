package events

import (
	"github.com/udisondev/veziizi/backend/internal/domain/support/entities"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"github.com/google/uuid"
)

const AggregateType = "support_ticket"

// Event type constants
const (
	TypeTicketCreated  = "support.ticket_created"
	TypeMessageAdded   = "support.message_added"
	TypeTicketClosed   = "support.ticket_closed"
	TypeTicketReopened = "support.ticket_reopened"
)

func init() {
	eventstore.RegisterEventType[TicketCreated](TypeTicketCreated)
	eventstore.RegisterEventType[MessageAdded](TypeMessageAdded)
	eventstore.RegisterEventType[TicketClosed](TypeTicketClosed)
	eventstore.RegisterEventType[TicketReopened](TypeTicketReopened)
}

// TicketCreated is emitted when a new support ticket is created
type TicketCreated struct {
	eventstore.BaseEvent
	TicketNumber   int64     `json:"ticket_number"`
	MemberID       uuid.UUID `json:"member_id"`
	OrgID          uuid.UUID `json:"org_id"`
	Subject        string    `json:"subject"`
	InitialMessage string    `json:"initial_message"`
}

func (e TicketCreated) EventType() string { return TypeTicketCreated }

// MessageAdded is emitted when a message is added to the ticket
type MessageAdded struct {
	eventstore.BaseEvent
	MessageID  uuid.UUID           `json:"message_id"`
	SenderType entities.SenderType `json:"sender_type"`
	SenderID   uuid.UUID           `json:"sender_id"`
	Content    string              `json:"content"`
}

func (e MessageAdded) EventType() string { return TypeMessageAdded }

// TicketClosed is emitted when admin closes the ticket
type TicketClosed struct {
	eventstore.BaseEvent
	ClosedByAdminID uuid.UUID `json:"closed_by_admin_id"`
	Resolution      string    `json:"resolution,omitempty"`
}

func (e TicketClosed) EventType() string { return TypeTicketClosed }

// TicketReopened is emitted when user reopens a closed ticket
type TicketReopened struct {
	eventstore.BaseEvent
	ReopenedByMemberID uuid.UUID `json:"reopened_by_member_id"`
}

func (e TicketReopened) EventType() string { return TypeTicketReopened }
