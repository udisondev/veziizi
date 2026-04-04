package main

import (
	"archive/zip"
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/udisondev/veziizi/backend/internal/pkg/config"
)

const (
	// GeoNames download URLs
	countriesURL     = "https://download.geonames.org/export/dump/countryInfo.txt"
	citiesURL        = "https://download.geonames.org/export/dump/cities1000.zip"
	alternateNameURL = "https://download.geonames.org/export/dump/alternateNamesV2.zip"

	batchSize = 5000
)

// Country from GeoNames countryInfo.txt
type Country struct {
	ISO2       string
	ISO3       string
	ISONumeric string
	FIPS       string
	Name       string
	Capital    string
	Area       float64
	Population int64
	Continent  string
	TLD        string
	Currency   string
	Phone      string
	GeonameID  int
}

// City from GeoNames cities1000.txt
type City struct {
	GeonameID   int
	Name        string
	ASCIIName   string
	CountryCode string
	Admin1Code  string
	Admin1Name  string
	Latitude    float64
	Longitude   float64
	Population  int64
	Timezone    string
}

// AlternateName from GeoNames alternateNamesV2.txt
type AlternateName struct {
	GeonameID   int
	Language    string
	Name        string
	IsPreferred bool
}

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

	// Check if data already exists
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

	tmpDir, err := os.MkdirTemp("", "geonames-*")
	if err != nil {
		slog.Error("failed to create temp dir", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			slog.Error("failed to remove temp dir", "error", err)
		}
	}()

	// Step 1: Download and parse countries
	slog.Info("downloading countries from GeoNames...")
	countries, err := downloadAndParseCountries(countriesURL)
	if err != nil {
		slog.Error("failed to download countries", "error", err)
		os.Exit(1)
	}
	slog.Info("parsed countries", "count", len(countries))

	// Build country code to geoname_id map
	countryCodeToID := make(map[string]int)
	for _, c := range countries {
		countryCodeToID[c.ISO2] = c.GeonameID
	}

	// Step 2: Download and parse cities
	slog.Info("downloading cities from GeoNames (this may take a while)...")
	cities, err := downloadAndParseCities(citiesURL, tmpDir)
	if err != nil {
		slog.Error("failed to download cities", "error", err)
		os.Exit(1)
	}
	slog.Info("parsed cities", "count", len(cities))

	// Step 3: Download and parse Russian translations
	slog.Info("downloading alternate names from GeoNames (this may take a while)...")
	russianNames, err := downloadAndParseAlternateNames(alternateNameURL, tmpDir, "ru")
	if err != nil {
		slog.Error("failed to download alternate names", "error", err)
		os.Exit(1)
	}
	slog.Info("parsed Russian translations", "count", len(russianNames))

	// Step 4: Seed countries
	if err := seedCountries(ctx, pool, countries, russianNames); err != nil {
		slog.Error("failed to seed countries", "error", err)
		os.Exit(1)
	}
	slog.Info("seeded countries", "count", len(countries))

	// Step 5: Seed cities with Russian names
	if err := seedCities(ctx, pool, cities, countryCodeToID, russianNames); err != nil {
		slog.Error("failed to seed cities", "error", err)
		os.Exit(1)
	}
	slog.Info("seeded cities", "count", len(cities))

	fmt.Println()
	fmt.Printf("Successfully seeded %d countries and %d cities with Russian translations!\n", len(countries), len(cities))
}

func downloadAndParseCountries(url string) ([]Country, error) {
	client := &http.Client{Timeout: 2 * time.Minute}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("http get: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Error("failed to close response body", "error", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var countries []Country
	scanner := bufio.NewScanner(resp.Body)
	// Increase buffer size for long lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		// Skip comments and empty lines
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Split(line, "\t")
		if len(fields) < 17 {
			continue
		}

		geonameID, _ := strconv.Atoi(fields[16])
		if geonameID == 0 {
			continue
		}

		countries = append(countries, Country{
			ISO2:      fields[0],
			ISO3:      fields[1],
			Name:      fields[4],
			Capital:   fields[5],
			Phone:     fields[12],
			GeonameID: geonameID,
		})
	}

	return countries, scanner.Err()
}

func downloadAndParseCities(url string, tmpDir string) ([]City, error) {
	zipPath := filepath.Join(tmpDir, "cities1000.zip")

	if err := downloadFile(url, zipPath); err != nil {
		return nil, fmt.Errorf("download cities: %w", err)
	}

	// Open and extract zip
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}
	defer func() {
		if err := reader.Close(); err != nil {
			slog.Error("failed to close zip reader", "error", err)
		}
	}()

	var cities []City

	for _, file := range reader.File {
		if !strings.HasSuffix(file.Name, ".txt") {
			continue
		}

		parsed, err := parseCitiesFromZipFile(file)
		if err != nil {
			return nil, err
		}
		cities = append(cities, parsed...)
	}

	return cities, nil
}

func parseCitiesFromZipFile(file *zip.File) ([]City, error) {
	rc, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("open file in zip: %w", err)
	}
	defer rc.Close()

	var cities []City
	scanner := bufio.NewScanner(rc)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, "\t")
		if len(fields) < 19 {
			continue
		}

		geonameID, _ := strconv.Atoi(fields[0])
		lat, _ := strconv.ParseFloat(fields[4], 64)
		lon, _ := strconv.ParseFloat(fields[5], 64)
		pop, _ := strconv.ParseInt(fields[14], 10, 64)

		// Feature class P = populated place
		if fields[6] != "P" {
			continue
		}

		cities = append(cities, City{
			GeonameID:   geonameID,
			Name:        fields[1],
			ASCIIName:   fields[2],
			Latitude:    lat,
			Longitude:   lon,
			CountryCode: fields[8],
			Admin1Code:  fields[10],
			Admin1Name:  fields[10], // Will be updated if we have admin names
			Population:  pop,
			Timezone:    fields[17],
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan cities: %w", err)
	}

	return cities, nil
}

func downloadAndParseAlternateNames(url string, tmpDir string, language string) (map[int]string, error) {
	zipPath := filepath.Join(tmpDir, "alternateNamesV2.zip")

	if err := downloadFile(url, zipPath); err != nil {
		return nil, fmt.Errorf("download alternate names: %w", err)
	}

	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}
	defer func() {
		if err := reader.Close(); err != nil {
			slog.Error("failed to close zip reader", "error", err)
		}
	}()

	names := make(map[int]string)
	preferred := make(map[int]bool)

	for _, file := range reader.File {
		if !strings.HasSuffix(file.Name, ".txt") {
			continue
		}

		if err := parseAlternateNamesFromZipFile(file, language, names, preferred); err != nil {
			return nil, err
		}
	}

	return names, nil
}

func parseAlternateNamesFromZipFile(file *zip.File, language string, names map[int]string, preferred map[int]bool) error {
	rc, err := file.Open()
	if err != nil {
		return fmt.Errorf("open file in zip: %w", err)
	}
	defer rc.Close()

	scanner := bufio.NewScanner(rc)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	lineCount := 0
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, "\t")
		if len(fields) < 4 {
			continue
		}

		// Check if this is the language we want
		if fields[2] != language {
			continue
		}

		geonameID, _ := strconv.Atoi(fields[1])
		name := fields[3]
		isPreferred := len(fields) > 4 && fields[4] == "1"

		// Prefer preferred names, otherwise take first one
		if _, exists := names[geonameID]; !exists || (isPreferred && !preferred[geonameID]) {
			names[geonameID] = name
			preferred[geonameID] = isPreferred
		}

		lineCount++
		if lineCount%1000000 == 0 {
			slog.Info("processing alternate names", "lines", lineCount, "found", len(names))
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan alternate names: %w", err)
	}

	return nil
}

func downloadFile(url string, destPath string) error {
	client := &http.Client{Timeout: 10 * time.Minute}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	file, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer file.Close()

	written, err := io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("copy: %w", err)
	}

	slog.Info("downloaded file", "path", destPath, "size_mb", written/1024/1024)
	return nil
}

func seedCountries(ctx context.Context, pool *pgxpool.Pool, countries []Country, russianNames map[int]string) error {
	rows := make([][]any, 0, len(countries))

	for _, c := range countries {
		nameRu := russianNames[c.GeonameID]
		if nameRu == "" {
			nameRu = c.Name // Fallback to English
		}

		rows = append(rows, []any{
			c.GeonameID,
			c.Name,
			c.ISO2,
			nullableString(c.ISO3),
			nullableString(c.Phone),
			nullableString(nameRu),
			nil, // latitude (not in countryInfo)
			nil, // longitude (not in countryInfo)
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

func seedCities(ctx context.Context, pool *pgxpool.Pool, cities []City, countryCodeToID map[string]int, russianNames map[int]string) error {
	// Build country_id lookup from geoname_id
	// First, get country geoname_ids from database
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

	// Seed in batches
	for i := 0; i < len(cities); i += batchSize {
		end := i + batchSize
		if end > len(cities) {
			end = len(cities)
		}

		batch := cities[i:end]
		rows := make([][]any, 0, len(batch))

		for _, c := range batch {
			countryID, ok := countryIDByCode[c.CountryCode]
			if !ok {
				continue // Skip cities without matching country
			}

			nameRu := russianNames[c.GeonameID]
			if nameRu == "" {
				nameRu = c.Name // Fallback to English
			}

			rows = append(rows, []any{
				c.GeonameID,
				c.Name,
				countryID,
				nullableString(c.Admin1Name),
				nil, // state_code
				c.Latitude,
				c.Longitude,
				nullableString(nameRu),
			})
		}

		_, err := pool.CopyFrom(
			ctx,
			pgx.Identifier{"geo_cities"},
			[]string{"id", "name", "country_id", "state_name", "state_code", "latitude", "longitude", "name_ru"},
			pgx.CopyFromRows(rows),
		)
		if err != nil {
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
