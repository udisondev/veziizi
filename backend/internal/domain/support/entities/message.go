package entities

import (
	"time"

	"github.com/google/uuid"
)

// SenderType represents who sent the message
type SenderType string

const (
	SenderTypeUser  SenderType = "user"
	SenderTypeAdmin SenderType = "admin"
)

// Message represents a message in a support ticket conversation
type Message struct {
	id         uuid.UUID
	senderType SenderType
	senderID   uuid.UUID // member_id for user, admin_id for admin
	content    string
	createdAt  time.Time
}

// NewMessage creates a new Message entity
func NewMessage(
	id uuid.UUID,
	senderType SenderType,
	senderID uuid.UUID,
	content string,
	createdAt time.Time,
) Message {
	return Message{
		id:         id,
		senderType: senderType,
		senderID:   senderID,
		content:    content,
		createdAt:  createdAt,
	}
}

// Getters
func (m Message) ID() uuid.UUID          { return m.id }
func (m Message) SenderType() SenderType { return m.senderType }
func (m Message) SenderID() uuid.UUID    { return m.senderID }
func (m Message) Content() string        { return m.content }
func (m Message) CreatedAt() time.Time   { return m.createdAt }
