//go:generate go-enum --marshal --sql --names --ptr

package values

// FreightRequestStatus represents the status of a freight request
// ENUM(published, selected, confirmed, partially_completed, completed, cancelled, cancelled_after_confirmed, expired)
type FreightRequestStatus string
