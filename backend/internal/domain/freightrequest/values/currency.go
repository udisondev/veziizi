//go:generate go-enum --marshal --sql --names --ptr

package values

// Currency represents currency codes
// ENUM(RUB, KZT, BYN, EUR, USD)
type Currency string
