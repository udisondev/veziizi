package values

import (
	"errors"
	"fmt"
)

// ErrDeferredDaysMustBePositive is returned when deferred_days is not positive for deferred payment terms
var ErrDeferredDaysMustBePositive = errors.New("deferred_days must be greater than 0")

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
		return fmt.Errorf("validate payment: %w", ErrNegativeAmount)
	}

	if p.Terms == PaymentTermsDeferred && p.DeferredDays <= 0 {
		return fmt.Errorf("validate payment: %w", ErrDeferredDaysMustBePositive)
	}

	return nil
}
