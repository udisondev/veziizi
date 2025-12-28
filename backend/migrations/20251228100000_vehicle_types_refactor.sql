-- +goose Up
-- Рефакторинг: замена body_types на vehicle_type + vehicle_subtype

-- Удаляем данные (НЕ organizations/users!)
DELETE FROM events WHERE aggregate_type IN ('freight_request', 'order', 'review', 'notification');

-- Очищаем lookup таблицы (они точно существуют)
TRUNCATE TABLE reviews_lookup CASCADE;
TRUNCATE TABLE orders_lookup CASCADE;
TRUNCATE TABLE offers_lookup CASCADE;
TRUNCATE TABLE freight_requests_lookup CASCADE;
TRUNCATE TABLE freight_subscriptions CASCADE;
TRUNCATE TABLE freight_subscription_route_points CASCADE;

-- Изменяем freight_requests_lookup
ALTER TABLE freight_requests_lookup
    DROP COLUMN IF EXISTS body_types,
    ADD COLUMN IF NOT EXISTS vehicle_type VARCHAR(30),
    ADD COLUMN IF NOT EXISTS vehicle_subtype VARCHAR(30);

DROP INDEX IF EXISTS idx_freight_requests_body_types;
CREATE INDEX IF NOT EXISTS idx_freight_requests_vehicle_type ON freight_requests_lookup(vehicle_type);
CREATE INDEX IF NOT EXISTS idx_freight_requests_vehicle_subtype ON freight_requests_lookup(vehicle_subtype);

-- Изменяем freight_subscriptions
ALTER TABLE freight_subscriptions
    DROP COLUMN IF EXISTS body_types,
    DROP COLUMN IF EXISTS cargo_types,
    ADD COLUMN IF NOT EXISTS vehicle_types TEXT[],
    ADD COLUMN IF NOT EXISTS vehicle_subtypes TEXT[];

DROP INDEX IF EXISTS idx_freight_subscriptions_body_types;
DROP INDEX IF EXISTS idx_freight_subscriptions_cargo_types;
CREATE INDEX IF NOT EXISTS idx_freight_subscriptions_vehicle_types ON freight_subscriptions USING GIN(vehicle_types);
CREATE INDEX IF NOT EXISTS idx_freight_subscriptions_vehicle_subtypes ON freight_subscriptions USING GIN(vehicle_subtypes);

-- +goose Down
-- Откат: возвращаем body_types

ALTER TABLE freight_requests_lookup
    DROP COLUMN IF EXISTS vehicle_type,
    DROP COLUMN IF EXISTS vehicle_subtype,
    ADD COLUMN IF NOT EXISTS body_types TEXT[];

DROP INDEX IF EXISTS idx_freight_requests_vehicle_type;
DROP INDEX IF EXISTS idx_freight_requests_vehicle_subtype;
CREATE INDEX IF NOT EXISTS idx_freight_requests_body_types ON freight_requests_lookup USING GIN(body_types);

ALTER TABLE freight_subscriptions
    DROP COLUMN IF EXISTS vehicle_types,
    DROP COLUMN IF EXISTS vehicle_subtypes,
    ADD COLUMN IF NOT EXISTS body_types TEXT[],
    ADD COLUMN IF NOT EXISTS cargo_types TEXT[];

DROP INDEX IF EXISTS idx_freight_subscriptions_vehicle_types;
DROP INDEX IF EXISTS idx_freight_subscriptions_vehicle_subtypes;
CREATE INDEX IF NOT EXISTS idx_freight_subscriptions_body_types ON freight_subscriptions USING GIN(body_types);
CREATE INDEX IF NOT EXISTS idx_freight_subscriptions_cargo_types ON freight_subscriptions USING GIN(cargo_types);
