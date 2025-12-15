//go:generate go-enum --marshal --sql --names --ptr

package values

// ADRClass represents ADR dangerous goods classification
// ENUM(none, class1, class2, class3, class4, class5, class6, class7, class8, class9)
type ADRClass string
