-- +goose Up
-- +goose StatementBegin

-- Настройки уведомлений member (1:1 к member)
CREATE TABLE notification_preferences (
    member_id UUID PRIMARY KEY,

    -- Telegram OAuth
    telegram_chat_id BIGINT,
    telegram_username VARCHAR(64),
    telegram_connected_at TIMESTAMPTZ,

    -- Включенные категории (JSONB для гибкости)
    -- Структура: {"offers":{"in_app":true,"telegram":false},"orders":{...},...}
    enabled_categories JSONB NOT NULL DEFAULT '{
        "offers": {"in_app": true, "telegram": false},
        "orders": {"in_app": true, "telegram": false},
        "reviews": {"in_app": true, "telegram": false},
        "organization": {"in_app": true, "telegram": false}
    }'::jsonb,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE notification_preferences IS 'Настройки уведомлений для каждого member';
COMMENT ON COLUMN notification_preferences.enabled_categories IS 'Включенные категории по каналам: offers, orders, reviews, organization';

-- In-app уведомления
CREATE TABLE inapp_notifications (
    id UUID PRIMARY KEY,
    member_id UUID NOT NULL,
    organization_id UUID NOT NULL,

    -- Тип и данные
    notification_type VARCHAR(50) NOT NULL,
    title VARCHAR(255) NOT NULL,
    body TEXT,
    link VARCHAR(255),

    -- Ссылка на сущность
    entity_type VARCHAR(50),  -- freight_request, order, organization
    entity_id UUID,

    -- Статус
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    read_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Индексы для быстрого доступа
CREATE INDEX idx_inapp_notifications_member_unread
    ON inapp_notifications(member_id, is_read, created_at DESC)
    WHERE is_read = FALSE;

CREATE INDEX idx_inapp_notifications_member_created
    ON inapp_notifications(member_id, created_at DESC);

COMMENT ON TABLE inapp_notifications IS 'In-app уведомления для notification center';

-- Лог доставки (для аудита и дебага)
CREATE TABLE notification_delivery_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    member_id UUID NOT NULL,
    notification_type VARCHAR(50) NOT NULL,
    channel VARCHAR(20) NOT NULL,  -- in_app, telegram
    status VARCHAR(20) NOT NULL,   -- sent, failed, skipped
    error_message TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_delivery_log_member_created
    ON notification_delivery_log(member_id, created_at DESC);

COMMENT ON TABLE notification_delivery_log IS 'Лог доставки уведомлений для аудита';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS notification_delivery_log;
DROP TABLE IF EXISTS inapp_notifications;
DROP TABLE IF EXISTS notification_preferences;

-- +goose StatementEnd
