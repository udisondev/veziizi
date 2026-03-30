package setup

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/udisondev/veziizi/backend/internal/pkg/config"
	"github.com/udisondev/veziizi/backend/migrations"
	wmSql "github.com/ThreeDotsLabs/watermill-sql/v4/pkg/sql"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
	"golang.org/x/crypto/bcrypt"
)

// runMigrations applies database migrations using embedded SQL files.
// Uses goose Provider API for reliable embedded migrations.
func runMigrations(cfg *config.Config) error {
	db, err := sql.Open("postgres", cfg.Database.URL)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrations.FS)
	if err != nil {
		return fmt.Errorf("failed to create goose provider: %w", err)
	}

	if _, err := provider.Up(context.Background()); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// initWatermillSchema creates Watermill tables for all topics.
// Uses SchemaInitializingQueries() to get DDL and execute directly.
func initWatermillSchema(cfg *config.Config) error {
	db, err := sql.Open("postgres", cfg.Database.URL)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	schema := wmSql.DefaultPostgreSQLSchema{}

	topics := []string{
		"organization.events",
		"freightrequest.events",
		"order.events",
		"review.events",
		"notification.events",
		"notification.send",
		"support.events",
	}

	for _, topic := range topics {
		queries, err := schema.SchemaInitializingQueries(wmSql.SchemaInitializingQueriesParams{
			Topic: topic,
		})
		if err != nil {
			return fmt.Errorf("get schema queries for %s: %w", topic, err)
		}

		for _, q := range queries {
			if _, err := db.Exec(q.Query, q.Args...); err != nil {
				return fmt.Errorf("execute schema query for %s: %w", topic, err)
			}
		}
	}

	return nil
}

// CleanDatabase removes test data while preserving schema.
// Use this between tests that need complete isolation.
func CleanDatabase(cfg *config.Config) error {
	db, err := sql.Open("postgres", cfg.Database.URL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Tables to clean (in order respecting foreign keys)
	tables := []string{
		"notification_delivery_log",
		"inapp_notifications",
		"notification_preferences",
		"telegram_link_codes",
		"freight_subscriptions",
		"reviews_lookup",
		"fraud_signals",
		"fraud_data",
		"interaction_stats",
		"session_fraud_signals",
		"order_fraud_signals",
		"orders_lookup",
		"offers_lookup",
		"freight_requests_lookup",
		"organization_ratings",
		"invitations_lookup",
		"members_lookup",
		"pending_organizations",
		"organizations_lookup",
		"support_tickets",
		"event_store",
		"platform_admins",
	}

	for _, table := range tables {
		if _, err := db.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)); err != nil {
			// Table might not exist, ignore
			continue
		}
	}

	// Reset sequences
	sequences := []string{
		"freight_requests_seq",
		"orders_seq",
	}

	for _, seq := range sequences {
		if _, err := db.Exec(fmt.Sprintf("ALTER SEQUENCE IF EXISTS %s RESTART WITH 1", seq)); err != nil {
			continue
		}
	}

	return nil
}

// CreateTestAdmin creates a test admin user.
func CreateTestAdmin(cfg *config.Config) error {
	db, err := sql.Open("postgres", cfg.Database.URL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Hash password with bcrypt MinCost for faster tests
	hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.MinCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	_, err = db.Exec(`
		INSERT INTO platform_admins (id, email, password_hash, name, is_active)
		VALUES ($1, $2, $3, $4, true)
		ON CONFLICT (email) DO NOTHING
	`, uuid.New(), "admin@veziizi.local", string(hash), "Test Admin")
	if err != nil {
		return fmt.Errorf("failed to create test admin: %w", err)
	}

	return nil
}

// SeedGeoData inserts test geographic data.
func SeedGeoData(cfg *config.Config) error {
	db, err := sql.Open("postgres", cfg.Database.URL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Insert test countries into geo_countries
	_, err = db.Exec(`
		INSERT INTO geo_countries (id, name, name_ru, iso2) VALUES
		(1, 'Russia', 'Россия', 'RU'),
		(2, 'Kazakhstan', 'Казахстан', 'KZ'),
		(3, 'Belarus', 'Беларусь', 'BY')
		ON CONFLICT (id) DO NOTHING
	`)
	if err != nil {
		return fmt.Errorf("failed to seed countries: %w", err)
	}

	// Insert test cities into geo_cities
	_, err = db.Exec(`
		INSERT INTO geo_cities (id, country_id, name, name_ru, latitude, longitude) VALUES
		(1, 1, 'Moscow', 'Москва', 55.7558, 37.6173),
		(2, 1, 'Saint Petersburg', 'Санкт-Петербург', 59.9311, 30.3609),
		(3, 1, 'Novosibirsk', 'Новосибирск', 55.0084, 82.9357),
		(4, 2, 'Almaty', 'Алматы', 43.2220, 76.8512),
		(5, 3, 'Minsk', 'Минск', 53.9045, 27.5615)
		ON CONFLICT (id) DO NOTHING
	`)
	if err != nil {
		return fmt.Errorf("failed to seed cities: %w", err)
	}

	return nil
}
