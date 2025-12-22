-- +goose Up
-- +goose StatementBegin

-- Временные коды для привязки Telegram через бота
CREATE TABLE telegram_link_codes (
    code VARCHAR(6) PRIMARY KEY,
    member_id UUID NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Индекс для поиска по member_id (чтобы удалять старые коды)
CREATE INDEX idx_telegram_link_codes_member ON telegram_link_codes(member_id);

-- Индекс для очистки истёкших кодов
CREATE INDEX idx_telegram_link_codes_expires ON telegram_link_codes(expires_at);

COMMENT ON TABLE telegram_link_codes IS 'Временные коды для привязки Telegram аккаунта через бота';
COMMENT ON COLUMN telegram_link_codes.code IS '6-символьный код (например ABC123)';
COMMENT ON COLUMN telegram_link_codes.expires_at IS 'Время истечения кода (обычно 10 минут)';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS telegram_link_codes;

-- +goose StatementEnd
