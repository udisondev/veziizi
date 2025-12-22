-- +goose Up
ALTER TABLE freight_requests_lookup ADD COLUMN route JSONB;

-- Backfill will be done by re-processing events or manual script

-- +goose Down
ALTER TABLE freight_requests_lookup DROP COLUMN route;
