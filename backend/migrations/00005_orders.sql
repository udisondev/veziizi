-- +goose Up
-- +goose StatementBegin

-- File storage
CREATE TABLE files (
    id UUID PRIMARY KEY,
    data BYTEA NOT NULL,
    mime_type VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Orders lookup (projection for list/search)
-- Только ID и колонки для фильтрации, полные данные из event store
CREATE TABLE orders_lookup (
    id UUID PRIMARY KEY,
    freight_request_id UUID NOT NULL,
    customer_org_id UUID NOT NULL,
    carrier_org_id UUID NOT NULL,
    status VARCHAR(30) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_orders_freight_request ON orders_lookup (freight_request_id);
CREATE INDEX idx_orders_customer_org ON orders_lookup (customer_org_id);
CREATE INDEX idx_orders_carrier_org ON orders_lookup (carrier_org_id);
CREATE INDEX idx_orders_status ON orders_lookup (status);
CREATE INDEX idx_orders_created ON orders_lookup (created_at DESC);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS orders_lookup;
DROP TABLE IF EXISTS files;

-- +goose StatementEnd
