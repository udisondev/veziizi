package events

import (
	"codeberg.org/udison/veziizi/backend/internal/domain/freightrequest/values"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"github.com/google/uuid"
)

const AggregateType = "freight_request"

// Event type constants
const (
	TypeFreightRequestCreated    = "freight_request.created"
	TypeFreightRequestUpdated    = "freight_request.updated"
	TypeFreightRequestReassigned = "freight_request.reassigned"
	TypeFreightRequestCancelled  = "freight_request.cancelled"
	TypeFreightRequestExpired    = "freight_request.expired"
	TypeOfferMade                = "offer.made"
	TypeOfferWithdrawn          = "offer.withdrawn"
	TypeOfferSelected           = "offer.selected"
	TypeOfferRejected           = "offer.rejected"
	TypeOfferConfirmed          = "offer.confirmed"
	TypeOfferDeclined           = "offer.declined"
)

func init() {
	eventstore.RegisterEventType[FreightRequestCreated](TypeFreightRequestCreated)
	eventstore.RegisterEventType[FreightRequestUpdated](TypeFreightRequestUpdated)
	eventstore.RegisterEventType[FreightRequestReassigned](TypeFreightRequestReassigned)
	eventstore.RegisterEventType[FreightRequestCancelled](TypeFreightRequestCancelled)
	eventstore.RegisterEventType[FreightRequestExpired](TypeFreightRequestExpired)
	eventstore.RegisterEventType[OfferMade](TypeOfferMade)
	eventstore.RegisterEventType[OfferWithdrawn](TypeOfferWithdrawn)
	eventstore.RegisterEventType[OfferSelected](TypeOfferSelected)
	eventstore.RegisterEventType[OfferRejected](TypeOfferRejected)
	eventstore.RegisterEventType[OfferConfirmed](TypeOfferConfirmed)
	eventstore.RegisterEventType[OfferDeclined](TypeOfferDeclined)
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
