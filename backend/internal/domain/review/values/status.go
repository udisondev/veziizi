package values

// ReviewStatus represents the current state of a review
type ReviewStatus string

const (
	// StatusPendingAnalysis - review just received, waiting for fraud analysis
	StatusPendingAnalysis ReviewStatus = "pending_analysis"

	// StatusPendingModeration - review has fraud signals, waiting for moderator
	StatusPendingModeration ReviewStatus = "pending_moderation"

	// StatusApproved - review approved (by moderator or auto), waiting for activation
	StatusApproved ReviewStatus = "approved"

	// StatusRejected - review rejected by moderator
	StatusRejected ReviewStatus = "rejected"

	// StatusActive - review is active and affects the rating
	StatusActive ReviewStatus = "active"

	// StatusDeactivated - review was invalidated (e.g., reviewer is fraudster)
	StatusDeactivated ReviewStatus = "deactivated"
)

func (s ReviewStatus) String() string {
	return string(s)
}

// IsTerminal returns true if the status is final and cannot change
func (s ReviewStatus) IsTerminal() bool {
	return s == StatusRejected || s == StatusDeactivated
}

// CanTransitionTo checks if transition to target status is valid
func (s ReviewStatus) CanTransitionTo(target ReviewStatus) bool {
	switch s {
	case StatusPendingAnalysis:
		return target == StatusPendingModeration || target == StatusApproved
	case StatusPendingModeration:
		return target == StatusApproved || target == StatusRejected || target == StatusDeactivated
	case StatusApproved:
		return target == StatusActive || target == StatusDeactivated
	case StatusActive:
		return target == StatusDeactivated
	default:
		return false
	}
}
