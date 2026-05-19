-- +goose Up
-- +goose StatementBegin

-- Denormalize first/last route points into separate arrays for role-aware
-- filtering: "origin = <city/country>" и "destination = <city/country>".
-- Существующий route_city_ids/route_country_ids остаётся как «любая точка
-- маршрута» — новые колонки не заменяют его.
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

-- Backfill из существующего route JSONB:
-- первая точка → origin, последняя → destination.
-- Поле в JSONB называется city_id/country_id (см. values.RoutePoint).
UPDATE freight_requests_lookup
SET
    origin_city_ids = CASE
        WHEN route IS NULL OR jsonb_array_length(route->'points') = 0 THEN NULL
        WHEN (route->'points'->0->>'city_id') IS NULL THEN ARRAY[]::INTEGER[]
        ELSE ARRAY[(route->'points'->0->>'city_id')::INTEGER]
    END,
    origin_country_ids = CASE
        WHEN route IS NULL OR jsonb_array_length(route->'points') = 0 THEN NULL
        WHEN (route->'points'->0->>'country_id') IS NULL THEN ARRAY[]::INTEGER[]
        ELSE ARRAY[(route->'points'->0->>'country_id')::INTEGER]
    END,
    destination_city_ids = CASE
        WHEN route IS NULL OR jsonb_array_length(route->'points') = 0 THEN NULL
        WHEN (route->'points'->-1->>'city_id') IS NULL THEN ARRAY[]::INTEGER[]
        ELSE ARRAY[(route->'points'->-1->>'city_id')::INTEGER]
    END,
    destination_country_ids = CASE
        WHEN route IS NULL OR jsonb_array_length(route->'points') = 0 THEN NULL
        WHEN (route->'points'->-1->>'country_id') IS NULL THEN ARRAY[]::INTEGER[]
        ELSE ARRAY[(route->'points'->-1->>'country_id')::INTEGER]
    END
WHERE route IS NOT NULL;

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
