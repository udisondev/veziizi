//go:generate go-enum --marshal --sql --names --ptr

package values

// OfferStatus represents the status of an offer
// ENUM(pending, selected, confirmed, rejected, withdrawn, declined)
type OfferStatus string
