-- +goose Up
-- +goose StatementBegin

-- Справочник стран мира
-- Данные из https://github.com/dr5hn/countries-states-cities-database
CREATE TABLE geo_countries (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    iso2 CHAR(2) NOT NULL UNIQUE,
    iso3 CHAR(3),
    phone_code VARCHAR(20),
    native_name TEXT,
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8)
);

CREATE INDEX idx_geo_countries_iso2 ON geo_countries(iso2);
CREATE INDEX idx_geo_countries_name ON geo_countries(name);

COMMENT ON TABLE geo_countries IS 'Справочник стран мира';
COMMENT ON COLUMN geo_countries.id IS 'ID из dr5hn database';
COMMENT ON COLUMN geo_countries.name IS 'Название страны на английском';
COMMENT ON COLUMN geo_countries.iso2 IS 'ISO 3166-1 alpha-2 код';
COMMENT ON COLUMN geo_countries.iso3 IS 'ISO 3166-1 alpha-3 код';
COMMENT ON COLUMN geo_countries.phone_code IS 'Телефонный код страны';
COMMENT ON COLUMN geo_countries.native_name IS 'Название на родном языке';
COMMENT ON COLUMN geo_countries.latitude IS 'Широта центра страны';
COMMENT ON COLUMN geo_countries.longitude IS 'Долгота центра страны';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS geo_countries;
-- +goose StatementEnd
