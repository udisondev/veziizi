package projections

import (
	"context"
	"fmt"

	"codeberg.org/udison/veziizi/backend/internal/pkg/dbtx"
	"github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/v2/pgxscan"
)

// GeoProjection provides read access to geo data (countries and cities)
type GeoProjection struct {
	db   dbtx.TxManager
	psql squirrel.StatementBuilderType
}

// NewGeoProjection creates a new GeoProjection
func NewGeoProjection(db dbtx.TxManager) *GeoProjection {
	return &GeoProjection{
		db:   db,
		psql: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

// Country represents a country from geo_countries table
type Country struct {
	ID         int      `db:"id" json:"id"`
	Name       string   `db:"name" json:"name"`
	NameRu     *string  `db:"name_ru" json:"name_ru,omitempty"`
	ISO2       string   `db:"iso2" json:"iso2"`
	ISO3       *string  `db:"iso3" json:"iso3,omitempty"`
	PhoneCode  *string  `db:"phone_code" json:"phone_code,omitempty"`
	NativeName *string  `db:"native_name" json:"native_name,omitempty"`
	Latitude   *float64 `db:"latitude" json:"latitude,omitempty"`
	Longitude  *float64 `db:"longitude" json:"longitude,omitempty"`
}

// City represents a city from geo_cities table
type City struct {
	ID            int      `db:"id" json:"id"`
	Name          string   `db:"name" json:"name"`
	NameRu        *string  `db:"name_ru" json:"name_ru,omitempty"`
	CountryID     int      `db:"country_id" json:"country_id"`
	StateName     *string  `db:"state_name" json:"state_name,omitempty"`
	StateCode     *string  `db:"state_code" json:"state_code,omitempty"`
	Latitude      float64  `db:"latitude" json:"latitude"`
	Longitude     float64  `db:"longitude" json:"longitude"`
	CountryName   *string  `db:"country_name" json:"country_name,omitempty"`     // joined field
	CountryNameRu *string  `db:"country_name_ru" json:"country_name_ru,omitempty"` // joined field
	CountryISO2   *string  `db:"country_iso2" json:"country_iso2,omitempty"`     // joined field
}

// ListCountries returns all countries ordered by Russian name (fallback to English)
func (p *GeoProjection) ListCountries(ctx context.Context) ([]Country, error) {
	query, args, err := p.psql.
		Select("id", "name", "name_ru", "iso2", "iso3", "phone_code", "native_name", "latitude", "longitude").
		From("geo_countries").
		OrderBy("COALESCE(name_ru, name)").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select query: %w", err)
	}

	var countries []Country
	if err := pgxscan.Select(ctx, p.db, &countries, query, args...); err != nil {
		return nil, fmt.Errorf("list countries: %w", err)
	}

	return countries, nil
}

// GetCountry returns a country by ID
func (p *GeoProjection) GetCountry(ctx context.Context, id int) (*Country, error) {
	query, args, err := p.psql.
		Select("id", "name", "name_ru", "iso2", "iso3", "phone_code", "native_name", "latitude", "longitude").
		From("geo_countries").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select query: %w", err)
	}

	var country Country
	if err := pgxscan.Get(ctx, p.db, &country, query, args...); err != nil {
		return nil, fmt.Errorf("get country: %w", err)
	}

	return &country, nil
}

// GetCountryByISO2 returns a country by ISO2 code
func (p *GeoProjection) GetCountryByISO2(ctx context.Context, iso2 string) (*Country, error) {
	query, args, err := p.psql.
		Select("id", "name", "name_ru", "iso2", "iso3", "phone_code", "native_name", "latitude", "longitude").
		From("geo_countries").
		Where(squirrel.Eq{"iso2": iso2}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select query: %w", err)
	}

	var country Country
	if err := pgxscan.Get(ctx, p.db, &country, query, args...); err != nil {
		return nil, fmt.Errorf("get country by iso2: %w", err)
	}

	return &country, nil
}

// SearchCitiesOptions contains options for city search
type SearchCitiesOptions struct {
	OnlyWithTranslation bool // Show only cities with Russian translation (name_ru != name)
}

// SearchCities searches cities by name (English or Russian) within a country
func (p *GeoProjection) SearchCities(ctx context.Context, countryID int, search string, limit int, opts *SearchCitiesOptions) ([]City, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	builder := p.psql.
		Select(
			"c.id", "c.name", "c.name_ru", "c.country_id", "c.state_name", "c.state_code",
			"c.latitude", "c.longitude",
			"co.name AS country_name", "co.name_ru AS country_name_ru", "co.iso2 AS country_iso2",
		).
		From("geo_cities c").
		Join("geo_countries co ON co.id = c.country_id").
		Where(squirrel.Eq{"c.country_id": countryID}).
		OrderBy("COALESCE(c.name_ru, c.name)").
		Limit(uint64(limit))

	// Filter only cities with actual Russian translations
	if opts != nil && opts.OnlyWithTranslation {
		builder = builder.Where("c.name_ru IS NOT NULL AND c.name_ru != c.name")
	}

	if search != "" {
		// Search by both English and Russian names
		builder = builder.Where("(c.name ILIKE ? OR c.name_ru ILIKE ?)", search+"%", search+"%")
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build search query: %w", err)
	}

	var cities []City
	if err := pgxscan.Select(ctx, p.db, &cities, query, args...); err != nil {
		return nil, fmt.Errorf("search cities: %w", err)
	}

	return cities, nil
}

// GetCity returns a city by ID with country info
func (p *GeoProjection) GetCity(ctx context.Context, id int) (*City, error) {
	query, args, err := p.psql.
		Select(
			"c.id", "c.name", "c.name_ru", "c.country_id", "c.state_name", "c.state_code",
			"c.latitude", "c.longitude",
			"co.name AS country_name", "co.name_ru AS country_name_ru", "co.iso2 AS country_iso2",
		).
		From("geo_cities c").
		Join("geo_countries co ON co.id = c.country_id").
		Where(squirrel.Eq{"c.id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select query: %w", err)
	}

	var city City
	if err := pgxscan.Get(ctx, p.db, &city, query, args...); err != nil {
		return nil, fmt.Errorf("get city: %w", err)
	}

	return &city, nil
}

// GetCities returns cities by IDs with country info
func (p *GeoProjection) GetCities(ctx context.Context, ids []int) ([]City, error) {
	if len(ids) == 0 {
		return []City{}, nil
	}

	query, args, err := p.psql.
		Select(
			"c.id", "c.name", "c.name_ru", "c.country_id", "c.state_name", "c.state_code",
			"c.latitude", "c.longitude",
			"co.name AS country_name", "co.name_ru AS country_name_ru", "co.iso2 AS country_iso2",
		).
		From("geo_cities c").
		Join("geo_countries co ON co.id = c.country_id").
		Where(squirrel.Eq{"c.id": ids}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select query: %w", err)
	}

	var cities []City
	if err := pgxscan.Select(ctx, p.db, &cities, query, args...); err != nil {
		return nil, fmt.Errorf("get cities: %w", err)
	}

	return cities, nil
}

// CountCities returns the number of cities in a country
func (p *GeoProjection) CountCities(ctx context.Context, countryID int) (int, error) {
	query, args, err := p.psql.
		Select("COUNT(*)").
		From("geo_cities").
		Where(squirrel.Eq{"country_id": countryID}).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build count query: %w", err)
	}

	var count int
	if err := p.db.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count cities: %w", err)
	}

	return count, nil
}
