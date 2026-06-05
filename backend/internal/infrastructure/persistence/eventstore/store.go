package eventstore

import (
	"context"

	"github.com/google/uuid"
)

type Store interface {
	// Save persists events for aggregates.
	// Handles optimistic locking via UNIQUE constraint on (aggregate_id, version).
	// Returns ErrConcurrentModification if version conflict detected.
	// NOTE: снапшоты НЕ создаются — для агрегатов с aggregate.Snapshotable
	// (все текущие) используй SaveWithState.
	Save(ctx context.Context, events ...Event) error

	// SaveWithState persists events and upserts a state snapshot every
	// snapshotThreshold versions. state — результат aggregate.State().
	SaveWithState(ctx context.Context, state any, events ...Event) error

	// Load retrieves events for an aggregate AFTER its snapshot version (if a
	// snapshot exists). НЕ строй агрегат по результату через NewFromEvents —
	// используй LoadWithSnapshot + <domain>.NewFromStore, иначе при наличии
	// снапшота состояние соберётся битым (события до снапшота не вернутся).
	// Returns ErrAggregateNotFound if no events exist.
	Load(ctx context.Context, aggregateID uuid.UUID, aggregateType string) ([]Event, error)

	// LoadWithSnapshot retrieves the snapshot state (raw JSON, nil if none)
	// and events after it. Пара к <domain>.NewFromStore.
	LoadWithSnapshot(ctx context.Context, aggregateID uuid.UUID, aggregateType string) (*LoadResult, error)

	// LoadByIDs retrieves events for multiple aggregates in a single batch.
	// Снапшоты не используются — возвращаются ВСЕ события агрегатов, поэтому
	// NewFromEvents по результату корректен.
	// Returns map[aggregateID][]Event. Missing aggregates are not included in the result.
	LoadByIDs(ctx context.Context, aggregateIDs []uuid.UUID, aggregateType string) (map[uuid.UUID][]Event, error)

	// LoadPaginated retrieves events for an aggregate with pagination.
	// Returns events in descending order (newest first).
	// Returns events, total count, and error.
	LoadPaginated(ctx context.Context, aggregateID uuid.UUID, aggregateType string, limit, offset int) ([]EventEnvelope, int, error)
}
