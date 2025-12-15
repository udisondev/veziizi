//go:generate go-enum --marshal --sql --names --ptr

package values

// CargoType represents the type of cargo
// ENUM(general, bulk, liquid, refrigerated, dangerous, oversized, container)
type CargoType string
