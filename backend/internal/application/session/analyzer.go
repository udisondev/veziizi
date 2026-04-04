package session

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/projections"
)

// SessionAnalyzer analyzes session events for fraud signals
type SessionAnalyzer struct {
	sessionFraud *projections.SessionFraudProjection
}

// NewSessionAnalyzer creates a new session analyzer
func NewSessionAnalyzer(sessionFraud *projections.SessionFraudProjection) *SessionAnalyzer {
	return &SessionAnalyzer{
		sessionFraud: sessionFraud,
	}
}

// LoginAnalysisInput contains data for login analysis
type LoginAnalysisInput struct {
	MemberID       uuid.UUID
	OrganizationID uuid.UUID
	IPAddress      string
	Fingerprint    string
	UserAgent      string
	GeoCountry     string
	GeoCity        string
	GeoLat         float64
	GeoLon         float64
	LoginTime      time.Time
}

// LoginAnalysisResult contains analysis results
type LoginAnalysisResult struct {
	IsSuspicious bool
	Signals      []string
	BlockLogin   bool
	BlockReason  string
}

// AnalyzeLogin analyzes a login event for fraud signals
func (a *SessionAnalyzer) AnalyzeLogin(ctx context.Context, input LoginAnalysisInput) (*LoginAnalysisResult, error) {
	result := &LoginAnalysisResult{
		Signals: make([]string, 0),
	}

	// Record the login event
	event := &projections.SessionEvent{
		MemberID:       input.MemberID,
		OrganizationID: input.OrganizationID,
		EventType:      "login",
		IPAddress:      &input.IPAddress,
		Fingerprint:    &input.Fingerprint,
		UserAgent:      &input.UserAgent,
		GeoCountry:     &input.GeoCountry,
		GeoCity:        &input.GeoCity,
		GeoLat:         &input.GeoLat,
		GeoLon:         &input.GeoLon,
		CreatedAt:      input.LoginTime,
	}
	if err := a.sessionFraud.RecordSessionEvent(ctx, event); err != nil {
		slog.Error("failed to record login event", slog.String("error", err.Error()))
		// Don't fail login on logging error
	}

	// Get behavior data
	behavior, err := a.sessionFraud.GetMemberSessionBehavior(ctx, input.MemberID)
	if err != nil {
		slog.Error("failed to get session behavior", slog.String("error", err.Error()))
	}

	// Check for geo jump (impossible travel)
	if err := a.checkGeoJump(ctx, input, behavior, result); err != nil {
		slog.Warn("failed to check geo jump", slog.String("error", err.Error()))
	}

	// Check for session anomaly (unusual login time)
	if err := a.checkSessionAnomaly(ctx, input, behavior, result); err != nil {
		slog.Warn("failed to check session anomaly", slog.String("error", err.Error()))
	}

	// Update behavior data
	if err := a.updateBehavior(ctx, input, behavior, result); err != nil {
		slog.Error("failed to update session behavior", slog.String("error", err.Error()))
	}

	return result, nil
}

// checkGeoJump checks for impossible travel (login from distant location in short time)
func (a *SessionAnalyzer) checkGeoJump(ctx context.Context, input LoginAnalysisInput, behavior *projections.MemberSessionBehavior, result *LoginAnalysisResult) error {
	// Need previous login with geo data
	if behavior == nil || behavior.LastLoginLat == nil || behavior.LastLoginLon == nil {
		return nil
	}

	// Need current geo data
	if input.GeoLat == 0 && input.GeoLon == 0 {
		return nil
	}

	// Calculate distance
	distance := projections.CalculateDistance(
		*behavior.LastLoginLat, *behavior.LastLoginLon,
		input.GeoLat, input.GeoLon,
	)

	// Skip if distance is too small
	if distance < projections.SessionFraudThresholds.MinDistanceForCheck {
		return nil
	}

	// Calculate time difference
	if behavior.LastLoginAt == nil {
		return nil
	}
	hoursSinceLastLogin := input.LoginTime.Sub(*behavior.LastLoginAt).Hours()
	if hoursSinceLastLogin <= 0 {
		return nil
	}

	// Calculate required speed
	requiredSpeed := distance / hoursSinceLastLogin

	// Check if impossible
	if requiredSpeed > projections.SessionFraudThresholds.MaxKmPerHour {
		result.IsSuspicious = true
		result.Signals = append(result.Signals, projections.SignalLoginGeoJump)

		signal := &projections.SessionFraudSignal{
			MemberID:       input.MemberID,
			OrganizationID: input.OrganizationID,
			SignalType:     projections.SignalLoginGeoJump,
			Severity:       "high",
			Description: fmt.Sprintf(
				"Impossible travel detected: %.0f km in %.1f hours (%.0f km/h required, max %.0f km/h)",
				distance, hoursSinceLastLogin, requiredSpeed, projections.SessionFraudThresholds.MaxKmPerHour,
			),
			ScoreImpact: 0.4,
			Evidence: fmt.Sprintf(
				`{"distance_km": %.2f, "hours": %.2f, "speed_kmh": %.2f, "from_country": %q, "to_country": %q}`,
				distance, hoursSinceLastLogin, requiredSpeed,
				stringVal(behavior.LastLoginCountry), input.GeoCountry,
			),
		}

		if err := a.sessionFraud.InsertSessionFraudSignal(ctx, signal); err != nil {
			return fmt.Errorf("insert fraud signal: %w", err)
		}

		slog.Warn("session fraud: impossible travel detected",
			slog.String("member_id", input.MemberID.String()),
			slog.Float64("distance_km", distance),
			slog.Float64("hours", hoursSinceLastLogin),
			slog.Float64("speed_kmh", requiredSpeed),
		)
	}

	return nil
}

// checkSessionAnomaly checks for unusual login patterns
func (a *SessionAnalyzer) checkSessionAnomaly(ctx context.Context, input LoginAnalysisInput, behavior *projections.MemberSessionBehavior, result *LoginAnalysisResult) error {
	// Need enough login history
	if behavior == nil || behavior.TotalLogins < projections.SessionFraudThresholds.MinLoginsForPattern {
		return nil
	}

	// Get typical login hour
	typicalHour, totalLogins, err := a.sessionFraud.GetTypicalLoginHour(ctx, input.MemberID)
	if err != nil {
		return fmt.Errorf("get typical login hour: %w", err)
	}

	if totalLogins < projections.SessionFraudThresholds.MinLoginsForPattern {
		return nil
	}

	// Calculate hour difference (circular, 0-23)
	currentHour := input.LoginTime.Hour()
	hourDiff := int(math.Abs(float64(currentHour - typicalHour)))
	if hourDiff > 12 {
		hourDiff = 24 - hourDiff
	}

	// Check if unusual
	if hourDiff >= projections.SessionFraudThresholds.UnusualHourThreshold {
		result.IsSuspicious = true
		result.Signals = append(result.Signals, projections.SignalSessionAnomaly)

		signal := &projections.SessionFraudSignal{
			MemberID:       input.MemberID,
			OrganizationID: input.OrganizationID,
			SignalType:     projections.SignalSessionAnomaly,
			Severity:       "medium",
			Description: fmt.Sprintf(
				"Unusual login time: %d:00 (typical: %d:00, %d hours difference)",
				currentHour, typicalHour, hourDiff,
			),
			ScoreImpact: 0.25,
			Evidence: fmt.Sprintf(
				`{"current_hour": %d, "typical_hour": %d, "hour_diff": %d, "total_logins": %d}`,
				currentHour, typicalHour, hourDiff, totalLogins,
			),
		}

		if err := a.sessionFraud.InsertSessionFraudSignal(ctx, signal); err != nil {
			return fmt.Errorf("insert fraud signal: %w", err)
		}

		slog.Info("session fraud: unusual login time detected",
			slog.String("member_id", input.MemberID.String()),
			slog.Int("current_hour", currentHour),
			slog.Int("typical_hour", typicalHour),
		)
	}

	// Check for country change without logout
	if behavior.LastLoginCountry != nil && input.GeoCountry != "" && *behavior.LastLoginCountry != input.GeoCountry {
		// Country changed - flag as suspicious if within short time
		if behavior.LastLoginAt != nil {
			hoursSinceLastLogin := input.LoginTime.Sub(*behavior.LastLoginAt).Hours()
			if hoursSinceLastLogin < 24 { // Country change within 24 hours
				result.IsSuspicious = true
				result.Signals = append(result.Signals, projections.SignalSessionAnomaly)

				signal := &projections.SessionFraudSignal{
					MemberID:       input.MemberID,
					OrganizationID: input.OrganizationID,
					SignalType:     projections.SignalSessionAnomaly,
					Severity:       "medium",
					Description: fmt.Sprintf(
						"Country changed within %.1f hours: %s → %s",
						hoursSinceLastLogin, *behavior.LastLoginCountry, input.GeoCountry,
					),
					ScoreImpact: 0.25,
					Evidence: fmt.Sprintf(
						`{"from_country": %q, "to_country": %q, "hours": %.2f}`,
						*behavior.LastLoginCountry, input.GeoCountry, hoursSinceLastLogin,
					),
				}

				if err := a.sessionFraud.InsertSessionFraudSignal(ctx, signal); err != nil {
					return fmt.Errorf("insert fraud signal: %w", err)
				}
			}
		}
	}

	return nil
}

// updateBehavior updates session behavior after login analysis
func (a *SessionAnalyzer) updateBehavior(ctx context.Context, input LoginAnalysisInput, behavior *projections.MemberSessionBehavior, result *LoginAnalysisResult) error {
	if behavior == nil {
		behavior = &projections.MemberSessionBehavior{
			MemberID:         input.MemberID,
			TypicalHours:     make(map[int]int),
			TypicalCountries: make([]string, 0),
			TypicalIPs:       make([]string, 0),
		}
	}

	// Update login hour histogram
	if behavior.TypicalHours == nil {
		behavior.TypicalHours = make(map[int]int)
	}
	behavior.TypicalHours[input.LoginTime.Hour()]++

	// Update last login info
	behavior.LastLoginAt = &input.LoginTime
	if input.IPAddress != "" {
		behavior.LastLoginIP = &input.IPAddress
	}
	if input.GeoCountry != "" {
		behavior.LastLoginCountry = &input.GeoCountry
	}
	if input.GeoLat != 0 {
		behavior.LastLoginLat = &input.GeoLat
	}
	if input.GeoLon != 0 {
		behavior.LastLoginLon = &input.GeoLon
	}

	// Update typical countries
	if input.GeoCountry != "" && !containsString(behavior.TypicalCountries, input.GeoCountry) {
		behavior.TypicalCountries = append(behavior.TypicalCountries, input.GeoCountry)
		if len(behavior.TypicalCountries) > 10 {
			behavior.TypicalCountries = behavior.TypicalCountries[1:]
		}
	}

	// Update typical IPs
	if input.IPAddress != "" && !containsString(behavior.TypicalIPs, input.IPAddress) {
		behavior.TypicalIPs = append(behavior.TypicalIPs, input.IPAddress)
		if len(behavior.TypicalIPs) > 10 {
			behavior.TypicalIPs = behavior.TypicalIPs[1:]
		}
	}

	// Update counts
	behavior.TotalLogins++
	if result.IsSuspicious {
		behavior.SuspiciousLogins++
		behavior.IsSuspicious = true
		reason := fmt.Sprintf("Suspicious login signals: %v", result.Signals)
		behavior.SuspiciousReason = &reason
	}

	return a.sessionFraud.UpsertMemberSessionBehavior(ctx, behavior)
}

// CheckAPIAbuse checks for API abuse patterns
func (a *SessionAnalyzer) CheckAPIAbuse(ctx context.Context, memberID, orgID uuid.UUID, endpoint string) (*LoginAnalysisResult, error) {
	result := &LoginAnalysisResult{
		Signals: make([]string, 0),
	}

	// Record API call
	event := &projections.SessionEvent{
		MemberID:       memberID,
		OrganizationID: orgID,
		EventType:      "api_call",
		Endpoint:       &endpoint,
	}
	if err := a.sessionFraud.RecordSessionEvent(ctx, event); err != nil {
		slog.Error("failed to record API event", slog.String("error", err.Error()))
	}

	// Check rate limit
	key := fmt.Sprintf("member:%s", memberID.String())
	rateResult, err := a.sessionFraud.CheckRateLimit(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("check rate limit: %w", err)
	}

	if rateResult.IsBlocked {
		result.IsSuspicious = true
		result.BlockLogin = true
		result.BlockReason = rateResult.Reason
		result.Signals = append(result.Signals, projections.SignalAPIAbuse)

		signal := &projections.SessionFraudSignal{
			MemberID:       memberID,
			OrganizationID: orgID,
			SignalType:     projections.SignalAPIAbuse,
			Severity:       "high",
			Description:    rateResult.Reason,
			ScoreImpact:    0.3,
			Evidence: fmt.Sprintf(
				`{"request_count": %d, "blocked_until": %q}`,
				rateResult.RequestCount,
				rateResult.BlockedUntil,
			),
		}

		if err := a.sessionFraud.InsertSessionFraudSignal(ctx, signal); err != nil {
			slog.Error("failed to insert API abuse signal", slog.String("error", err.Error()))
		}

		slog.Warn("session fraud: API abuse detected",
			slog.String("member_id", memberID.String()),
			slog.Int("request_count", rateResult.RequestCount),
		)
	}

	// Check for scraping pattern (many GETs without actions)
	activity, err := a.sessionFraud.GetRecentAPIActivity(ctx, memberID, 10)
	if err != nil {
		slog.Error("failed to get API activity", slog.String("error", err.Error()))
	} else if activity != nil && activity.GetRequests >= projections.SessionFraudThresholds.ScrapingThreshold && activity.PostRequests == 0 {
		result.IsSuspicious = true
		result.Signals = append(result.Signals, projections.SignalAPIAbuse)

		signal := &projections.SessionFraudSignal{
			MemberID:       memberID,
			OrganizationID: orgID,
			SignalType:     projections.SignalAPIAbuse,
			Severity:       "medium",
			Description:    fmt.Sprintf("Scraping pattern detected: %d GET requests without actions", activity.GetRequests),
			ScoreImpact:    0.2,
			Evidence: fmt.Sprintf(
				`{"get_requests": %d, "post_requests": %d, "minutes": 10}`,
				activity.GetRequests, activity.PostRequests,
			),
		}

		if err := a.sessionFraud.InsertSessionFraudSignal(ctx, signal); err != nil {
			slog.Error("failed to insert scraping signal", slog.String("error", err.Error()))
		}
	}

	return result, nil
}

func stringVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
