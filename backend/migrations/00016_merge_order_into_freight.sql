-- +goose Up
-- +goose StatementBegin

-- Add carrier info to freight_requests_lookup (populated when offer is confirmed)
ALTER TABLE freight_requests_lookup
ADD COLUMN carrier_org_id UUID,
ADD COLUMN carrier_member_id UUID,
ADD COLUMN confirmed_at TIMESTAMPTZ;

-- Add completion tracking
ALTER TABLE freight_requests_lookup
ADD COLUMN customer_completed BOOLEAN DEFAULT FALSE,
ADD COLUMN carrier_completed BOOLEAN DEFAULT FALSE,
ADD COLUMN completed_at TIMESTAMPTZ;

-- Add cancellation after confirmed tracking
ALTER TABLE freight_requests_lookup
ADD COLUMN cancelled_after_confirmed_at TIMESTAMPTZ;

-- Create indexes for new columns
CREATE INDEX idx_freight_requests_carrier_org ON freight_requests_lookup(carrier_org_id);
CREATE INDEX idx_freight_requests_carrier_member ON freight_requests_lookup(carrier_member_id);
CREATE INDEX idx_freight_requests_completed ON freight_requests_lookup(completed_at);
CREATE INDEX idx_freight_requests_confirmed ON freight_requests_lookup(confirmed_at);

-- Drop order_id column (no longer needed as Order aggregate is removed)
ALTER TABLE freight_requests_lookup DROP COLUMN IF EXISTS order_id;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Remove new columns
ALTER TABLE freight_requests_lookup DROP COLUMN IF EXISTS carrier_org_id;
ALTER TABLE freight_requests_lookup DROP COLUMN IF EXISTS carrier_member_id;
ALTER TABLE freight_requests_lookup DROP COLUMN IF EXISTS confirmed_at;
ALTER TABLE freight_requests_lookup DROP COLUMN IF EXISTS customer_completed;
ALTER TABLE freight_requests_lookup DROP COLUMN IF EXISTS carrier_completed;
ALTER TABLE freight_requests_lookup DROP COLUMN IF EXISTS completed_at;
ALTER TABLE freight_requests_lookup DROP COLUMN IF EXISTS cancelled_after_confirmed_at;

-- Restore order_id column
ALTER TABLE freight_requests_lookup ADD COLUMN order_id UUID;

-- Drop indexes
DROP INDEX IF EXISTS idx_freight_requests_carrier_org;
DROP INDEX IF EXISTS idx_freight_requests_carrier_member;
DROP INDEX IF EXISTS idx_freight_requests_completed;
DROP INDEX IF EXISTS idx_freight_requests_confirmed;

-- +goose StatementEnd
