-- +goose Up
-- +goose StatementBegin

-- Enable pg_trgm extension for fast substring search
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Countries reference table
-- Data from https://github.com/dr5hn/countries-states-cities-database
-- Consolidated from: 20251222100000, 20251222150000
CREATE TABLE geo_countries (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    name_ru TEXT,
    iso2 CHAR(2) NOT NULL UNIQUE,
    iso3 CHAR(3),
    phone_code VARCHAR(20),
    native_name TEXT,
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8)
);

CREATE INDEX idx_geo_countries_iso2 ON geo_countries(iso2);
CREATE INDEX idx_geo_countries_name ON geo_countries(name);
CREATE INDEX idx_geo_countries_name_ru ON geo_countries(name_ru);

COMMENT ON TABLE geo_countries IS 'Countries reference table';
COMMENT ON COLUMN geo_countries.id IS 'ID from dr5hn database';
COMMENT ON COLUMN geo_countries.name IS 'Country name in English';
COMMENT ON COLUMN geo_countries.name_ru IS 'Country name in Russian';
COMMENT ON COLUMN geo_countries.iso2 IS 'ISO 3166-1 alpha-2 code';
COMMENT ON COLUMN geo_countries.iso3 IS 'ISO 3166-1 alpha-3 code';
COMMENT ON COLUMN geo_countries.phone_code IS 'Phone country code';
COMMENT ON COLUMN geo_countries.native_name IS 'Country name in native language';
COMMENT ON COLUMN geo_countries.latitude IS 'Country center latitude';
COMMENT ON COLUMN geo_countries.longitude IS 'Country center longitude';

-- Cities reference table
-- Data from https://github.com/dr5hn/countries-states-cities-database
-- Consolidated from: 20251222100001, 20251222150000
CREATE TABLE geo_cities (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    name_ru TEXT,
    country_id INTEGER NOT NULL REFERENCES geo_countries(id) ON DELETE CASCADE,
    state_name TEXT,
    state_code VARCHAR(10),
    latitude DECIMAL(10, 8) NOT NULL,
    longitude DECIMAL(11, 8) NOT NULL
);

-- Index by country for filtering
CREATE INDEX idx_geo_cities_country ON geo_cities(country_id);

-- Trigram indexes for fast substring search (LIKE '%query%')
CREATE INDEX idx_geo_cities_name_trgm ON geo_cities USING gin(name gin_trgm_ops);
CREATE INDEX idx_geo_cities_name_ru_trgm ON geo_cities USING gin(name_ru gin_trgm_ops);

-- Regular indexes for exact match
CREATE INDEX idx_geo_cities_name ON geo_cities(name);
CREATE INDEX idx_geo_cities_name_ru ON geo_cities(name_ru);

COMMENT ON TABLE geo_cities IS 'Cities reference table (~153K records)';
COMMENT ON COLUMN geo_cities.id IS 'ID from dr5hn database';
COMMENT ON COLUMN geo_cities.name IS 'City name in English';
COMMENT ON COLUMN geo_cities.name_ru IS 'City name in Russian';
COMMENT ON COLUMN geo_cities.country_id IS 'Country ID';
COMMENT ON COLUMN geo_cities.state_name IS 'State/region/oblast name';
COMMENT ON COLUMN geo_cities.state_code IS 'State code';
COMMENT ON COLUMN geo_cities.latitude IS 'Latitude';
COMMENT ON COLUMN geo_cities.longitude IS 'Longitude';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS geo_cities;
DROP TABLE IF EXISTS geo_countries;
-- Don't drop pg_trgm as it may be used by other tables

-- +goose StatementEnd
