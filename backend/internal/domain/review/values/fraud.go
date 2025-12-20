package values

// FraudSignalType represents types of detected anomalies
type FraudSignalType string

const (
	// SignalMutualReviews - organizations exchange reviews too frequently (>5/month)
	SignalMutualReviews FraudSignalType = "mutual_reviews"

	// SignalFastCompletion - order completed too quickly (<2 hours)
	SignalFastCompletion FraudSignalType = "fast_completion"

	// SignalPerfectRatings - reviewer always gives 5 stars to this org (>3 reviews)
	SignalPerfectRatings FraudSignalType = "perfect_ratings"

	// SignalNewOrgBurst - new org received too many reviews in first week (>10)
	SignalNewOrgBurst FraudSignalType = "new_org_burst"

	// SignalSameIP - organizations registered from same IP address
	SignalSameIP FraudSignalType = "same_ip"

	// SignalSameFingerprint - organizations registered from same device
	SignalSameFingerprint FraudSignalType = "same_fingerprint"

	// SignalTextSimilarity - reviewer leaves similar/identical review texts (>80% match)
	SignalTextSimilarity FraudSignalType = "review_text_similarity"

	// SignalTimingPattern - reviews always posted at same time of day (bot behavior)
	SignalTimingPattern FraudSignalType = "review_timing_pattern"

	// SignalRatingManipulation - 5★ to friends, 1★ to competitors
	SignalRatingManipulation FraudSignalType = "rating_manipulation"

	// SignalBurstAfterLow - burst of 5★ reviews right after a low rating
	SignalBurstAfterLow FraudSignalType = "burst_after_low_rating"

	// SignalDormantReviewer - dormant org suddenly active with many reviews
	SignalDormantReviewer FraudSignalType = "dormant_reviewer"
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

// FraudThresholds contains configurable thresholds for fraud detection
var FraudThresholds = struct {
	MutualReviewsPerMonth     int     // >5 mutual reviews per month
	FastCompletionHours       int     // <2 hours is suspicious
	PerfectRatingsCount       int     // >3 perfect ratings from same reviewer
	NewOrgBurstReviewsPerWeek int     // >10 reviews in first week
	ModerationScoreThreshold  float64 // fraud_score > 0.3 requires moderation
	ActivationDelayDays       int     // 7 days for normal reviews
	SuspiciousDelayDays       int     // 14 days for suspicious reviews

	// New thresholds
	TextSimilarityThreshold   float64 // >0.8 similarity is suspicious
	TextSimilarityMinReviews  int     // minimum reviews with similar text
	TimingPatternWindowHours  int     // reviews within X hours window = bot
	TimingPatternMinReviews   int     // minimum reviews to detect pattern
	RatingManipFriendAvgMin   float64 // avg rating to "friends" >= 4.5
	RatingManipOtherAvgMax    float64 // avg rating to "others" <= 2.5
	RatingManipMinFriendReviews int   // min mutual reviews to be "friend"
	BurstAfterLowDays         int     // check 5★ burst within X days after low
	BurstAfterLowCount        int     // >5 five-star reviews = burst
	DormantDays               int     // >90 days inactive = dormant
	DormantBurstCount         int     // >5 reviews after dormancy = suspicious
}{
	MutualReviewsPerMonth:     5,
	FastCompletionHours:       2,
	PerfectRatingsCount:       3,
	NewOrgBurstReviewsPerWeek: 10,
	ModerationScoreThreshold:  0.3,
	ActivationDelayDays:       7,
	SuspiciousDelayDays:       14,

	TextSimilarityThreshold:   0.8,
	TextSimilarityMinReviews:  3,
	TimingPatternWindowHours:  2,
	TimingPatternMinReviews:   10,
	RatingManipFriendAvgMin:   4.5,
	RatingManipOtherAvgMax:    2.5,
	RatingManipMinFriendReviews: 3,
	BurstAfterLowDays:         7,
	BurstAfterLowCount:        5,
	DormantDays:               90,
	DormantBurstCount:         5,
}
