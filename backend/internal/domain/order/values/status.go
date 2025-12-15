//go:generate go-enum --marshal --sql --names --ptr

package values

// OrderStatus represents the status of an order
// ENUM(active, customer_completed, carrier_completed, completed, cancelled_by_customer, cancelled_by_carrier)
type OrderStatus string

// IsCancelled returns true if order was cancelled
func (s OrderStatus) IsCancelled() bool {
	return s == OrderStatusCancelledByCustomer || s == OrderStatusCancelledByCarrier
}

// IsFinished returns true if order is in terminal state
func (s OrderStatus) IsFinished() bool {
	return s == OrderStatusCompleted || s.IsCancelled()
}
