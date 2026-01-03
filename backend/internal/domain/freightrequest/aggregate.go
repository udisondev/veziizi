package freightrequest

import (
	"time"

	"codeberg.org/udison/veziizi/backend/internal/domain/freightrequest/entities"
	"codeberg.org/udison/veziizi/backend/internal/domain/freightrequest/events"
	"codeberg.org/udison/veziizi/backend/internal/domain/freightrequest/values"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"codeberg.org/udison/veziizi/backend/internal/pkg/aggregate"
	"github.com/google/uuid"
)

type FreightRequest struct {
	aggregate.Base

	requestNumber       int64
	customerOrgID       uuid.UUID
	customerMemberID    uuid.UUID
	route               values.Route
	cargo               values.CargoInfo
	vehicleRequirements values.VehicleRequirements
	payment             values.Payment
	comment             string
	status              values.FreightRequestStatus
	freightVersion      int
	expiresAt           time.Time
	createdAt           time.Time
	cancelledAt         *time.Time

	offers         map[uuid.UUID]*entities.Offer
	selectedOffer  *uuid.UUID
}

// New creates a new FreightRequest (published immediately)
func New(
	id uuid.UUID,
	requestNumber int64,
	customerOrgID uuid.UUID,
	customerMemberID uuid.UUID,
	route values.Route,
	cargo values.CargoInfo,
	vehicleRequirements values.VehicleRequirements,
	payment values.Payment,
	comment string,
	expiresAt time.Time,
) *FreightRequest {
	fr := &FreightRequest{
		Base:   aggregate.NewBase(id),
		offers: make(map[uuid.UUID]*entities.Offer),
	}

	fr.Apply(events.FreightRequestCreated{
		BaseEvent:           eventstore.NewBaseEvent(id, events.AggregateType, fr.Version()+1),
		RequestNumber:       requestNumber,
		CustomerOrgID:       customerOrgID,
		CustomerMemberID:    customerMemberID,
		Route:               route,
		Cargo:               cargo,
		VehicleRequirements: vehicleRequirements,
		Payment:             payment,
		Comment:             comment,
		ExpiresAt:           expiresAt.Unix(),
	})

	return fr
}

// NewFromEvents reconstructs FreightRequest from events
func NewFromEvents(id uuid.UUID, evts []eventstore.Event) *FreightRequest {
	fr := &FreightRequest{
		Base:   aggregate.NewBase(id),
		offers: make(map[uuid.UUID]*entities.Offer),
	}

	for _, evt := range evts {
		fr.apply(evt)
		fr.Replay(evt)
	}

	return fr
}

// Getters
func (f *FreightRequest) RequestNumber() int64                      { return f.requestNumber }
func (f *FreightRequest) CustomerOrgID() uuid.UUID                  { return f.customerOrgID }
func (f *FreightRequest) CustomerMemberID() uuid.UUID               { return f.customerMemberID }
func (f *FreightRequest) Route() values.Route                       { return f.route }
func (f *FreightRequest) Cargo() values.CargoInfo                   { return f.cargo }
func (f *FreightRequest) VehicleRequirements() values.VehicleRequirements { return f.vehicleRequirements }
func (f *FreightRequest) Payment() values.Payment                   { return f.payment }
func (f *FreightRequest) Comment() string                           { return f.comment }
func (f *FreightRequest) Status() values.FreightRequestStatus       { return f.status }
func (f *FreightRequest) FreightVersion() int                       { return f.freightVersion }
func (f *FreightRequest) ExpiresAt() time.Time                      { return f.expiresAt }
func (f *FreightRequest) CreatedAt() time.Time                      { return f.createdAt }
func (f *FreightRequest) CancelledAt() *time.Time                   { return f.cancelledAt }
func (f *FreightRequest) Offers() map[uuid.UUID]*entities.Offer     { return f.offers }
func (f *FreightRequest) SelectedOfferID() *uuid.UUID               { return f.selectedOffer }

func (f *FreightRequest) GetOffer(id uuid.UUID) (*entities.Offer, bool) {
	o, ok := f.offers[id]
	return o, ok
}

func (f *FreightRequest) IsPublished() bool {
	return f.status == values.FreightRequestStatusPublished
}

func (f *FreightRequest) IsExpired() bool {
	return f.status == values.FreightRequestStatusExpired || time.Now().After(f.expiresAt)
}

func (f *FreightRequest) CanAcceptOffers() bool {
	return f.status == values.FreightRequestStatusPublished ||
		f.status == values.FreightRequestStatusSelected
}

func (f *FreightRequest) HasSelectedOffer() bool {
	return f.selectedOffer != nil
}

// Commands

func (f *FreightRequest) Update(
	actorID uuid.UUID,
	route *values.Route,
	cargo *values.CargoInfo,
	vehicleRequirements *values.VehicleRequirements,
	payment *values.Payment,
	comment *string,
) error {
	if f.customerMemberID != actorID {
		return ErrNotFreightRequestOwner
	}
	if !f.IsPublished() {
		return ErrFreightRequestNotPublished
	}

	f.Apply(events.FreightRequestUpdated{
		BaseEvent:           eventstore.NewBaseEvent(f.ID(), events.AggregateType, f.Version()+1),
		UpdatedBy:           actorID,
		Route:               route,
		Cargo:               cargo,
		VehicleRequirements: vehicleRequirements,
		Payment:             payment,
		Comment:             comment,
	})

	return nil
}

func (f *FreightRequest) Cancel(actorID uuid.UUID, reason string) error {
	if f.customerMemberID != actorID {
		return ErrNotFreightRequestOwner
	}

	// Разрешаем отмену для published и selected
	if f.status != values.FreightRequestStatusPublished &&
		f.status != values.FreightRequestStatusSelected {
		return ErrCannotCancelFreightRequest
	}

	// 1. Отклоняем все pending офферы
	for offerID, offer := range f.offers {
		if offer.IsPending() {
			f.Apply(events.OfferRejected{
				BaseEvent:  eventstore.NewBaseEvent(f.ID(), events.AggregateType, f.Version()+1),
				OfferID:    offerID,
				RejectedBy: actorID,
				Reason:     "freight request cancelled",
			})
		}
	}

	// 2. Отклоняем selected оффер (если есть)
	if f.selectedOffer != nil {
		f.Apply(events.OfferCancelledWithRequest{
			BaseEvent: eventstore.NewBaseEvent(f.ID(), events.AggregateType, f.Version()+1),
			OfferID:   *f.selectedOffer,
			Reason:    reason,
		})
	}

	// 3. Отменяем заявку
	f.Apply(events.FreightRequestCancelled{
		BaseEvent:   eventstore.NewBaseEvent(f.ID(), events.AggregateType, f.Version()+1),
		CancelledBy: actorID,
		Reason:      reason,
	})

	return nil
}

func (f *FreightRequest) Reassign(actorID uuid.UUID, newMemberID uuid.UUID) error {
	if f.status == values.FreightRequestStatusCancelled {
		return ErrFreightRequestCancelled
	}
	if f.status == values.FreightRequestStatusConfirmed {
		return ErrFreightRequestConfirmed
	}
	if f.customerMemberID == newMemberID {
		return nil // no-op
	}

	f.Apply(events.FreightRequestReassigned{
		BaseEvent:    eventstore.NewBaseEvent(f.ID(), events.AggregateType, f.Version()+1),
		OldMemberID:  f.customerMemberID,
		NewMemberID:  newMemberID,
		ReassignedBy: actorID,
	})

	return nil
}

func (f *FreightRequest) Expire() error {
	if !f.IsPublished() {
		return ErrFreightRequestNotPublished
	}

	f.Apply(events.FreightRequestExpired{
		BaseEvent: eventstore.NewBaseEvent(f.ID(), events.AggregateType, f.Version()+1),
	})

	return nil
}

func (f *FreightRequest) MakeOffer(
	offerID uuid.UUID,
	carrierOrgID uuid.UUID,
	carrierMemberID uuid.UUID,
	price values.Money,
	comment string,
	vatType values.VatType,
	paymentMethod values.PaymentMethod,
) error {
	if !f.CanAcceptOffers() {
		return ErrFreightRequestNotPublished
	}
	if f.IsExpired() {
		return ErrFreightRequestExpired
	}
	if f.customerOrgID == carrierOrgID {
		return ErrCannotOfferOwnRequest
	}

	// Check if carrier already has an active offer
	for _, offer := range f.offers {
		if offer.CarrierOrgID() == carrierOrgID && offer.IsPending() {
			return ErrOfferAlreadyExists
		}
	}

	f.Apply(events.OfferMade{
		BaseEvent:       eventstore.NewBaseEvent(f.ID(), events.AggregateType, f.Version()+1),
		OfferID:         offerID,
		CarrierOrgID:    carrierOrgID,
		CarrierMemberID: carrierMemberID,
		Price:           price,
		Comment:         comment,
		FreightVersion:  f.freightVersion,
		VatType:         vatType,
		PaymentMethod:   paymentMethod,
	})

	return nil
}

func (f *FreightRequest) WithdrawOffer(offerID uuid.UUID, actorOrgID uuid.UUID, reason string) error {
	offer, ok := f.offers[offerID]
	if !ok {
		return ErrOfferNotFound
	}
	if offer.CarrierOrgID() != actorOrgID {
		return ErrNotOfferOwner
	}
	if !offer.IsPending() {
		return ErrOfferNotPending
	}

	f.Apply(events.OfferWithdrawn{
		BaseEvent:   eventstore.NewBaseEvent(f.ID(), events.AggregateType, f.Version()+1),
		OfferID:     offerID,
		WithdrawnBy: actorOrgID,
		Reason:      reason,
	})

	return nil
}

func (f *FreightRequest) SelectOffer(offerID uuid.UUID, actorID uuid.UUID, actorOrgID uuid.UUID) error {
	// Проверка: актор должен быть из организации-заказчика
	if f.customerOrgID != actorOrgID {
		return ErrNotFreightRequestOwner
	}

	// Проверка: только ответственный член может выбирать офферы
	if f.customerMemberID != actorID {
		return ErrNotResponsibleMember
	}

	if !f.IsPublished() {
		return ErrFreightRequestNotPublished
	}
	if f.HasSelectedOffer() {
		return ErrHasSelectedOffer
	}

	offer, ok := f.offers[offerID]
	if !ok {
		return ErrOfferNotFound
	}
	if !offer.IsPending() {
		return ErrOfferNotPending
	}

	f.Apply(events.OfferSelected{
		BaseEvent:  eventstore.NewBaseEvent(f.ID(), events.AggregateType, f.Version()+1),
		OfferID:    offerID,
		SelectedBy: actorID,
	})

	return nil
}

func (f *FreightRequest) RejectOffer(offerID uuid.UUID, actorID uuid.UUID, actorOrgID uuid.UUID, reason string) error {
	// Проверка: актор должен быть из организации-заказчика
	if f.customerOrgID != actorOrgID {
		return ErrNotFreightRequestOwner
	}

	// Проверка: только ответственный член может отклонять офферы
	if f.customerMemberID != actorID {
		return ErrNotResponsibleMember
	}

	offer, ok := f.offers[offerID]
	if !ok {
		return ErrOfferNotFound
	}
	if !offer.IsPending() {
		return ErrOfferNotPending
	}

	f.Apply(events.OfferRejected{
		BaseEvent:  eventstore.NewBaseEvent(f.ID(), events.AggregateType, f.Version()+1),
		OfferID:    offerID,
		RejectedBy: actorID,
		Reason:     reason,
	})

	return nil
}

func (f *FreightRequest) ConfirmOffer(offerID uuid.UUID, actorMemberID uuid.UUID, actorOrgID uuid.UUID, actorRole string) error {
	offer, ok := f.offers[offerID]
	if !ok {
		return ErrOfferNotFound
	}
	if offer.CarrierOrgID() != actorOrgID {
		return ErrNotOfferOwner
	}
	// Проверка: владелец, администратор организации или создатель предложения
	isOwnerOrAdmin := actorRole == "owner" || actorRole == "administrator"
	isCreator := offer.CarrierMemberID() == actorMemberID
	if !isOwnerOrAdmin && !isCreator {
		return ErrNotResponsibleMember
	}
	if !offer.IsSelected() {
		return ErrOfferNotSelected
	}

	f.Apply(events.OfferConfirmed{
		BaseEvent:   eventstore.NewBaseEvent(f.ID(), events.AggregateType, f.Version()+1),
		OfferID:     offerID,
		ConfirmedBy: actorOrgID,
	})

	return nil
}

func (f *FreightRequest) DeclineOffer(offerID uuid.UUID, actorMemberID uuid.UUID, actorOrgID uuid.UUID, actorRole string, reason string) error {
	offer, ok := f.offers[offerID]
	if !ok {
		return ErrOfferNotFound
	}
	if offer.CarrierOrgID() != actorOrgID {
		return ErrNotOfferOwner
	}
	// Проверка: владелец, администратор организации или создатель предложения
	isOwnerOrAdmin := actorRole == "owner" || actorRole == "administrator"
	isCreator := offer.CarrierMemberID() == actorMemberID
	if !isOwnerOrAdmin && !isCreator {
		return ErrNotResponsibleMember
	}
	if !offer.IsSelected() {
		return ErrOfferNotSelected
	}

	f.Apply(events.OfferDeclined{
		BaseEvent:  eventstore.NewBaseEvent(f.ID(), events.AggregateType, f.Version()+1),
		OfferID:    offerID,
		DeclinedBy: actorOrgID,
		Reason:     reason,
	})

	return nil
}

func (f *FreightRequest) UnselectOffer(offerID uuid.UUID, actorID uuid.UUID, actorOrgID uuid.UUID, reason string) error {
	// Проверка: актор должен быть из организации-заказчика
	if f.customerOrgID != actorOrgID {
		return ErrNotFreightRequestOwner
	}

	// Проверка: только ответственный член может отменять выбор офферов
	if f.customerMemberID != actorID {
		return ErrNotResponsibleMember
	}

	// Проверка: заявка должна быть в статусе selected
	if f.status != values.FreightRequestStatusSelected {
		return ErrFreightRequestNotSelected
	}

	// Проверка: оффер должен существовать и быть выбранным
	offer, ok := f.offers[offerID]
	if !ok {
		return ErrOfferNotFound
	}
	if !offer.IsSelected() {
		return ErrOfferNotSelected
	}

	// Проверка: это должен быть именно выбранный оффер
	if f.selectedOffer == nil || *f.selectedOffer != offerID {
		return ErrOfferNotSelected
	}

	f.Apply(events.OfferUnselected{
		BaseEvent:    eventstore.NewBaseEvent(f.ID(), events.AggregateType, f.Version()+1),
		OfferID:      offerID,
		UnselectedBy: actorID,
		Reason:       reason,
	})

	return nil
}

// Apply applies event and records it as change
func (f *FreightRequest) Apply(evt eventstore.Event) {
	f.apply(evt)
	f.Base.Apply(evt)
}

// apply updates state from event (used by both Apply and Replay)
func (f *FreightRequest) apply(evt eventstore.Event) {
	switch e := evt.(type) {
	case events.FreightRequestCreated:
		f.requestNumber = e.RequestNumber
		f.customerOrgID = e.CustomerOrgID
		f.customerMemberID = e.CustomerMemberID
		f.route = e.Route
		f.cargo = e.Cargo
		f.vehicleRequirements = e.VehicleRequirements
		f.payment = e.Payment
		f.comment = e.Comment
		f.status = values.FreightRequestStatusPublished
		f.freightVersion = 1
		f.expiresAt = time.Unix(e.ExpiresAt, 0)
		f.createdAt = e.OccurredAt()

	case events.FreightRequestUpdated:
		if e.Route != nil {
			f.route = *e.Route
		}
		if e.Cargo != nil {
			f.cargo = *e.Cargo
		}
		if e.VehicleRequirements != nil {
			f.vehicleRequirements = *e.VehicleRequirements
		}
		if e.Payment != nil {
			f.payment = *e.Payment
		}
		if e.Comment != nil {
			f.comment = *e.Comment
		}
		f.freightVersion++

	case events.FreightRequestCancelled:
		f.status = values.FreightRequestStatusCancelled
		now := e.OccurredAt()
		f.cancelledAt = &now

	case events.FreightRequestExpired:
		f.status = values.FreightRequestStatusExpired

	case events.FreightRequestReassigned:
		f.customerMemberID = e.NewMemberID

	case events.OfferMade:
		offer := entities.NewOffer(
			e.OfferID,
			e.CarrierOrgID,
			e.CarrierMemberID,
			e.Price,
			e.Comment,
			e.FreightVersion,
			e.VatType,
			e.PaymentMethod,
			e.OccurredAt(),
		)
		f.offers[e.OfferID] = &offer

	case events.OfferWithdrawn:
		if offer, ok := f.offers[e.OfferID]; ok {
			// In Replay, events are already validated - ignore FSM errors
			_ = offer.Withdraw()
		}

	case events.OfferSelected:
		if offer, ok := f.offers[e.OfferID]; ok {
			_ = offer.Select()
			f.selectedOffer = &e.OfferID
			f.status = values.FreightRequestStatusSelected
		}

	case events.OfferRejected:
		if offer, ok := f.offers[e.OfferID]; ok {
			_ = offer.Reject()
		}

	case events.OfferConfirmed:
		if offer, ok := f.offers[e.OfferID]; ok {
			_ = offer.Confirm()
			f.status = values.FreightRequestStatusConfirmed
		}

	case events.OfferDeclined:
		if offer, ok := f.offers[e.OfferID]; ok {
			_ = offer.Decline()
			f.selectedOffer = nil
			f.status = values.FreightRequestStatusPublished
		}

	case events.OfferUnselected:
		if offer, ok := f.offers[e.OfferID]; ok {
			_ = offer.Unselect()
			f.selectedOffer = nil
			f.status = values.FreightRequestStatusPublished
		}

	case events.OfferCancelledWithRequest:
		if offer, ok := f.offers[e.OfferID]; ok {
			_ = offer.CancelWithRequest()
		}
		f.selectedOffer = nil
	}
}
