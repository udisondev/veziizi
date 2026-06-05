//go:generate go-enum --marshal --sql --names --ptr

package transport

// LoadingType represents the type of cargo loading
// ENUM(rear, side, top, full_untarp)
type LoadingType string
