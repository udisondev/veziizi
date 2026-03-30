package projections

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/udisondev/veziizi/backend/internal/pkg/dbtx"
	"github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/google/uuid"
)

// Session fraud signal types
const (
	SignalLoginGeoJump   = "login_geo_jump"
	SignalSessionAnomaly = "session_anomaly"
	SignalAPIAbuse       = "api_abuse"
)

// Session fraud thresholds
var SessionFraudThresholds = struct {
	// login_geo_jump
	MaxKmPerHour           float64 // impossible travel speed
	MinDistanceForCheck    float64 // minimum km to check
	// session_anomaly
	UnusualHourThreshold   int     // hours outside typical range
	MinLoginsForPattern    int     // minimum logins to establish pattern
	// api_abuse
	MaxRequestsPerMinute   int
	MaxRequestsPerHour     int
	BlockDurationMinutes   int
	ScrapingThreshold      int     // GET requests without actions
}{
	MaxKmPerHour:           900,   // ~airplane speed
	MinDistanceForCheck:    100,   // 100km minimum
	UnusualHourThreshold:   3,     // 3+ hours from typical
	MinLoginsForPattern:    5,     // need 5+ logins for pattern
	MaxRequestsPerMinute:   100,
	MaxRequestsPerHour:     1000,
	BlockDurationMinutes:   15,
	ScrapingThreshold:      50,    // 50 GETs without POST/PUT
}

// SetSessionFraudLimits allows configuring session fraud limits for testing.
func SetSessionFraudLimits(maxRequestsPerMinute, maxRequestsPerHour int) {
	SessionFraudThresholds.MaxRequestsPerMinute = maxRequestsPerMinute
	SessionFraudThresholds.MaxRequestsPerHour = maxRequestsPerHour
}

type SessionFraudProjection struct {
	db   dbtx.TxManager
	psql squirrel.StatementBuilderType
}

func NewSessionFraudProjection(db dbtx.TxManager) *SessionFraudProjection {
	return &SessionFraudProjection{
		db:   db,
		psql: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

// SessionEvent represents a session event
type SessionEvent struct {
	ID             uuid.UUID  `db:"id"`
	MemberID       uuid.UUID  `db:"member_id"`
	OrganizationID uuid.UUID  `db:"organization_id"`
	EventType      string     `db:"event_type"`
	IPAddress      *string    `db:"ip_address"`
	Fingerprint    *string    `db:"fingerprint"`
	UserAgent      *string    `db:"user_agent"`
	GeoCountry     *string    `db:"geo_country"`
	GeoCity        *string    `db:"geo_city"`
	GeoLat         *float64   `db:"geo_lat"`
	GeoLon         *float64   `db:"geo_lon"`
	Endpoint       *string    `db:"endpoint"`
	CreatedAt      time.Time  `db:"created_at"`
}

// RecordSessionEvent records a session event
func (p *SessionFraudProjection) RecordSessionEvent(ctx context.Context, event *SessionEvent) error {
	query, args, err := p.psql.
		Insert("session_events").
		Columns(
			"member_id", "organization_id", "event_type",
			"ip_address", "fingerprint", "user_agent",
			"geo_country", "geo_city", "geo_lat", "geo_lon",
			"endpoint",
		).
		Values(
			event.MemberID, event.OrganizationID, event.EventType,
			event.IPAddress, event.Fingerprint, event.UserAgent,
			event.GeoCountry, event.GeoCity, event.GeoLat, event.GeoLon,
			event.Endpoint,
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build insert query: %w", err)
	}

	if _, err := p.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to insert session event: %w", err)
	}

	return nil
}

// SessionFraudSignal represents a fraud signal
type SessionFraudSignal struct {
	ID             uuid.UUID `db:"id"`
	MemberID       uuid.UUID `db:"member_id"`
	OrganizationID uuid.UUID `db:"organization_id"`
	SignalType     string    `db:"signal_type"`
	Severity       string    `db:"severity"`
	Description    string    `db:"description"`
	ScoreImpact    float64   `db:"score_impact"`
	Evidence       string    `db:"evidence"`
	CreatedAt      time.Time `db:"created_at"`
}

// InsertSessionFraudSignal inserts a fraud signal
func (p *SessionFraudProjection) InsertSessionFraudSignal(ctx context.Context, signal *SessionFraudSignal) error {
	query, args, err := p.psql.
		Insert("session_fraud_signals").
		Columns("member_id", "organization_id", "signal_type", "severity", "description", "score_impact", "evidence").
		Values(signal.MemberID, signal.OrganizationID, signal.SignalType, signal.Severity, signal.Description, signal.ScoreImpact, signal.Evidence).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build insert query: %w", err)
	}

	if _, err := p.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to insert session fraud signal: %w", err)
	}

	return nil
}

// MemberSessionBehavior tracks typical behavior for anomaly detection
type MemberSessionBehavior struct {
	MemberID         uuid.UUID          `db:"member_id"`
	TypicalHours     map[int]int        `db:"-"` // hour -> count
	TypicalHoursJSON json.RawMessage    `db:"typical_login_hours"`
	TypicalCountries []string           `db:"typical_countries"`
	TypicalIPs       []string           `db:"typical_ips"`
	LastLoginAt      *time.Time         `db:"last_login_at"`
	LastLoginIP      *string            `db:"last_login_ip"`
	LastLoginCountry *string            `db:"last_login_country"`
	LastLoginLat     *float64           `db:"last_login_lat"`
	LastLoginLon     *float64           `db:"last_login_lon"`
	TotalLogins      int                `db:"total_logins"`
	SuspiciousLogins int                `db:"suspicious_logins"`
	IsSuspicious     bool               `db:"is_suspicious"`
	SuspiciousReason *string            `db:"suspicious_reason"`
	UpdatedAt        time.Time          `db:"updated_at"`
}

// GetMemberSessionBehavior retrieves behavior data for a member
func (p *SessionFraudProjection) GetMemberSessionBehavior(ctx context.Context, memberID uuid.UUID) (*MemberSessionBehavior, error) {
	query, args, err := p.psql.
		Select(
			"member_id", "typical_login_hours", "typical_countries", "typical_ips",
			"last_login_at", "last_login_ip::TEXT as last_login_ip", "last_login_country",
			"last_login_lat", "last_login_lon", "total_logins", "suspicious_logins",
			"is_suspicious", "suspicious_reason", "updated_at",
		).
		From("member_session_behavior").
		Where(squirrel.Eq{"member_id": memberID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	var behavior MemberSessionBehavior
	if err := pgxscan.Get(ctx, p.db, &behavior, query, args...); err != nil {
		if pgxscan.NotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get member session behavior: %w", err)
	}

	// Parse JSON
	if behavior.TypicalHoursJSON != nil {
		behavior.TypicalHours = make(map[int]int)
		if err := json.Unmarshal(behavior.TypicalHoursJSON, &behavior.TypicalHours); err != nil {
			behavior.TypicalHours = make(map[int]int)
		}
	}

	return &behavior, nil
}

// UpsertMemberSessionBehavior creates or updates behavior data
func (p *SessionFraudProjection) UpsertMemberSessionBehavior(ctx context.Context, behavior *MemberSessionBehavior) error {
	hoursJSON, err := json.Marshal(behavior.TypicalHours)
	if err != nil {
		hoursJSON = []byte("{}")
	}

	query := `
		INSERT INTO member_session_behavior (
			member_id, typical_login_hours, typical_countries, typical_ips,
			last_login_at, last_login_ip, last_login_country, last_login_lat, last_login_lon,
			total_logins, suspicious_logins, is_suspicious, suspicious_reason, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW())
		ON CONFLICT (member_id) DO UPDATE SET
			typical_login_hours = EXCLUDED.typical_login_hours,
			typical_countries = EXCLUDED.typical_countries,
			typical_ips = EXCLUDED.typical_ips,
			last_login_at = EXCLUDED.last_login_at,
			last_login_ip = EXCLUDED.last_login_ip,
			last_login_country = EXCLUDED.last_login_country,
			last_login_lat = EXCLUDED.last_login_lat,
			last_login_lon = EXCLUDED.last_login_lon,
			total_logins = EXCLUDED.total_logins,
			suspicious_logins = EXCLUDED.suspicious_logins,
			is_suspicious = EXCLUDED.is_suspicious,
			suspicious_reason = EXCLUDED.suspicious_reason,
			updated_at = NOW()
	`

	_, err = p.db.Exec(ctx, query,
		behavior.MemberID, hoursJSON, behavior.TypicalCountries, behavior.TypicalIPs,
		behavior.LastLoginAt, behavior.LastLoginIP, behavior.LastLoginCountry,
		behavior.LastLoginLat, behavior.LastLoginLon,
		behavior.TotalLogins, behavior.SuspiciousLogins,
		behavior.IsSuspicious, behavior.SuspiciousReason,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert member session behavior: %w", err)
	}

	return nil
}

// GetLastLogin returns the last login event for a member
func (p *SessionFraudProjection) GetLastLogin(ctx context.Context, memberID uuid.UUID) (*SessionEvent, error) {
	query, args, err := p.psql.
		Select(
			"id", "member_id", "organization_id", "event_type",
			"ip_address::TEXT as ip_address", "fingerprint", "user_agent",
			"geo_country", "geo_city", "geo_lat", "geo_lon", "endpoint", "created_at",
		).
		From("session_events").
		Where(squirrel.And{
			squirrel.Eq{"member_id": memberID},
			squirrel.Eq{"event_type": "login"},
		}).
		OrderBy("created_at DESC").
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	var event SessionEvent
	if err := pgxscan.Get(ctx, p.db, &event, query, args...); err != nil {
		if pgxscan.NotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get last login: %w", err)
	}

	return &event, nil
}

// RateLimitResult contains rate limit check result
type RateLimitResult struct {
	IsBlocked      bool
	RequestCount   int
	BlockedUntil   *time.Time
	Reason         string
}

// CheckRateLimit checks if request should be rate limited
func (p *SessionFraudProjection) CheckRateLimit(ctx context.Context, key string) (*RateLimitResult, error) {
	// Get or create rate limit entry
	query := `
		INSERT INTO api_rate_limits (key, request_count, window_start)
		VALUES ($1, 1, NOW())
		ON CONFLICT (key) DO UPDATE SET
			request_count = CASE
				WHEN api_rate_limits.window_start < NOW() - INTERVAL '1 minute'
				THEN 1
				ELSE api_rate_limits.request_count + 1
			END,
			window_start = CASE
				WHEN api_rate_limits.window_start < NOW() - INTERVAL '1 minute'
				THEN NOW()
				ELSE api_rate_limits.window_start
			END
		RETURNING request_count, window_start, blocked_until
	`

	var count int
	var windowStart time.Time
	var blockedUntil *time.Time

	if err := p.db.QueryRow(ctx, query, key).Scan(&count, &windowStart, &blockedUntil); err != nil {
		return nil, fmt.Errorf("failed to check rate limit: %w", err)
	}

	result := &RateLimitResult{
		RequestCount: count,
		BlockedUntil: blockedUntil,
	}

	// Check if currently blocked
	if blockedUntil != nil && blockedUntil.After(time.Now()) {
		result.IsBlocked = true
		result.Reason = "temporarily blocked due to rate limit"
		return result, nil
	}

	// Check if exceeded limit
	if count > SessionFraudThresholds.MaxRequestsPerMinute {
		// Block the key
		blockUntil := time.Now().Add(time.Duration(SessionFraudThresholds.BlockDurationMinutes) * time.Minute)
		if _, err := p.db.Exec(ctx,
			"UPDATE api_rate_limits SET blocked_until = $1 WHERE key = $2",
			blockUntil, key,
		); err != nil {
			return nil, fmt.Errorf("failed to set block: %w", err)
		}
		result.IsBlocked = true
		result.BlockedUntil = &blockUntil
		result.Reason = fmt.Sprintf("rate limit exceeded: %d requests/minute", count)
	}

	return result, nil
}

// ResetRateLimit resets rate limit for a key
func (p *SessionFraudProjection) ResetRateLimit(ctx context.Context, key string) error {
	_, err := p.db.Exec(ctx,
		"DELETE FROM api_rate_limits WHERE key = $1",
		key,
	)
	if err != nil {
		return fmt.Errorf("failed to reset rate limit: %w", err)
	}
	return nil
}

// CleanupOldRateLimits removes old rate limit entries
func (p *SessionFraudProjection) CleanupOldRateLimits(ctx context.Context) error {
	_, err := p.db.Exec(ctx,
		"DELETE FROM api_rate_limits WHERE window_start < NOW() - INTERVAL '1 hour' AND (blocked_until IS NULL OR blocked_until < NOW())",
	)
	if err != nil {
		return fmt.Errorf("failed to cleanup rate limits: %w", err)
	}
	return nil
}

// CalculateDistance calculates distance between two geo points using Haversine formula
func CalculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0

	dLat := degreesToRadians(lat2 - lat1)
	dLon := degreesToRadians(lon2 - lon1)

	lat1Rad := degreesToRadians(lat1)
	lat2Rad := degreesToRadians(lat2)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(lat1Rad)*math.Cos(lat2Rad)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusKm * c
}

func degreesToRadians(deg float64) float64 {
	return deg * math.Pi / 180
}

// GetLoginHourHistogram returns login hour distribution for a member
func (p *SessionFraudProjection) GetLoginHourHistogram(ctx context.Context, memberID uuid.UUID) (map[int]int, error) {
	query := `
		SELECT EXTRACT(HOUR FROM created_at)::int as hour, COUNT(*) as count
		FROM session_events
		WHERE member_id = $1 AND event_type = 'login'
		GROUP BY EXTRACT(HOUR FROM created_at)
	`

	type hourCount struct {
		Hour  int `db:"hour"`
		Count int `db:"count"`
	}

	var rows []hourCount
	if err := pgxscan.Select(ctx, p.db, &rows, query, memberID); err != nil {
		return nil, fmt.Errorf("failed to get login hour histogram: %w", err)
	}

	result := make(map[int]int)
	for _, row := range rows {
		result[row.Hour] = row.Count
	}

	return result, nil
}

// GetTypicalLoginHour returns the most common login hour for a member
func (p *SessionFraudProjection) GetTypicalLoginHour(ctx context.Context, memberID uuid.UUID) (int, int, error) {
	histogram, err := p.GetLoginHourHistogram(ctx, memberID)
	if err != nil {
		return 0, 0, err
	}

	maxHour := 0
	maxCount := 0
	totalLogins := 0

	for hour, count := range histogram {
		totalLogins += count
		if count > maxCount {
			maxCount = count
			maxHour = hour
		}
	}

	return maxHour, totalLogins, nil
}

// GetRecentAPIActivity returns recent API activity for scraping detection
type APIActivitySummary struct {
	GetRequests    int `db:"get_requests"`
	PostRequests   int `db:"post_requests"`
	TotalRequests  int `db:"total_requests"`
}

func (p *SessionFraudProjection) GetRecentAPIActivity(ctx context.Context, memberID uuid.UUID, minutes int) (*APIActivitySummary, error) {
	query := `
		SELECT
			COUNT(*) FILTER (WHERE endpoint LIKE 'GET %') as get_requests,
			COUNT(*) FILTER (WHERE endpoint LIKE 'POST %' OR endpoint LIKE 'PUT %' OR endpoint LIKE 'DELETE %') as post_requests,
			COUNT(*) as total_requests
		FROM session_events
		WHERE member_id = $1
		  AND event_type = 'api_call'
		  AND created_at > NOW() - INTERVAL '1 minute' * $2
	`

	var summary APIActivitySummary
	if err := p.db.QueryRow(ctx, query, memberID, minutes).Scan(
		&summary.GetRequests,
		&summary.PostRequests,
		&summary.TotalRequests,
	); err != nil {
		return nil, fmt.Errorf("failed to get API activity: %w", err)
	}

	return &summary, nil
}
