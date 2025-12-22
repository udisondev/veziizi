-- +goose Up
-- +goose StatementBegin

-- Добавляем русские названия для стран
ALTER TABLE geo_countries ADD COLUMN name_ru TEXT;
CREATE INDEX idx_geo_countries_name_ru ON geo_countries(name_ru);

COMMENT ON COLUMN geo_countries.name_ru IS 'Название страны на русском языке';

-- Добавляем русские названия для городов
ALTER TABLE geo_cities ADD COLUMN name_ru TEXT;

-- Trigram индекс для быстрого поиска по русскому названию
CREATE INDEX idx_geo_cities_name_ru_trgm ON geo_cities USING gin(name_ru gin_trgm_ops);

-- Обычный индекс для точного поиска
CREATE INDEX idx_geo_cities_name_ru ON geo_cities(name_ru);

COMMENT ON COLUMN geo_cities.name_ru IS 'Название города на русском языке';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_geo_cities_name_ru;
DROP INDEX IF EXISTS idx_geo_cities_name_ru_trgm;
ALTER TABLE geo_cities DROP COLUMN IF EXISTS name_ru;

DROP INDEX IF EXISTS idx_geo_countries_name_ru;
ALTER TABLE geo_countries DROP COLUMN IF EXISTS name_ru;

-- +goose StatementEnd
