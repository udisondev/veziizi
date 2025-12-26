-- +goose Up
-- Система подписок (трафаретов) на заявки v2
-- Opt-in модель: уведомления только по созданным подпискам

-- Удаляем старую таблицу (opt-out модель)
DROP TABLE IF EXISTS freight_request_subscriptions;

-- Основная таблица подписок (трафаретов)
CREATE TABLE freight_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    member_id UUID NOT NULL REFERENCES members_lookup(id) ON DELETE CASCADE,
    name TEXT NOT NULL,

    -- Числовые диапазоны (NULL = без ограничения)
    min_weight NUMERIC(10, 2),
    max_weight NUMERIC(10, 2),
    min_price BIGINT,
    max_price BIGINT,
    min_volume NUMERIC(10, 2),
    max_volume NUMERIC(10, 2),

    -- ENUM массивы (NULL = все подходят)
    cargo_types TEXT[],
    body_types TEXT[],
    payment_methods TEXT[],
    payment_terms TEXT[],
    vat_types TEXT[],

    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Точки маршрута для subsequence matching
CREATE TABLE freight_subscription_route_points (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id UUID NOT NULL REFERENCES freight_subscriptions(id) ON DELETE CASCADE,
    country_id INTEGER NOT NULL REFERENCES geo_countries(id),
    city_id INTEGER REFERENCES geo_cities(id),
    point_order INTEGER NOT NULL,
    UNIQUE(subscription_id, point_order)
);

-- Индексы для freight_subscriptions
CREATE INDEX idx_freight_subscriptions_member ON freight_subscriptions(member_id);
CREATE INDEX idx_freight_subscriptions_active ON freight_subscriptions(is_active) WHERE is_active = true;

-- GIN индексы для ENUM массивов
CREATE INDEX idx_freight_subscriptions_cargo_types ON freight_subscriptions USING GIN(cargo_types);
CREATE INDEX idx_freight_subscriptions_body_types ON freight_subscriptions USING GIN(body_types);
CREATE INDEX idx_freight_subscriptions_payment_methods ON freight_subscriptions USING GIN(payment_methods);
CREATE INDEX idx_freight_subscriptions_payment_terms ON freight_subscriptions USING GIN(payment_terms);
CREATE INDEX idx_freight_subscriptions_vat_types ON freight_subscriptions USING GIN(vat_types);

-- Индексы для route_points
CREATE INDEX idx_subscription_route_points_sub ON freight_subscription_route_points(subscription_id);
CREATE INDEX idx_subscription_route_points_country ON freight_subscription_route_points(country_id);
CREATE INDEX idx_subscription_route_points_city ON freight_subscription_route_points(city_id) WHERE city_id IS NOT NULL;

COMMENT ON TABLE freight_subscriptions IS 'Трафареты (подписки) пользователей на уведомления о заявках (opt-in модель)';
COMMENT ON COLUMN freight_subscriptions.name IS 'Название трафарета для UI';
COMMENT ON COLUMN freight_subscriptions.min_weight IS 'Мин. вес груза в тоннах (NULL = без ограничения)';
COMMENT ON COLUMN freight_subscriptions.max_weight IS 'Макс. вес груза в тоннах (NULL = без ограничения)';
COMMENT ON COLUMN freight_subscriptions.min_price IS 'Мин. цена в минорных единицах (NULL = без ограничения)';
COMMENT ON COLUMN freight_subscriptions.max_price IS 'Макс. цена в минорных единицах (NULL = без ограничения)';
COMMENT ON COLUMN freight_subscriptions.cargo_types IS 'Типы груза (NULL = все подходят)';
COMMENT ON COLUMN freight_subscriptions.body_types IS 'Типы кузова (NULL = все подходят)';
COMMENT ON COLUMN freight_subscriptions.payment_methods IS 'Способы оплаты (NULL = все подходят)';
COMMENT ON COLUMN freight_subscriptions.payment_terms IS 'Условия оплаты (NULL = все подходят)';
COMMENT ON COLUMN freight_subscriptions.vat_types IS 'Типы НДС (NULL = все подходят)';
COMMENT ON COLUMN freight_subscriptions.is_active IS 'Активна ли подписка';

COMMENT ON TABLE freight_subscription_route_points IS 'Точки маршрута для subsequence matching';
COMMENT ON COLUMN freight_subscription_route_points.country_id IS 'ID страны (обязательно)';
COMMENT ON COLUMN freight_subscription_route_points.city_id IS 'ID города (NULL = любой город в стране)';
COMMENT ON COLUMN freight_subscription_route_points.point_order IS 'Порядок точки в последовательности (1, 2, 3...)';

-- +goose Down
DROP TABLE IF EXISTS freight_subscription_route_points;
DROP TABLE IF EXISTS freight_subscriptions;

-- Восстанавливаем старую таблицу
CREATE TABLE freight_request_subscriptions (
    member_id UUID PRIMARY KEY REFERENCES members_lookup(id) ON DELETE CASCADE,
    origin_country_ids INTEGER[],
    destination_country_ids INTEGER[],
    cargo_types TEXT[],
    min_weight NUMERIC(10, 2),
    max_weight NUMERIC(10, 2),
    body_types TEXT[],
    unsubscribed BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_fr_subscriptions_unsubscribed ON freight_request_subscriptions(unsubscribed) WHERE unsubscribed = FALSE;
CREATE INDEX idx_fr_subscriptions_origin ON freight_request_subscriptions USING GIN(origin_country_ids);
CREATE INDEX idx_fr_subscriptions_destination ON freight_request_subscriptions USING GIN(destination_country_ids);
CREATE INDEX idx_fr_subscriptions_cargo_types ON freight_request_subscriptions USING GIN(cargo_types);
CREATE INDEX idx_fr_subscriptions_body_types ON freight_request_subscriptions USING GIN(body_types);
