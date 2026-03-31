package freightrequest

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/udisondev/veziizi/backend/internal/domain/freightrequest"
	"github.com/udisondev/veziizi/backend/internal/domain/freightrequest/events"
	"github.com/udisondev/veziizi/backend/internal/domain/freightrequest/values"
	orgValues "github.com/udisondev/veziizi/backend/internal/domain/organization/values"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/messaging"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/sequence"
	"github.com/udisondev/veziizi/backend/internal/pkg/dbtx"
	"github.com/google/uuid"
)

const defaultTTL = 30 * 24 * time.Hour // 30 days

// MemberChecker проверяет принадлежность и права членов организации.
type MemberChecker interface {
	MemberExists(ctx context.Context, orgID, memberID uuid.UUID) error
	CanManageMembers(ctx context.Context, orgID, memberID uuid.UUID) (orgValues.MemberRole, error)
}

type Service struct {
	db            dbtx.TxManager
	eventStore    eventstore.Store
	publisher     *messaging.EventPublisher
	seqGen        *sequence.Generator
	memberChecker MemberChecker
}

func NewService(
	db dbtx.TxManager,
	eventStore eventstore.Store,
	publisher *messaging.EventPublisher,
	seqGen *sequence.Generator,
	memberChecker MemberChecker,
) *Service {
	return &Service{
		db:            db,
		eventStore:    eventStore,
		publisher:     publisher,
		seqGen:        seqGen,
		memberChecker: memberChecker,
	}
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*freightrequest.FreightRequest, error) {
	evts, err := s.eventStore.Load(ctx, id, events.AggregateType)
	if err != nil {
		if errors.Is(err, eventstore.ErrAggregateNotFound) {
			return nil, freightrequest.ErrFreightRequestNotFound
		}
		slog.Error("failed to load freight request",
			slog.String("freight_request_id", id.String()),
			slog.String("error", err.Error()))
		return nil, fmt.Errorf("load freight request: %w", err)
	}

	fr := freightrequest.NewFromEvents(id, evts)
	if fr.Version() == 0 {
		return nil, freightrequest.ErrFreightRequestNotFound
	}

	return fr, nil
}

// GetByIDs загружает несколько freight requests одним batch запросом.
// Возвращает map[id]*FreightRequest. Отсутствующие ID не включаются в результат.
func (s *Service) GetByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*freightrequest.FreightRequest, error) {
	if len(ids) == 0 {
		return make(map[uuid.UUID]*freightrequest.FreightRequest), nil
	}

	eventsMap, err := s.eventStore.LoadByIDs(ctx, ids, events.AggregateType)
	if err != nil {
		slog.Error("failed to batch load freight requests",
			slog.Int("count", len(ids)),
			slog.String("error", err.Error()))
		return nil, fmt.Errorf("batch load freight requests: %w", err)
	}

	result := make(map[uuid.UUID]*freightrequest.FreightRequest, len(eventsMap))
	for id, evts := range eventsMap {
		fr := freightrequest.NewFromEvents(id, evts)
		if fr.Version() > 0 {
			result[id] = fr
		}
	}

	return result, nil
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

	slog.Info("freight request created",
		slog.String("id", resultID.String()),
		slog.String("customer_org_id", input.CustomerOrgID.String()))

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
	if _, err := s.memberChecker.CanManageMembers(ctx, input.ActorOrgID, input.ActorID); err != nil {
		return err
	}

	// Check new member exists in organization
	if err := s.memberChecker.MemberExists(ctx, input.ActorOrgID, input.NewMemberID); err != nil {
		return err
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

	slog.Info("offer made",
		slog.String("offer_id", offerID.String()),
		slog.String("freight_request_id", input.FreightRequestID.String()),
		slog.String("carrier_org_id", input.CarrierOrgID.String()))

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
		if _, err := s.memberChecker.CanManageMembers(ctx, input.ActorOrgID, input.ActorMemberID); err != nil {
			return err
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

	if err := fr.SelectOffer(input.OfferID, input.ActorID, input.ActorOrgID); err != nil {
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

	if err := fr.RejectOffer(input.OfferID, input.ActorID, input.ActorOrgID, input.Reason); err != nil {
		return err
	}

	return s.saveAndPublish(ctx, fr)
}

type ConfirmOfferInput struct {
	FreightRequestID uuid.UUID
	OfferID          uuid.UUID
	ActorMemberID    uuid.UUID
	ActorOrgID       uuid.UUID
	ActorRole        orgValues.MemberRole
}

func (s *Service) ConfirmOffer(ctx context.Context, input ConfirmOfferInput) error {
	fr, err := s.Get(ctx, input.FreightRequestID)
	if err != nil {
		return err
	}

	if err := fr.ConfirmOffer(input.OfferID, input.ActorMemberID, input.ActorOrgID, input.ActorRole); err != nil {
		return err
	}

	if err := s.saveAndPublish(ctx, fr); err != nil {
		return err
	}

	slog.Info("offer confirmed",
		slog.String("offer_id", input.OfferID.String()),
		slog.String("freight_request_id", input.FreightRequestID.String()),
		slog.String("carrier_org_id", input.ActorOrgID.String()))

	return nil
}

type DeclineOfferInput struct {
	FreightRequestID uuid.UUID
	OfferID          uuid.UUID
	ActorMemberID    uuid.UUID
	ActorOrgID       uuid.UUID
	ActorRole        orgValues.MemberRole
	Reason           string
}

func (s *Service) DeclineOffer(ctx context.Context, input DeclineOfferInput) error {
	fr, err := s.Get(ctx, input.FreightRequestID)
	if err != nil {
		return err
	}

	if err := fr.DeclineOffer(input.OfferID, input.ActorMemberID, input.ActorOrgID, input.ActorRole, input.Reason); err != nil {
		return err
	}

	return s.saveAndPublish(ctx, fr)
}

type UnselectOfferInput struct {
	FreightRequestID uuid.UUID
	OfferID          uuid.UUID
	ActorID          uuid.UUID
	ActorOrgID       uuid.UUID
	Reason           string
}

func (s *Service) UnselectOffer(ctx context.Context, input UnselectOfferInput) error {
	fr, err := s.Get(ctx, input.FreightRequestID)
	if err != nil {
		return err
	}

	if err := fr.UnselectOffer(input.OfferID, input.ActorID, input.ActorOrgID, input.Reason); err != nil {
		return err
	}

	return s.saveAndPublish(ctx, fr)
}

// CompleteInput represents input for completing a freight request by one party
type CompleteInput struct {
	FreightRequestID uuid.UUID
	OrgID            uuid.UUID
	MemberID         uuid.UUID
}

// Complete marks the freight as completed by one party (customer or carrier)
func (s *Service) Complete(ctx context.Context, input CompleteInput) error {
	fr, err := s.Get(ctx, input.FreightRequestID)
	if err != nil {
		return err
	}

	if err := fr.Complete(input.OrgID, input.MemberID); err != nil {
		return err
	}

	return s.saveAndPublish(ctx, fr)
}

// LeaveReviewInput represents input for leaving a review
type LeaveReviewInput struct {
	FreightRequestID uuid.UUID
	ReviewerOrgID    uuid.UUID
	ReviewerMemberID uuid.UUID
	Rating           int
	Comment          string
}

// LeaveReview leaves a review for the other party
func (s *Service) LeaveReview(ctx context.Context, input LeaveReviewInput) (uuid.UUID, error) {
	fr, err := s.Get(ctx, input.FreightRequestID)
	if err != nil {
		return uuid.Nil, err
	}

	reviewID := uuid.New()
	if err := fr.LeaveReview(reviewID, input.ReviewerOrgID, input.ReviewerMemberID, input.Rating, input.Comment); err != nil {
		return uuid.Nil, err
	}

	if err := s.saveAndPublish(ctx, fr); err != nil {
		return uuid.Nil, err
	}

	return reviewID, nil
}

// EditReviewInput represents input for editing a review
type EditReviewInput struct {
	FreightRequestID uuid.UUID
	ReviewerOrgID    uuid.UUID
	ReviewerMemberID uuid.UUID
	Rating           int
	Comment          string
}

// EditReview edits an existing review (within 24h window)
func (s *Service) EditReview(ctx context.Context, input EditReviewInput) error {
	fr, err := s.Get(ctx, input.FreightRequestID)
	if err != nil {
		return err
	}

	if err := fr.EditReview(input.ReviewerOrgID, input.ReviewerMemberID, input.Rating, input.Comment); err != nil {
		return err
	}

	return s.saveAndPublish(ctx, fr)
}

// CancelAfterConfirmedInput represents input for cancelling after confirmed
type CancelAfterConfirmedInput struct {
	FreightRequestID uuid.UUID
	OrgID            uuid.UUID
	MemberID         uuid.UUID
	Reason           string
}

// CancelAfterConfirmed cancels freight after offer was confirmed
func (s *Service) CancelAfterConfirmed(ctx context.Context, input CancelAfterConfirmedInput) error {
	fr, err := s.Get(ctx, input.FreightRequestID)
	if err != nil {
		return err
	}

	if err := fr.CancelAfterConfirmed(input.OrgID, input.MemberID, input.Reason); err != nil {
		return err
	}

	return s.saveAndPublish(ctx, fr)
}

// ReassignCarrierMemberInput represents input for reassigning carrier's responsible member
type ReassignCarrierMemberInput struct {
	FreightRequestID uuid.UUID
	ActorID          uuid.UUID
	ActorOrgID       uuid.UUID
	NewMemberID      uuid.UUID
}

// ReassignCarrierMember reassigns the carrier's responsible member
func (s *Service) ReassignCarrierMember(ctx context.Context, input ReassignCarrierMemberInput) error {
	// Check actor has permission (admin or owner)
	actorRole, err := s.memberChecker.CanManageMembers(ctx, input.ActorOrgID, input.ActorID)
	if err != nil {
		return err
	}

	// Check new member exists in organization
	if err := s.memberChecker.MemberExists(ctx, input.ActorOrgID, input.NewMemberID); err != nil {
		return err
	}

	fr, err := s.Get(ctx, input.FreightRequestID)
	if err != nil {
		return err
	}

	// Verify freight request has carrier assigned and actor is from carrier org
	if fr.CarrierOrgID() == nil || *fr.CarrierOrgID() != input.ActorOrgID {
		return freightrequest.ErrNotConfirmed
	}

	if err := fr.ReassignCarrierMember(input.ActorID, input.NewMemberID, actorRole); err != nil {
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
			slog.Error("failed to save freight request events",
				slog.String("freight_request_id", fr.ID().String()),
				slog.Int("event_count", len(changes)),
				slog.String("error", err.Error()))
			return fmt.Errorf("save events: %w", err)
		}

		if err := s.publisher.Publish(ctx, "freightrequest.events", changes...); err != nil {
			slog.Error("failed to publish freight request events",
				slog.String("freight_request_id", fr.ID().String()),
				slog.Int("event_count", len(changes)),
				slog.String("error", err.Error()))
			return fmt.Errorf("publish events: %w", err)
		}

		fr.ClearChanges()
		return nil
	})
}
