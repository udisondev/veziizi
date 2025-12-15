package entities

import (
	"time"

	"github.com/google/uuid"
)

type Message struct {
	id             uuid.UUID
	senderOrgID    uuid.UUID
	senderMemberID uuid.UUID
	content        string
	createdAt      time.Time
}

func NewMessage(
	id uuid.UUID,
	senderOrgID uuid.UUID,
	senderMemberID uuid.UUID,
	content string,
	createdAt time.Time,
) Message {
	return Message{
		id:             id,
		senderOrgID:    senderOrgID,
		senderMemberID: senderMemberID,
		content:        content,
		createdAt:      createdAt,
	}
}

func (m Message) ID() uuid.UUID           { return m.id }
func (m Message) SenderOrgID() uuid.UUID  { return m.senderOrgID }
func (m Message) SenderMemberID() uuid.UUID { return m.senderMemberID }
func (m Message) Content() string         { return m.content }
func (m Message) CreatedAt() time.Time    { return m.createdAt }
