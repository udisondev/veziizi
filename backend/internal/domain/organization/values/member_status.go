//go:generate go-enum --marshal --sql --names --ptr

package values

// MemberStatus represents member's status in organization
// ENUM(active, blocked)
type MemberStatus string

func (s MemberStatus) IsActive() bool {
	return s == MemberStatusActive
}
