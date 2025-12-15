//go:generate go-enum --marshal --sql --names --ptr

package values

// VatType represents VAT inclusion type
// ENUM(included, excluded, none)
type VatType string
