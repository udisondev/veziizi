package values

import "errors"

var (
	ErrNegativeAmount   = errors.New("amount cannot be negative")
	ErrCurrencyMismatch = errors.New("currency mismatch")
)

// Money represents a monetary amount with currency
type Money struct {
	Amount   int64    `json:"amount"`
	Currency Currency `json:"currency"`
}

// NewMoney creates a new Money value. Returns error if amount is negative.
func NewMoney(amount int64, currency Currency) (Money, error) {
	if amount < 0 {
		return Money{}, ErrNegativeAmount
	}
	return Money{Amount: amount, Currency: currency}, nil
}

// MustNewMoney creates a new Money value, panics if amount is negative.
// Use only for known valid values (e.g., constants, tests).
func MustNewMoney(amount int64, currency Currency) Money {
	m, err := NewMoney(amount, currency)
	if err != nil {
		panic(err)
	}
	return m
}

// ZeroMoney returns a zero Money value with the given currency.
func ZeroMoney(currency Currency) Money {
	return Money{Amount: 0, Currency: currency}
}

func (m Money) IsZero() bool {
	return m.Amount == 0
}

// IsPositive returns true if amount is greater than zero.
func (m Money) IsPositive() bool {
	return m.Amount > 0
}

// Add adds two Money values. Returns error if currencies don't match.
func (m Money) Add(other Money) (Money, error) {
	if m.Currency != other.Currency {
		return Money{}, ErrCurrencyMismatch
	}
	return Money{Amount: m.Amount + other.Amount, Currency: m.Currency}, nil
}

// Subtract subtracts other from m. Returns error if currencies don't match or result is negative.
func (m Money) Subtract(other Money) (Money, error) {
	if m.Currency != other.Currency {
		return Money{}, ErrCurrencyMismatch
	}
	result := m.Amount - other.Amount
	if result < 0 {
		return Money{}, ErrNegativeAmount
	}
	return Money{Amount: result, Currency: m.Currency}, nil
}

// Equals compares two Money values (amount and currency).
func (m Money) Equals(other Money) bool {
	return m.Amount == other.Amount && m.Currency == other.Currency
}

// GreaterThan returns true if m is greater than other (same currency required).
func (m Money) GreaterThan(other Money) bool {
	return m.Currency == other.Currency && m.Amount > other.Amount
}

// LessThan returns true if m is less than other (same currency required).
func (m Money) LessThan(other Money) bool {
	return m.Currency == other.Currency && m.Amount < other.Amount
}
