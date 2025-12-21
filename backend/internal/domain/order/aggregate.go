package order

import (
	"strings"
	"time"

	frValues "codeberg.org/udison/veziizi/backend/internal/domain/freightrequest/values"
	"codeberg.org/udison/veziizi/backend/internal/domain/order/entities"
	"codeberg.org/udison/veziizi/backend/internal/domain/order/events"
	"codeberg.org/udison/veziizi/backend/internal/domain/order/values"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"codeberg.org/udison/veziizi/backend/internal/pkg/aggregate"
	"github.com/google/uuid"
)

type Order struct {
	aggregate.Base

	orderNumber      int64
	freightRequestID uuid.UUID
	offerID          uuid.UUID
	customerOrgID    uuid.UUID
	customerMemberID uuid.UUID
	carrierOrgID     uuid.UUID
	carrierMemberID  uuid.UUID

	route   frValues.Route
	cargo   frValues.CargoInfo
	payment frValues.Payment

	status            values.OrderStatus
	customerCompleted bool
	carrierCompleted  bool

	messages  map[uuid.UUID]*entities.Message
	documents map[uuid.UUID]*entities.Document
	reviews   map[uuid.UUID]*entities.Review

	createdAt   time.Time
	completedAt *time.Time
	cancelledAt *time.Time
}

// New creates a new Order from confirmed offer
func New(
	id uuid.UUID,
	orderNumber int64,
	freightRequestID uuid.UUID,
	offerID uuid.UUID,
	customerOrgID uuid.UUID,
	customerMemberID uuid.UUID,
	carrierOrgID uuid.UUID,
	carrierMemberID uuid.UUID,
	route frValues.Route,
	cargo frValues.CargoInfo,
	payment frValues.Payment,
) *Order {
	o := &Order{
		Base:      aggregate.NewBase(id),
		messages:  make(map[uuid.UUID]*entities.Message),
		documents: make(map[uuid.UUID]*entities.Document),
		reviews:   make(map[uuid.UUID]*entities.Review),
	}

	o.Apply(events.OrderCreated{
		BaseEvent:        eventstore.NewBaseEvent(id, events.AggregateType, o.Version()+1),
		OrderNumber:      orderNumber,
		FreightRequestID: freightRequestID,
		OfferID:          offerID,
		CustomerOrgID:    customerOrgID,
		CustomerMemberID: customerMemberID,
		CarrierOrgID:     carrierOrgID,
		CarrierMemberID:  carrierMemberID,
		Route:            route,
		Cargo:            cargo,
		Payment:          payment,
	})

	return o
}

// NewFromEvents reconstructs Order from events
func NewFromEvents(id uuid.UUID, evts []eventstore.Event) *Order {
	o := &Order{
		Base:      aggregate.NewBase(id),
		messages:  make(map[uuid.UUID]*entities.Message),
		documents: make(map[uuid.UUID]*entities.Document),
		reviews:   make(map[uuid.UUID]*entities.Review),
	}

	for _, evt := range evts {
		o.apply(evt)
		o.Replay(evt)
	}

	return o
}

// Getters
func (o *Order) OrderNumber() int64            { return o.orderNumber }
func (o *Order) FreightRequestID() uuid.UUID   { return o.freightRequestID }
func (o *Order) OfferID() uuid.UUID            { return o.offerID }
func (o *Order) CustomerOrgID() uuid.UUID      { return o.customerOrgID }
func (o *Order) CustomerMemberID() uuid.UUID   { return o.customerMemberID }
func (o *Order) CarrierOrgID() uuid.UUID       { return o.carrierOrgID }
func (o *Order) CarrierMemberID() uuid.UUID    { return o.carrierMemberID }
func (o *Order) Route() frValues.Route         { return o.route }
func (o *Order) Cargo() frValues.CargoInfo     { return o.cargo }
func (o *Order) Payment() frValues.Payment     { return o.payment }
func (o *Order) Status() values.OrderStatus    { return o.status }
func (o *Order) CustomerCompleted() bool       { return o.customerCompleted }
func (o *Order) CarrierCompleted() bool        { return o.carrierCompleted }
func (o *Order) CreatedAt() time.Time          { return o.createdAt }
func (o *Order) CompletedAt() *time.Time       { return o.completedAt }
func (o *Order) CancelledAt() *time.Time       { return o.cancelledAt }

func (o *Order) Messages() map[uuid.UUID]*entities.Message   { return o.messages }
func (o *Order) Documents() map[uuid.UUID]*entities.Document { return o.documents }
func (o *Order) Reviews() map[uuid.UUID]*entities.Review     { return o.reviews }

func (o *Order) GetDocument(id uuid.UUID) (*entities.Document, bool) {
	d, ok := o.documents[id]
	return d, ok
}

func (o *Order) IsParticipant(orgID uuid.UUID) bool {
	return o.customerOrgID == orgID || o.carrierOrgID == orgID
}

func (o *Order) IsCustomer(orgID uuid.UUID) bool {
	return o.customerOrgID == orgID
}

func (o *Order) IsCarrier(orgID uuid.UUID) bool {
	return o.carrierOrgID == orgID
}

func (o *Order) HasReviewFrom(orgID uuid.UUID) bool {
	for _, r := range o.reviews {
		if r.ReviewerOrgID() == orgID {
			return true
		}
	}
	return false
}

// CanAccess проверяет может ли пользователь видеть заказ
// Owner/Admin видят все заказы своей организации
// Обычные сотрудники видят только свои заказы (где они ответственные)
func (o *Order) CanAccess(orgID, memberID uuid.UUID, role string) bool {
	if (role == "owner" || role == "administrator") && o.IsParticipant(orgID) {
		return true
	}
	return o.customerMemberID == memberID || o.carrierMemberID == memberID
}

// Commands

func (o *Order) SendMessage(senderOrgID, senderMemberID uuid.UUID, content string) error {
	if !o.IsParticipant(senderOrgID) {
		return ErrNotOrderParticipant
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return ErrEmptyMessage
	}
	if o.status.IsCancelled() {
		return ErrOrderCancelled
	}

	o.Apply(events.MessageSent{
		BaseEvent:      eventstore.NewBaseEvent(o.ID(), events.AggregateType, o.Version()+1),
		MessageID:      uuid.New(),
		SenderOrgID:    senderOrgID,
		SenderMemberID: senderMemberID,
		Content:        content,
	})

	return nil
}

func (o *Order) AttachDocument(
	uploaderOrgID, uploaderMemberID uuid.UUID,
	name, mimeType string,
	size int64,
	fileID uuid.UUID,
) error {
	if !o.IsParticipant(uploaderOrgID) {
		return ErrNotOrderParticipant
	}
	if o.status.IsFinished() {
		if o.status.IsCancelled() {
			return ErrOrderCancelled
		}
		return ErrOrderCompleted
	}

	o.Apply(events.DocumentAttached{
		BaseEvent:  eventstore.NewBaseEvent(o.ID(), events.AggregateType, o.Version()+1),
		DocumentID: uuid.New(),
		Name:       name,
		MimeType:   mimeType,
		Size:       size,
		FileID:     fileID,
		UploadedBy: uploaderMemberID,
	})

	return nil
}

func (o *Order) RemoveDocument(removerOrgID uuid.UUID, documentID uuid.UUID) error {
	if !o.IsParticipant(removerOrgID) {
		return ErrNotOrderParticipant
	}
	doc, ok := o.documents[documentID]
	if !ok {
		return ErrDocumentNotFound
	}
	// Check if remover is the uploader or from same org
	// For simplicity, allow any participant to remove any document
	_ = doc

	o.Apply(events.DocumentRemoved{
		BaseEvent:  eventstore.NewBaseEvent(o.ID(), events.AggregateType, o.Version()+1),
		DocumentID: documentID,
		RemovedBy:  removerOrgID,
	})

	return nil
}

func (o *Order) Complete(orgID, memberID uuid.UUID) error {
	if !o.IsParticipant(orgID) {
		return ErrNotOrderParticipant
	}
	if o.status.IsCancelled() {
		return ErrOrderCancelled
	}
	if o.status == values.OrderStatusCompleted {
		return ErrOrderCompleted
	}

	isCustomer := o.IsCustomer(orgID)

	if isCustomer {
		if o.customerCompleted {
			return ErrAlreadyCompleted
		}
		o.Apply(events.CustomerCompleted{
			BaseEvent: eventstore.NewBaseEvent(o.ID(), events.AggregateType, o.Version()+1),
			MemberID:  memberID,
		})
	} else {
		if o.carrierCompleted {
			return ErrAlreadyCompleted
		}
		o.Apply(events.CarrierCompleted{
			BaseEvent: eventstore.NewBaseEvent(o.ID(), events.AggregateType, o.Version()+1),
			MemberID:  memberID,
		})
	}

	// Check if both completed after applying the event
	if o.customerCompleted && o.carrierCompleted {
		o.Apply(events.OrderCompleted{
			BaseEvent: eventstore.NewBaseEvent(o.ID(), events.AggregateType, o.Version()+1),
		})
	}

	return nil
}

func (o *Order) Cancel(orgID, memberID uuid.UUID, reason string) error {
	if !o.IsParticipant(orgID) {
		return ErrNotOrderParticipant
	}
	if o.status != values.OrderStatusActive {
		if o.status.IsCancelled() {
			return ErrOrderCancelled
		}
		return ErrCannotCancelAfterComplete
	}

	o.Apply(events.OrderCancelled{
		BaseEvent:           eventstore.NewBaseEvent(o.ID(), events.AggregateType, o.Version()+1),
		CancelledByOrgID:    orgID,
		CancelledByMemberID: memberID,
		Reason:              reason,
	})

	return nil
}

func (o *Order) LeaveReview(reviewerOrgID uuid.UUID, rating int, comment string) error {
	if !o.IsParticipant(reviewerOrgID) {
		return ErrNotOrderParticipant
	}
	if o.status.IsCancelled() {
		return ErrCannotLeaveReview
	}
	// Allow review after own side completed (not waiting for both sides)
	isCustomer := reviewerOrgID == o.customerOrgID
	isCarrier := reviewerOrgID == o.carrierOrgID
	if isCustomer && !o.customerCompleted {
		return ErrCannotLeaveReview
	}
	if isCarrier && !o.carrierCompleted {
		return ErrCannotLeaveReview
	}
	if rating < 1 || rating > 5 {
		return ErrInvalidRating
	}
	if o.HasReviewFrom(reviewerOrgID) {
		return ErrAlreadyLeftReview
	}

	o.Apply(events.ReviewLeft{
		BaseEvent:     eventstore.NewBaseEvent(o.ID(), events.AggregateType, o.Version()+1),
		ReviewID:      uuid.New(),
		ReviewerOrgID: reviewerOrgID,
		Rating:        rating,
		Comment:       comment,
	})

	return nil
}

// CanReassign проверяет можно ли переназначить ответственного
// Разрешено в статусах: active, customer_completed, carrier_completed
func (o *Order) CanReassign() bool {
	switch o.status {
	case values.OrderStatusActive,
		values.OrderStatusCustomerCompleted,
		values.OrderStatusCarrierCompleted:
		return true
	default:
		return false
	}
}

// ReassignCustomerMember переназначает ответственного со стороны заказчика
func (o *Order) ReassignCustomerMember(actorID, newMemberID uuid.UUID) error {
	if !o.CanReassign() {
		return ErrCannotReassignFinishedOrder
	}
	if o.customerMemberID == newMemberID {
		return nil // no-op
	}

	o.Apply(events.CustomerMemberReassigned{
		BaseEvent:    eventstore.NewBaseEvent(o.ID(), events.AggregateType, o.Version()+1),
		OldMemberID:  o.customerMemberID,
		NewMemberID:  newMemberID,
		ReassignedBy: actorID,
	})

	return nil
}

// ReassignCarrierMember переназначает ответственного со стороны перевозчика
func (o *Order) ReassignCarrierMember(actorID, newMemberID uuid.UUID) error {
	if !o.CanReassign() {
		return ErrCannotReassignFinishedOrder
	}
	if o.carrierMemberID == newMemberID {
		return nil // no-op
	}

	o.Apply(events.CarrierMemberReassigned{
		BaseEvent:    eventstore.NewBaseEvent(o.ID(), events.AggregateType, o.Version()+1),
		OldMemberID:  o.carrierMemberID,
		NewMemberID:  newMemberID,
		ReassignedBy: actorID,
	})

	return nil
}

// Apply applies event and records it as change
func (o *Order) Apply(evt eventstore.Event) {
	o.apply(evt)
	o.Base.Apply(evt)
}

// apply updates state from event (used by both Apply and Replay)
func (o *Order) apply(evt eventstore.Event) {
	switch e := evt.(type) {
	case events.OrderCreated:
		o.orderNumber = e.OrderNumber
		o.freightRequestID = e.FreightRequestID
		o.offerID = e.OfferID
		o.customerOrgID = e.CustomerOrgID
		o.customerMemberID = e.CustomerMemberID
		o.carrierOrgID = e.CarrierOrgID
		o.carrierMemberID = e.CarrierMemberID
		o.route = e.Route
		o.cargo = e.Cargo
		o.payment = e.Payment
		o.status = values.OrderStatusActive
		o.createdAt = e.OccurredAt()

	case events.OrderCancelled:
		if o.IsCustomer(e.CancelledByOrgID) {
			o.status = values.OrderStatusCancelledByCustomer
		} else {
			o.status = values.OrderStatusCancelledByCarrier
		}
		now := e.OccurredAt()
		o.cancelledAt = &now

	case events.CustomerCompleted:
		o.customerCompleted = true
		if !o.carrierCompleted {
			o.status = values.OrderStatusCustomerCompleted
		}

	case events.CarrierCompleted:
		o.carrierCompleted = true
		if !o.customerCompleted {
			o.status = values.OrderStatusCarrierCompleted
		}

	case events.OrderCompleted:
		o.status = values.OrderStatusCompleted
		now := e.OccurredAt()
		o.completedAt = &now

	case events.MessageSent:
		msg := entities.NewMessage(
			e.MessageID,
			e.SenderOrgID,
			e.SenderMemberID,
			e.Content,
			e.OccurredAt(),
		)
		o.messages[e.MessageID] = &msg

	case events.DocumentAttached:
		doc := entities.NewDocument(
			e.DocumentID,
			e.Name,
			e.MimeType,
			e.Size,
			e.FileID,
			e.UploadedBy,
			e.OccurredAt(),
		)
		o.documents[e.DocumentID] = &doc

	case events.DocumentRemoved:
		delete(o.documents, e.DocumentID)

	case events.ReviewLeft:
		review := entities.NewReview(
			e.ReviewID,
			e.ReviewerOrgID,
			e.Rating,
			e.Comment,
			e.OccurredAt(),
		)
		o.reviews[e.ReviewID] = &review

	case events.CustomerMemberReassigned:
		o.customerMemberID = e.NewMemberID

	case events.CarrierMemberReassigned:
		o.carrierMemberID = e.NewMemberID
	}
}
