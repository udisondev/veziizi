package entities

import (
	"time"

	"codeberg.org/udison/veziizi/backend/internal/domain/freightrequest/values"
	"github.com/google/uuid"
)

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

func (o *Offer) Select()   { o.status = values.OfferStatusSelected }
func (o *Offer) Confirm()  { o.status = values.OfferStatusConfirmed }
func (o *Offer) Reject()   { o.status = values.OfferStatusRejected }
func (o *Offer) Withdraw() { o.status = values.OfferStatusWithdrawn }
func (o *Offer) Decline()  { o.status = values.OfferStatusDeclined }
