package eventstore

import "errors"

var (
	ErrAggregateNotFound      = errors.New("aggregate not found")
	ErrConcurrentModification = errors.New("concurrent modification detected")
	ErrEventVersionConflict   = errors.New("event version conflict")
)
