package organization

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"codeberg.org/udison/veziizi/backend/internal/domain/organization"
	"codeberg.org/udison/veziizi/backend/internal/domain/organization/events"
	"codeberg.org/udison/veziizi/backend/internal/domain/organization/values"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/messaging"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/projections"
	"codeberg.org/udison/veziizi/backend/internal/pkg/dbtx"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const invitationTTL = 7 * 24 * time.Hour // 7 days

type Service struct {
	db          dbtx.TxManager
	eventStore  eventstore.Store
	publisher   *messaging.EventPublisher
	members     *projections.MembersProjection
	invitations *projections.InvitationsProjection
}

func NewService(
	db dbtx.TxManager,
	eventStore eventstore.Store,
	publisher *messaging.EventPublisher,
	members *projections.MembersProjection,
	invitations *projections.InvitationsProjection,
) *Service {
	return &Service{
		db:          db,
		eventStore:  eventStore,
		publisher:   publisher,
		members:     members,
		invitations: invitations,
	}
}

type RegisterInput struct {
	Name          string
	INN           string
	LegalName     string
	Country       values.Country
	Phone         string
	Email         string
	Address       values.Address
	OwnerEmail    string
	OwnerPassword string
	OwnerName     string
	OwnerPhone    string
}

type RegisterOutput struct {
	OrganizationID uuid.UUID
	MemberID       uuid.UUID
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (*RegisterOutput, error) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.OwnerPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	orgID := uuid.New()
	memberID := uuid.New()

	org := organization.New(
		orgID,
		input.Name,
		input.INN,
		input.LegalName,
		input.Country,
		input.Phone,
		input.Email,
		input.Address,
		memberID,
		input.OwnerEmail,
		string(passwordHash),
		input.OwnerName,
		input.OwnerPhone,
	)

	if err := s.saveAndPublish(ctx, org); err != nil {
		return nil, err
	}

	return &RegisterOutput{
		OrganizationID: orgID,
		MemberID:       memberID,
	}, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*organization.Organization, error) {
	evts, err := s.eventStore.Load(ctx, id, events.AggregateType)
	if err != nil {
		return nil, fmt.Errorf("failed to load organization: %w", err)
	}
	return organization.NewFromEvents(id, evts), nil
}

type CreateInvitationInput struct {
	OrganizationID uuid.UUID
	ActorID        uuid.UUID
	Email          string
	Role           values.MemberRole
}

func (s *Service) CreateInvitation(ctx context.Context, input CreateInvitationInput) (uuid.UUID, error) {
	org, err := s.Get(ctx, input.OrganizationID)
	if err != nil {
		return uuid.Nil, err
	}

	invitationID := uuid.New()
	token := rand.Text()
	expiresAt := time.Now().UTC().Add(invitationTTL)

	if err := org.CreateInvitation(input.ActorID, invitationID, input.Email, input.Role, token, expiresAt); err != nil {
		return uuid.Nil, err
	}

	if err := s.saveAndPublish(ctx, org); err != nil {
		return uuid.Nil, err
	}

	return invitationID, nil
}

type AcceptInvitationInput struct {
	Token    string
	Password string
	Name     string
	Phone    string
}

type AcceptInvitationOutput struct {
	OrganizationID uuid.UUID
	MemberID       uuid.UUID
}

func (s *Service) AcceptInvitation(ctx context.Context, input AcceptInvitationInput) (*AcceptInvitationOutput, error) {
	inv, err := s.invitations.GetByToken(ctx, input.Token)
	if err != nil {
		return nil, organization.ErrInvitationNotFound
	}

	org, err := s.Get(ctx, inv.OrganizationID)
	if err != nil {
		return nil, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	memberID := uuid.New()

	if err := org.AcceptInvitation(inv.ID, memberID, string(passwordHash), input.Name, input.Phone); err != nil {
		return nil, err
	}

	if err := s.saveAndPublish(ctx, org); err != nil {
		return nil, err
	}

	return &AcceptInvitationOutput{
		OrganizationID: inv.OrganizationID,
		MemberID:       memberID,
	}, nil
}

type ChangeMemberRoleInput struct {
	OrganizationID uuid.UUID
	ActorID        uuid.UUID
	MemberID       uuid.UUID
	NewRole        values.MemberRole
}

func (s *Service) ChangeMemberRole(ctx context.Context, input ChangeMemberRoleInput) error {
	org, err := s.Get(ctx, input.OrganizationID)
	if err != nil {
		return err
	}

	if err := org.ChangeMemberRole(input.ActorID, input.MemberID, input.NewRole); err != nil {
		return err
	}

	return s.saveAndPublish(ctx, org)
}

type BlockMemberInput struct {
	OrganizationID uuid.UUID
	ActorID        uuid.UUID
	MemberID       uuid.UUID
	Reason         string
}

func (s *Service) BlockMember(ctx context.Context, input BlockMemberInput) error {
	org, err := s.Get(ctx, input.OrganizationID)
	if err != nil {
		return err
	}

	if err := org.BlockMember(input.ActorID, input.MemberID, input.Reason); err != nil {
		return err
	}

	return s.saveAndPublish(ctx, org)
}

type UnblockMemberInput struct {
	OrganizationID uuid.UUID
	ActorID        uuid.UUID
	MemberID       uuid.UUID
}

func (s *Service) UnblockMember(ctx context.Context, input UnblockMemberInput) error {
	org, err := s.Get(ctx, input.OrganizationID)
	if err != nil {
		return err
	}

	if err := org.UnblockMember(input.ActorID, input.MemberID); err != nil {
		return err
	}

	return s.saveAndPublish(ctx, org)
}

type SetCarrierProfileInput struct {
	OrganizationID uuid.UUID
	ActorID        uuid.UUID
	Profile        values.CarrierProfile
}

func (s *Service) SetCarrierProfile(ctx context.Context, input SetCarrierProfileInput) error {
	org, err := s.Get(ctx, input.OrganizationID)
	if err != nil {
		return err
	}

	if err := org.SetCarrierProfile(input.ActorID, input.Profile); err != nil {
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
		if err := s.eventStore.Save(ctx, changes...); err != nil {
			return fmt.Errorf("failed to save events: %w", err)
		}

		// Update projections
		for _, evt := range changes {
			if err := s.members.Handle(ctx, evt); err != nil {
				return fmt.Errorf("failed to update members projection: %w", err)
			}
			if err := s.invitations.Handle(ctx, evt); err != nil {
				return fmt.Errorf("failed to update invitations projection: %w", err)
			}
		}

		// Publish to message bus
		if err := s.publisher.Publish(ctx, "organization.events", changes...); err != nil {
			return fmt.Errorf("failed to publish events: %w", err)
		}

		org.ClearChanges()
		return nil
	})
}
