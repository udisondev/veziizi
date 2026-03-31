package freightrequest

import (
	"errors"
	"time"

	"github.com/udisondev/veziizi/backend/internal/domain/freightrequest/entities"
	"github.com/udisondev/veziizi/backend/internal/domain/freightrequest/events"
	"github.com/udisondev/veziizi/backend/internal/domain/freightrequest/values"
	orgValues "github.com/udisondev/veziizi/backend/internal/domain/organization/values"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"github.com/udisondev/veziizi/backend/internal/pkg/aggregate"
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

	offers      map[uuid.UUID]*entities.Offer
	offersCache []*entities.Offer
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
func (f *FreightRequest) OffersList() []*entities.Offer {
	if f.offersCache == nil {
		f.offersCache = make([]*entities.Offer, 0, len(f.offers))
		for _, o := range f.offers {
			f.offersCache = append(f.offersCache, o)
		}
	}
	return f.offersCache
}
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

func (f *FreightRequest) Reassign(actorID uuid.UUID, newMemberID uuid.UUID, actorRole orgValues.MemberRole) error {
	// Только owner/admin может переназначать
	if actorRole != orgValues.MemberRoleOwner && actorRole != orgValues.MemberRoleAdministrator {
		return ErrNotResponsibleMember
	}
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

func (f *FreightRequest) WithdrawOffer(offerID uuid.UUID, actorMemberID uuid.UUID, actorOrgID uuid.UUID, actorRole orgValues.MemberRole, reason string) error {
	offer, ok := f.offers[offerID]
	if !ok {
		return ErrOfferNotFound
	}
	if offer.CarrierOrgID() != actorOrgID {
		return ErrNotOfferOwner
	}
	// Создатель оффера или owner/admin организации
	isCreator := offer.CarrierMemberID() == actorMemberID
	isOwnerOrAdmin := actorRole == orgValues.MemberRoleOwner || actorRole == orgValues.MemberRoleAdministrator
	if !isCreator && !isOwnerOrAdmin {
		return ErrNotResponsibleMember
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

func (f *FreightRequest) ConfirmOffer(offerID uuid.UUID, actorMemberID uuid.UUID, actorOrgID uuid.UUID, actorRole orgValues.MemberRole) error {
	offer, ok := f.offers[offerID]
	if !ok {
		return ErrOfferNotFound
	}
	if offer.CarrierOrgID() != actorOrgID {
		return ErrNotOfferOwner
	}
	// Проверка: владелец, администратор организации или создатель предложения
	isOwnerOrAdmin := actorRole == orgValues.MemberRoleOwner || actorRole == orgValues.MemberRoleAdministrator
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

func (f *FreightRequest) DeclineOffer(offerID uuid.UUID, actorMemberID uuid.UUID, actorOrgID uuid.UUID, actorRole orgValues.MemberRole, reason string) error {
	offer, ok := f.offers[offerID]
	if !ok {
		return ErrOfferNotFound
	}
	if offer.CarrierOrgID() != actorOrgID {
		return ErrNotOfferOwner
	}
	// Проверка: владелец, администратор организации или создатель предложения
	isOwnerOrAdmin := actorRole == orgValues.MemberRoleOwner || actorRole == orgValues.MemberRoleAdministrator
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
		if f.carrierOrgID == nil {
			return ErrNotConfirmed
		}
		reviewedOrgID = *f.carrierOrgID
	} else {
		reviewedOrgID = f.customerOrgID
	}

	// Получаем цену из подтверждённого оффера (реальная согласованная цена)
	var freightAmount int64
	var freightCurrency string
	if f.confirmedOfferID != nil {
		if offer, ok := f.offers[*f.confirmedOfferID]; ok {
			freightAmount = offer.Price().Amount
			freightCurrency = string(offer.Price().Currency)
		}
	}

	// Используем реальное время завершения перевозки
	var completedAt int64
	if f.completedAt != nil {
		completedAt = f.completedAt.Unix()
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
		CompletedAt:      completedAt,
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
func (f *FreightRequest) ReassignCarrierMember(actorID uuid.UUID, newMemberID uuid.UUID, actorRole orgValues.MemberRole) error {
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
	isOwnerOrAdmin := actorRole == orgValues.MemberRoleOwner || actorRole == orgValues.MemberRoleAdministrator
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
	f.offersCache = nil

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

// =====================================
// Snapshot support for efficient loading
// =====================================

// FreightRequestSnapshot represents serializable state of FreightRequest aggregate
type FreightRequestSnapshot struct {
	ID                            uuid.UUID                      `json:"id"`
	Version                       int64                          `json:"version"`
	RequestNumber                 int64                          `json:"request_number"`
	CustomerOrgID                 uuid.UUID                      `json:"customer_org_id"`
	CustomerMemberID              uuid.UUID                      `json:"customer_member_id"`
	Route                         values.Route                   `json:"route"`
	Cargo                         values.CargoInfo               `json:"cargo"`
	VehicleRequirements           values.VehicleRequirements     `json:"vehicle_requirements"`
	Payment                       values.Payment                 `json:"payment"`
	Comment                       string                         `json:"comment"`
	Status                        values.FreightRequestStatus    `json:"status"`
	FreightVersion                int                            `json:"freight_version"`
	ExpiresAt                     time.Time                      `json:"expires_at"`
	CreatedAt                     time.Time                      `json:"created_at"`
	CancelledAt                   *time.Time                     `json:"cancelled_at,omitempty"`
	Offers                        map[uuid.UUID]OfferSnapshot    `json:"offers"`
	SelectedOffer                 *uuid.UUID                     `json:"selected_offer,omitempty"`
	ConfirmedAt                   *time.Time                     `json:"confirmed_at,omitempty"`
	ConfirmedOfferID              *uuid.UUID                     `json:"confirmed_offer_id,omitempty"`
	CarrierOrgID                  *uuid.UUID                     `json:"carrier_org_id,omitempty"`
	CarrierMemberID               *uuid.UUID                     `json:"carrier_member_id,omitempty"`
	CustomerCompleted             bool                           `json:"customer_completed"`
	CustomerCompletedAt           *time.Time                     `json:"customer_completed_at,omitempty"`
	CarrierCompleted              bool                           `json:"carrier_completed"`
	CarrierCompletedAt            *time.Time                     `json:"carrier_completed_at,omitempty"`
	CompletedAt                   *time.Time                     `json:"completed_at,omitempty"`
	CancelledAfterConfirmedAt     *time.Time                     `json:"cancelled_after_confirmed_at,omitempty"`
	CancelledAfterConfirmedBy     *uuid.UUID                     `json:"cancelled_after_confirmed_by,omitempty"`
	CancelledAfterConfirmedReason string                         `json:"cancelled_after_confirmed_reason,omitempty"`
	CustomerReview                *ReviewSnapshot                `json:"customer_review,omitempty"`
	CarrierReview                 *ReviewSnapshot                `json:"carrier_review,omitempty"`
}

// OfferSnapshot represents serializable state of Offer entity
type OfferSnapshot struct {
	ID              uuid.UUID            `json:"id"`
	CarrierOrgID    uuid.UUID            `json:"carrier_org_id"`
	CarrierMemberID uuid.UUID            `json:"carrier_member_id"`
	Price           values.Money         `json:"price"`
	Comment         string               `json:"comment"`
	FreightVersion  int                  `json:"freight_version"`
	VatType         values.VatType       `json:"vat_type"`
	PaymentMethod   values.PaymentMethod `json:"payment_method"`
	Status          values.OfferStatus   `json:"status"`
	CreatedAt       time.Time            `json:"created_at"`
}

// ReviewSnapshot represents serializable state of Review entity in FreightRequest
type ReviewSnapshot struct {
	ID            uuid.UUID `json:"id"`
	ReviewerOrgID uuid.UUID `json:"reviewer_org_id"`
	Rating        int       `json:"rating"`
	Comment       string    `json:"comment"`
	CreatedAt     time.Time `json:"created_at"`
}

// State returns current aggregate state for snapshot storage.
// Implements aggregate.Snapshotable interface.
func (f *FreightRequest) State() any {
	offers := make(map[uuid.UUID]OfferSnapshot, len(f.offers))
	for id, o := range f.offers {
		offers[id] = OfferSnapshot{
			ID:              o.ID(),
			CarrierOrgID:    o.CarrierOrgID(),
			CarrierMemberID: o.CarrierMemberID(),
			Price:           o.Price(),
			Comment:         o.Comment(),
			FreightVersion:  o.FreightVersion(),
			VatType:         o.VatType(),
			PaymentMethod:   o.PaymentMethod(),
			Status:          o.Status(),
			CreatedAt:       o.CreatedAt(),
		}
	}

	var customerReview, carrierReview *ReviewSnapshot
	if f.customerReview != nil {
		customerReview = &ReviewSnapshot{
			ID:            f.customerReview.ID(),
			ReviewerOrgID: f.customerReview.ReviewerOrgID(),
			Rating:        f.customerReview.Rating(),
			Comment:       f.customerReview.Comment(),
			CreatedAt:     f.customerReview.CreatedAt(),
		}
	}
	if f.carrierReview != nil {
		carrierReview = &ReviewSnapshot{
			ID:            f.carrierReview.ID(),
			ReviewerOrgID: f.carrierReview.ReviewerOrgID(),
			Rating:        f.carrierReview.Rating(),
			Comment:       f.carrierReview.Comment(),
			CreatedAt:     f.carrierReview.CreatedAt(),
		}
	}

	return FreightRequestSnapshot{
		ID:                            f.ID(),
		Version:                       f.Version(),
		RequestNumber:                 f.requestNumber,
		CustomerOrgID:                 f.customerOrgID,
		CustomerMemberID:              f.customerMemberID,
		Route:                         f.route,
		Cargo:                         f.cargo,
		VehicleRequirements:           f.vehicleRequirements,
		Payment:                       f.payment,
		Comment:                       f.comment,
		Status:                        f.status,
		FreightVersion:                f.freightVersion,
		ExpiresAt:                     f.expiresAt,
		CreatedAt:                     f.createdAt,
		CancelledAt:                   f.cancelledAt,
		Offers:                        offers,
		SelectedOffer:                 f.selectedOffer,
		ConfirmedAt:                   f.confirmedAt,
		ConfirmedOfferID:              f.confirmedOfferID,
		CarrierOrgID:                  f.carrierOrgID,
		CarrierMemberID:               f.carrierMemberID,
		CustomerCompleted:             f.customerCompleted,
		CustomerCompletedAt:           f.customerCompletedAt,
		CarrierCompleted:              f.carrierCompleted,
		CarrierCompletedAt:            f.carrierCompletedAt,
		CompletedAt:                   f.completedAt,
		CancelledAfterConfirmedAt:     f.cancelledAfterConfirmedAt,
		CancelledAfterConfirmedBy:     f.cancelledAfterConfirmedBy,
		CancelledAfterConfirmedReason: f.cancelledAfterConfirmedReason,
		CustomerReview:                customerReview,
		CarrierReview:                 carrierReview,
	}
}

// FromSnapshot restores aggregate from snapshot state.
// Implements aggregate.Snapshotable interface.
func (f *FreightRequest) FromSnapshot(state any) error {
	snap, ok := state.(FreightRequestSnapshot)
	if !ok {
		return ErrInvalidSnapshotType
	}

	f.Base.SetID(snap.ID)
	f.Base.SetVersion(snap.Version)

	f.requestNumber = snap.RequestNumber
	f.customerOrgID = snap.CustomerOrgID
	f.customerMemberID = snap.CustomerMemberID
	f.route = snap.Route
	f.cargo = snap.Cargo
	f.vehicleRequirements = snap.VehicleRequirements
	f.payment = snap.Payment
	f.comment = snap.Comment
	f.status = snap.Status
	f.freightVersion = snap.FreightVersion
	f.expiresAt = snap.ExpiresAt
	f.createdAt = snap.CreatedAt
	f.cancelledAt = snap.CancelledAt
	f.selectedOffer = snap.SelectedOffer
	f.confirmedAt = snap.ConfirmedAt
	f.confirmedOfferID = snap.ConfirmedOfferID
	f.carrierOrgID = snap.CarrierOrgID
	f.carrierMemberID = snap.CarrierMemberID
	f.customerCompleted = snap.CustomerCompleted
	f.customerCompletedAt = snap.CustomerCompletedAt
	f.carrierCompleted = snap.CarrierCompleted
	f.carrierCompletedAt = snap.CarrierCompletedAt
	f.completedAt = snap.CompletedAt
	f.cancelledAfterConfirmedAt = snap.CancelledAfterConfirmedAt
	f.cancelledAfterConfirmedBy = snap.CancelledAfterConfirmedBy
	f.cancelledAfterConfirmedReason = snap.CancelledAfterConfirmedReason

	f.offers = make(map[uuid.UUID]*entities.Offer, len(snap.Offers))
	for id, os := range snap.Offers {
		o := entities.NewOffer(
			os.ID,
			os.CarrierOrgID,
			os.CarrierMemberID,
			os.Price,
			os.Comment,
			os.FreightVersion,
			os.VatType,
			os.PaymentMethod,
			os.CreatedAt,
		)
		// Restore status via state transitions
		restoreOfferStatus(&o, os.Status)
		f.offers[id] = &o
	}

	if snap.CustomerReview != nil {
		r := entities.NewReview(
			snap.CustomerReview.ID,
			snap.CustomerReview.ReviewerOrgID,
			snap.CustomerReview.Rating,
			snap.CustomerReview.Comment,
			snap.CustomerReview.CreatedAt,
		)
		f.customerReview = &r
	}

	if snap.CarrierReview != nil {
		r := entities.NewReview(
			snap.CarrierReview.ID,
			snap.CarrierReview.ReviewerOrgID,
			snap.CarrierReview.Rating,
			snap.CarrierReview.Comment,
			snap.CarrierReview.CreatedAt,
		)
		f.carrierReview = &r
	}

	return nil
}

// restoreOfferStatus sets offer to target status
func restoreOfferStatus(o *entities.Offer, target values.OfferStatus) {
	switch target {
	case values.OfferStatusSelected:
		_ = o.Select()
	case values.OfferStatusConfirmed:
		_ = o.Select()
		_ = o.Confirm()
	case values.OfferStatusRejected:
		_ = o.Reject()
	case values.OfferStatusWithdrawn:
		_ = o.Withdraw()
	case values.OfferStatusDeclined:
		_ = o.Select()
		_ = o.Decline()
	}
}

// NewFromSnapshot creates FreightRequest from snapshot state.
func NewFromSnapshot(id uuid.UUID, state any) (*FreightRequest, error) {
	fr := &FreightRequest{
		Base:   aggregate.NewBase(id),
		offers: make(map[uuid.UUID]*entities.Offer),
	}

	if err := fr.FromSnapshot(state); err != nil {
		return nil, err
	}

	return fr, nil
}

// ErrInvalidSnapshotType is returned when snapshot type doesn't match
var ErrInvalidSnapshotType = errors.New("invalid snapshot type")
