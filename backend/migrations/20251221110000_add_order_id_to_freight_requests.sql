-- +goose Up
-- +goose StatementBegin
ALTER TABLE freight_requests_lookup ADD COLUMN order_id UUID;

-- Index for quick lookup by order_id
CREATE INDEX idx_freight_requests_lookup_order_id ON freight_requests_lookup(order_id) WHERE order_id IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_freight_requests_lookup_order_id;
ALTER TABLE freight_requests_lookup DROP COLUMN IF EXISTS order_id;
-- +goose StatementEnd
