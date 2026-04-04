package aggregate

import (
	"github.com/google/uuid"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
)

// Snapshotable defines interface for aggregates that support state snapshots.
// Aggregates implementing this interface can be efficiently restored from
// a snapshot state instead of replaying all events.
type Snapshotable interface {
	// State returns the current state of the aggregate for snapshot storage.
	// The returned value must be JSON-serializable.
	State() any

	// FromSnapshot restores the aggregate from a snapshot state.
	// Returns error if the state cannot be applied.
	FromSnapshot(state any) error
}

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

// SetVersion sets the aggregate version directly.
// Used when restoring aggregate from snapshot state.
func (b *Base) SetVersion(version int64) {
	b.version = version
}

// SetID sets the aggregate ID.
// Used when restoring aggregate from snapshot state.
func (b *Base) SetID(id uuid.UUID) {
	b.id = id
}
