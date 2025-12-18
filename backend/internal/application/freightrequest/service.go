package freightrequest

import (
	"context"
	"fmt"
	"time"

	"codeberg.org/udison/veziizi/backend/internal/domain/freightrequest"
	"codeberg.org/udison/veziizi/backend/internal/domain/freightrequest/events"
	"codeberg.org/udison/veziizi/backend/internal/domain/freightrequest/values"
	"codeberg.org/udison/veziizi/backend/internal/domain/organization"
	orgEvents "codeberg.org/udison/veziizi/backend/internal/domain/organization/events"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/messaging"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/sequence"
	"codeberg.org/udison/veziizi/backend/internal/pkg/dbtx"
	"github.com/google/uuid"
)

const defaultTTL = 30 * 24 * time.Hour // 30 days

type Service struct {
	db         dbtx.TxManager
	eventStore eventstore.Store
	publisher  *messaging.EventPublisher
	seqGen     *sequence.Generator
}

func NewService(
	db dbtx.TxManager,
	eventStore eventstore.Store,
	publisher *messaging.EventPublisher,
	seqGen *sequence.Generator,
) *Service {
	return &Service{
		db:         db,
		eventStore: eventStore,
		publisher:  publisher,
		seqGen:     seqGen,
	}
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*freightrequest.FreightRequest, error) {
	evts, err := s.eventStore.Load(ctx, id, events.AggregateType)
	if err != nil {
		return nil, fmt.Errorf("failed to load freight request: %w", err)
	}
	return freightrequest.NewFromEvents(id, evts), nil
}

func (s *Service) getOrganization(ctx context.Context, id uuid.UUID) (*organization.Organization, error) {
	evts, err := s.eventStore.Load(ctx, id, orgEvents.AggregateType)
	if err != nil {
		return nil, fmt.Errorf("failed to load organization: %w", err)
	}
	return organization.NewFromEvents(id, evts), nil
}

type CreateInput struct {
	CustomerOrgID       uuid.UUID
	CustomerMemberID    uuid.UUID
	Route               values.Route
	Cargo               values.CargoInfo
	VehicleRequirements values.VehicleRequirements
	Payment             values.Payment
	Comment             string
	ExpiresAt           *time.Time
}

func (s *Service) Create(ctx context.Context, input CreateInput) (uuid.UUID, error) {
	expiresAt := time.Now().UTC().Add(defaultTTL)
	if input.ExpiresAt != nil {
		expiresAt = *input.ExpiresAt
	}

	var resultID uuid.UUID

	err := s.db.InTx(ctx, func(ctx context.Context) error {
		requestNumber, err := s.seqGen.NextRequestNumber(ctx)
		if err != nil {
			return fmt.Errorf("get next request number: %w", err)
		}

		id := uuid.New()
		fr := freightrequest.New(
			id,
			requestNumber,
			input.CustomerOrgID,
			input.CustomerMemberID,
			input.Route,
			input.Cargo,
			input.VehicleRequirements,
			input.Payment,
			input.Comment,
			expiresAt,
		)

		if err := s.saveAndPublish(ctx, fr); err != nil {
			return err
		}

		resultID = id
		return nil
	})

	if err != nil {
		return uuid.Nil, err
	}

	return resultID, nil
}

type UpdateInput struct {
	ID                  uuid.UUID
	ActorID             uuid.UUID
	Route               *values.Route
	Cargo               *values.CargoInfo
	VehicleRequirements *values.VehicleRequirements
	Payment             *values.Payment
	Comment             *string
}

func (s *Service) Update(ctx context.Context, input UpdateInput) error {
	fr, err := s.Get(ctx, input.ID)
	if err != nil {
		return err
	}

	if err := fr.Update(input.ActorID, input.Route, input.Cargo, input.VehicleRequirements, input.Payment, input.Comment); err != nil {
		return err
	}

	return s.saveAndPublish(ctx, fr)
}

type CancelInput struct {
	ID      uuid.UUID
	ActorID uuid.UUID
	Reason  string
}

func (s *Service) Cancel(ctx context.Context, input CancelInput) error {
	fr, err := s.Get(ctx, input.ID)
	if err != nil {
		return err
	}

	if err := fr.Cancel(input.ActorID, input.Reason); err != nil {
		return err
	}

	return s.saveAndPublish(ctx, fr)
}

type ReassignInput struct {
	ID          uuid.UUID
	ActorID     uuid.UUID
	ActorOrgID  uuid.UUID
	NewMemberID uuid.UUID
}

func (s *Service) Reassign(ctx context.Context, input ReassignInput) error {
	// Check actor has permission (admin or owner)
	org, err := s.getOrganization(ctx, input.ActorOrgID)
	if err != nil {
		return err
	}

	actor, ok := org.GetMember(input.ActorID)
	if !ok {
		return organization.ErrMemberNotFound
	}
	if !actor.CanManageMembers() {
		return organization.ErrInsufficientPermissions
	}

	// Check new member exists in organization
	if _, ok := org.GetMember(input.NewMemberID); !ok {
		return organization.ErrMemberNotFound
	}

	fr, err := s.Get(ctx, input.ID)
	if err != nil {
		return err
	}

	// Verify freight request belongs to this organization
	if fr.CustomerOrgID() != input.ActorOrgID {
		return freightrequest.ErrNotFreightRequestOwner
	}

	if err := fr.Reassign(input.ActorID, input.NewMemberID); err != nil {
		return err
	}

	return s.saveAndPublish(ctx, fr)
}

type MakeOfferInput struct {
	FreightRequestID uuid.UUID
	CarrierOrgID     uuid.UUID
	CarrierMemberID  uuid.UUID
	Price            values.Money
	Comment          string
	VatType          values.VatType
	PaymentMethod    values.PaymentMethod
}

func (s *Service) MakeOffer(ctx context.Context, input MakeOfferInput) (uuid.UUID, error) {
	fr, err := s.Get(ctx, input.FreightRequestID)
	if err != nil {
		return uuid.Nil, err
	}

	offerID := uuid.New()
	if err := fr.MakeOffer(
		offerID,
		input.CarrierOrgID,
		input.CarrierMemberID,
		input.Price,
		input.Comment,
		input.VatType,
		input.PaymentMethod,
	); err != nil {
		return uuid.Nil, err
	}

	if err := s.saveAndPublish(ctx, fr); err != nil {
		return uuid.Nil, err
	}

	return offerID, nil
}

type WithdrawOfferInput struct {
	FreightRequestID uuid.UUID
	OfferID          uuid.UUID
	ActorOrgID       uuid.UUID
	ActorMemberID    uuid.UUID
	Reason           string
}

func (s *Service) WithdrawOffer(ctx context.Context, input WithdrawOfferInput) error {
	fr, err := s.Get(ctx, input.FreightRequestID)
	if err != nil {
		return err
	}

	// Получаем оффер для проверки прав
	offer, ok := fr.GetOffer(input.OfferID)
	if !ok {
		return freightrequest.ErrOfferNotFound
	}

	// Проверка: создатель оффера или admin/owner организации
	isOfferCreator := offer.CarrierMemberID() == input.ActorMemberID
	if !isOfferCreator {
		org, err := s.getOrganization(ctx, input.ActorOrgID)
		if err != nil {
			return fmt.Errorf("failed to get organization: %w", err)
		}

		actor, ok := org.GetMember(input.ActorMemberID)
		if !ok {
			return organization.ErrMemberNotFound
		}

		if !actor.CanManageMembers() {
			return organization.ErrInsufficientPermissions
		}
	}

	if err := fr.WithdrawOffer(input.OfferID, input.ActorOrgID, input.Reason); err != nil {
		return err
	}

	return s.saveAndPublish(ctx, fr)
}

type SelectOfferInput struct {
	FreightRequestID uuid.UUID
	OfferID          uuid.UUID
	ActorID          uuid.UUID
	ActorOrgID       uuid.UUID
}

func (s *Service) SelectOffer(ctx context.Context, input SelectOfferInput) error {
	fr, err := s.Get(ctx, input.FreightRequestID)
	if err != nil {
		return err
	}

	// Получаем роль актора для проверки в доменной логике
	canManage := false
	if fr.CustomerOrgID() == input.ActorOrgID {
		org, err := s.getOrganization(ctx, input.ActorOrgID)
		if err != nil {
			return fmt.Errorf("get organization: %w", err)
		}

		if member, ok := org.GetMember(input.ActorID); ok {
			canManage = member.Role().CanManageFreightRequests()
		}
	}

	if err := fr.SelectOffer(input.OfferID, input.ActorID, input.ActorOrgID, canManage); err != nil {
		return err
	}

	return s.saveAndPublish(ctx, fr)
}

type RejectOfferInput struct {
	FreightRequestID uuid.UUID
	OfferID          uuid.UUID
	ActorID          uuid.UUID
	ActorOrgID       uuid.UUID
	Reason           string
}

func (s *Service) RejectOffer(ctx context.Context, input RejectOfferInput) error {
	fr, err := s.Get(ctx, input.FreightRequestID)
	if err != nil {
		return err
	}

	// Получаем роль актора для проверки в доменной логике
	canManage := false
	if fr.CustomerOrgID() == input.ActorOrgID {
		org, err := s.getOrganization(ctx, input.ActorOrgID)
		if err != nil {
			return fmt.Errorf("get organization: %w", err)
		}

		if member, ok := org.GetMember(input.ActorID); ok {
			canManage = member.Role().CanManageFreightRequests()
		}
	}

	if err := fr.RejectOffer(input.OfferID, input.ActorID, input.ActorOrgID, canManage, input.Reason); err != nil {
		return err
	}

	return s.saveAndPublish(ctx, fr)
}

type ConfirmOfferInput struct {
	FreightRequestID uuid.UUID
	OfferID          uuid.UUID
	ActorOrgID       uuid.UUID
}

func (s *Service) ConfirmOffer(ctx context.Context, input ConfirmOfferInput) error {
	fr, err := s.Get(ctx, input.FreightRequestID)
	if err != nil {
		return err
	}

	if err := fr.ConfirmOffer(input.OfferID, input.ActorOrgID); err != nil {
		return err
	}

	return s.saveAndPublish(ctx, fr)
}

type DeclineOfferInput struct {
	FreightRequestID uuid.UUID
	OfferID          uuid.UUID
	ActorOrgID       uuid.UUID
	Reason           string
}

func (s *Service) DeclineOffer(ctx context.Context, input DeclineOfferInput) error {
	fr, err := s.Get(ctx, input.FreightRequestID)
	if err != nil {
		return err
	}

	if err := fr.DeclineOffer(input.OfferID, input.ActorOrgID, input.Reason); err != nil {
		return err
	}

	return s.saveAndPublish(ctx, fr)
}

func (s *Service) saveAndPublish(ctx context.Context, fr *freightrequest.FreightRequest) error {
	changes := fr.Changes()
	if len(changes) == 0 {
		return nil
	}

	return s.db.InTx(ctx, func(ctx context.Context) error {
		if err := s.eventStore.Save(ctx, changes...); err != nil {
			return fmt.Errorf("failed to save events: %w", err)
		}

		if err := s.publisher.Publish(ctx, "freightrequest.events", changes...); err != nil {
			return fmt.Errorf("failed to publish events: %w", err)
		}

		fr.ClearChanges()
		return nil
	})
}
