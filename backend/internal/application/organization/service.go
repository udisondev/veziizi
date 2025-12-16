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
	invitations *projections.InvitationsProjection
}

func NewService(
	db dbtx.TxManager,
	eventStore eventstore.Store,
	publisher *messaging.EventPublisher,
	invitations *projections.InvitationsProjection,
) *Service {
	return &Service{
		db:          db,
		eventStore:  eventStore,
		publisher:   publisher,
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

// GetNames возвращает названия организаций по их ID
func (s *Service) GetNames(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error) {
	result := make(map[uuid.UUID]string, len(ids))
	for _, id := range ids {
		org, err := s.Get(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("failed to get organization %s: %w", id, err)
		}
		if org.Version() > 0 {
			result[id] = org.Name()
		}
	}
	return result, nil
}

type CreateInvitationInput struct {
	OrganizationID uuid.UUID
	ActorID        uuid.UUID
	Email          string
	Role           values.MemberRole
	Name           *string // предзаполненное ФИО (опционально)
	Phone          *string // предзаполненный телефон (опционально)
}

type CreateInvitationOutput struct {
	InvitationID uuid.UUID
	Token        string // возвращаем токен для ручного тестирования (пока нет отправки email)
}

func (s *Service) CreateInvitation(ctx context.Context, input CreateInvitationInput) (*CreateInvitationOutput, error) {
	org, err := s.Get(ctx, input.OrganizationID)
	if err != nil {
		return nil, err
	}

	invitationID := uuid.New()
	token := rand.Text()
	expiresAt := time.Now().UTC().Add(invitationTTL)

	if err := org.CreateInvitation(input.ActorID, invitationID, input.Email, input.Role, token, expiresAt, input.Name, input.Phone); err != nil {
		return nil, err
	}

	if err := s.saveAndPublish(ctx, org); err != nil {
		return nil, err
	}

	return &CreateInvitationOutput{
		InvitationID: invitationID,
		Token:        token,
	}, nil
}

type AcceptInvitationInput struct {
	Token    string
	Password string
	Name     *string // опционально, если не предзаполнено в приглашении
	Phone    *string // опционально, если не предзаполнено в приглашении
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

// InvitationInfo содержит данные приглашения для отображения
type InvitationInfo struct {
	ID               uuid.UUID
	OrganizationID   uuid.UUID
	OrganizationName string
	Email            string
	Role             string
	Name             *string
	Phone            *string
	Status           string
	ExpiresAt        time.Time
	CreatedAt        time.Time
}

// GetInvitationByToken возвращает данные приглашения по токену (для формы принятия)
func (s *Service) GetInvitationByToken(ctx context.Context, token string) (*InvitationInfo, error) {
	inv, err := s.invitations.GetByToken(ctx, token)
	if err != nil {
		return nil, organization.ErrInvitationNotFound
	}

	// Получаем название организации
	org, err := s.Get(ctx, inv.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	return &InvitationInfo{
		ID:               inv.ID,
		OrganizationID:   inv.OrganizationID,
		OrganizationName: org.Name(),
		Email:            inv.Email,
		Role:             inv.Role,
		Name:             inv.Name,
		Phone:            inv.Phone,
		Status:           inv.Status,
		ExpiresAt:        inv.ExpiresAt,
		CreatedAt:        inv.CreatedAt,
	}, nil
}

// ListInvitationsInput параметры для получения списка приглашений
type ListInvitationsInput struct {
	OrganizationID uuid.UUID
	Status         *string // фильтр по статусу (pending, accepted, expired)
}

// ListInvitations возвращает список приглашений организации
func (s *Service) ListInvitations(ctx context.Context, input ListInvitationsInput) ([]InvitationInfo, error) {
	invitations, err := s.invitations.ListByOrganization(ctx, input.OrganizationID, input.Status)
	if err != nil {
		return nil, fmt.Errorf("failed to list invitations: %w", err)
	}

	result := make([]InvitationInfo, 0, len(invitations))
	for _, inv := range invitations {
		result = append(result, InvitationInfo{
			ID:             inv.ID,
			OrganizationID: inv.OrganizationID,
			Email:          inv.Email,
			Role:           inv.Role,
			Name:           inv.Name,
			Phone:          inv.Phone,
			Status:         inv.Status,
			ExpiresAt:      inv.ExpiresAt,
			CreatedAt:      inv.CreatedAt,
		})
	}

	return result, nil
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

func (s *Service) saveAndPublish(ctx context.Context, org *organization.Organization) error {
	changes := org.Changes()
	if len(changes) == 0 {
		return nil
	}

	return s.db.InTx(ctx, func(ctx context.Context) error {
		if err := s.eventStore.Save(ctx, changes...); err != nil {
			return fmt.Errorf("failed to save events: %w", err)
		}

		// Publish to message bus - watermill subscribers will update projections async
		if err := s.publisher.Publish(ctx, "organization.events", changes...); err != nil {
			return fmt.Errorf("failed to publish events: %w", err)
		}

		org.ClearChanges()
		return nil
	})
}
