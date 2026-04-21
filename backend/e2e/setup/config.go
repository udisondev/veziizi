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
			MaxJSONBodySize:   1 << 20,  // 1 MiB — default envDefault
			MaxFileUploadSize: 10 << 20, // 10 MiB — default envDefault
			// Высокие пороги, чтобы blocked-логин-тесты (AUTH008 wrong password и
			// т.п.) не запирали аккаунт случайно. 0 здесь неверный выбор —
			// запрос `failed_login_count + 1 >= 0` всегда true, и каждый failed
			// login ставит locked_until.
			MaxFailedLoginAttempts: 1000,
			AccountLockoutDuration: time.Millisecond, // ≈ мгновенно разблокируется
			ShutdownTimeout:        30 * time.Second,
		},
		Worker: config.WorkerConfig{
			ShutdownTimeout:            30 * time.Second,
			HeartbeatInterval:          30 * time.Second,
			ReviewActivatorInterval:    1 * time.Minute,
			ReviewActivatorBatchSize:   100,
			RateLimiterCleanupInterval: 10 * time.Minute,
		},
		Fraud: config.FraudConfig{
			// Session fraud
			MaxKmPerHour:         900,
			MinDistanceForCheck:  100,
			UnusualHourThreshold: 3,
			MinLoginsForPattern:  5,
			MaxRequestsPerMinute: 100,
			MaxRequestsPerHour:   1000,
			BlockDurationMinutes: 15,
			ScrapingThreshold:    50,
			// Review fraud — совпадает с envDefault в config.go
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
