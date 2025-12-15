//go:generate go-enum --marshal --sql --names --ptr

package values

// PaymentTerms represents when payment is due
// ENUM(prepaid, on_loading, on_unloading, deferred)
type PaymentTerms string
