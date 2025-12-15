//go:generate go-enum --marshal --sql --names --ptr

package values

// PaymentMethod represents how payment will be made
// ENUM(bank_transfer, cash, card)
type PaymentMethod string
