package values

// FraudSignalType represents types of detected anomalies
type FraudSignalType string

const (
	SignalMutualReviews    FraudSignalType = "mutual_reviews"
	SignalFastCompletion   FraudSignalType = "fast_completion"
	SignalPerfectRatings   FraudSignalType = "perfect_ratings"
	SignalNewOrgBurst      FraudSignalType = "new_org_burst"
	SignalSameIP           FraudSignalType = "same_ip"
	SignalSameFingerprint  FraudSignalType = "same_fingerprint"
	SignalTextSimilarity   FraudSignalType = "review_text_similarity"
	SignalTimingPattern    FraudSignalType = "review_timing_pattern"
	SignalRatingManipulation FraudSignalType = "rating_manipulation"
	SignalBurstAfterLow    FraudSignalType = "burst_after_low_rating"
	SignalDormantReviewer  FraudSignalType = "dormant_reviewer"
)

func (s FraudSignalType) String() string {
	return string(s)
}

// Severity represents the severity level of a fraud signal
type Severity string

const (
	SeverityLow    Severity = "low"
	SeverityMedium Severity = "medium"
	SeverityHigh   Severity = "high"
)

func (s Severity) String() string {
	return string(s)
}

// DefaultSeverity returns the default severity for a signal type
func (s FraudSignalType) DefaultSeverity() Severity {
	switch s {
	case SignalMutualReviews, SignalSameIP, SignalSameFingerprint,
		SignalTextSimilarity, SignalRatingManipulation:
		return SeverityHigh
	case SignalFastCompletion, SignalPerfectRatings, SignalNewOrgBurst,
		SignalTimingPattern, SignalBurstAfterLow, SignalDormantReviewer:
		return SeverityMedium
	default:
		return SeverityLow
	}
}

// DefaultScoreImpact returns the default fraud score impact for a signal type
func (s FraudSignalType) DefaultScoreImpact() float64 {
	switch s {
	case SignalMutualReviews:
		return 0.4
	case SignalSameIP, SignalSameFingerprint:
		return 0.5
	case SignalFastCompletion:
		return 0.2
	case SignalPerfectRatings:
		return 0.15
	case SignalNewOrgBurst:
		return 0.25
	case SignalTextSimilarity:
		return 0.4
	case SignalRatingManipulation:
		return 0.45
	case SignalTimingPattern, SignalDormantReviewer:
		return 0.2
	case SignalBurstAfterLow:
		return 0.25
	default:
		return 0.1
	}
}
