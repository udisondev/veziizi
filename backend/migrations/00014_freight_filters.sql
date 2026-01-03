-- +goose Up
-- +goose StatementBegin

-- Add columns for extended filtering to freight_requests_lookup
ALTER TABLE freight_requests_lookup
    ADD COLUMN IF NOT EXISTS cargo_volume NUMERIC(12, 2),
    ADD COLUMN IF NOT EXISTS payment_method VARCHAR(30),
    ADD COLUMN IF NOT EXISTS payment_terms VARCHAR(30),
    ADD COLUMN IF NOT EXISTS vat_type VARCHAR(30);

-- Indexes for new filter columns
CREATE INDEX IF NOT EXISTS idx_freight_requests_cargo_volume ON freight_requests_lookup(cargo_volume);
CREATE INDEX IF NOT EXISTS idx_freight_requests_payment_method ON freight_requests_lookup(payment_method);
CREATE INDEX IF NOT EXISTS idx_freight_requests_payment_terms ON freight_requests_lookup(payment_terms);
CREATE INDEX IF NOT EXISTS idx_freight_requests_vat_type ON freight_requests_lookup(vat_type);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_freight_requests_vat_type;
DROP INDEX IF EXISTS idx_freight_requests_payment_terms;
DROP INDEX IF EXISTS idx_freight_requests_payment_method;
DROP INDEX IF EXISTS idx_freight_requests_cargo_volume;

ALTER TABLE freight_requests_lookup
    DROP COLUMN IF EXISTS vat_type,
    DROP COLUMN IF EXISTS payment_terms,
    DROP COLUMN IF EXISTS payment_method,
    DROP COLUMN IF EXISTS cargo_volume;

-- +goose StatementEnd
