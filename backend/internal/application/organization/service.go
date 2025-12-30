package organization

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"sort"
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

// SEC-007: bcrypt cost 12 вместо DefaultCost (10) для защиты от GPU перебора
const bcryptCost = 12

type Service struct {
	db            dbtx.TxManager
	eventStore    eventstore.Store
	publisher     *messaging.EventPublisher
	invitations   *projections.InvitationsProjection
	members       *projections.MembersProjection
	organizations *projections.OrganizationsProjection
}

func NewService(
	db dbtx.TxManager,
	eventStore eventstore.Store,
	publisher *messaging.EventPublisher,
	invitations *projections.InvitationsProjection,
	members *projections.MembersProjection,
	organizations *projections.OrganizationsProjection,
) *Service {
	return &Service{
		db:            db,
		eventStore:    eventStore,
		publisher:     publisher,
		invitations:   invitations,
		members:       members,
		organizations: organizations,
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
	// Registration metadata for fraud detection
	RegistrationIP          string
	RegistrationFingerprint string
	RegistrationUserAgent   string
}

type RegisterOutput struct {
	OrganizationID uuid.UUID
	MemberID       uuid.UUID
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (*RegisterOutput, error) {
	// Validate registration (email pattern + velocity)
	if err := s.validateRegistration(ctx, input.OwnerEmail, input.RegistrationIP, input.RegistrationFingerprint); err != nil {
		return nil, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.OwnerPassword), bcryptCost)
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
		input.RegistrationIP,
		input.RegistrationFingerprint,
		input.RegistrationUserAgent,
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
// Использует batch query через projection вместо N+1 запросов к event store
func (s *Service) GetNames(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error) {
	return s.organizations.GetNames(ctx, ids)
}

// MemberInfo содержит данные сотрудника для отображения
type MemberInfo struct {
	ID               uuid.UUID
	OrganizationID   uuid.UUID
	OrganizationName string
	Name             string
	Email            string
	Phone            string
	Role             string
	Status           string
	CreatedAt        time.Time
}

// GetMemberByID возвращает данные сотрудника по ID из event store
func (s *Service) GetMemberByID(ctx context.Context, memberID uuid.UUID) (*MemberInfo, error) {
	// 1. Получить org_id из проекции (lookup)
	lookup, err := s.members.GetByID(ctx, memberID)
	if err != nil {
		return nil, organization.ErrMemberNotFound
	}

	// 2. Загрузить organization из event store
	org, err := s.Get(ctx, lookup.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	// 3. Найти member в organization
	member, ok := org.GetMember(memberID)
	if !ok {
		return nil, organization.ErrMemberNotFound
	}

	// 4. Вернуть MemberInfo
	return &MemberInfo{
		ID:               member.ID(),
		OrganizationID:   org.ID(),
		OrganizationName: org.Name(),
		Name:             member.Name(),
		Email:            member.Email(),
		Phone:            member.Phone(),
		Role:             member.Role().String(),
		Status:           member.Status().String(),
		CreatedAt:        member.CreatedAt(),
	}, nil
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

type CancelInvitationInput struct {
	OrganizationID uuid.UUID
	ActorID        uuid.UUID
	InvitationID   uuid.UUID
}

func (s *Service) CancelInvitation(ctx context.Context, input CancelInvitationInput) error {
	org, err := s.Get(ctx, input.OrganizationID)
	if err != nil {
		return err
	}

	if err := org.CancelInvitation(input.ActorID, input.InvitationID); err != nil {
		return err
	}

	return s.saveAndPublish(ctx, org)
}

type AcceptInvitationInput struct {
	Token    string
	Password string
	Name     *string // опционально, если не предзаполнено в приглашении
	Phone    *string // опционально, если не предзаполнено в приглашении
	// Registration metadata for fraud detection
	RegistrationIP          string
	RegistrationFingerprint string
	RegistrationUserAgent   string
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

	// Validate registration (email pattern + velocity)
	if err := s.validateRegistration(ctx, inv.Email, input.RegistrationIP, input.RegistrationFingerprint); err != nil {
		return nil, err
	}

	org, err := s.Get(ctx, inv.OrganizationID)
	if err != nil {
		return nil, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	memberID := uuid.New()

	if err := org.AcceptInvitation(
		inv.ID,
		memberID,
		string(passwordHash),
		input.Name,
		input.Phone,
		input.RegistrationIP,
		input.RegistrationFingerprint,
		input.RegistrationUserAgent,
	); err != nil {
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
	Status         *string // фильтр по статусу (pending, accepted, expired, cancelled)
}

// ListInvitations возвращает список приглашений организации (напрямую из event store)
func (s *Service) ListInvitations(ctx context.Context, input ListInvitationsInput) ([]InvitationInfo, error) {
	org, err := s.Get(ctx, input.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	result := make([]InvitationInfo, 0, len(org.Invitations()))
	for _, inv := range org.Invitations() {
		// Фильтр по статусу
		if input.Status != nil && inv.Status().String() != *input.Status {
			continue
		}

		result = append(result, InvitationInfo{
			ID:             inv.ID(),
			OrganizationID: input.OrganizationID,
			Email:          inv.Email(),
			Role:           inv.Role().String(),
			Name:           inv.Name(),
			Phone:          inv.Phone(),
			Status:         inv.Status().String(),
			ExpiresAt:      inv.ExpiresAt(),
			CreatedAt:      inv.CreatedAt(),
		})
	}

	// Сортировка по дате создания (новые сначала)
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

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

// DevRemoveMember removes member from organization (dev only, no permission checks)
func (s *Service) DevRemoveMember(ctx context.Context, orgID, memberID uuid.UUID) error {
	org, err := s.Get(ctx, orgID)
	if err != nil {
		return err
	}

	if err := org.RemoveMember(memberID); err != nil {
		return err
	}

	return s.saveAndPublish(ctx, org)
}

// AddMemberInput for direct member addition (seeding/dev)
type AddMemberInput struct {
	OrganizationID uuid.UUID
	Email          string
	Password       string
	Name           string
	Phone          string
	Role           values.MemberRole
}

// AddMemberDirect adds member directly without invitation (for seeding/dev)
func (s *Service) AddMemberDirect(ctx context.Context, input AddMemberInput) (uuid.UUID, error) {
	org, err := s.Get(ctx, input.OrganizationID)
	if err != nil {
		return uuid.Nil, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcryptCost)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to hash password: %w", err)
	}

	memberID := uuid.New()

	if err := org.AddMemberDirect(memberID, input.Email, string(passwordHash), input.Name, input.Phone, input.Role); err != nil {
		return uuid.Nil, err
	}

	if err := s.saveAndPublish(ctx, org); err != nil {
		return uuid.Nil, err
	}

	return memberID, nil
}

// validateRegistration checks email patterns and registration velocity
// Returns error if registration should be blocked
func (s *Service) validateRegistration(ctx context.Context, email, ip, fingerprint string) error {
	// 1. Check email pattern (disposable domains, suspicious patterns)
	emailResult := values.ValidateEmail(email)
	if !emailResult.IsValid {
		return fmt.Errorf("invalid email: %s", emailResult.BlockReason)
	}
	if emailResult.IsSuspect && emailResult.BlockReason != "" {
		// Disposable email - block
		slog.Warn("registration blocked: suspicious email",
			slog.String("email", email),
			slog.String("reason", emailResult.Reason),
		)
		return organization.ErrDisposableEmail
	}
	if emailResult.IsSuspect {
		// Suspicious pattern but not blocking - just log
		slog.Info("registration: suspicious email pattern detected",
			slog.String("email", email),
			slog.String("reason", emailResult.Reason),
		)
	}

	// 2. Check registration velocity (IP + fingerprint)
	velocityResult, err := s.members.CheckRegistrationVelocity(ctx, ip, fingerprint)
	if err != nil {
		slog.Error("failed to check registration velocity",
			slog.String("error", err.Error()),
			slog.String("ip", ip),
		)
		// Don't block on error, just log
		return nil
	}
	if velocityResult.IsTooFast {
		slog.Warn("registration blocked: velocity exceeded",
			slog.String("ip", ip),
			slog.String("reason", velocityResult.BlockReason),
			slog.Int("ip_registrations", velocityResult.IPRegistrations),
			slog.Int("fp_registrations", velocityResult.FPRegistrations),
		)
		return fmt.Errorf("%w: %s", organization.ErrRegistrationVelocity, velocityResult.BlockReason)
	}

	return nil
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
