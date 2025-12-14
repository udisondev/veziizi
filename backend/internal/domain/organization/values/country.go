//go:generate go-enum --marshal --sql --names --ptr

package values

// Country represents supported countries
// ENUM(RU, KZ, BY)
type Country string
