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
	Redis     RedisConfig
	HTTP      HTTPConfig
	Session   SessionConfig
	Telegram  TelegramConfig
	Email     EmailConfig
	App       AppConfig
	GeoIP     GeoIPConfig
	RateLimit RateLimitConfig
	Metrics   MetricsConfig
	Security  SecurityConfig
	Worker    WorkerConfig
	Fraud     FraudConfig
}

type MetricsConfig struct {
	Addr    string `env:"METRICS_ADDR" envDefault:":9090"`
	Enabled bool   `env:"METRICS_ENABLED" envDefault:"true"`
}

type DatabaseConfig struct {
	URL string `env:"DATABASE_URL" envDefault:"postgres://veziizi:veziizi@localhost:5432/veziizi?sslmode=disable" validate:"required,url"`
}

// RedisConfig — настройки Redis Streams (транспорт событий между forwarder'ом
// и воркерами). Postgres outbox остаётся источником истины: события публикуются
// в watermill_messages_<OutboxTopic> транзакционно, forwarder перекладывает их
// в Redis-стримы, воркеры читают стримы через consumer groups.
type RedisConfig struct {
	URL string `env:"REDIS_URL" envDefault:"redis://localhost:6379/0" validate:"required"`
	// ConsumerName — имя консьюмера внутри consumer group. ДОЛЖНО быть уникальным
	// на инстанс воркера (иначе реплики делят один pending-список и дерутся за
	// XCLAIM). Пустое значение → worker.Run подставит "<worker>-<hostname>".
	ConsumerName string `env:"REDIS_CONSUMER_NAME" envDefault:""`
	// MaxIdleTime — сколько pending-сообщение упавшего консьюмера висит до того,
	// как его заберёт (XAUTOCLAIM) живой консьюмер той же группы.
	MaxIdleTime time.Duration `env:"REDIS_MAX_IDLE_TIME" envDefault:"60s"`
	// ClaimInterval — как часто консьюмер проверяет чужие зависшие pending.
	ClaimInterval time.Duration `env:"REDIS_CLAIM_INTERVAL" envDefault:"5s"`
	// NackResendSleep — пауза перед повторной доставкой nack'нутого сообщения.
	NackResendSleep time.Duration `env:"REDIS_NACK_RESEND_SLEEP" envDefault:"100ms"`
	// BlockTime — таймаут блокирующего XREADGROUP.
	BlockTime time.Duration `env:"REDIS_BLOCK_TIME" envDefault:"100ms"`
	// MaxLen — приблизительный потолок длины стрима (XADD MAXLEN ~). Защита от
	// безграничного роста; источник истины — event store, стрим можно терять.
	MaxLen int64 `env:"REDIS_STREAM_MAXLEN" envDefault:"100000"`
}

type HTTPConfig struct {
	Addr           string        `env:"HTTP_ADDR" envDefault:":8080" validate:"required"`
	ReadTimeout    time.Duration `env:"HTTP_READ_TIMEOUT" envDefault:"15s" validate:"required"`
	WriteTimeout   time.Duration `env:"HTTP_WRITE_TIMEOUT" envDefault:"15s" validate:"required"`
	IdleTimeout    time.Duration `env:"HTTP_IDLE_TIMEOUT" envDefault:"60s" validate:"required"`
	TrustedProxies string        `env:"HTTP_TRUSTED_PROXIES" envDefault:"127.0.0.1,::1"`
	CORSOrigins    string        `env:"CORS_ORIGINS"`
}

type SessionConfig struct {
	Secret      string `env:"SESSION_SECRET"`
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
	Env      string `env:"APP_ENV" envDefault:"development" validate:"required,oneof=development staging production"`
	LogLevel string `env:"LOG_LEVEL" envDefault:"debug" validate:"required,oneof=debug info warn error"`
	LogFile  string `env:"LOG_FILE" envDefault:""`                          // Path to log file (empty = stdout only)
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
	// Cleanup threshold — entries older than this are removed
	CleanupThreshold time.Duration `env:"RATE_LIMIT_CLEANUP_THRESHOLD" envDefault:"1h"`
	// Cleanup interval — how often cleanup runs
	CleanupInterval time.Duration `env:"RATE_LIMIT_CLEANUP_INTERVAL" envDefault:"10m"`
}

type SecurityConfig struct {
	// Max JSON body size in bytes
	MaxJSONBodySize int64 `env:"MAX_JSON_BODY_SIZE" envDefault:"1048576"` // 1MB
	// Max file upload size in bytes
	MaxFileUploadSize int64 `env:"MAX_FILE_UPLOAD_SIZE" envDefault:"10485760"` // 10MB
	// Max failed login attempts before lockout
	MaxFailedLoginAttempts int `env:"MAX_FAILED_LOGIN_ATTEMPTS" envDefault:"5"`
	// Account lockout duration
	AccountLockoutDuration time.Duration `env:"ACCOUNT_LOCKOUT_DURATION" envDefault:"15m"`
	// Shutdown timeout for HTTP server
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"30s"`
}

type WorkerConfig struct {
	// Shutdown timeout for workers
	ShutdownTimeout time.Duration `env:"WORKER_SHUTDOWN_TIMEOUT" envDefault:"30s"`
	// Heartbeat interval
	HeartbeatInterval time.Duration `env:"WORKER_HEARTBEAT_INTERVAL" envDefault:"30s"`
	// Review activator interval
	ReviewActivatorInterval time.Duration `env:"REVIEW_ACTIVATOR_INTERVAL" envDefault:"1m"`
	// Review activator batch size
	ReviewActivatorBatchSize int `env:"REVIEW_ACTIVATOR_BATCH_SIZE" envDefault:"100"`
	// Rate limiter cleanup interval
	RateLimiterCleanupInterval time.Duration `env:"RATE_LIMITER_CLEANUP_INTERVAL" envDefault:"10m"`

	// Retry middleware parameters applied to every event-driven worker.
	RetryMaxRetries      int           `env:"WORKER_RETRY_MAX" envDefault:"5"`
	RetryInitialInterval time.Duration `env:"WORKER_RETRY_INITIAL_INTERVAL" envDefault:"1s"`
	RetryMaxInterval     time.Duration `env:"WORKER_RETRY_MAX_INTERVAL" envDefault:"1m"`
	RetryMultiplier      float64       `env:"WORKER_RETRY_MULTIPLIER" envDefault:"2"`

	// DeadLetterTopic — куда уезжают сообщения, которые Retry исчерпал. Пустая
	// строка отключает PoisonQueue (тогда после исчерпания попыток сообщение
	// останется в горячем nack-цикле, как было до этапа 3).
	DeadLetterTopic string `env:"WORKER_DEADLETTER_TOPIC" envDefault:"deadletter"`
}

type FraudConfig struct {
	// Session fraud
	MaxKmPerHour         float64 `env:"FRAUD_MAX_KM_PER_HOUR" envDefault:"900"`
	MinDistanceForCheck  float64 `env:"FRAUD_MIN_DISTANCE_FOR_CHECK" envDefault:"100"`
	UnusualHourThreshold int     `env:"FRAUD_UNUSUAL_HOUR_THRESHOLD" envDefault:"3"`
	MinLoginsForPattern  int     `env:"FRAUD_MIN_LOGINS_FOR_PATTERN" envDefault:"5"`
	MaxRequestsPerMinute int     `env:"FRAUD_MAX_REQUESTS_PER_MINUTE" envDefault:"100"`
	MaxRequestsPerHour   int     `env:"FRAUD_MAX_REQUESTS_PER_HOUR" envDefault:"1000"`
	BlockDurationMinutes int     `env:"FRAUD_BLOCK_DURATION_MINUTES" envDefault:"15"`
	ScrapingThreshold    int     `env:"FRAUD_SCRAPING_THRESHOLD" envDefault:"50"`

	// Review fraud
	MutualReviewsPerMonth       int     `env:"FRAUD_MUTUAL_REVIEWS_PER_MONTH" envDefault:"5"`
	FastCompletionHours         int     `env:"FRAUD_FAST_COMPLETION_HOURS" envDefault:"2"`
	PerfectRatingsCount         int     `env:"FRAUD_PERFECT_RATINGS_COUNT" envDefault:"3"`
	NewOrgBurstReviewsPerWeek   int     `env:"FRAUD_NEW_ORG_BURST_REVIEWS_PER_WEEK" envDefault:"10"`
	ModerationScoreThreshold    float64 `env:"FRAUD_MODERATION_SCORE_THRESHOLD" envDefault:"0.3"`
	ActivationDelayDays         int     `env:"FRAUD_ACTIVATION_DELAY_DAYS" envDefault:"7"`
	SuspiciousDelayDays         int     `env:"FRAUD_SUSPICIOUS_DELAY_DAYS" envDefault:"14"`
	TextSimilarityThreshold     float64 `env:"FRAUD_TEXT_SIMILARITY_THRESHOLD" envDefault:"0.8"`
	TextSimilarityMinReviews    int     `env:"FRAUD_TEXT_SIMILARITY_MIN_REVIEWS" envDefault:"3"`
	TimingPatternWindowHours    int     `env:"FRAUD_TIMING_PATTERN_WINDOW_HOURS" envDefault:"2"`
	TimingPatternMinReviews     int     `env:"FRAUD_TIMING_PATTERN_MIN_REVIEWS" envDefault:"10"`
	RatingManipFriendAvgMin     float64 `env:"FRAUD_RATING_MANIP_FRIEND_AVG_MIN" envDefault:"4.5"`
	RatingManipOtherAvgMax      float64 `env:"FRAUD_RATING_MANIP_OTHER_AVG_MAX" envDefault:"2.5"`
	RatingManipMinFriendReviews int     `env:"FRAUD_RATING_MANIP_MIN_FRIEND_REVIEWS" envDefault:"3"`
	BurstAfterLowDays           int     `env:"FRAUD_BURST_AFTER_LOW_DAYS" envDefault:"7"`
	BurstAfterLowCount          int     `env:"FRAUD_BURST_AFTER_LOW_COUNT" envDefault:"5"`
	DormantDays                 int     `env:"FRAUD_DORMANT_DAYS" envDefault:"90"`
	DormantBurstCount           int     `env:"FRAUD_DORMANT_BURST_COUNT" envDefault:"5"`
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

	// SESSION_SECRET обязателен в production (required_if не работает кросс-struct)
	if cfg.IsProduction() && cfg.Session.Secret == "" {
		return nil, fmt.Errorf("config validation failed: SESSION_SECRET is required in production")
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
