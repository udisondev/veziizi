//go:generate go-enum --marshal --sql --names --ptr

package values

// LoadingType represents the type of cargo loading
// ENUM(rear, side, top)
type LoadingType string
