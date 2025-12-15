-- +goose Up
-- +goose StatementBegin

-- Freight requests lookup (for list/search)
-- Only ID and filter columns, full data from event store
CREATE TABLE freight_requests_lookup (
    id UUID PRIMARY KEY,
    customer_org_id UUID NOT NULL,
    status VARCHAR(20) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_freight_requests_customer_org ON freight_requests_lookup (customer_org_id);
CREATE INDEX idx_freight_requests_status ON freight_requests_lookup (status);
CREATE INDEX idx_freight_requests_expires ON freight_requests_lookup (expires_at);
CREATE INDEX idx_freight_requests_created ON freight_requests_lookup (created_at DESC);

-- Offers lookup (for list/search)
-- Only ID and filter columns, full data from event store
CREATE TABLE offers_lookup (
    id UUID PRIMARY KEY,
    freight_request_id UUID NOT NULL,
    carrier_org_id UUID NOT NULL,
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_offers_freight_request ON offers_lookup (freight_request_id);
CREATE INDEX idx_offers_carrier_org ON offers_lookup (carrier_org_id);
CREATE INDEX idx_offers_status ON offers_lookup (status);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS offers_lookup;
DROP TABLE IF EXISTS freight_requests_lookup;

-- +goose StatementEnd
