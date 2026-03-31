-- +goose Up

ALTER TABLE members_lookup
    ADD COLUMN failed_login_count INT NOT NULL DEFAULT 0,
    ADD COLUMN last_failed_login_at TIMESTAMPTZ,
    ADD COLUMN locked_until TIMESTAMPTZ;

CREATE INDEX idx_members_lookup_locked_until ON members_lookup (locked_until) WHERE locked_until IS NOT NULL;

-- +goose Down

DROP INDEX IF EXISTS idx_members_lookup_locked_until;

ALTER TABLE members_lookup
    DROP COLUMN IF EXISTS failed_login_count,
    DROP COLUMN IF EXISTS last_failed_login_at,
    DROP COLUMN IF EXISTS locked_until;
