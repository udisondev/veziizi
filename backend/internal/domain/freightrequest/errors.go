package freightrequest

import "errors"

var (
	ErrFreightRequestNotFound     = errors.New("freight request not found")
	ErrFreightRequestNotPublished = errors.New("freight request is not published")
	ErrFreightRequestExpired      = errors.New("freight request has expired")
	ErrFreightRequestCancelled    = errors.New("freight request is cancelled")
	ErrFreightRequestConfirmed    = errors.New("freight request is confirmed")
	ErrOfferNotFound              = errors.New("offer not found")
	ErrOfferNotPending            = errors.New("offer is not pending")
	ErrOfferNotSelected           = errors.New("offer is not selected")
	ErrOfferAlreadyExists         = errors.New("offer from this carrier already exists")
	ErrCannotOfferOwnRequest      = errors.New("cannot make offer on own freight request")
	ErrNotFreightRequestOwner     = errors.New("not freight request owner")
	ErrNotResponsibleMember       = errors.New("not responsible member for this freight request")
	ErrNotOfferOwner              = errors.New("not offer owner")
	ErrHasSelectedOffer           = errors.New("freight request already has selected offer")
	ErrFreightVersionMismatch     = errors.New("freight version mismatch")
)
