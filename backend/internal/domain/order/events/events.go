package events

import (
	frValues "codeberg.org/udison/veziizi/backend/internal/domain/freightrequest/values"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"github.com/google/uuid"
)

const AggregateType = "order"

// Event type constants
const (
	TypeOrderCreated       = "order.created"
	TypeOrderCancelled     = "order.cancelled"
	TypeCustomerCompleted  = "order.customer_completed"
	TypeCarrierCompleted   = "order.carrier_completed"
	TypeOrderCompleted     = "order.completed"
	TypeMessageSent        = "order.message_sent"
	TypeDocumentAttached   = "order.document_attached"
	TypeDocumentRemoved    = "order.document_removed"
	TypeReviewLeft         = "order.review_left"
)

func init() {
	eventstore.RegisterEventType[OrderCreated](TypeOrderCreated)
	eventstore.RegisterEventType[OrderCancelled](TypeOrderCancelled)
	eventstore.RegisterEventType[CustomerCompleted](TypeCustomerCompleted)
	eventstore.RegisterEventType[CarrierCompleted](TypeCarrierCompleted)
	eventstore.RegisterEventType[OrderCompleted](TypeOrderCompleted)
	eventstore.RegisterEventType[MessageSent](TypeMessageSent)
	eventstore.RegisterEventType[DocumentAttached](TypeDocumentAttached)
	eventstore.RegisterEventType[DocumentRemoved](TypeDocumentRemoved)
	eventstore.RegisterEventType[ReviewLeft](TypeReviewLeft)
}

// OrderCreated is emitted when a new order is created from confirmed offer
type OrderCreated struct {
	eventstore.BaseEvent
	OrderNumber      int64                  `json:"order_number"`
	FreightRequestID uuid.UUID              `json:"freight_request_id"`
	OfferID          uuid.UUID              `json:"offer_id"`
	CustomerOrgID    uuid.UUID              `json:"customer_org_id"`
	CustomerMemberID uuid.UUID              `json:"customer_member_id"`
	CarrierOrgID     uuid.UUID              `json:"carrier_org_id"`
	CarrierMemberID  uuid.UUID              `json:"carrier_member_id"`
	Route            frValues.Route         `json:"route"`
	Cargo            frValues.CargoInfo     `json:"cargo"`
	Payment          frValues.Payment       `json:"payment"`
}

func (e OrderCreated) EventType() string { return TypeOrderCreated }

// OrderCancelled is emitted when order is cancelled by either party
type OrderCancelled struct {
	eventstore.BaseEvent
	CancelledByOrgID    uuid.UUID `json:"cancelled_by_org_id"`
	CancelledByMemberID uuid.UUID `json:"cancelled_by_member_id"`
	Reason              string    `json:"reason,omitempty"`
}

func (e OrderCancelled) EventType() string { return TypeOrderCancelled }

// CustomerCompleted is emitted when customer marks order as completed from their side
type CustomerCompleted struct {
	eventstore.BaseEvent
	MemberID uuid.UUID `json:"member_id"`
}

func (e CustomerCompleted) EventType() string { return TypeCustomerCompleted }

// CarrierCompleted is emitted when carrier marks order as completed from their side
type CarrierCompleted struct {
	eventstore.BaseEvent
	MemberID uuid.UUID `json:"member_id"`
}

func (e CarrierCompleted) EventType() string { return TypeCarrierCompleted }

// OrderCompleted is emitted automatically when both parties have marked completion
type OrderCompleted struct {
	eventstore.BaseEvent
}

func (e OrderCompleted) EventType() string { return TypeOrderCompleted }

// MessageSent is emitted when a message is sent in order chat
type MessageSent struct {
	eventstore.BaseEvent
	MessageID      uuid.UUID `json:"message_id"`
	SenderOrgID    uuid.UUID `json:"sender_org_id"`
	SenderMemberID uuid.UUID `json:"sender_member_id"`
	Content        string    `json:"content"`
}

func (e MessageSent) EventType() string { return TypeMessageSent }

// DocumentAttached is emitted when a document is attached to order
type DocumentAttached struct {
	eventstore.BaseEvent
	DocumentID uuid.UUID `json:"document_id"`
	Name       string    `json:"name"`
	MimeType   string    `json:"mime_type"`
	Size       int64     `json:"size"`
	FileID     uuid.UUID `json:"file_id"`
	UploadedBy uuid.UUID `json:"uploaded_by"`
}

func (e DocumentAttached) EventType() string { return TypeDocumentAttached }

// DocumentRemoved is emitted when a document is removed from order
type DocumentRemoved struct {
	eventstore.BaseEvent
	DocumentID uuid.UUID `json:"document_id"`
	RemovedBy  uuid.UUID `json:"removed_by"`
}

func (e DocumentRemoved) EventType() string { return TypeDocumentRemoved }

// ReviewLeft is emitted when a party leaves a review
type ReviewLeft struct {
	eventstore.BaseEvent
	ReviewID      uuid.UUID `json:"review_id"`
	ReviewerOrgID uuid.UUID `json:"reviewer_org_id"`
	Rating        int       `json:"rating"`
	Comment       string    `json:"comment,omitempty"`
}

func (e ReviewLeft) EventType() string { return TypeReviewLeft }
