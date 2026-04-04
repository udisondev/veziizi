package review

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/udisondev/veziizi/backend/internal/domain/review"
	"github.com/udisondev/veziizi/backend/internal/domain/review/events"
	"github.com/udisondev/veziizi/backend/internal/domain/review/values"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/projections"
)

// Analyzer performs fraud detection and weight calculation for reviews
type Analyzer struct {
	fraudData *projections.FraudDataProjection
	members   *projections.MembersProjection
}

func NewAnalyzer(
	fraudData *projections.FraudDataProjection,
	members *projections.MembersProjection,
) *Analyzer {
	return &Analyzer{
		fraudData: fraudData,
		members:   members,
	}
}

// AnalysisResult contains the results of fraud analysis
type AnalysisResult struct {
	RawWeight          float64
	FraudSignals       []events.FraudSignal
	FraudScore         float64
	RequiresModeration bool
	ActivationDate     time.Time
}

// Analyze performs fraud analysis on a review
func (a *Analyzer) Analyze(ctx context.Context, r *review.Review) (*AnalysisResult, error) {
	slog.Info("starting fraud analysis",
		slog.String("review_id", r.ID().String()),
		slog.String("reviewer_org_id", r.ReviewerOrgID().String()),
		slog.String("reviewed_org_id", r.ReviewedOrgID().String()),
	)

	// Calculate weight components
	orderAmountWeight := a.calculateOrderAmountWeight(r.OrderAmount())
	orgAgeWeight, err := a.calculateOrgAgeWeight(ctx, r.ReviewerOrgID())
	if err != nil {
		slog.Warn("failed to calculate org age weight, using default",
			slog.String("error", err.Error()),
		)
		orgAgeWeight = 0.5
	}

	diversityWeight, err := a.calculateDiversityWeight(ctx, r.ReviewerOrgID(), r.ReviewedOrgID())
	if err != nil {
		slog.Warn("failed to calculate diversity weight, using default",
			slog.String("error", err.Error()),
		)
		diversityWeight = 1.0
	}

	reputationWeight, err := a.calculateReputationWeight(ctx, r.ReviewerOrgID())
	if err != nil {
		slog.Warn("failed to calculate reputation weight, using default",
			slog.String("error", err.Error()),
		)
		reputationWeight = 1.0
	}

	// Calculate raw weight
	rawWeight := orderAmountWeight * orgAgeWeight * diversityWeight * reputationWeight

	slog.Info("calculated weight components",
		slog.Float64("order_amount_weight", orderAmountWeight),
		slog.Float64("org_age_weight", orgAgeWeight),
		slog.Float64("diversity_weight", diversityWeight),
		slog.Float64("reputation_weight", reputationWeight),
		slog.Float64("raw_weight", rawWeight),
	)

	// Detect fraud signals
	signals, err := a.detectFraudSignals(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("detect fraud signals: %w", err)
	}

	// Calculate fraud score (sum of signal impacts, capped at 1.0)
	var fraudScore float64
	for _, signal := range signals {
		fraudScore += signal.ScoreImpact
	}
	if fraudScore > 1.0 {
		fraudScore = 1.0
	}

	// Determine if moderation is required
	requiresModeration := fraudScore >= values.FraudModerationScoreThreshold

	// Calculate activation date
	var activationDate time.Time
	if requiresModeration || fraudScore > 0.1 {
		// Suspicious reviews have longer delay
		activationDate = time.Now().AddDate(0, 0, values.FraudSuspiciousDelayDays)
	} else {
		activationDate = time.Now().AddDate(0, 0, values.FraudActivationDelayDays)
	}

	slog.Info("fraud analysis completed",
		slog.Float64("fraud_score", fraudScore),
		slog.Int("signals_count", len(signals)),
		slog.Bool("requires_moderation", requiresModeration),
		slog.Time("activation_date", activationDate),
	)

	return &AnalysisResult{
		RawWeight:          rawWeight,
		FraudSignals:       signals,
		FraudScore:         fraudScore,
		RequiresModeration: requiresModeration,
		ActivationDate:     activationDate,
	}, nil
}

// calculateOrderAmountWeight returns weight based on order amount
// 100K+ RUB = 1.0, 50K = 0.9, 10K = 0.7, 1K = 0.5, less = 0.3
func (a *Analyzer) calculateOrderAmountWeight(amount int64) float64 {
	// Amounts are in kopeks (1 RUB = 100 kopeks)
	amountRub := amount / 100

	switch {
	case amountRub >= 100_000:
		return 1.0
	case amountRub >= 50_000:
		return 0.9
	case amountRub >= 10_000:
		return 0.7
	case amountRub >= 1_000:
		return 0.5
	default:
		return 0.3
	}
}

// calculateOrgAgeWeight returns weight based on reviewer organization age
// >12 months = 1.0, 6-12 = 0.8, 3-6 = 0.6, <3 months = 0.3
func (a *Analyzer) calculateOrgAgeWeight(ctx context.Context, orgID uuid.UUID) (float64, error) {
	createdAt, err := a.fraudData.GetOrgCreatedAt(ctx, orgID)
	if err != nil {
		return 0, fmt.Errorf("get org created at: %w", err)
	}

	ageMonths := time.Since(createdAt).Hours() / 24 / 30

	switch {
	case ageMonths >= 12:
		return 1.0, nil
	case ageMonths >= 6:
		return 0.8, nil
	case ageMonths >= 3:
		return 0.6, nil
	default:
		return 0.3, nil
	}
}

// calculateDiversityWeight returns weight based on review diversity
// 1st review from counterparty = 1.0, 2nd = 0.5, 3+ = 0.1
func (a *Analyzer) calculateDiversityWeight(ctx context.Context, reviewerOrgID, reviewedOrgID uuid.UUID) (float64, error) {
	count, _, err := a.fraudData.GetPreviousReviewsFromReviewer(ctx, reviewerOrgID, reviewedOrgID)
	if err != nil {
		return 0, fmt.Errorf("get previous reviews: %w", err)
	}

	switch count {
	case 0:
		return 1.0, nil
	case 1:
		return 0.5, nil
	default:
		return 0.1, nil
	}
}

// calculateReputationWeight returns weight based on reviewer reputation
// fraudster = 0.0, suspected = 0.3, normal = 1.0
func (a *Analyzer) calculateReputationWeight(ctx context.Context, reviewerOrgID uuid.UUID) (float64, error) {
	rep, err := a.fraudData.GetReviewerReputation(ctx, reviewerOrgID)
	if err != nil {
		return 0, fmt.Errorf("get reputation: %w", err)
	}

	if rep.IsConfirmedFraudster {
		return 0.0, nil
	}
	if rep.IsSuspectedFraudster {
		return 0.3, nil
	}
	return rep.ReputationScore, nil
}

// detectFraudSignals detects various fraud signals in a review
func (a *Analyzer) detectFraudSignals(ctx context.Context, r *review.Review) ([]events.FraudSignal, error) {
	var signals []events.FraudSignal

	// Check for fast completion
	if signal := a.checkFastCompletion(r); signal != nil {
		signals = append(signals, *signal)
	}

	// Check for mutual reviews
	if signal, err := a.checkMutualReviews(ctx, r.ReviewerOrgID(), r.ReviewedOrgID()); err != nil {
		slog.Warn("failed to check mutual reviews", slog.String("error", err.Error()))
	} else if signal != nil {
		signals = append(signals, *signal)
	}

	// Check for perfect ratings
	if signal, err := a.checkPerfectRatings(ctx, r.ReviewerOrgID(), r.ReviewedOrgID(), r.Rating()); err != nil {
		slog.Warn("failed to check perfect ratings", slog.String("error", err.Error()))
	} else if signal != nil {
		signals = append(signals, *signal)
	}

	// Check for new org burst
	if signal, err := a.checkNewOrgBurst(ctx, r.ReviewedOrgID()); err != nil {
		slog.Warn("failed to check new org burst", slog.String("error", err.Error()))
	} else if signal != nil {
		signals = append(signals, *signal)
	}

	// Check for same IP between reviewer and reviewed organization
	if signal, err := a.checkSameIP(ctx, r.ReviewerOrgID(), r.ReviewedOrgID()); err != nil {
		slog.Warn("failed to check same IP", slog.String("error", err.Error()))
	} else if signal != nil {
		signals = append(signals, *signal)
	}

	// Check for same fingerprint between reviewer and reviewed organization
	if signal, err := a.checkSameFingerprint(ctx, r.ReviewerOrgID(), r.ReviewedOrgID()); err != nil {
		slog.Warn("failed to check same fingerprint", slog.String("error", err.Error()))
	} else if signal != nil {
		signals = append(signals, *signal)
	}

	// Check for timing pattern (bot behavior)
	if signal, err := a.checkTimingPattern(ctx, r.ReviewerOrgID()); err != nil {
		slog.Warn("failed to check timing pattern", slog.String("error", err.Error()))
	} else if signal != nil {
		signals = append(signals, *signal)
	}

	// Check for dormant reviewer
	if signal, err := a.checkDormantReviewer(ctx, r.ReviewerOrgID()); err != nil {
		slog.Warn("failed to check dormant reviewer", slog.String("error", err.Error()))
	} else if signal != nil {
		signals = append(signals, *signal)
	}

	// Check for burst of 5★ after low rating
	if signal, err := a.checkBurstAfterLow(ctx, r.ReviewedOrgID()); err != nil {
		slog.Warn("failed to check burst after low", slog.String("error", err.Error()))
	} else if signal != nil {
		signals = append(signals, *signal)
	}

	// Check for rating manipulation (5★ to friends, 1★ to others)
	if signal, err := a.checkRatingManipulation(ctx, r.ReviewerOrgID()); err != nil {
		slog.Warn("failed to check rating manipulation", slog.String("error", err.Error()))
	} else if signal != nil {
		signals = append(signals, *signal)
	}

	// Check for text similarity
	if signal, err := a.checkTextSimilarity(ctx, r.ReviewerOrgID(), r.Comment()); err != nil {
		slog.Warn("failed to check text similarity", slog.String("error", err.Error()))
	} else if signal != nil {
		signals = append(signals, *signal)
	}

	return signals, nil
}

// checkFastCompletion checks if order was completed too quickly (<2 hours)
func (a *Analyzer) checkFastCompletion(r *review.Review) *events.FraudSignal {
	completionDuration := r.OrderCompletedAt().Sub(r.OrderCreatedAt())
	thresholdHours := time.Duration(values.FraudFastCompletionHours) * time.Hour

	if completionDuration < thresholdHours {
		signalType := values.SignalFastCompletion
		return &events.FraudSignal{
			Type:        signalType.String(),
			Severity:    signalType.DefaultSeverity().String(),
			Description: fmt.Sprintf("Order completed in %.1f hours (threshold: %d hours)", completionDuration.Hours(), values.FraudFastCompletionHours),
			ScoreImpact: signalType.DefaultScoreImpact(),
			Evidence:    fmt.Sprintf(`{"completion_hours": %.2f}`, completionDuration.Hours()),
		}
	}

	return nil
}

// checkMutualReviews checks for excessive mutual reviews between organizations
func (a *Analyzer) checkMutualReviews(ctx context.Context, reviewerOrgID, reviewedOrgID uuid.UUID) (*events.FraudSignal, error) {
	since := time.Now().AddDate(0, -1, 0) // Last month
	aToB, bToA, err := a.fraudData.CountMutualReviewsInPeriod(ctx, reviewerOrgID, reviewedOrgID, since)
	if err != nil {
		return nil, fmt.Errorf("count mutual reviews: %w", err)
	}

	totalMutual := aToB + bToA
	if totalMutual > values.FraudMutualReviewsPerMonth {
		signalType := values.SignalMutualReviews
		return &events.FraudSignal{
			Type:        signalType.String(),
			Severity:    signalType.DefaultSeverity().String(),
			Description: fmt.Sprintf("Excessive mutual reviews: %d in last month (threshold: %d)", totalMutual, values.FraudMutualReviewsPerMonth),
			ScoreImpact: signalType.DefaultScoreImpact(),
			Evidence:    fmt.Sprintf(`{"a_to_b": %d, "b_to_a": %d, "total": %d}`, aToB, bToA, totalMutual),
		}, nil
	}

	return nil, nil
}

// checkPerfectRatings checks if reviewer always gives 5 stars to this org (>3 reviews)
func (a *Analyzer) checkPerfectRatings(ctx context.Context, reviewerOrgID, reviewedOrgID uuid.UUID, currentRating int) (*events.FraudSignal, error) {
	count, sumRating, err := a.fraudData.GetPreviousReviewsFromReviewer(ctx, reviewerOrgID, reviewedOrgID)
	if err != nil {
		return nil, fmt.Errorf("get previous reviews: %w", err)
	}

	// Include current rating
	count++
	sumRating += currentRating

	if count > values.FraudPerfectRatingsCount {
		avgRating := float64(sumRating) / float64(count)
		if avgRating == 5.0 {
			signalType := values.SignalPerfectRatings
			return &events.FraudSignal{
				Type:        signalType.String(),
				Severity:    signalType.DefaultSeverity().String(),
				Description: fmt.Sprintf("100%% perfect ratings from this reviewer (%d reviews)", count),
				ScoreImpact: signalType.DefaultScoreImpact(),
				Evidence:    fmt.Sprintf(`{"review_count": %d, "avg_rating": %.1f}`, count, avgRating),
			}, nil
		}
	}

	return nil, nil
}

// checkNewOrgBurst checks if new org received too many reviews in first week
func (a *Analyzer) checkNewOrgBurst(ctx context.Context, reviewedOrgID uuid.UUID) (*events.FraudSignal, error) {
	createdAt, err := a.fraudData.GetOrgCreatedAt(ctx, reviewedOrgID)
	if err != nil {
		return nil, fmt.Errorf("get org created at: %w", err)
	}

	// Only check if org is less than 1 week old
	orgAge := time.Since(createdAt)
	if orgAge > 7*24*time.Hour {
		return nil, nil
	}

	count, err := a.fraudData.CountReviewsReceivedInPeriod(ctx, reviewedOrgID, createdAt)
	if err != nil {
		return nil, fmt.Errorf("count reviews: %w", err)
	}

	if count > values.FraudNewOrgBurstReviewsPerWeek {
		signalType := values.SignalNewOrgBurst
		return &events.FraudSignal{
			Type:        signalType.String(),
			Severity:    signalType.DefaultSeverity().String(),
			Description: fmt.Sprintf("New organization received %d reviews in first week (threshold: %d)", count, values.FraudNewOrgBurstReviewsPerWeek),
			ScoreImpact: signalType.DefaultScoreImpact(),
			Evidence:    fmt.Sprintf(`{"org_age_hours": %.1f, "review_count": %d}`, orgAge.Hours(), count),
		}, nil
	}

	return nil, nil
}

// checkSameIP checks if reviewer and reviewed organizations share IP addresses
// This indicates potential sock puppet accounts
func (a *Analyzer) checkSameIP(ctx context.Context, reviewerOrgID, reviewedOrgID uuid.UUID) (*events.FraudSignal, error) {
	// Get member metadata for both organizations
	reviewerMembers, err := a.members.GetMemberMetadata(ctx, reviewerOrgID)
	if err != nil {
		return nil, fmt.Errorf("get reviewer members metadata: %w", err)
	}

	reviewedMembers, err := a.members.GetMemberMetadata(ctx, reviewedOrgID)
	if err != nil {
		return nil, fmt.Errorf("get reviewed members metadata: %w", err)
	}

	// Collect all IPs from reviewer organization
	reviewerIPs := make(map[string]bool)
	for _, m := range reviewerMembers {
		if m.RegistrationIP != nil && *m.RegistrationIP != "" {
			reviewerIPs[*m.RegistrationIP] = true
		}
		if m.LastLoginIP != nil && *m.LastLoginIP != "" {
			reviewerIPs[*m.LastLoginIP] = true
		}
	}

	// Check if any IP from reviewed organization matches
	var matchedIPs []string
	for _, m := range reviewedMembers {
		if m.RegistrationIP != nil && reviewerIPs[*m.RegistrationIP] {
			matchedIPs = append(matchedIPs, *m.RegistrationIP)
		}
		if m.LastLoginIP != nil && reviewerIPs[*m.LastLoginIP] {
			matchedIPs = append(matchedIPs, *m.LastLoginIP)
		}
	}

	if len(matchedIPs) > 0 {
		signalType := values.SignalSameIP
		return &events.FraudSignal{
			Type:        signalType.String(),
			Severity:    signalType.DefaultSeverity().String(),
			Description: fmt.Sprintf("Organizations share %d IP address(es)", len(matchedIPs)),
			ScoreImpact: signalType.DefaultScoreImpact(),
			Evidence:    fmt.Sprintf(`{"matched_ips_count": %d}`, len(matchedIPs)),
		}, nil
	}

	return nil, nil
}

// checkSameFingerprint checks if reviewer and reviewed organizations share device fingerprints
// This indicates potential sock puppet accounts from the same device
func (a *Analyzer) checkSameFingerprint(ctx context.Context, reviewerOrgID, reviewedOrgID uuid.UUID) (*events.FraudSignal, error) {
	// Get member metadata for both organizations
	reviewerMembers, err := a.members.GetMemberMetadata(ctx, reviewerOrgID)
	if err != nil {
		return nil, fmt.Errorf("get reviewer members metadata: %w", err)
	}

	reviewedMembers, err := a.members.GetMemberMetadata(ctx, reviewedOrgID)
	if err != nil {
		return nil, fmt.Errorf("get reviewed members metadata: %w", err)
	}

	// Collect all fingerprints from reviewer organization
	reviewerFingerprints := make(map[string]bool)
	for _, m := range reviewerMembers {
		if m.RegistrationFingerprint != nil && *m.RegistrationFingerprint != "" {
			reviewerFingerprints[*m.RegistrationFingerprint] = true
		}
		if m.LastLoginFingerprint != nil && *m.LastLoginFingerprint != "" {
			reviewerFingerprints[*m.LastLoginFingerprint] = true
		}
	}

	// Check if any fingerprint from reviewed organization matches
	var matchedFPs []string
	for _, m := range reviewedMembers {
		if m.RegistrationFingerprint != nil && reviewerFingerprints[*m.RegistrationFingerprint] {
			matchedFPs = append(matchedFPs, *m.RegistrationFingerprint)
		}
		if m.LastLoginFingerprint != nil && reviewerFingerprints[*m.LastLoginFingerprint] {
			matchedFPs = append(matchedFPs, *m.LastLoginFingerprint)
		}
	}

	if len(matchedFPs) > 0 {
		signalType := values.SignalSameFingerprint
		return &events.FraudSignal{
			Type:        signalType.String(),
			Severity:    signalType.DefaultSeverity().String(),
			Description: fmt.Sprintf("Organizations share %d device fingerprint(s)", len(matchedFPs)),
			ScoreImpact: signalType.DefaultScoreImpact(),
			Evidence:    fmt.Sprintf(`{"matched_fingerprints_count": %d}`, len(matchedFPs)),
		}, nil
	}

	return nil, nil
}

// checkTimingPattern checks if reviewer always posts reviews at the same time of day
// This indicates potential bot behavior
func (a *Analyzer) checkTimingPattern(ctx context.Context, reviewerOrgID uuid.UUID) (*events.FraudSignal, error) {
	timings, err := a.fraudData.GetReviewTimings(ctx, reviewerOrgID, values.FraudTimingPatternMinReviews)
	if err != nil {
		return nil, fmt.Errorf("get review timings: %w", err)
	}

	if len(timings) < values.FraudTimingPatternMinReviews {
		return nil, nil
	}

	// Count reviews per hour of day
	hourCounts := make(map[int]int)
	for _, t := range timings {
		hour := t.Hour()
		hourCounts[hour]++
	}

	// Check if all reviews fall within a 2-hour window
	windowSize := values.FraudTimingPatternWindowHours
	maxInWindow := 0
	peakHour := 0

	for hour := 0; hour < 24; hour++ {
		count := 0
		for h := 0; h < windowSize; h++ {
			count += hourCounts[(hour+h)%24]
		}
		if count > maxInWindow {
			maxInWindow = count
			peakHour = hour
		}
	}

	// If 90% or more of reviews are within the window, flag as suspicious
	if float64(maxInWindow)/float64(len(timings)) >= 0.9 {
		signalType := values.SignalTimingPattern
		return &events.FraudSignal{
			Type:        signalType.String(),
			Severity:    signalType.DefaultSeverity().String(),
			Description: fmt.Sprintf("%d of %d reviews posted between %02d:00-%02d:00", maxInWindow, len(timings), peakHour, (peakHour+windowSize)%24),
			ScoreImpact: signalType.DefaultScoreImpact(),
			Evidence:    fmt.Sprintf(`{"peak_hour": %d, "reviews_in_window": %d, "total_reviews": %d}`, peakHour, maxInWindow, len(timings)),
		}, nil
	}

	return nil, nil
}

// checkDormantReviewer checks if a dormant org suddenly becomes active with many reviews
func (a *Analyzer) checkDormantReviewer(ctx context.Context, reviewerOrgID uuid.UUID) (*events.FraudSignal, error) {
	activity, err := a.fraudData.GetOrgLastActivity(ctx, reviewerOrgID, 7) // Last 7 days
	if err != nil {
		return nil, fmt.Errorf("get org activity: %w", err)
	}

	// Calculate last activity time
	var lastActivity time.Time
	if activity.LastOrderAt != nil && (lastActivity.IsZero() || activity.LastOrderAt.After(lastActivity)) {
		lastActivity = *activity.LastOrderAt
	}
	if activity.LastReviewLeftAt != nil && (lastActivity.IsZero() || activity.LastReviewLeftAt.After(lastActivity)) {
		lastActivity = *activity.LastReviewLeftAt
	}

	// If no prior activity, skip (new org)
	if lastActivity.IsZero() {
		return nil, nil
	}

	// Check if org was dormant (no activity for X days)
	dormantThreshold := time.Duration(values.FraudDormantDays) * 24 * time.Hour
	daysSinceActivity := time.Since(lastActivity)

	// If org was dormant and now has burst of reviews
	if daysSinceActivity > dormantThreshold && activity.RecentReviewsCount > values.FraudDormantBurstCount {
		signalType := values.SignalDormantReviewer
		return &events.FraudSignal{
			Type:        signalType.String(),
			Severity:    signalType.DefaultSeverity().String(),
			Description: fmt.Sprintf("Dormant org (%.0f days inactive) left %d reviews in last week", daysSinceActivity.Hours()/24, activity.RecentReviewsCount),
			ScoreImpact: signalType.DefaultScoreImpact(),
			Evidence:    fmt.Sprintf(`{"days_inactive": %.0f, "recent_reviews": %d}`, daysSinceActivity.Hours()/24, activity.RecentReviewsCount),
		}, nil
	}

	return nil, nil
}

// checkBurstAfterLow checks if org received burst of 5★ reviews right after a low rating
func (a *Analyzer) checkBurstAfterLow(ctx context.Context, reviewedOrgID uuid.UUID) (*events.FraudSignal, error) {
	data, err := a.fraudData.GetBurstAfterLowRating(
		ctx, reviewedOrgID,
		2, // Low threshold (rating <= 2)
		values.FraudBurstAfterLowDays,
	)
	if err != nil {
		return nil, fmt.Errorf("get burst after low: %w", err)
	}

	if data.LastLowRatingAt == nil {
		return nil, nil // No low rating found
	}

	if data.FiveStarCountAfter >= values.FraudBurstAfterLowCount {
		signalType := values.SignalBurstAfterLow
		return &events.FraudSignal{
			Type:        signalType.String(),
			Severity:    signalType.DefaultSeverity().String(),
			Description: fmt.Sprintf("%d five-star reviews within %d days after low rating", data.FiveStarCountAfter, values.FraudBurstAfterLowDays),
			ScoreImpact: signalType.DefaultScoreImpact(),
			Evidence:    fmt.Sprintf(`{"five_star_count": %d, "days_window": %d, "low_rating_at": "%s"}`, data.FiveStarCountAfter, values.FraudBurstAfterLowDays, data.LastLowRatingAt.Format(time.RFC3339)),
		}, nil
	}

	return nil, nil
}

// checkRatingManipulation checks if reviewer gives 5★ to "friends" and 1★ to others
func (a *Analyzer) checkRatingManipulation(ctx context.Context, reviewerOrgID uuid.UUID) (*events.FraudSignal, error) {
	data, err := a.fraudData.GetRatingPatternsByRelationship(
		ctx, reviewerOrgID,
		values.FraudRatingManipMinFriendReviews,
	)
	if err != nil {
		return nil, fmt.Errorf("get rating patterns: %w", err)
	}

	// Need at least some reviews to both friends and others
	if data.FriendsCount < 2 || data.OthersCount < 2 {
		return nil, nil
	}

	friendsAvg := float64(data.FriendsRatingSum) / float64(data.FriendsCount)
	othersAvg := float64(data.OthersRatingSum) / float64(data.OthersCount)

	// Check for manipulation pattern
	if friendsAvg >= values.FraudRatingManipFriendAvgMin &&
		othersAvg <= values.FraudRatingManipOtherAvgMax {
		signalType := values.SignalRatingManipulation
		return &events.FraudSignal{
			Type:        signalType.String(),
			Severity:    signalType.DefaultSeverity().String(),
			Description: fmt.Sprintf("Avg rating to friends: %.1f (%d reviews), to others: %.1f (%d reviews)", friendsAvg, data.FriendsCount, othersAvg, data.OthersCount),
			ScoreImpact: signalType.DefaultScoreImpact(),
			Evidence:    fmt.Sprintf(`{"friends_avg": %.2f, "friends_count": %d, "others_avg": %.2f, "others_count": %d}`, friendsAvg, data.FriendsCount, othersAvg, data.OthersCount),
		}, nil
	}

	return nil, nil
}

// checkTextSimilarity checks if reviewer leaves similar/identical review texts
func (a *Analyzer) checkTextSimilarity(ctx context.Context, reviewerOrgID uuid.UUID, currentComment string) (*events.FraudSignal, error) {
	if currentComment == "" {
		return nil, nil
	}

	texts, err := a.fraudData.GetRecentReviewTexts(ctx, reviewerOrgID, 100)
	if err != nil {
		return nil, fmt.Errorf("get review texts: %w", err)
	}

	if len(texts) == 0 {
		return nil, nil
	}

	// Count similar reviews
	similarCount := 0
	for _, t := range texts {
		if t.Comment == "" {
			continue
		}
		similarity := calculateTextSimilarity(currentComment, t.Comment)
		if similarity >= values.FraudTextSimilarityThreshold {
			similarCount++
		}
	}

	if similarCount >= values.FraudTextSimilarityMinReviews {
		signalType := values.SignalTextSimilarity
		return &events.FraudSignal{
			Type:        signalType.String(),
			Severity:    signalType.DefaultSeverity().String(),
			Description: fmt.Sprintf("Found %d reviews with >%.0f%% text similarity", similarCount, values.FraudTextSimilarityThreshold*100),
			ScoreImpact: signalType.DefaultScoreImpact(),
			Evidence:    fmt.Sprintf(`{"similar_count": %d, "threshold": %.2f}`, similarCount, values.FraudTextSimilarityThreshold),
		}, nil
	}

	return nil, nil
}

// calculateTextSimilarity calculates Levenshtein similarity between two strings
func calculateTextSimilarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}

	// Normalize strings
	s1 = normalizeText(s1)
	s2 = normalizeText(s2)

	if s1 == s2 {
		return 1.0
	}

	// Calculate Levenshtein distance (rune-based)
	distance := levenshteinDistance(s1, s2)
	maxLen := max(len([]rune(s1)), len([]rune(s2)))

	if maxLen == 0 {
		return 1.0
	}

	return 1.0 - float64(distance)/float64(maxLen)
}

// normalizeText normalizes text for comparison
func normalizeText(s string) string {
	// Convert to lowercase and trim whitespace
	s = strings.ToLower(strings.TrimSpace(s))
	// Remove extra whitespace
	return strings.Join(strings.Fields(s), " ")
}

// levenshteinDistance calculates the Levenshtein distance between two strings
func levenshteinDistance(s1, s2 string) int {
	r1 := []rune(s1)
	r2 := []rune(s2)

	if len(r1) == 0 {
		return len(r2)
	}
	if len(r2) == 0 {
		return len(r1)
	}

	d := make([][]int, len(r1)+1)
	for i := range d {
		d[i] = make([]int, len(r2)+1)
		d[i][0] = i
	}
	for j := range d[0] {
		d[0][j] = j
	}

	for i := 1; i <= len(r1); i++ {
		for j := 1; j <= len(r2); j++ {
			cost := 1
			if r1[i-1] == r2[j-1] {
				cost = 0
			}
			d[i][j] = min(
				d[i-1][j]+1,      // deletion
				d[i][j-1]+1,      // insertion
				d[i-1][j-1]+cost, // substitution
			)
		}
	}

	return d[len(r1)][len(r2)]
}
