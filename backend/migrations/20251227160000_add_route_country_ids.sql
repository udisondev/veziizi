-- +goose Up
-- +goose StatementBegin

-- Add route_country_ids for filtering by route countries (when city is not selected)
ALTER TABLE freight_requests_lookup
    ADD COLUMN route_country_ids INTEGER[];

-- GIN index for fast overlap queries (&&)
CREATE INDEX idx_freight_requests_route_country_ids
    ON freight_requests_lookup USING GIN (route_country_ids);

-- Backfill existing records: extract country_id from route JSON
UPDATE freight_requests_lookup
SET route_country_ids = (
    SELECT array_agg((point->>'country_id')::integer)
    FROM jsonb_array_elements(route->'points') AS point
    WHERE point->>'country_id' IS NOT NULL
)
WHERE route IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_freight_requests_route_country_ids;

ALTER TABLE freight_requests_lookup
    DROP COLUMN IF EXISTS route_country_ids;

-- +goose StatementEnd
