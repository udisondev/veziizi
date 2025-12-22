-- +goose Up
-- +goose StatementBegin

-- Включаем расширение pg_trgm для быстрого поиска по подстроке
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Справочник городов мира
-- Данные из https://github.com/dr5hn/countries-states-cities-database
CREATE TABLE geo_cities (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    country_id INTEGER NOT NULL REFERENCES geo_countries(id) ON DELETE CASCADE,
    state_name TEXT,
    state_code VARCHAR(10),
    latitude DECIMAL(10, 8) NOT NULL,
    longitude DECIMAL(11, 8) NOT NULL
);

-- Индекс по стране для фильтрации
CREATE INDEX idx_geo_cities_country ON geo_cities(country_id);

-- Trigram индекс для быстрого поиска по названию (LIKE '%query%')
CREATE INDEX idx_geo_cities_name_trgm ON geo_cities USING gin(name gin_trgm_ops);

-- Обычный индекс для точного поиска
CREATE INDEX idx_geo_cities_name ON geo_cities(name);

COMMENT ON TABLE geo_cities IS 'Справочник городов мира (~153K записей)';
COMMENT ON COLUMN geo_cities.id IS 'ID из dr5hn database';
COMMENT ON COLUMN geo_cities.name IS 'Название города';
COMMENT ON COLUMN geo_cities.country_id IS 'ID страны';
COMMENT ON COLUMN geo_cities.state_name IS 'Название региона/области/штата';
COMMENT ON COLUMN geo_cities.state_code IS 'Код региона';
COMMENT ON COLUMN geo_cities.latitude IS 'Широта';
COMMENT ON COLUMN geo_cities.longitude IS 'Долгота';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS geo_cities;
-- Не удаляем pg_trgm, так как он может использоваться другими таблицами
-- +goose StatementEnd
