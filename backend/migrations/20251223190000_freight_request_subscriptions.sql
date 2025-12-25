-- +goose Up
-- Таблица подписок на уведомления о заявках
-- По умолчанию все подписаны на всё (opt-out модель)
CREATE TABLE freight_request_subscriptions (
    member_id UUID PRIMARY KEY REFERENCES members_lookup(id) ON DELETE CASCADE,

    -- Географические фильтры (NULL = все страны)
    origin_country_ids INTEGER[],
    destination_country_ids INTEGER[],

    -- Фильтры по грузу
    cargo_types TEXT[],
    min_weight NUMERIC(10, 2),
    max_weight NUMERIC(10, 2),

    -- Фильтры по транспорту
    body_types TEXT[],

    -- Полная отписка от уведомлений о новых заявках
    unsubscribed BOOLEAN NOT NULL DEFAULT FALSE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Индексы для эффективного поиска подписчиков
CREATE INDEX idx_fr_subscriptions_unsubscribed ON freight_request_subscriptions(unsubscribed) WHERE unsubscribed = FALSE;
CREATE INDEX idx_fr_subscriptions_origin ON freight_request_subscriptions USING GIN(origin_country_ids);
CREATE INDEX idx_fr_subscriptions_destination ON freight_request_subscriptions USING GIN(destination_country_ids);
CREATE INDEX idx_fr_subscriptions_cargo_types ON freight_request_subscriptions USING GIN(cargo_types);
CREATE INDEX idx_fr_subscriptions_body_types ON freight_request_subscriptions USING GIN(body_types);

COMMENT ON TABLE freight_request_subscriptions IS 'Настройки подписок пользователей на уведомления о новых заявках';
COMMENT ON COLUMN freight_request_subscriptions.origin_country_ids IS 'NULL = все страны отправления';
COMMENT ON COLUMN freight_request_subscriptions.destination_country_ids IS 'NULL = все страны назначения';
COMMENT ON COLUMN freight_request_subscriptions.cargo_types IS 'NULL = все типы груза';
COMMENT ON COLUMN freight_request_subscriptions.body_types IS 'NULL = все типы кузова';
COMMENT ON COLUMN freight_request_subscriptions.unsubscribed IS 'TRUE = полная отписка от уведомлений о новых заявках';

-- +goose Down
DROP TABLE IF EXISTS freight_request_subscriptions;
