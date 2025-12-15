//go:generate go-enum --marshal --sql --names --ptr

package values

// BodyType represents the type of vehicle body
// ENUM(tent, refrigerator, isothermal, container, openbed, lowbed, jumbo, tank, tipper)
type BodyType string
