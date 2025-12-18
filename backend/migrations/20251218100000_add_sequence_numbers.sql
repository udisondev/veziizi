-- +goose Up
-- +goose StatementBegin

-- Create sequences for order and freight request numbers
CREATE SEQUENCE order_number_seq START 1;
CREATE SEQUENCE request_number_seq START 1;

-- Add columns (nullable first for backfill)
ALTER TABLE orders_lookup ADD COLUMN order_number BIGINT;
ALTER TABLE freight_requests_lookup ADD COLUMN request_number BIGINT;

-- Backfill existing orders with sequential numbers by creation date
WITH numbered_orders AS (
    SELECT id, ROW_NUMBER() OVER (ORDER BY created_at ASC) as num
    FROM orders_lookup
)
UPDATE orders_lookup o
SET order_number = n.num
FROM numbered_orders n
WHERE o.id = n.id;

-- Backfill existing freight requests with sequential numbers by creation date
WITH numbered_requests AS (
    SELECT id, ROW_NUMBER() OVER (ORDER BY created_at ASC) as num
    FROM freight_requests_lookup
)
UPDATE freight_requests_lookup fr
SET request_number = n.num
FROM numbered_requests n
WHERE fr.id = n.id;

-- Set sequences to max + 1 (or 1 if no records exist)
SELECT setval('order_number_seq', COALESCE((SELECT MAX(order_number) FROM orders_lookup), 0) + 1, false);
SELECT setval('request_number_seq', COALESCE((SELECT MAX(request_number) FROM freight_requests_lookup), 0) + 1, false);

-- Make columns NOT NULL and add unique indexes
ALTER TABLE orders_lookup ALTER COLUMN order_number SET NOT NULL;
ALTER TABLE freight_requests_lookup ALTER COLUMN request_number SET NOT NULL;

CREATE UNIQUE INDEX idx_orders_order_number ON orders_lookup (order_number);
CREATE UNIQUE INDEX idx_freight_requests_request_number ON freight_requests_lookup (request_number);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_orders_order_number;
DROP INDEX IF EXISTS idx_freight_requests_request_number;

ALTER TABLE orders_lookup DROP COLUMN IF EXISTS order_number;
ALTER TABLE freight_requests_lookup DROP COLUMN IF EXISTS request_number;

DROP SEQUENCE IF EXISTS order_number_seq;
DROP SEQUENCE IF EXISTS request_number_seq;

-- +goose StatementEnd
