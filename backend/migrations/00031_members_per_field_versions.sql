-- +goose Up
-- +goose StatementBegin

-- Per-field version-guard для members_lookup.
--
-- role и status — ортогональные поля, которые меняются независимыми событиями
-- агрегата Organization. Общий version-столбец (00030) терял обновление одного
-- поля при out-of-order доставке: MemberBlocked(v5) поднимал version до 5, и
-- пришедший позже MemberRoleChanged(v3) молча ack'ался как «устаревший», хотя
-- роль он нёс актуальную. Отдельные version-колонки на поле решают это:
-- guard сравнивает только версии событий, трогающих одно и то же поле.
ALTER TABLE members_lookup ADD COLUMN IF NOT EXISTS role_version BIGINT NOT NULL DEFAULT 0;
ALTER TABLE members_lookup ADD COLUMN IF NOT EXISTS status_version BIGINT NOT NULL DEFAULT 0;
UPDATE members_lookup SET role_version = version, status_version = version;
ALTER TABLE members_lookup DROP COLUMN IF EXISTS version;

-- Tombstone удалённых участников. Закрывает обе дыры незащищённого DELETE при
-- at-least-once/out-of-order:
--   - повторно доставленный MemberAdded после MemberRemoved не воскрешает
--     строку (INSERT проверяет tombstone);
--   - статусное событие, пришедшее после удаления строки, ack'ается (хендлер
--     видит tombstone), а не уходит в бесконечный retry → DLQ.
CREATE TABLE IF NOT EXISTS members_removed (
    id         UUID PRIMARY KEY,
    removed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS members_removed;
ALTER TABLE members_lookup ADD COLUMN IF NOT EXISTS version BIGINT NOT NULL DEFAULT 0;
UPDATE members_lookup SET version = GREATEST(role_version, status_version);
ALTER TABLE members_lookup DROP COLUMN IF EXISTS role_version;
ALTER TABLE members_lookup DROP COLUMN IF EXISTS status_version;

-- +goose StatementEnd
