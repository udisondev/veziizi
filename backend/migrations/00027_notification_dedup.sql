-- +goose Up
-- +goose StatementBegin

-- Дедупликация исходящих уведомлений. При at-least-once доставке Telegram/Email
-- сообщение может прийти sender-воркеру повторно (например, после рестарта
-- между внешним вызовом и Ack). UUID берётся из message.UUID — он стабилен
-- через retry'и одного сообщения.
CREATE TABLE notification_dedup (
    message_uuid UUID PRIMARY KEY,
    channel      TEXT NOT NULL,
    sent_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- TTL-чистка не нужна на старте: ~1KB на запись, миллионы записей в год терпимы.
-- При росте — добавить partial-индекс по channel или партиционирование по дате.
CREATE INDEX idx_notification_dedup_sent_at ON notification_dedup(sent_at);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS notification_dedup;
-- +goose StatementEnd
