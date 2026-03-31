package events

import (
	"github.com/udisondev/veziizi/backend/internal/domain/freightrequest/values"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"github.com/google/uuid"
)

const AggregateType = "freight_request"

// Event type constants
const (
	TypeFreightRequestCreated              = "freight_request.created"
	TypeFreightRequestUpdated              = "freight_request.updated"
	TypeFreightRequestReassigned           = "freight_request.reassigned"
	TypeFreightRequestCancelled            = "freight_request.cancelled"
	TypeFreightRequestExpired              = "freight_request.expired"
	TypeCustomerCompleted                  = "freight_request.customer_completed"
	TypeCarrierCompleted                   = "freight_request.carrier_completed"
	TypeFreightRequestCompleted            = "freight_request.completed"
	TypeReviewLeft                         = "freight_request.review_left"
	TypeReviewEdited                       = "freight_request.review_edited"
	TypeCancelledAfterConfirmed            = "freight_request.cancelled_after_confirmed"
	TypeCarrierMemberReassigned            = "freight_request.carrier_member_reassigned"
	TypeOfferMade                          = "offer.made"
	TypeOfferWithdrawn                     = "offer.withdrawn"
	TypeOfferSelected                      = "offer.selected"
	TypeOfferRejected                      = "offer.rejected"
	TypeOfferConfirmed                     = "offer.confirmed"
	TypeOfferDeclined                      = "offer.declined"
	TypeOfferUnselected                    = "offer.unselected"
	TypeOfferCancelledWithRequest          = "offer.cancelled_with_request"
)

func init() {
	eventstore.RegisterEventType[FreightRequestCreated](TypeFreightRequestCreated)
	eventstore.RegisterEventType[FreightRequestUpdated](TypeFreightRequestUpdated)
	eventstore.RegisterEventType[FreightRequestReassigned](TypeFreightRequestReassigned)
	eventstore.RegisterEventType[FreightRequestCancelled](TypeFreightRequestCancelled)
	eventstore.RegisterEventType[FreightRequestExpired](TypeFreightRequestExpired)
	eventstore.RegisterEventType[CustomerCompleted](TypeCustomerCompleted)
	eventstore.RegisterEventType[CarrierCompleted](TypeCarrierCompleted)
	eventstore.RegisterEventType[FreightRequestCompleted](TypeFreightRequestCompleted)
	eventstore.RegisterEventType[ReviewLeft](TypeReviewLeft)
	eventstore.RegisterEventType[ReviewEdited](TypeReviewEdited)
	eventstore.RegisterEventType[CancelledAfterConfirmed](TypeCancelledAfterConfirmed)
	eventstore.RegisterEventType[CarrierMemberReassigned](TypeCarrierMemberReassigned)
	eventstore.RegisterEventType[OfferMade](TypeOfferMade)
	eventstore.RegisterEventType[OfferWithdrawn](TypeOfferWithdrawn)
	eventstore.RegisterEventType[OfferSelected](TypeOfferSelected)
	eventstore.RegisterEventType[OfferRejected](TypeOfferRejected)
	eventstore.RegisterEventType[OfferConfirmed](TypeOfferConfirmed)
	eventstore.RegisterEventType[OfferDeclined](TypeOfferDeclined)
	eventstore.RegisterEventType[OfferUnselected](TypeOfferUnselected)
	eventstore.RegisterEventType[OfferCancelledWithRequest](TypeOfferCancelledWithRequest)
}

// FreightRequestCreated is emitted when a new freight request is published
type FreightRequestCreated struct {
	eventstore.BaseEvent
	RequestNumber       int64                      `json:"request_number"`
	CustomerOrgID       uuid.UUID                  `json:"customer_org_id"`
	CustomerMemberID    uuid.UUID                  `json:"customer_member_id"`
	Route               values.Route               `json:"route"`
	Cargo               values.CargoInfo           `json:"cargo"`
	VehicleRequirements values.VehicleRequirements `json:"vehicle_requirements"`
	Payment             values.Payment             `json:"payment"`
	Comment             string                     `json:"comment,omitempty"`
	ExpiresAt           int64                      `json:"expires_at"`
}

func (e FreightRequestCreated) EventType() string { return TypeFreightRequestCreated }

// FreightRequestUpdated is emitted when freight request data is updated
// Increments freightVersion in aggregate
type FreightRequestUpdated struct {
	eventstore.BaseEvent
	UpdatedBy           uuid.UUID                   `json:"updated_by"`
	Route               *values.Route               `json:"route,omitempty"`
	Cargo               *values.CargoInfo           `json:"cargo,omitempty"`
	VehicleRequirements *values.VehicleRequirements `json:"vehicle_requirements,omitempty"`
	Payment             *values.Payment             `json:"payment,omitempty"`
	Comment             *string                     `json:"comment,omitempty"`
}

func (e FreightRequestUpdated) EventType() string { return TypeFreightRequestUpdated }

// FreightRequestReassigned is emitted when freight request is reassigned to another member
type FreightRequestReassigned struct {
	eventstore.BaseEvent
	OldMemberID  uuid.UUID `json:"old_member_id"`
	NewMemberID  uuid.UUID `json:"new_member_id"`
	ReassignedBy uuid.UUID `json:"reassigned_by"`
}

func (e FreightRequestReassigned) EventType() string { return TypeFreightRequestReassigned }

// FreightRequestCancelled is emitted when customer cancels freight request
type FreightRequestCancelled struct {
	eventstore.BaseEvent
	CancelledBy uuid.UUID `json:"cancelled_by"`
	Reason      string    `json:"reason,omitempty"`
}

func (e FreightRequestCancelled) EventType() string { return TypeFreightRequestCancelled }

// FreightRequestExpired is emitted when freight request expires
type FreightRequestExpired struct {
	eventstore.BaseEvent
}

func (e FreightRequestExpired) EventType() string { return TypeFreightRequestExpired }

// OfferMade is emitted when carrier makes an offer
type OfferMade struct {
	eventstore.BaseEvent
	OfferID         uuid.UUID            `json:"offer_id"`
	CarrierOrgID    uuid.UUID            `json:"carrier_org_id"`
	CarrierMemberID uuid.UUID            `json:"carrier_member_id"`
	Price           values.Money         `json:"price"`
	Comment         string               `json:"comment,omitempty"`
	FreightVersion  int                  `json:"freight_version"`
	VatType         values.VatType       `json:"vat_type"`
	PaymentMethod   values.PaymentMethod `json:"payment_method"`
}

func (e OfferMade) EventType() string { return TypeOfferMade }

// OfferWithdrawn is emitted when carrier withdraws their offer
type OfferWithdrawn struct {
	eventstore.BaseEvent
	OfferID     uuid.UUID `json:"offer_id"`
	WithdrawnBy uuid.UUID `json:"withdrawn_by"`
	Reason      string    `json:"reason,omitempty"`
}

func (e OfferWithdrawn) EventType() string { return TypeOfferWithdrawn }

// OfferSelected is emitted when customer selects an offer
type OfferSelected struct {
	eventstore.BaseEvent
	OfferID    uuid.UUID `json:"offer_id"`
	SelectedBy uuid.UUID `json:"selected_by"`
}

func (e OfferSelected) EventType() string { return TypeOfferSelected }

// OfferRejected is emitted when customer rejects an offer
type OfferRejected struct {
	eventstore.BaseEvent
	OfferID    uuid.UUID `json:"offer_id"`
	RejectedBy uuid.UUID `json:"rejected_by"`
	Reason     string    `json:"reason,omitempty"`
}

func (e OfferRejected) EventType() string { return TypeOfferRejected }

// OfferConfirmed is emitted when carrier confirms the selected offer
type OfferConfirmed struct {
	eventstore.BaseEvent
	OfferID     uuid.UUID `json:"offer_id"`
	ConfirmedBy uuid.UUID `json:"confirmed_by"`
}

func (e OfferConfirmed) EventType() string { return TypeOfferConfirmed }

// OfferDeclined is emitted when carrier declines the selected offer
type OfferDeclined struct {
	eventstore.BaseEvent
	OfferID    uuid.UUID `json:"offer_id"`
	DeclinedBy uuid.UUID `json:"declined_by"`
	Reason     string    `json:"reason,omitempty"`
}

func (e OfferDeclined) EventType() string { return TypeOfferDeclined }

// OfferUnselected is emitted when customer unselects a previously selected offer
// (carrier did not respond, customer wants to choose another offer)
type OfferUnselected struct {
	eventstore.BaseEvent
	OfferID      uuid.UUID `json:"offer_id"`
	UnselectedBy uuid.UUID `json:"unselected_by"`
	Reason       string    `json:"reason,omitempty"`
}

func (e OfferUnselected) EventType() string { return TypeOfferUnselected }

// OfferCancelledWithRequest is emitted when a selected offer is cancelled
// due to freight request cancellation by customer
type OfferCancelledWithRequest struct {
	eventstore.BaseEvent
	OfferID uuid.UUID `json:"offer_id"`
	Reason  string    `json:"reason,omitempty"`
}

func (e OfferCancelledWithRequest) EventType() string { return TypeOfferCancelledWithRequest }

// CustomerCompleted is emitted when customer confirms freight completion
type CustomerCompleted struct {
	eventstore.BaseEvent
	CompletedBy uuid.UUID `json:"completed_by"`
}

func (e CustomerCompleted) EventType() string { return TypeCustomerCompleted }

// CarrierCompleted is emitted when carrier confirms freight completion
type CarrierCompleted struct {
	eventstore.BaseEvent
	CompletedBy uuid.UUID `json:"completed_by"`
}

func (e CarrierCompleted) EventType() string { return TypeCarrierCompleted }

// FreightRequestCompleted is emitted when both parties have confirmed completion
type FreightRequestCompleted struct {
	eventstore.BaseEvent
	CustomerCompletedAt int64 `json:"customer_completed_at"`
	CarrierCompletedAt  int64 `json:"carrier_completed_at"`
}

func (e FreightRequestCompleted) EventType() string { return TypeFreightRequestCompleted }

// ReviewLeft is emitted when a party leaves a review after completion
type ReviewLeft struct {
	eventstore.BaseEvent
	ReviewID          uuid.UUID `json:"review_id"`
	ReviewerOrgID     uuid.UUID `json:"reviewer_org_id"`
	ReviewerMemberID  uuid.UUID `json:"reviewer_member_id"`
	ReviewedOrgID     uuid.UUID `json:"reviewed_org_id"`
	Rating            int       `json:"rating"`
	Comment           string    `json:"comment,omitempty"`
	// Context for Review aggregate (fraud analysis, weight calculation)
	FreightAmount    int64  `json:"freight_amount"`
	FreightCurrency  string `json:"freight_currency"`
	FreightCreatedAt int64  `json:"freight_created_at"`
	CompletedAt      int64  `json:"completed_at"`
}

func (e ReviewLeft) EventType() string { return TypeReviewLeft }

// ReviewEdited is emitted when a party edits their review (within 24h window)
type ReviewEdited struct {
	eventstore.BaseEvent
	ReviewID      uuid.UUID `json:"review_id"`
	ReviewerOrgID uuid.UUID `json:"reviewer_org_id"`
	OldRating     int       `json:"old_rating"`
	NewRating     int       `json:"new_rating"`
	OldComment    string    `json:"old_comment,omitempty"`
	NewComment    string    `json:"new_comment,omitempty"`
	EditedBy      uuid.UUID `json:"edited_by"`
}

func (e ReviewEdited) EventType() string { return TypeReviewEdited }

// CancelledAfterConfirmed is emitted when freight is cancelled after offer confirmation
type CancelledAfterConfirmed struct {
	eventstore.BaseEvent
	CancelledBy   uuid.UUID `json:"cancelled_by"`
	CancelledRole string    `json:"cancelled_role"` // "customer" or "carrier"
	Reason        string    `json:"reason,omitempty"`
}

func (e CancelledAfterConfirmed) EventType() string { return TypeCancelledAfterConfirmed }

// CarrierMemberReassigned is emitted when carrier's responsible member is reassigned
type CarrierMemberReassigned struct {
	eventstore.BaseEvent
	OldMemberID  uuid.UUID `json:"old_member_id"`
	NewMemberID  uuid.UUID `json:"new_member_id"`
	ReassignedBy uuid.UUID `json:"reassigned_by"`
}

func (e CarrierMemberReassigned) EventType() string { return TypeCarrierMemberReassigned }
