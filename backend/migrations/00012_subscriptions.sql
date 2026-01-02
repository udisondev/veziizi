-- +goose Up
-- +goose StatementBegin

-- Freight subscriptions (templates) v2
-- Opt-in model: notifications only for created subscriptions
-- Consolidated from: 20251226, 20251228
CREATE TABLE freight_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    member_id UUID NOT NULL REFERENCES members_lookup(id) ON DELETE CASCADE,
    name TEXT NOT NULL,

    -- Numeric ranges (NULL = no limit)
    min_weight NUMERIC(10, 2),
    max_weight NUMERIC(10, 2),
    min_price BIGINT,
    max_price BIGINT,
    min_volume NUMERIC(10, 2),
    max_volume NUMERIC(10, 2),

    -- ENUM arrays (NULL = all match)
    vehicle_types TEXT[],
    vehicle_subtypes TEXT[],
    payment_methods TEXT[],
    payment_terms TEXT[],
    vat_types TEXT[],

    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Route points for subsequence matching
CREATE TABLE freight_subscription_route_points (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id UUID NOT NULL REFERENCES freight_subscriptions(id) ON DELETE CASCADE,
    country_id INTEGER NOT NULL REFERENCES geo_countries(id),
    city_id INTEGER REFERENCES geo_cities(id),
    point_order INTEGER NOT NULL,
    UNIQUE(subscription_id, point_order)
);

-- Indexes for freight_subscriptions
CREATE INDEX idx_freight_subscriptions_member ON freight_subscriptions(member_id);
CREATE INDEX idx_freight_subscriptions_active ON freight_subscriptions(is_active) WHERE is_active = true;

-- GIN indexes for ENUM arrays
CREATE INDEX idx_freight_subscriptions_vehicle_types ON freight_subscriptions USING GIN(vehicle_types);
CREATE INDEX idx_freight_subscriptions_vehicle_subtypes ON freight_subscriptions USING GIN(vehicle_subtypes);
CREATE INDEX idx_freight_subscriptions_payment_methods ON freight_subscriptions USING GIN(payment_methods);
CREATE INDEX idx_freight_subscriptions_payment_terms ON freight_subscriptions USING GIN(payment_terms);
CREATE INDEX idx_freight_subscriptions_vat_types ON freight_subscriptions USING GIN(vat_types);

-- Indexes for route_points
CREATE INDEX idx_subscription_route_points_sub ON freight_subscription_route_points(subscription_id);
CREATE INDEX idx_subscription_route_points_country ON freight_subscription_route_points(country_id);
CREATE INDEX idx_subscription_route_points_city ON freight_subscription_route_points(city_id) WHERE city_id IS NOT NULL;

COMMENT ON TABLE freight_subscriptions IS 'Subscription templates for freight request notifications (opt-in model)';
COMMENT ON COLUMN freight_subscriptions.name IS 'Template name for UI';
COMMENT ON COLUMN freight_subscriptions.min_weight IS 'Min cargo weight in tons (NULL = no limit)';
COMMENT ON COLUMN freight_subscriptions.max_weight IS 'Max cargo weight in tons (NULL = no limit)';
COMMENT ON COLUMN freight_subscriptions.min_price IS 'Min price in minor units (NULL = no limit)';
COMMENT ON COLUMN freight_subscriptions.max_price IS 'Max price in minor units (NULL = no limit)';
COMMENT ON COLUMN freight_subscriptions.vehicle_types IS 'Vehicle types (NULL = all match)';
COMMENT ON COLUMN freight_subscriptions.vehicle_subtypes IS 'Vehicle subtypes (NULL = all match)';
COMMENT ON COLUMN freight_subscriptions.payment_methods IS 'Payment methods (NULL = all match)';
COMMENT ON COLUMN freight_subscriptions.payment_terms IS 'Payment terms (NULL = all match)';
COMMENT ON COLUMN freight_subscriptions.vat_types IS 'VAT types (NULL = all match)';
COMMENT ON COLUMN freight_subscriptions.is_active IS 'Whether subscription is active';

COMMENT ON TABLE freight_subscription_route_points IS 'Route points for subsequence matching';
COMMENT ON COLUMN freight_subscription_route_points.country_id IS 'Country ID (required)';
COMMENT ON COLUMN freight_subscription_route_points.city_id IS 'City ID (NULL = any city in country)';
COMMENT ON COLUMN freight_subscription_route_points.point_order IS 'Point order in sequence (1, 2, 3...)';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS freight_subscription_route_points;
DROP TABLE IF EXISTS freight_subscriptions;

-- +goose StatementEnd
