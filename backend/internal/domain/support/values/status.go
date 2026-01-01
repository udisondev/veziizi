//go:generate go-enum --marshal --sql --names --ptr

package values

// TicketStatus represents the status of a support ticket
// ENUM(open, answered, awaiting_reply, closed)
type TicketStatus string

// IsOpen returns true if ticket is not closed
func (s TicketStatus) IsOpen() bool {
	return s != TicketStatusClosed
}

// IsClosed returns true if ticket is closed
func (s TicketStatus) IsClosed() bool {
	return s == TicketStatusClosed
}
