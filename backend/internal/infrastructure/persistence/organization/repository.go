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

	if err := r.store.Save(ctx, changes...); err != nil {
		if errors.Is(err, eventstore.ErrConcurrentModification) {
			return fmt.Errorf("organization was modified concurrently: %w", err)
		}
		return fmt.Errorf("failed to save organization events: %w", err)
	}

	org.ClearChanges()
	return nil
}

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (*organization.Organization, error) {
	evts, err := r.store.Load(ctx, id, events.AggregateType)
	if err != nil {
		if errors.Is(err, eventstore.ErrAggregateNotFound) {
			return nil, organization.ErrOrganizationNotFound
		}
		return nil, fmt.Errorf("failed to load organization events: %w", err)
	}

	return organization.NewFromEvents(id, evts), nil
}
