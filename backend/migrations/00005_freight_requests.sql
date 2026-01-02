-- +goose Up
-- +goose StatementBegin

-- Sequences for request and order numbers
CREATE SEQUENCE request_number_seq START 1;

-- Freight requests lookup (for list/search)
-- Consolidated from: 00004, 20251215, 20251217, 20251218, 20251221110000, 20251222160000, 20251227, 20251228
CREATE TABLE freight_requests_lookup (
    id UUID PRIMARY KEY,
    customer_org_id UUID NOT NULL,
    status VARCHAR(20) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    -- Display data (denormalized for list performance)
    origin_address TEXT,
    destination_address TEXT,
    cargo_weight NUMERIC(12, 2),
    price_amount BIGINT,
    price_currency VARCHAR(3),
    vehicle_type VARCHAR(30),
    vehicle_subtype VARCHAR(30),
    -- Organization data (for filtering)
    customer_org_name VARCHAR(255),
    customer_org_inn VARCHAR(20),
    customer_org_country VARCHAR(10),
    customer_member_id UUID,
    -- Sequence number
    request_number BIGINT NOT NULL,
    -- Order link
    order_id UUID,
    -- Route (JSON with points)
    route JSONB,
    route_city_ids INTEGER[],
    route_country_ids INTEGER[]
);

CREATE INDEX idx_freight_requests_customer_org ON freight_requests_lookup (customer_org_id);
CREATE INDEX idx_freight_requests_status ON freight_requests_lookup (status);
CREATE INDEX idx_freight_requests_expires ON freight_requests_lookup (expires_at);
CREATE INDEX idx_freight_requests_created ON freight_requests_lookup (created_at DESC);
CREATE INDEX idx_freight_requests_org_name ON freight_requests_lookup (customer_org_name);
CREATE INDEX idx_freight_requests_org_inn ON freight_requests_lookup (customer_org_inn);
CREATE INDEX idx_freight_requests_org_country ON freight_requests_lookup (customer_org_country);
CREATE INDEX idx_freight_requests_member ON freight_requests_lookup (customer_member_id);
CREATE UNIQUE INDEX idx_freight_requests_request_number ON freight_requests_lookup (request_number);
CREATE INDEX idx_freight_requests_lookup_order_id ON freight_requests_lookup(order_id) WHERE order_id IS NOT NULL;
CREATE INDEX idx_freight_requests_vehicle_type ON freight_requests_lookup(vehicle_type);
CREATE INDEX idx_freight_requests_vehicle_subtype ON freight_requests_lookup(vehicle_subtype);
CREATE INDEX idx_freight_requests_route_city_ids ON freight_requests_lookup USING GIN (route_city_ids);
CREATE INDEX idx_freight_requests_route_country_ids ON freight_requests_lookup USING GIN (route_country_ids);

-- Offers lookup (for list/search)
-- Consolidated from: 00004, 20251223
CREATE TABLE offers_lookup (
    id UUID PRIMARY KEY,
    freight_request_id UUID NOT NULL,
    carrier_org_id UUID NOT NULL,
    carrier_member_id UUID,
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_offers_freight_request ON offers_lookup (freight_request_id);
CREATE INDEX idx_offers_carrier_org ON offers_lookup (carrier_org_id);
CREATE INDEX idx_offers_status ON offers_lookup (status);
CREATE INDEX idx_offers_carrier_member ON offers_lookup(carrier_member_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS offers_lookup;
DROP TABLE IF EXISTS freight_requests_lookup;
DROP SEQUENCE IF EXISTS request_number_seq;

-- +goose StatementEnd
