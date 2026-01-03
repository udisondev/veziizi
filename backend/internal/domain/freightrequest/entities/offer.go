package entities

import (
	"errors"
	"time"

	"codeberg.org/udison/veziizi/backend/internal/domain/freightrequest/values"
	"github.com/google/uuid"
)

// ErrInvalidStatusTransition is returned when an invalid offer status transition is attempted
var ErrInvalidStatusTransition = errors.New("invalid offer status transition")

type Offer struct {
	id              uuid.UUID
	carrierOrgID    uuid.UUID
	carrierMemberID uuid.UUID
	price           values.Money
	comment         string
	freightVersion  int
	vatType         values.VatType
	paymentMethod   values.PaymentMethod
	status          values.OfferStatus
	createdAt       time.Time
}

func NewOffer(
	id uuid.UUID,
	carrierOrgID uuid.UUID,
	carrierMemberID uuid.UUID,
	price values.Money,
	comment string,
	freightVersion int,
	vatType values.VatType,
	paymentMethod values.PaymentMethod,
	createdAt time.Time,
) Offer {
	return Offer{
		id:              id,
		carrierOrgID:    carrierOrgID,
		carrierMemberID: carrierMemberID,
		price:           price,
		comment:         comment,
		freightVersion:  freightVersion,
		vatType:         vatType,
		paymentMethod:   paymentMethod,
		status:          values.OfferStatusPending,
		createdAt:       createdAt,
	}
}

func (o Offer) ID() uuid.UUID                      { return o.id }
func (o Offer) CarrierOrgID() uuid.UUID            { return o.carrierOrgID }
func (o Offer) CarrierMemberID() uuid.UUID         { return o.carrierMemberID }
func (o Offer) Price() values.Money                { return o.price }
func (o Offer) Comment() string                    { return o.comment }
func (o Offer) FreightVersion() int                { return o.freightVersion }
func (o Offer) VatType() values.VatType            { return o.vatType }
func (o Offer) PaymentMethod() values.PaymentMethod { return o.paymentMethod }
func (o Offer) Status() values.OfferStatus         { return o.status }
func (o Offer) CreatedAt() time.Time               { return o.createdAt }

func (o Offer) IsPending() bool   { return o.status == values.OfferStatusPending }
func (o Offer) IsSelected() bool  { return o.status == values.OfferStatusSelected }
func (o Offer) IsConfirmed() bool { return o.status == values.OfferStatusConfirmed }

// Select transitions offer from pending to selected (customer selects offer)
func (o *Offer) Select() error {
	if !o.IsPending() {
		return ErrInvalidStatusTransition
	}
	o.status = values.OfferStatusSelected
	return nil
}

// Confirm transitions offer from selected to confirmed (carrier confirms)
func (o *Offer) Confirm() error {
	if !o.IsSelected() {
		return ErrInvalidStatusTransition
	}
	o.status = values.OfferStatusConfirmed
	return nil
}

// Reject transitions offer from pending to rejected (customer rejects offer)
func (o *Offer) Reject() error {
	if !o.IsPending() {
		return ErrInvalidStatusTransition
	}
	o.status = values.OfferStatusRejected
	return nil
}

// Withdraw transitions offer from pending to withdrawn (carrier withdraws offer)
func (o *Offer) Withdraw() error {
	if !o.IsPending() {
		return ErrInvalidStatusTransition
	}
	o.status = values.OfferStatusWithdrawn
	return nil
}

// Decline transitions offer from selected to declined (carrier declines)
func (o *Offer) Decline() error {
	if !o.IsSelected() {
		return ErrInvalidStatusTransition
	}
	o.status = values.OfferStatusDeclined
	return nil
}

// CancelWithRequest transitions offer from selected to rejected
// when the freight request is cancelled by customer
func (o *Offer) CancelWithRequest() error {
	if !o.IsSelected() {
		return ErrInvalidStatusTransition
	}
	o.status = values.OfferStatusRejected
	return nil
}

// Unselect transitions offer from selected back to pending
// when customer unselects the offer (carrier did not respond)
func (o *Offer) Unselect() error {
	if !o.IsSelected() {
		return ErrInvalidStatusTransition
	}
	o.status = values.OfferStatusPending
	return nil
}
