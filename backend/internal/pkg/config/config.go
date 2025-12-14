package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/go-playground/validator/v10"
)

type Config struct {
	Database DatabaseConfig
	HTTP     HTTPConfig
	Session  SessionConfig
	Telegram TelegramConfig
	App      AppConfig
}

type DatabaseConfig struct {
	URL string `env:"DATABASE_URL" envDefault:"postgres://veziizi:veziizi@localhost:5432/veziizi?sslmode=disable" validate:"required,url"`
}

type HTTPConfig struct {
	Addr         string        `env:"HTTP_ADDR" envDefault:":8080" validate:"required"`
	ReadTimeout  time.Duration `env:"HTTP_READ_TIMEOUT" envDefault:"15s" validate:"required"`
	WriteTimeout time.Duration `env:"HTTP_WRITE_TIMEOUT" envDefault:"15s" validate:"required"`
	IdleTimeout  time.Duration `env:"HTTP_IDLE_TIMEOUT" envDefault:"60s" validate:"required"`
}

type SessionConfig struct {
	Secret string `env:"SESSION_SECRET" validate:"required_if=App.Env production"`
	Name   string `env:"SESSION_NAME" envDefault:"veziizi_session" validate:"required"`
	MaxAge int    `env:"SESSION_MAX_AGE" envDefault:"86400" validate:"required,min=1"`
}

type TelegramConfig struct {
	BotToken string `env:"TELEGRAM_BOT_TOKEN"`
}

type AppConfig struct {
	Env      string `env:"APP_ENV" envDefault:"development" validate:"required,oneof=development production"`
	LogLevel string `env:"LOG_LEVEL" envDefault:"debug" validate:"required,oneof=debug info warn error"`
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

	return cfg, nil
}

func (c *Config) IsDevelopment() bool {
	return c.App.Env == "development"
}

func (c *Config) IsProduction() bool {
	return c.App.Env == "production"
}
