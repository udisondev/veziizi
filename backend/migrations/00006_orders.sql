-- +goose Up
-- +goose StatementBegin

-- Sequence for order numbers
CREATE SEQUENCE order_number_seq START 1;

-- File storage
CREATE TABLE files (
    id UUID PRIMARY KEY,
    data BYTEA NOT NULL,
    mime_type VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Orders lookup (projection for list/search)
-- Consolidated from: 00005, 20251218
CREATE TABLE orders_lookup (
    id UUID PRIMARY KEY,
    freight_request_id UUID NOT NULL,
    customer_org_id UUID NOT NULL,
    carrier_org_id UUID NOT NULL,
    customer_member_id UUID,
    carrier_member_id UUID,
    status VARCHAR(30) NOT NULL,
    order_number BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_orders_freight_request ON orders_lookup (freight_request_id);
CREATE INDEX idx_orders_customer_org ON orders_lookup (customer_org_id);
CREATE INDEX idx_orders_carrier_org ON orders_lookup (carrier_org_id);
CREATE INDEX idx_orders_status ON orders_lookup (status);
CREATE INDEX idx_orders_created ON orders_lookup (created_at DESC);
CREATE UNIQUE INDEX idx_orders_order_number ON orders_lookup (order_number);
CREATE INDEX idx_orders_customer_member ON orders_lookup(customer_member_id);
CREATE INDEX idx_orders_carrier_member ON orders_lookup(carrier_member_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS orders_lookup;
DROP TABLE IF EXISTS files;
DROP SEQUENCE IF EXISTS order_number_seq;

-- +goose StatementEnd
