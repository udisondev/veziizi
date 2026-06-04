package admin

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/udisondev/veziizi/backend/internal/domain/organization"
	"github.com/udisondev/veziizi/backend/internal/domain/organization/events"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/messaging"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/projections"
	"github.com/udisondev/veziizi/backend/internal/pkg/dbtx"
)

type Service struct {
	db                   dbtx.TxManager
	eventStore           eventstore.Store
	publisher            *messaging.EventPublisher
	pendingOrganizations *projections.PendingOrganizationsProjection
}

func NewService(
	db dbtx.TxManager,
	eventStore eventstore.Store,
	publisher *messaging.EventPublisher,
	pendingOrganizations *projections.PendingOrganizationsProjection,
) *Service {
	return &Service{
		db:                   db,
		eventStore:           eventStore,
		publisher:            publisher,
		pendingOrganizations: pendingOrganizations,
	}
}

func (s *Service) GetOrganization(ctx context.Context, id uuid.UUID) (*organization.Organization, error) {
	res, err := s.eventStore.LoadWithSnapshot(ctx, id, events.AggregateType)
	if err != nil {
		if errors.Is(err, eventstore.ErrAggregateNotFound) {
			return nil, organization.ErrOrganizationNotFound
		}
		return nil, fmt.Errorf("failed to load organization: %w", err)
	}
	org, err := organization.NewFromStore(id, res.SnapshotState, res.Events)
	if err != nil {
		return nil, fmt.Errorf("restore organization: %w", err)
	}
	if org.Version() == 0 {
		return nil, organization.ErrOrganizationNotFound
	}
	return org, nil
}

func (s *Service) ListPendingOrganizations(ctx context.Context) ([]projections.PendingOrganization, error) {
	return s.pendingOrganizations.List(ctx)
}

type ApproveInput struct {
	AdminID        uuid.UUID
	OrganizationID uuid.UUID
}

func (s *Service) Approve(ctx context.Context, input ApproveInput) error {
	org, err := s.GetOrganization(ctx, input.OrganizationID)
	if err != nil {
		return err
	}

	if err := org.Approve(input.AdminID); err != nil {
		return err
	}

	return s.saveAndPublish(ctx, org)
}

type RejectInput struct {
	AdminID        uuid.UUID
	OrganizationID uuid.UUID
	Reason         string
}

func (s *Service) Reject(ctx context.Context, input RejectInput) error {
	org, err := s.GetOrganization(ctx, input.OrganizationID)
	if err != nil {
		return err
	}

	if err := org.Reject(input.AdminID, input.Reason); err != nil {
		return err
	}

	return s.saveAndPublish(ctx, org)
}

func (s *Service) saveAndPublish(ctx context.Context, org *organization.Organization) error {
	changes := org.Changes()
	if len(changes) == 0 {
		return nil
	}

	return s.db.InTx(ctx, func(ctx context.Context) error {
		// SaveWithState пишет снапшот каждые snapshotThreshold версий.
		if err := s.eventStore.SaveWithState(ctx, org.State(), changes...); err != nil {
			return fmt.Errorf("failed to save events: %w", err)
		}

		if err := s.publisher.Publish(ctx, changes...); err != nil {
			return fmt.Errorf("failed to publish events: %w", err)
		}

		org.ClearChanges()
		return nil
	})
}

// MarkFraudsterInput contains data for marking organization as fraudster
type MarkFraudsterInput struct {
	AdminID        uuid.UUID
	OrganizationID uuid.UUID
	IsConfirmed    bool
	Reason         string
}

// MarkFraudster marks organization as fraudster
func (s *Service) MarkFraudster(ctx context.Context, input MarkFraudsterInput) error {
	org, err := s.GetOrganization(ctx, input.OrganizationID)
	if err != nil {
		return err
	}

	if err := org.MarkAsFraudster(input.AdminID, input.IsConfirmed, input.Reason); err != nil {
		return err
	}

	return s.saveAndPublish(ctx, org)
}

// UnmarkFraudsterInput contains data for unmarking organization as fraudster
type UnmarkFraudsterInput struct {
	AdminID        uuid.UUID
	OrganizationID uuid.UUID
	Reason         string
}

// UnmarkFraudster removes fraudster status from organization
func (s *Service) UnmarkFraudster(ctx context.Context, input UnmarkFraudsterInput) error {
	org, err := s.GetOrganization(ctx, input.OrganizationID)
	if err != nil {
		return err
	}

	if err := org.UnmarkFraudster(input.AdminID, input.Reason); err != nil {
		return err
	}

	return s.saveAndPublish(ctx, org)
}
