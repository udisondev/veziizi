-- +goose Up
ALTER TABLE orders_lookup ADD COLUMN customer_member_id UUID;
ALTER TABLE orders_lookup ADD COLUMN carrier_member_id UUID;
CREATE INDEX idx_orders_customer_member ON orders_lookup(customer_member_id);
CREATE INDEX idx_orders_carrier_member ON orders_lookup(carrier_member_id);

-- +goose Down
DROP INDEX IF EXISTS idx_orders_carrier_member;
DROP INDEX IF EXISTS idx_orders_customer_member;
ALTER TABLE orders_lookup DROP COLUMN IF EXISTS carrier_member_id;
ALTER TABLE orders_lookup DROP COLUMN IF EXISTS customer_member_id;
