package organization

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/udisondev/veziizi/backend/internal/domain/organization"
	"github.com/udisondev/veziizi/backend/internal/domain/organization/events"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
)

type Repository struct {
	store eventstore.Store
}

func NewRepository(store eventstore.Store) *Repository {
	return &Repository{store: store}
}

func (r *Repository) Save(ctx context.Context, org *organization.Organization) error {
	changes := org.Changes()
	if len(changes) == 0 {
		return nil
	}

	// SaveWithState пишет снапшот каждые snapshotThreshold версий.
	if err := r.store.SaveWithState(ctx, org.State(), changes...); err != nil {
		if errors.Is(err, eventstore.ErrConcurrentModification) {
			return fmt.Errorf("organization was modified concurrently: %w", err)
		}
		return fmt.Errorf("failed to save organization events: %w", err)
	}

	org.ClearChanges()
	return nil
}

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (*organization.Organization, error) {
	res, err := r.store.LoadWithSnapshot(ctx, id, events.AggregateType)
	if err != nil {
		if errors.Is(err, eventstore.ErrAggregateNotFound) {
			return nil, organization.ErrOrganizationNotFound
		}
		return nil, fmt.Errorf("failed to load organization events: %w", err)
	}

	org, err := organization.NewFromStore(id, res.SnapshotState, res.Events)
	if err != nil {
		return nil, fmt.Errorf("restore organization: %w", err)
	}
	return org, nil
}
