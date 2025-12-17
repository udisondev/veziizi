-- +goose Up
-- +goose StatementBegin

-- Add organization data columns for filtering freight requests
-- These are denormalized for list/filter performance
ALTER TABLE freight_requests_lookup
    ADD COLUMN customer_org_name VARCHAR(255),
    ADD COLUMN customer_org_inn VARCHAR(20),
    ADD COLUMN customer_org_country VARCHAR(10),
    ADD COLUMN customer_member_id UUID;

CREATE INDEX idx_freight_requests_org_name ON freight_requests_lookup (customer_org_name);
CREATE INDEX idx_freight_requests_org_inn ON freight_requests_lookup (customer_org_inn);
CREATE INDEX idx_freight_requests_org_country ON freight_requests_lookup (customer_org_country);
CREATE INDEX idx_freight_requests_member ON freight_requests_lookup (customer_member_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_freight_requests_member;
DROP INDEX IF EXISTS idx_freight_requests_org_country;
DROP INDEX IF EXISTS idx_freight_requests_org_inn;
DROP INDEX IF EXISTS idx_freight_requests_org_name;

ALTER TABLE freight_requests_lookup
    DROP COLUMN IF EXISTS customer_member_id,
    DROP COLUMN IF EXISTS customer_org_country,
    DROP COLUMN IF EXISTS customer_org_inn,
    DROP COLUMN IF EXISTS customer_org_name;

-- +goose StatementEnd
