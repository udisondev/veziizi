-- +goose Up
-- +goose StatementBegin

-- Denormalize first/last route points into separate arrays for role-aware
-- filtering: "origin = <city/country>" и "destination = <city/country>".
-- Существующий route_city_ids/route_country_ids остаётся как «любая точка
-- маршрута» — новые колонки не заменяют его.
--
-- Семантика «пусто» — NULL (а не пустой массив). Совпадает с тем, как
-- extractRouteCityIDs/CountryIDs в Go-handler возвращает nil → pgx пишет
-- NULL. Это инвариант: `<column> IS NULL` означает «у точки нет id такого
-- типа», и не путается с пустым массивом из ARRAY[]::INTEGER[].
ALTER TABLE freight_requests_lookup
    ADD COLUMN IF NOT EXISTS origin_city_ids         INTEGER[],
    ADD COLUMN IF NOT EXISTS origin_country_ids      INTEGER[],
    ADD COLUMN IF NOT EXISTS destination_city_ids    INTEGER[],
    ADD COLUMN IF NOT EXISTS destination_country_ids INTEGER[];

CREATE INDEX IF NOT EXISTS idx_freight_requests_origin_city_ids
    ON freight_requests_lookup USING GIN (origin_city_ids);
CREATE INDEX IF NOT EXISTS idx_freight_requests_origin_country_ids
    ON freight_requests_lookup USING GIN (origin_country_ids);
CREATE INDEX IF NOT EXISTS idx_freight_requests_destination_city_ids
    ON freight_requests_lookup USING GIN (destination_city_ids);
CREATE INDEX IF NOT EXISTS idx_freight_requests_destination_country_ids
    ON freight_requests_lookup USING GIN (destination_country_ids);

-- +goose StatementEnd

-- Backfill из существующего route JSONB по батчам.
-- Зачем батчи: один большой UPDATE по freight_requests_lookup держит
-- EXCLUSIVE-лок таблицы и переписывает все строки — на горячем эндпоинте
-- (GET /freight-requests) это видимая просадка. Батчи по 1000 строк дают
-- автовакууму прорваться между ними, а другим транзакциям — попасть в
-- окно между батчами.
--
-- Guard'ы:
--   1. route IS NOT NULL — иначе backfill не нужен;
--   2. route ? 'points'  — на случай битых данных без ключа;
--   3. jsonb_typeof(route->'points') = 'array' — защита от подмены типа;
--   4. id > last_id      — keyset-пагинация по PK; продвигаем через MAX(id)
--      батча независимо от того, обновились ли колонки в NULL (битый
--      маршрут без city/country: backfill оставит NULL, но мы не должны
--      зациклиться на этой же строке).
-- +goose StatementBegin
DO $$
DECLARE
    last_id UUID := '00000000-0000-0000-0000-000000000000';
    max_id  UUID;
BEGIN
    LOOP
        WITH batch AS (
            SELECT id
            FROM freight_requests_lookup
            WHERE id > last_id
              AND route IS NOT NULL
              AND route ? 'points'
              AND jsonb_typeof(route->'points') = 'array'
              AND jsonb_array_length(route->'points') > 0
            ORDER BY id
            LIMIT 1000
        ),
        updated AS (
            UPDATE freight_requests_lookup l
            SET
                origin_city_ids = CASE
                    WHEN (l.route->'points'->0->>'city_id') IS NULL THEN NULL
                    ELSE ARRAY[(l.route->'points'->0->>'city_id')::INTEGER]
                END,
                origin_country_ids = CASE
                    WHEN (l.route->'points'->0->>'country_id') IS NULL THEN NULL
                    ELSE ARRAY[(l.route->'points'->0->>'country_id')::INTEGER]
                END,
                destination_city_ids = CASE
                    WHEN (l.route->'points'->-1->>'city_id') IS NULL THEN NULL
                    ELSE ARRAY[(l.route->'points'->-1->>'city_id')::INTEGER]
                END,
                destination_country_ids = CASE
                    WHEN (l.route->'points'->-1->>'country_id') IS NULL THEN NULL
                    ELSE ARRAY[(l.route->'points'->-1->>'country_id')::INTEGER]
                END
            FROM batch
            WHERE l.id = batch.id
            RETURNING l.id
        )
        SELECT id INTO max_id FROM updated ORDER BY id DESC LIMIT 1;

        EXIT WHEN max_id IS NULL;
        last_id := max_id;
    END LOOP;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_freight_requests_destination_country_ids;
DROP INDEX IF EXISTS idx_freight_requests_destination_city_ids;
DROP INDEX IF EXISTS idx_freight_requests_origin_country_ids;
DROP INDEX IF EXISTS idx_freight_requests_origin_city_ids;

ALTER TABLE freight_requests_lookup
    DROP COLUMN IF EXISTS destination_country_ids,
    DROP COLUMN IF EXISTS destination_city_ids,
    DROP COLUMN IF EXISTS origin_country_ids,
    DROP COLUMN IF EXISTS origin_city_ids;

-- +goose StatementEnd
