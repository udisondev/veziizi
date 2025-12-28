-- +goose Up
-- +goose StatementBegin

-- Add route_city_ids for filtering by route cities
ALTER TABLE freight_requests_lookup
    ADD COLUMN route_city_ids INTEGER[];

-- GIN index for fast overlap queries (&&)
CREATE INDEX idx_freight_requests_route_city_ids
    ON freight_requests_lookup USING GIN (route_city_ids);

-- Backfill existing records: extract city_id from route JSON
UPDATE freight_requests_lookup
SET route_city_ids = (
    SELECT array_agg((point->>'city_id')::integer)
    FROM jsonb_array_elements(route->'points') AS point
    WHERE point->>'city_id' IS NOT NULL
)
WHERE route IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_freight_requests_route_city_ids;

ALTER TABLE freight_requests_lookup
    DROP COLUMN IF EXISTS route_city_ids;

-- +goose StatementEnd
