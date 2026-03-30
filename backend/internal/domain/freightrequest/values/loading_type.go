//go:generate go-enum --marshal --sql --names --ptr

package values

// LoadingType represents the type of cargo loading
// ENUM(rear, side, top, full_untarp)
type LoadingType string
