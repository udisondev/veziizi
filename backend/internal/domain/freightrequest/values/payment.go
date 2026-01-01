package values

import "fmt"

// Payment represents payment information for a freight request
type Payment struct {
	Price        *Money        `json:"price,omitempty"` // Optional: if nil, carriers propose their own price
	VatType      VatType       `json:"vat_type"`
	Method       PaymentMethod `json:"method"`
	Terms        PaymentTerms  `json:"terms"`
	DeferredDays int           `json:"deferred_days,omitempty"`
}

// Validate validates payment information
func (p Payment) Validate() error {
	if p.Price != nil && p.Price.Amount < 0 {
		return fmt.Errorf("price cannot be negative")
	}
	return nil
}
