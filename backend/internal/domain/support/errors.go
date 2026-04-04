package support

import "errors"

var (
	ErrTicketNotFound      = errors.New("ticket not found")
	ErrTicketClosed        = errors.New("ticket is closed")
	ErrTicketNotClosed     = errors.New("ticket is not closed")
	ErrNotTicketOwner      = errors.New("not ticket owner")
	ErrEmptyMessage        = errors.New("message content is empty")
	ErrEmptySubject        = errors.New("subject is empty")
	ErrSubjectTooLong      = errors.New("subject is too long (max 255 characters)")
	ErrMessageTooLong      = errors.New("message is too long (max 10000 characters)")
	ErrInvalidSnapshotType = errors.New("invalid snapshot type")
)
