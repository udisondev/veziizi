package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/geodata"
	"github.com/udisondev/veziizi/backend/internal/pkg/config"
)

const batchSize = 5000

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.Database.URL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	var countryCount int
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM geo_countries").Scan(&countryCount); err != nil {
		slog.Error("failed to check existing data", "error", err)
		os.Exit(1)
	}

	if countryCount > 0 {
		slog.Info("geo data already exists", "countries", countryCount)
		fmt.Println("Geo data already exists. Use --force to re-seed (will delete existing data).")

		if len(os.Args) > 1 && os.Args[1] == "--force" {
			slog.Info("force flag detected, clearing existing data")
			if _, err := pool.Exec(ctx, "TRUNCATE geo_cities, geo_countries RESTART IDENTITY CASCADE"); err != nil {
				slog.Error("failed to truncate tables", "error", err)
				os.Exit(1)
			}
		} else {
			return
		}
	}

	slog.Info("loading embedded countries...")
	countries, err := geodata.LoadCountries()
	if err != nil {
		slog.Error("failed to load countries", "error", err)
		os.Exit(1)
	}
	slog.Info("loaded countries", "count", len(countries))

	slog.Info("loading embedded cities...")
	cities, err := geodata.LoadCities()
	if err != nil {
		slog.Error("failed to load cities", "error", err)
		os.Exit(1)
	}
	slog.Info("loaded cities", "count", len(cities))

	if err := seedCountries(ctx, pool, countries); err != nil {
		slog.Error("failed to seed countries", "error", err)
		os.Exit(1)
	}
	slog.Info("seeded countries", "count", len(countries))

	if err := seedCities(ctx, pool, cities); err != nil {
		slog.Error("failed to seed cities", "error", err)
		os.Exit(1)
	}
	slog.Info("seeded cities", "count", len(cities))

	fmt.Printf("\nSuccessfully seeded %d countries and %d cities with Russian translations!\n", len(countries), len(cities))
}

func seedCountries(ctx context.Context, pool *pgxpool.Pool, countries []geodata.Country) error {
	rows := make([][]any, 0, len(countries))

	for _, c := range countries {
		rows = append(rows, []any{
			c.GeonameID,
			c.Name,
			c.ISO2,
			nullableString(c.ISO3),
			nullableString(c.Phone),
			nullableString(c.NameRu),
			nil,
			nil,
		})
	}

	_, err := pool.CopyFrom(
		ctx,
		pgx.Identifier{"geo_countries"},
		[]string{"id", "name", "iso2", "iso3", "phone_code", "name_ru", "latitude", "longitude"},
		pgx.CopyFromRows(rows),
	)
	return err
}

func seedCities(ctx context.Context, pool *pgxpool.Pool, cities []geodata.City) error {
	countryRows, err := pool.Query(ctx, "SELECT id, iso2 FROM geo_countries")
	if err != nil {
		return fmt.Errorf("query countries: %w", err)
	}

	countryIDByCode := make(map[string]int)
	for countryRows.Next() {
		var id int
		var iso2 string
		if err := countryRows.Scan(&id, &iso2); err != nil {
			countryRows.Close()
			return fmt.Errorf("scan country: %w", err)
		}
		countryIDByCode[iso2] = id
	}
	countryRows.Close()
	if err := countryRows.Err(); err != nil {
		return fmt.Errorf("iterate countries: %w", err)
	}

	for i := 0; i < len(cities); i += batchSize {
		end := min(i+batchSize, len(cities))

		batch := cities[i:end]
		rows := make([][]any, 0, len(batch))

		for _, c := range batch {
			countryID, ok := countryIDByCode[c.CountryCode]
			if !ok {
				continue
			}
			rows = append(rows, []any{
				c.GeonameID,
				c.Name,
				countryID,
				nullableString(c.Admin1Code),
				nil,
				c.Latitude,
				c.Longitude,
				nullableString(c.NameRu),
			})
		}

		if _, err := pool.CopyFrom(
			ctx,
			pgx.Identifier{"geo_cities"},
			[]string{"id", "name", "country_id", "state_name", "state_code", "latitude", "longitude", "name_ru"},
			pgx.CopyFromRows(rows),
		); err != nil {
			return fmt.Errorf("batch %d-%d: %w", i, end, err)
		}

		slog.Info("seeded batch", "progress", fmt.Sprintf("%d/%d", end, len(cities)))
	}

	return nil
}

func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
