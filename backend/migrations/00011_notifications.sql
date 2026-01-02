-- +goose Up
-- +goose StatementBegin

-- Member notification preferences (1:1 to member)
-- Consolidated from: 20251222170000
CREATE TABLE notification_preferences (
    member_id UUID PRIMARY KEY,

    -- Telegram OAuth
    telegram_chat_id BIGINT,
    telegram_username VARCHAR(64),
    telegram_connected_at TIMESTAMPTZ,

    -- Enabled categories (JSONB for flexibility)
    -- Structure: {"offers":{"in_app":true,"telegram":false},"orders":{...},...}
    enabled_categories JSONB NOT NULL DEFAULT '{
        "offers": {"in_app": true, "telegram": false},
        "orders": {"in_app": true, "telegram": false},
        "reviews": {"in_app": true, "telegram": false},
        "organization": {"in_app": true, "telegram": false}
    }'::jsonb,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE notification_preferences IS 'Notification settings for each member';
COMMENT ON COLUMN notification_preferences.enabled_categories IS 'Enabled categories by channel: offers, orders, reviews, organization';

-- In-app notifications
CREATE TABLE inapp_notifications (
    id UUID PRIMARY KEY,
    member_id UUID NOT NULL,
    organization_id UUID NOT NULL,

    -- Type and data
    notification_type VARCHAR(50) NOT NULL,
    title VARCHAR(255) NOT NULL,
    body TEXT,
    link VARCHAR(255),

    -- Entity reference
    entity_type VARCHAR(50),  -- freight_request, order, organization
    entity_id UUID,

    -- Status
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    read_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for fast access
CREATE INDEX idx_inapp_notifications_member_unread
    ON inapp_notifications(member_id, is_read, created_at DESC)
    WHERE is_read = FALSE;

CREATE INDEX idx_inapp_notifications_member_created
    ON inapp_notifications(member_id, created_at DESC);

COMMENT ON TABLE inapp_notifications IS 'In-app notifications for notification center';

-- Delivery log (for audit and debug)
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

COMMENT ON TABLE notification_delivery_log IS 'Notification delivery log for audit';

-- Temporary codes for linking Telegram via bot
-- Consolidated from: 20251222185322
CREATE TABLE telegram_link_codes (
    code VARCHAR(6) PRIMARY KEY,
    member_id UUID NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for finding by member_id (to delete old codes)
CREATE INDEX idx_telegram_link_codes_member ON telegram_link_codes(member_id);

-- Index for cleaning expired codes
CREATE INDEX idx_telegram_link_codes_expires ON telegram_link_codes(expires_at);

COMMENT ON TABLE telegram_link_codes IS 'Temporary codes for linking Telegram account via bot';
COMMENT ON COLUMN telegram_link_codes.code IS '6-character code (e.g. ABC123)';
COMMENT ON COLUMN telegram_link_codes.expires_at IS 'Code expiration time (usually 10 minutes)';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS telegram_link_codes;
DROP TABLE IF EXISTS notification_delivery_log;
DROP TABLE IF EXISTS inapp_notifications;
DROP TABLE IF EXISTS notification_preferences;

-- +goose StatementEnd
