package aggregate

import (
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"github.com/google/uuid"
)

type Base struct {
	id      uuid.UUID
	version int64
	changes []eventstore.Event
}

func NewBase(id uuid.UUID) Base {
	return Base{
		id:      id,
		version: 0,
		changes: make([]eventstore.Event, 0),
	}
}

func (b *Base) ID() uuid.UUID {
	return b.id
}

func (b *Base) Version() int64 {
	return b.version
}

func (b *Base) Changes() []eventstore.Event {
	return b.changes
}

func (b *Base) ClearChanges() {
	b.changes = make([]eventstore.Event, 0)
}

// Apply records new event and updates version
func (b *Base) Apply(event eventstore.Event) {
	b.version = event.Version()
	b.changes = append(b.changes, event)
}

// Replay applies event from history (version updates, but no changes recorded)
func (b *Base) Replay(event eventstore.Event) {
	b.version = event.Version()
}
