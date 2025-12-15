package order

import "errors"

var (
	ErrOrderNotFound           = errors.New("order not found")
	ErrOrderCancelled          = errors.New("order is cancelled")
	ErrOrderCompleted          = errors.New("order is completed")
	ErrOrderNotActive          = errors.New("order is not active")
	ErrNotOrderParticipant     = errors.New("not an order participant")
	ErrAlreadyCompleted        = errors.New("already marked as completed")
	ErrCannotCancelAfterComplete = errors.New("cannot cancel after completion started")
	ErrCannotLeaveReview       = errors.New("can only leave review after order is finished")
	ErrAlreadyLeftReview       = errors.New("already left a review")
	ErrInvalidRating           = errors.New("rating must be between 1 and 5")
	ErrDocumentNotFound        = errors.New("document not found")
	ErrNotDocumentOwner        = errors.New("not document owner")
	ErrEmptyMessage            = errors.New("message content is empty")
)
