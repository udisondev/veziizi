//go:generate go-enum --marshal --sql --names --ptr

package values

// InvitationStatus represents invitation lifecycle status
// ENUM(pending, accepted, expired, cancelled)
type InvitationStatus string

func (s InvitationStatus) CanBeAccepted() bool {
	return s == InvitationStatusPending
}

func (s InvitationStatus) CanBeCancelled() bool {
	return s == InvitationStatusPending
}

func (s InvitationStatus) IsCancelled() bool {
	return s == InvitationStatusCancelled
}
