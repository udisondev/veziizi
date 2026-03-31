package config

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/go-playground/validator/v10"
)

type Config struct {
	Database  DatabaseConfig
	HTTP      HTTPConfig
	Session   SessionConfig
	Telegram  TelegramConfig
	Email     EmailConfig
	App       AppConfig
	GeoIP     GeoIPConfig
	RateLimit RateLimitConfig
}

type DatabaseConfig struct {
	URL string `env:"DATABASE_URL" envDefault:"postgres://veziizi:veziizi@localhost:5432/veziizi?sslmode=disable" validate:"required,url"`
}

type HTTPConfig struct {
	Addr           string        `env:"HTTP_ADDR" envDefault:":8080" validate:"required"`
	ReadTimeout    time.Duration `env:"HTTP_READ_TIMEOUT" envDefault:"15s" validate:"required"`
	WriteTimeout   time.Duration `env:"HTTP_WRITE_TIMEOUT" envDefault:"15s" validate:"required"`
	IdleTimeout    time.Duration `env:"HTTP_IDLE_TIMEOUT" envDefault:"60s" validate:"required"`
	TrustedProxies string        `env:"HTTP_TRUSTED_PROXIES" envDefault:"127.0.0.1,::1"` // Comma-separated list of trusted proxy IPs
}

type SessionConfig struct {
	Secret      string `env:"SESSION_SECRET" validate:"required_if=App.Env production"`
	AdminSecret string `env:"SESSION_ADMIN_SECRET"` // SEC-006: Отдельный ключ для admin сессий
	Name        string `env:"SESSION_NAME" envDefault:"veziizi_session" validate:"required"`
	AdminName   string `env:"SESSION_ADMIN_NAME" envDefault:"veziizi_admin_session" validate:"required"`
	MaxAge      int    `env:"SESSION_MAX_AGE" envDefault:"86400" validate:"required,min=1"`
}

type TelegramConfig struct {
	BotToken    string `env:"TELEGRAM_BOT_TOKEN"`
	BotUsername string `env:"TELEGRAM_BOT_USERNAME"` // Имя бота для Telegram Login Widget
}

type EmailConfig struct {
	// Provider: resend, smtp (default: resend)
	Provider string `env:"EMAIL_PROVIDER" envDefault:"resend" validate:"oneof=resend smtp"`
	// Resend API key (required when provider=resend)
	ResendAPIKey string `env:"RESEND_API_KEY"`
	// From address for outgoing emails
	FromAddress string `env:"EMAIL_FROM_ADDRESS" envDefault:"noreply@veziizi.ru"`
	// From name for outgoing emails
	FromName string `env:"EMAIL_FROM_NAME" envDefault:"Veziizi"`
	// SMTP settings (when provider=smtp)
	SMTPHost     string `env:"SMTP_HOST"`
	SMTPPort     int    `env:"SMTP_PORT" envDefault:"587"`
	SMTPUsername string `env:"SMTP_USERNAME"`
	SMTPPassword string `env:"SMTP_PASSWORD"`
	SMTPUseTLS   bool   `env:"SMTP_USE_TLS" envDefault:"true"`
	// Enabled flag - if false, emails won't be sent (useful for dev)
	Enabled bool `env:"EMAIL_ENABLED" envDefault:"false"`
}

type AppConfig struct {
	Env      string `env:"APP_ENV" envDefault:"development" validate:"required,oneof=development production"`
	LogLevel string `env:"LOG_LEVEL" envDefault:"debug" validate:"required,oneof=debug info warn error"`
	LogFile  string `env:"LOG_FILE" envDefault:""` // Path to log file (empty = stdout only)
	BaseURL  string `env:"APP_BASE_URL" envDefault:"http://localhost:5173"` // URL для ссылок в уведомлениях
}

type GeoIPConfig struct {
	// Path to MaxMind GeoLite2-City.mmdb database file
	// Download from: https://dev.maxmind.com/geoip/geolite2-free-geolocation-data
	DatabasePath string `env:"GEOIP_DATABASE_PATH" envDefault:""`
}

type RateLimitConfig struct {
	// Public endpoints (login, register, invitations)
	PublicMaxRequests int `env:"RATE_LIMIT_PUBLIC_MAX" envDefault:"10"`
	// Geo endpoints (higher limit for autocomplete)
	GeoMaxRequests int `env:"RATE_LIMIT_GEO_MAX" envDefault:"200"`
	// Admin endpoints
	AdminMaxRequests int `env:"RATE_LIMIT_ADMIN_MAX" envDefault:"50"`
	// Window duration for rate limiting
	WindowDuration time.Duration `env:"RATE_LIMIT_WINDOW" envDefault:"1m"`
	// Block duration when rate limited
	BlockDuration time.Duration `env:"RATE_LIMIT_BLOCK" envDefault:"15m"`
}

func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	validate := validator.New()
	if err := validate.Struct(cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	// SEC-013: Предупреждение о небезопасном SSL режиме в production
	cfg.validateSecuritySettings()

	return cfg, nil
}

// validateSecuritySettings проверяет критические настройки безопасности
func (c *Config) validateSecuritySettings() {
	if c.IsProduction() {
		// SEC-013: Проверка SSL для PostgreSQL
		if strings.Contains(c.Database.URL, "sslmode=disable") {
			slog.Warn("SEC-013: CRITICAL - PostgreSQL sslmode=disable in production!",
				slog.String("recommendation", "use sslmode=require or sslmode=verify-full"))
		}

		// SEC-006: Проверка отдельного ключа для admin сессий
		if c.Session.AdminSecret == "" {
			slog.Warn("SEC-006: SESSION_ADMIN_SECRET not set, using SESSION_SECRET for admin sessions")
		}

		// Email configuration warnings
		if c.Email.Enabled && c.Email.Provider == "resend" && c.Email.ResendAPIKey == "" {
			slog.Warn("EMAIL: RESEND_API_KEY not set but EMAIL_ENABLED=true",
				slog.String("recommendation", "set RESEND_API_KEY or disable email with EMAIL_ENABLED=false"))
		}
	}
}

func (c *Config) IsDevelopment() bool {
	return c.App.Env == "development"
}

func (c *Config) IsProduction() bool {
	return c.App.Env == "production"
}
