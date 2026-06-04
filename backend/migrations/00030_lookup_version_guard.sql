-- +goose Up
-- +goose StatementBegin

-- Version-guard для проекций при at-least-once доставке через Redis Streams.
--
-- Проекционные хендлеры переходят на паттерн rebuild-from-aggregate: перечитывают
-- агрегат из event store и пишут ПОЛНОЕ состояние UPSERT'ом с условием
-- `WHERE <table>.version <= excluded.version`. version — версия агрегата на
-- момент Load. Это закрывает обе проблемы конкурентной обработки:
--   - out-of-order: устаревший rebuild (меньшая версия) не перетирает свежий;
--   - повторная доставка: повторный rebuild той же версии идемпотентен.
--
-- DEFAULT 0 — существующие строки примут первый же rebuild (любая версия >= 0).
ALTER TABLE freight_requests_lookup ADD COLUMN IF NOT EXISTS version BIGINT NOT NULL DEFAULT 0;
ALTER TABLE offers_lookup ADD COLUMN IF NOT EXISTS version BIGINT NOT NULL DEFAULT 0;
ALTER TABLE organizations_lookup ADD COLUMN IF NOT EXISTS version BIGINT NOT NULL DEFAULT 0;
ALTER TABLE support_tickets_lookup ADD COLUMN IF NOT EXISTS version BIGINT NOT NULL DEFAULT 0;
ALTER TABLE members_lookup ADD COLUMN IF NOT EXISTS version BIGINT NOT NULL DEFAULT 0;
ALTER TABLE invitations_lookup ADD COLUMN IF NOT EXISTS version BIGINT NOT NULL DEFAULT 0;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE freight_requests_lookup DROP COLUMN IF EXISTS version;
ALTER TABLE offers_lookup DROP COLUMN IF EXISTS version;
ALTER TABLE organizations_lookup DROP COLUMN IF EXISTS version;
ALTER TABLE support_tickets_lookup DROP COLUMN IF EXISTS version;
ALTER TABLE members_lookup DROP COLUMN IF EXISTS version;
ALTER TABLE invitations_lookup DROP COLUMN IF EXISTS version;

-- +goose StatementEnd
