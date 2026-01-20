package setup

import (
	"os"
	"time"

	"codeberg.org/udison/veziizi/backend/internal/pkg/config"
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
	}
}
