package eventstore

import (
	"context"

	"github.com/google/uuid"
)

type Store interface {
	// Save persists events for aggregates.
	// Handles optimistic locking via UNIQUE constraint on (aggregate_id, version).
	// Creates snapshots automatically when threshold is reached.
	// Returns ErrConcurrentModification if version conflict detected.
	Save(ctx context.Context, events ...Event) error

	// Load retrieves events for an aggregate (snapshot + subsequent events).
	// Returns ErrAggregateNotFound if no events exist.
	Load(ctx context.Context, aggregateID uuid.UUID, aggregateType string) ([]Event, error)
}
