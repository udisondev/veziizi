//go:generate go-enum --marshal --sql --names --ptr

package values

// FreightRequestStatus represents the status of a freight request
// ENUM(published, selected, confirmed, cancelled, expired)
type FreightRequestStatus string
