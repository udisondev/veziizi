-- +goose Up
-- +goose StatementBegin

-- Add display columns for freight requests list
-- These are denormalized for list performance, full data from event store
ALTER TABLE freight_requests_lookup
    ADD COLUMN origin_address TEXT,
    ADD COLUMN destination_address TEXT,
    ADD COLUMN cargo_type VARCHAR(20),
    ADD COLUMN cargo_weight NUMERIC(12, 2),
    ADD COLUMN price_amount BIGINT,
    ADD COLUMN price_currency VARCHAR(3),
    ADD COLUMN body_types TEXT[];

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE freight_requests_lookup
    DROP COLUMN IF EXISTS origin_address,
    DROP COLUMN IF EXISTS destination_address,
    DROP COLUMN IF EXISTS cargo_type,
    DROP COLUMN IF EXISTS cargo_weight,
    DROP COLUMN IF EXISTS price_amount,
    DROP COLUMN IF EXISTS price_currency,
    DROP COLUMN IF EXISTS body_types;

-- +goose StatementEnd
