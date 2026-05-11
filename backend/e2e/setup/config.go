package setup

import (
	"os"
	"time"

	"github.com/udisondev/veziizi/backend/internal/pkg/config"
)

// testConfigWithDSN creates a configuration with the provided database DSN.
func testConfigWithDSN(databaseURL string) *config.Config {
	sessionSecret := os.Getenv("TEST_SESSION_SECRET")
	if sessionSecret == "" {
		sessionSecret = "test-session-secret-32-bytes!!"
	}

	adminSessionSecret := os.Getenv("TEST_SESSION_ADMIN_SECRET")
	if adminSessionSecret == "" {
		adminSessionSecret = "test-admin-session-secret-32by!"
	}

	return &config.Config{
		App: config.AppConfig{
			Env:      "development", // Enable dev features for testing
			LogLevel: "error",
			BaseURL:  "http://localhost:5173",
		},
		HTTP: config.HTTPConfig{
			Addr:         "127.0.0.1:0", // Will be overwritten with random port
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		Database: config.DatabaseConfig{
			URL: databaseURL,
		},
		Session: config.SessionConfig{
			Secret:      sessionSecret,
			AdminSecret: adminSessionSecret,
			Name:        "veziizi_session",
			AdminName:   "veziizi_admin_session",
			MaxAge:      86400,
		},
		GeoIP: config.GeoIPConfig{
			DatabasePath: "", // Disabled in tests
		},
		Telegram: config.TelegramConfig{
			BotToken:    "",
			BotUsername: "testbot",
		},
		Email: config.EmailConfig{
			Enabled:     false, // Email disabled in tests, uses NoopEmailProvider
			Provider:    "resend",
			FromAddress: "test@veziizi.local",
			FromName:    "Veziizi Test",
		},
		Security: config.SecurityConfig{
			// Must match production defaults (config.go envDefault) — otherwise
			// BodyLimit middleware silently truncates JSON requests to 0 bytes
			// and every endpoint returns "invalid request body".
			MaxJSONBodySize:        1 * 1024 * 1024,  // 1 MB
			MaxFileUploadSize:      10 * 1024 * 1024, // 10 MB
			MaxFailedLoginAttempts: 100000,           // effectively unlimited for tests
			AccountLockoutDuration: 15 * time.Minute,
			ShutdownTimeout:        30 * time.Second,
		},
		RateLimit: config.RateLimitConfig{
			// Inflate test limits so we never trip on rate limiting during e2e runs.
			PublicMaxRequests: 100000,
			GeoMaxRequests:    100000,
			AdminMaxRequests:  100000,
			WindowDuration:    time.Minute,
			BlockDuration:     15 * time.Minute,
			CleanupThreshold:  time.Hour,
			CleanupInterval:   10 * time.Minute,
		},
		Worker: config.WorkerConfig{
			ShutdownTimeout:            30 * time.Second,
			HeartbeatInterval:          30 * time.Second,
			ReviewActivatorInterval:    time.Minute,
			ReviewActivatorBatchSize:   100,
			RateLimiterCleanupInterval: 10 * time.Minute,
		},
		Fraud: config.FraudConfig{
			// Session/request limits (irrelevant for review fraud tests, but keep
			// values aligned with config defaults so tests don't hit unexpected gates).
			MaxRequestsPerMinute: 100,
			MaxRequestsPerHour:   1000,
			BlockDurationMinutes: 15,
			ScrapingThreshold:    50,

			// Review fraud — match config.go envDefault values, otherwise all
			// thresholds collapse to zero and detectors misbehave.
			MutualReviewsPerMonth:       5,
			FastCompletionHours:         2,
			PerfectRatingsCount:         3,
			NewOrgBurstReviewsPerWeek:   10,
			ModerationScoreThreshold:    0.3,
			ActivationDelayDays:         7,
			SuspiciousDelayDays:         14,
			TextSimilarityThreshold:     0.8,
			TextSimilarityMinReviews:    3,
			TimingPatternWindowHours:    2,
			TimingPatternMinReviews:     10,
			RatingManipFriendAvgMin:     4.5,
			RatingManipOtherAvgMax:      2.5,
			RatingManipMinFriendReviews: 3,
			BurstAfterLowDays:           7,
			BurstAfterLowCount:          5,
			DormantDays:                 90,
			DormantBurstCount:           5,
		},
	}
}
