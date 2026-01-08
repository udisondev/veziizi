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

	offers        map[uuid.UUID]*entities.Offer
	selectedOffer *uuid.UUID

	// After offer confirmed
	confirmedAt      *time.Time
	confirmedOfferID *uuid.UUID
	carrierOrgID     *uuid.UUID
	carrierMemberID  *uuid.UUID

	// Completion
	customerCompleted   bool
	customerCompletedAt *time.Time
	carrierCompleted    bool
	carrierCompletedAt  *time.Time
	completedAt         *time.Time

	// Cancellation after confirmed
	cancelledAfterConfirmedAt     *time.Time
	cancelledAfterConfirmedBy     *uuid.UUID
	cancelledAfterConfirmedReason string

	// Reviews
	customerReview *entities.Review
	carrierReview  *entities.Review
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
func (f *FreightRequest) ConfirmedAt() *time.Time                   { return f.confirmedAt }
func (f *FreightRequest) ConfirmedOfferID() *uuid.UUID              { return f.confirmedOfferID }
func (f *FreightRequest) CarrierOrgID() *uuid.UUID                  { return f.carrierOrgID }
func (f *FreightRequest) CarrierMemberID() *uuid.UUID               { return f.carrierMemberID }
func (f *FreightRequest) CustomerCompleted() bool                   { return f.customerCompleted }
func (f *FreightRequest) CustomerCompletedAt() *time.Time           { return f.customerCompletedAt }
func (f *FreightRequest) CarrierCompleted() bool                    { return f.carrierCompleted }
func (f *FreightRequest) CarrierCompletedAt() *time.Time            { return f.carrierCompletedAt }
func (f *FreightRequest) CompletedAt() *time.Time                   { return f.completedAt }
func (f *FreightRequest) CancelledAfterConfirmedAt() *time.Time     { return f.cancelledAfterConfirmedAt }
func (f *FreightRequest) CustomerReview() *entities.Review          { return f.customerReview }
func (f *FreightRequest) CarrierReview() *entities.Review           { return f.carrierReview }

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

func (f *FreightRequest) IsConfirmed() bool {
	return f.status == values.FreightRequestStatusConfirmed
}

func (f *FreightRequest) IsPartiallyCompleted() bool {
	return f.status == values.FreightRequestStatusPartiallyCompleted
}

func (f *FreightRequest) IsCompleted() bool {
	return f.status == values.FreightRequestStatusCompleted
}

func (f *FreightRequest) IsCancelledAfterConfirmed() bool {
	return f.status == values.FreightRequestStatusCancelledAfterConfirmed
}

func (f *FreightRequest) IsFinished() bool {
	return f.IsCompleted() || f.IsCancelledAfterConfirmed() ||
		f.status == values.FreightRequestStatusCancelled
}

func (f *FreightRequest) CanComplete() bool {
	return f.IsConfirmed() || f.IsPartiallyCompleted()
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
	// Запрещаем переназначение только для терминальных статусов
	if f.status == values.FreightRequestStatusCancelled ||
		f.status == values.FreightRequestStatusCancelledAfterConfirmed ||
		f.status == values.FreightRequestStatusCompleted {
		return ErrFreightRequestCancelled
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

	// Отклоняем все другие pending офферы
	for otherOfferID, otherOffer := range f.offers {
		if otherOfferID != offerID && otherOffer.IsPending() {
			f.Apply(events.OfferRejected{
				BaseEvent:  eventstore.NewBaseEvent(f.ID(), events.AggregateType, f.Version()+1),
				OfferID:    otherOfferID,
				RejectedBy: actorOrgID,
				Reason:     "another offer was confirmed",
			})
		}
	}

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

// Complete marks freight as completed by one party
func (f *FreightRequest) Complete(orgID, memberID uuid.UUID) error {
	if !f.CanComplete() {
		return ErrNotConfirmed
	}

	isCustomer := f.customerOrgID == orgID
	isCarrier := f.carrierOrgID != nil && *f.carrierOrgID == orgID

	if !isCustomer && !isCarrier {
		return ErrCannotCompleteNotParticipant
	}

	if isCustomer {
		// Проверка что это ответственный заказчика
		if f.customerMemberID != memberID {
			return ErrNotResponsibleMember
		}
		if f.customerCompleted {
			return ErrAlreadyCompleted
		}
		f.Apply(events.CustomerCompleted{
			BaseEvent:   eventstore.NewBaseEvent(f.ID(), events.AggregateType, f.Version()+1),
			CompletedBy: memberID,
		})
	} else {
		// Проверка что это ответственный перевозчика
		if f.carrierMemberID == nil || *f.carrierMemberID != memberID {
			return ErrNotResponsibleMember
		}
		if f.carrierCompleted {
			return ErrAlreadyCompleted
		}
		f.Apply(events.CarrierCompleted{
			BaseEvent:   eventstore.NewBaseEvent(f.ID(), events.AggregateType, f.Version()+1),
			CompletedBy: memberID,
		})
	}

	// Check if both parties have completed
	if f.customerCompleted && f.carrierCompleted {
		f.Apply(events.FreightRequestCompleted{
			BaseEvent:           eventstore.NewBaseEvent(f.ID(), events.AggregateType, f.Version()+1),
			CustomerCompletedAt: f.customerCompletedAt.Unix(),
			CarrierCompletedAt:  f.carrierCompletedAt.Unix(),
		})
	}

	return nil
}

// LeaveReview leaves a review for the other party
func (f *FreightRequest) LeaveReview(reviewID uuid.UUID, reviewerOrgID uuid.UUID, memberID uuid.UUID, rating int, comment string) error {
	if rating < 1 || rating > 5 {
		return ErrInvalidRating
	}

	isCustomer := f.customerOrgID == reviewerOrgID
	isCarrier := f.carrierOrgID != nil && *f.carrierOrgID == reviewerOrgID

	if !isCustomer && !isCarrier {
		return ErrCannotLeaveReview
	}

	// Customer can leave review only after completing their part
	if isCustomer {
		if !f.customerCompleted {
			return ErrCannotLeaveReview
		}
		if f.customerReview != nil {
			return ErrAlreadyLeftReview
		}
	}

	// Carrier can leave review only after completing their part
	if isCarrier {
		if !f.carrierCompleted {
			return ErrCannotLeaveReview
		}
		if f.carrierReview != nil {
			return ErrAlreadyLeftReview
		}
	}

	var reviewedOrgID uuid.UUID
	if isCustomer {
		reviewedOrgID = *f.carrierOrgID
	} else {
		reviewedOrgID = f.customerOrgID
	}

	// Безопасное получение цены (может быть nil если заявка без ставки)
	var freightAmount int64
	var freightCurrency string
	if f.payment.Price != nil {
		freightAmount = f.payment.Price.Amount
		freightCurrency = string(f.payment.Price.Currency)
	}

	f.Apply(events.ReviewLeft{
		BaseEvent:        eventstore.NewBaseEvent(f.ID(), events.AggregateType, f.Version()+1),
		ReviewID:         reviewID,
		ReviewerOrgID:    reviewerOrgID,
		ReviewedOrgID:    reviewedOrgID,
		Rating:           rating,
		Comment:          comment,
		FreightAmount:    freightAmount,
		FreightCurrency:  freightCurrency,
		FreightCreatedAt: f.createdAt.Unix(),
		CompletedAt:      time.Now().Unix(),
	})

	return nil
}

// EditReview edits a previously left review (only within 24h window)
func (f *FreightRequest) EditReview(reviewerOrgID uuid.UUID, memberID uuid.UUID, rating int, comment string) error {
	if rating < 1 || rating > 5 {
		return ErrInvalidRating
	}

	isCustomer := f.customerOrgID == reviewerOrgID
	isCarrier := f.carrierOrgID != nil && *f.carrierOrgID == reviewerOrgID

	if !isCustomer && !isCarrier {
		return ErrCannotEditReview
	}

	var existingReview *entities.Review
	if isCustomer {
		existingReview = f.customerReview
	} else {
		existingReview = f.carrierReview
	}

	if existingReview == nil {
		return ErrReviewNotFound
	}

	if !existingReview.CanEdit() {
		return ErrReviewEditWindowExpired
	}

	f.Apply(events.ReviewEdited{
		BaseEvent:     eventstore.NewBaseEvent(f.ID(), events.AggregateType, f.Version()+1),
		ReviewID:      existingReview.ID(),
		ReviewerOrgID: reviewerOrgID,
		OldRating:     existingReview.Rating(),
		NewRating:     rating,
		OldComment:    existingReview.Comment(),
		NewComment:    comment,
		EditedBy:      memberID,
	})

	return nil
}

// CancelAfterConfirmed cancels freight after offer was confirmed
func (f *FreightRequest) CancelAfterConfirmed(orgID, memberID uuid.UUID, reason string) error {
	if !f.CanComplete() {
		return ErrCannotCancelAfterConfirmed
	}

	isCustomer := f.customerOrgID == orgID
	isCarrier := f.carrierOrgID != nil && *f.carrierOrgID == orgID

	if !isCustomer && !isCarrier {
		return ErrCannotCompleteNotParticipant
	}

	role := "customer"
	if isCarrier {
		role = "carrier"
	}

	f.Apply(events.CancelledAfterConfirmed{
		BaseEvent:     eventstore.NewBaseEvent(f.ID(), events.AggregateType, f.Version()+1),
		CancelledBy:   memberID,
		CancelledRole: role,
		Reason:        reason,
	})

	return nil
}

// ReassignCarrierMember reassigns the carrier's responsible member
func (f *FreightRequest) ReassignCarrierMember(actorID uuid.UUID, newMemberID uuid.UUID, actorRole string) error {
	if !f.CanComplete() {
		return ErrNotConfirmed
	}
	if f.carrierMemberID == nil {
		return ErrNotConfirmed
	}
	if *f.carrierMemberID == newMemberID {
		return nil // no-op
	}

	// Only owner/admin of carrier org can reassign
	isOwnerOrAdmin := actorRole == "owner" || actorRole == "administrator"
	if !isOwnerOrAdmin {
		return ErrNotResponsibleMember
	}

	f.Apply(events.CarrierMemberReassigned{
		BaseEvent:    eventstore.NewBaseEvent(f.ID(), events.AggregateType, f.Version()+1),
		OldMemberID:  *f.carrierMemberID,
		NewMemberID:  newMemberID,
		ReassignedBy: actorID,
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
			f.confirmedOfferID = &e.OfferID
			carrierOrgID := offer.CarrierOrgID()
			f.carrierOrgID = &carrierOrgID
			carrierMemberID := offer.CarrierMemberID()
			f.carrierMemberID = &carrierMemberID
			now := e.OccurredAt()
			f.confirmedAt = &now
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

	case events.CustomerCompleted:
		f.customerCompleted = true
		now := e.OccurredAt()
		f.customerCompletedAt = &now
		if !f.carrierCompleted {
			f.status = values.FreightRequestStatusPartiallyCompleted
		}

	case events.CarrierCompleted:
		f.carrierCompleted = true
		now := e.OccurredAt()
		f.carrierCompletedAt = &now
		if !f.customerCompleted {
			f.status = values.FreightRequestStatusPartiallyCompleted
		}

	case events.FreightRequestCompleted:
		f.status = values.FreightRequestStatusCompleted
		now := e.OccurredAt()
		f.completedAt = &now

	case events.ReviewLeft:
		review := entities.NewReview(
			e.ReviewID,
			e.ReviewerOrgID,
			e.Rating,
			e.Comment,
			e.OccurredAt(),
		)
		if e.ReviewerOrgID == f.customerOrgID {
			f.customerReview = &review
		} else {
			f.carrierReview = &review
		}

	case events.ReviewEdited:
		if e.ReviewerOrgID == f.customerOrgID && f.customerReview != nil {
			updated := f.customerReview.WithUpdatedRatingAndComment(e.NewRating, e.NewComment)
			f.customerReview = &updated
		} else if f.carrierReview != nil {
			updated := f.carrierReview.WithUpdatedRatingAndComment(e.NewRating, e.NewComment)
			f.carrierReview = &updated
		}

	case events.CancelledAfterConfirmed:
		f.status = values.FreightRequestStatusCancelledAfterConfirmed
		now := e.OccurredAt()
		f.cancelledAfterConfirmedAt = &now
		f.cancelledAfterConfirmedBy = &e.CancelledBy
		f.cancelledAfterConfirmedReason = e.Reason

	case events.CarrierMemberReassigned:
		f.carrierMemberID = &e.NewMemberID
	}
}
