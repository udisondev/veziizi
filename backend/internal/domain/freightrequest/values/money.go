package values

// Money represents a monetary amount with currency
type Money struct {
	Amount   int64    `json:"amount"`
	Currency Currency `json:"currency"`
}

func NewMoney(amount int64, currency Currency) Money {
	return Money{Amount: amount, Currency: currency}
}

func (m Money) IsZero() bool {
	return m.Amount == 0
}
