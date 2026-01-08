package freightrequest

import "errors"

var (
	ErrFreightRequestNotFound      = errors.New("freight request not found")
	ErrFreightRequestNotPublished  = errors.New("freight request is not published")
	ErrFreightRequestExpired       = errors.New("freight request has expired")
	ErrFreightRequestCancelled     = errors.New("freight request is cancelled")
	ErrFreightRequestConfirmed     = errors.New("freight request is confirmed")
	ErrCannotCancelFreightRequest  = errors.New("cannot cancel freight request in current status")
	ErrOfferNotFound               = errors.New("offer not found")
	ErrOfferNotPending             = errors.New("offer is not pending")
	ErrOfferNotSelected            = errors.New("offer is not selected")
	ErrOfferAlreadyExists          = errors.New("offer from this carrier already exists")
	ErrCannotOfferOwnRequest       = errors.New("cannot make offer on own freight request")
	ErrNotFreightRequestOwner      = errors.New("not freight request owner")
	ErrNotResponsibleMember        = errors.New("not responsible member for this freight request")
	ErrNotOfferOwner               = errors.New("not offer owner")
	ErrHasSelectedOffer            = errors.New("freight request already has selected offer")
	ErrFreightRequestNotSelected   = errors.New("freight request has no selected offer")
	ErrFreightVersionMismatch      = errors.New("freight version mismatch")

	// Completion errors
	ErrNotConfirmed                = errors.New("freight request is not confirmed")
	ErrAlreadyCompleted            = errors.New("already completed by this party")
	ErrCannotCompleteNotParticipant = errors.New("not a participant of this freight request")

	// Review errors
	ErrCannotLeaveReview           = errors.New("cannot leave review in current state")
	ErrAlreadyLeftReview           = errors.New("already left a review")
	ErrInvalidRating               = errors.New("rating must be between 1 and 5")
	ErrCannotEditReview            = errors.New("cannot edit review")
	ErrReviewNotFound              = errors.New("review not found")
	ErrReviewEditWindowExpired     = errors.New("review edit window has expired (24 hours)")

	// Cancellation errors
	ErrCannotCancelAfterConfirmed  = errors.New("cannot cancel in current status")
)
