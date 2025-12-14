//go:generate go-enum --marshal --sql --names --ptr

package values

// InvitationStatus represents invitation lifecycle status
// ENUM(pending, accepted, expired)
type InvitationStatus string

func (s InvitationStatus) CanBeAccepted() bool {
	return s == InvitationStatusPending
}
