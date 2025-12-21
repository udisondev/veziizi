package order

import (
	"context"
	"fmt"
	"net/http"

	"codeberg.org/udison/veziizi/backend/internal/domain/freightrequest"
	frEvents "codeberg.org/udison/veziizi/backend/internal/domain/freightrequest/events"
	"codeberg.org/udison/veziizi/backend/internal/domain/order"
	"codeberg.org/udison/veziizi/backend/internal/domain/order/events"
	"codeberg.org/udison/veziizi/backend/internal/domain/organization"
	orgEvents "codeberg.org/udison/veziizi/backend/internal/domain/organization/events"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/messaging"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/filestorage"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/sequence"
	"codeberg.org/udison/veziizi/backend/internal/pkg/dbtx"
	"github.com/google/uuid"
)

type Service struct {
	db          dbtx.TxManager
	eventStore  eventstore.Store
	publisher   *messaging.EventPublisher
	fileStorage filestorage.FileStorage
	seqGen      *sequence.Generator
}

func NewService(
	db dbtx.TxManager,
	eventStore eventstore.Store,
	publisher *messaging.EventPublisher,
	fileStorage filestorage.FileStorage,
	seqGen *sequence.Generator,
) *Service {
	return &Service{
		db:          db,
		eventStore:  eventStore,
		publisher:   publisher,
		fileStorage: fileStorage,
		seqGen:      seqGen,
	}
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*order.Order, error) {
	evts, err := s.eventStore.Load(ctx, id, events.AggregateType)
	if err != nil {
		return nil, fmt.Errorf("load order: %w", err)
	}
	return order.NewFromEvents(id, evts), nil
}

func (s *Service) getFreightRequest(ctx context.Context, id uuid.UUID) (*freightrequest.FreightRequest, error) {
	evts, err := s.eventStore.Load(ctx, id, frEvents.AggregateType)
	if err != nil {
		return nil, fmt.Errorf("load freight request: %w", err)
	}
	return freightrequest.NewFromEvents(id, evts), nil
}

type CreateFromOfferInput struct {
	FreightRequestID uuid.UUID
	OfferID          uuid.UUID
}

// CreateFromConfirmedOffer creates an order from a confirmed offer
// Called by order-creator worker when OfferConfirmed event is received
func (s *Service) CreateFromConfirmedOffer(ctx context.Context, input CreateFromOfferInput) (uuid.UUID, error) {
	var resultID uuid.UUID

	err := s.db.InTx(ctx, func(ctx context.Context) error {
		orderNumber, err := s.seqGen.NextOrderNumber(ctx)
		if err != nil {
			return fmt.Errorf("get next order number: %w", err)
		}

		fr, err := s.getFreightRequest(ctx, input.FreightRequestID)
		if err != nil {
			return err
		}

		offer, ok := fr.GetOffer(input.OfferID)
		if !ok {
			return freightrequest.ErrOfferNotFound
		}

		id := uuid.New()
		o := order.New(
			id,
			orderNumber,
			input.FreightRequestID,
			input.OfferID,
			fr.CustomerOrgID(),
			fr.CustomerMemberID(),
			offer.CarrierOrgID(),
			offer.CarrierMemberID(),
			fr.Route(),
			fr.Cargo(),
			fr.Payment(),
		)

		if err := s.saveAndPublish(ctx, o); err != nil {
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

type SendMessageInput struct {
	OrderID        uuid.UUID
	SenderOrgID    uuid.UUID
	SenderMemberID uuid.UUID
	Content        string
}

func (s *Service) SendMessage(ctx context.Context, input SendMessageInput) error {
	o, err := s.Get(ctx, input.OrderID)
	if err != nil {
		return err
	}

	if err := o.SendMessage(input.SenderOrgID, input.SenderMemberID, input.Content); err != nil {
		return err
	}

	return s.saveAndPublish(ctx, o)
}

type AttachDocumentInput struct {
	OrderID          uuid.UUID
	UploaderOrgID    uuid.UUID
	UploaderMemberID uuid.UUID
	Name             string
	Data             []byte
}

func (s *Service) AttachDocument(ctx context.Context, input AttachDocumentInput) error {
	o, err := s.Get(ctx, input.OrderID)
	if err != nil {
		return err
	}

	// Detect MIME type from file content
	mimeType := http.DetectContentType(input.Data)

	// Save file first
	fileID, err := s.fileStorage.Save(ctx, input.Data, mimeType)
	if err != nil {
		return fmt.Errorf("save file: %w", err)
	}

	if err := o.AttachDocument(
		input.UploaderOrgID,
		input.UploaderMemberID,
		input.Name,
		mimeType,
		int64(len(input.Data)),
		fileID,
	); err != nil {
		// Try to delete the file if we fail to attach
		_ = s.fileStorage.Delete(ctx, fileID)
		return err
	}

	return s.saveAndPublish(ctx, o)
}

type RemoveDocumentInput struct {
	OrderID         uuid.UUID
	DocumentID      uuid.UUID
	RemoverOrgID    uuid.UUID
	RemoverMemberID uuid.UUID
}

func (s *Service) RemoveDocument(ctx context.Context, input RemoveDocumentInput) error {
	o, err := s.Get(ctx, input.OrderID)
	if err != nil {
		return err
	}

	// Get file ID before removing document
	doc, ok := o.GetDocument(input.DocumentID)
	if !ok {
		return order.ErrDocumentNotFound
	}
	fileID := doc.FileID()

	if err := o.RemoveDocument(input.RemoverOrgID, input.RemoverMemberID, input.DocumentID); err != nil {
		return err
	}

	if err := s.saveAndPublish(ctx, o); err != nil {
		return err
	}

	// Delete file after successful event save (best effort)
	_ = s.fileStorage.Delete(ctx, fileID)

	return nil
}

func (s *Service) GetDocumentFile(ctx context.Context, orderID, documentID uuid.UUID) ([]byte, string, error) {
	o, err := s.Get(ctx, orderID)
	if err != nil {
		return nil, "", err
	}

	doc, ok := o.GetDocument(documentID)
	if !ok {
		return nil, "", order.ErrDocumentNotFound
	}

	data, mimeType, err := s.fileStorage.Get(ctx, doc.FileID())
	if err != nil {
		return nil, "", fmt.Errorf("get file: %w", err)
	}

	return data, mimeType, nil
}

type CompleteInput struct {
	OrderID  uuid.UUID
	OrgID    uuid.UUID
	MemberID uuid.UUID
}

func (s *Service) Complete(ctx context.Context, input CompleteInput) error {
	o, err := s.Get(ctx, input.OrderID)
	if err != nil {
		return err
	}

	if err := o.Complete(input.OrgID, input.MemberID); err != nil {
		return err
	}

	return s.saveAndPublish(ctx, o)
}

type CancelInput struct {
	OrderID  uuid.UUID
	OrgID    uuid.UUID
	MemberID uuid.UUID
	Reason   string
}

func (s *Service) Cancel(ctx context.Context, input CancelInput) error {
	o, err := s.Get(ctx, input.OrderID)
	if err != nil {
		return err
	}

	if err := o.Cancel(input.OrgID, input.MemberID, input.Reason); err != nil {
		return err
	}

	return s.saveAndPublish(ctx, o)
}

type LeaveReviewInput struct {
	OrderID          uuid.UUID
	ReviewerOrgID    uuid.UUID
	ReviewerMemberID uuid.UUID
	Rating           int
	Comment          string
}

func (s *Service) LeaveReview(ctx context.Context, input LeaveReviewInput) error {
	o, err := s.Get(ctx, input.OrderID)
	if err != nil {
		return err
	}

	if err := o.LeaveReview(input.ReviewerOrgID, input.ReviewerMemberID, input.Rating, input.Comment); err != nil {
		return err
	}

	return s.saveAndPublish(ctx, o)
}

func (s *Service) saveAndPublish(ctx context.Context, o *order.Order) error {
	changes := o.Changes()
	if len(changes) == 0 {
		return nil
	}

	return s.db.InTx(ctx, func(ctx context.Context) error {
		if err := s.eventStore.Save(ctx, changes...); err != nil {
			return fmt.Errorf("save events: %w", err)
		}

		if err := s.publisher.Publish(ctx, "order.events", changes...); err != nil {
			return fmt.Errorf("publish events: %w", err)
		}

		o.ClearChanges()
		return nil
	})
}

func (s *Service) getOrganization(ctx context.Context, id uuid.UUID) (*organization.Organization, error) {
	evts, err := s.eventStore.Load(ctx, id, orgEvents.AggregateType)
	if err != nil {
		return nil, fmt.Errorf("load organization: %w", err)
	}
	return organization.NewFromEvents(id, evts), nil
}

type ReassignMemberInput struct {
	OrderID     uuid.UUID
	ActorID     uuid.UUID
	ActorOrgID  uuid.UUID
	NewMemberID uuid.UUID
}

// ReassignCustomerMember переназначает ответственного со стороны заказчика
func (s *Service) ReassignCustomerMember(ctx context.Context, input ReassignMemberInput) error {
	// Проверяем права актора (owner/administrator организации-заказчика)
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

	// Проверяем что новый member существует в организации
	if _, ok := org.GetMember(input.NewMemberID); !ok {
		return organization.ErrMemberNotFound
	}

	o, err := s.Get(ctx, input.OrderID)
	if err != nil {
		return err
	}

	// Проверяем что актор из организации-заказчика
	if o.CustomerOrgID() != input.ActorOrgID {
		return order.ErrNotOrderParticipant
	}

	if err := o.ReassignCustomerMember(input.ActorID, input.NewMemberID); err != nil {
		return err
	}

	return s.saveAndPublish(ctx, o)
}

// ReassignCarrierMember переназначает ответственного со стороны перевозчика
func (s *Service) ReassignCarrierMember(ctx context.Context, input ReassignMemberInput) error {
	// Проверяем права актора (owner/administrator организации-перевозчика)
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

	// Проверяем что новый member существует в организации
	if _, ok := org.GetMember(input.NewMemberID); !ok {
		return organization.ErrMemberNotFound
	}

	o, err := s.Get(ctx, input.OrderID)
	if err != nil {
		return err
	}

	// Проверяем что актор из организации-перевозчика
	if o.CarrierOrgID() != input.ActorOrgID {
		return order.ErrNotOrderParticipant
	}

	if err := o.ReassignCarrierMember(input.ActorID, input.NewMemberID); err != nil {
		return err
	}

	return s.saveAndPublish(ctx, o)
}
