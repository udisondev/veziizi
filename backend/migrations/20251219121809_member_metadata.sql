-- +goose Up
-- +goose StatementBegin

-- Расширение members_lookup для хранения metadata
ALTER TABLE members_lookup
    ADD COLUMN registration_ip INET,
    ADD COLUMN registration_fingerprint VARCHAR(64),
    ADD COLUMN registration_user_agent TEXT,
    ADD COLUMN last_login_at TIMESTAMPTZ,
    ADD COLUMN last_login_ip INET,
    ADD COLUMN last_login_fingerprint VARCHAR(64);

CREATE INDEX idx_members_registration_ip ON members_lookup(registration_ip);
CREATE INDEX idx_members_registration_fingerprint ON members_lookup(registration_fingerprint)
    WHERE registration_fingerprint IS NOT NULL;
CREATE INDEX idx_members_last_login_ip ON members_lookup(last_login_ip);

-- История логинов для fraud detection и аудита
CREATE TABLE member_login_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    member_id UUID NOT NULL REFERENCES members_lookup(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL,
    ip_address INET,
    fingerprint VARCHAR(64),
    user_agent TEXT,
    status VARCHAR(20) NOT NULL, -- 'success', 'failed_password', 'failed_blocked'
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_login_history_member ON member_login_history(member_id, created_at DESC);
CREATE INDEX idx_login_history_ip ON member_login_history(ip_address);
CREATE INDEX idx_login_history_fingerprint ON member_login_history(fingerprint)
    WHERE fingerprint IS NOT NULL;
CREATE INDEX idx_login_history_created ON member_login_history(created_at);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS member_login_history;

DROP INDEX IF EXISTS idx_members_last_login_ip;
DROP INDEX IF EXISTS idx_members_registration_fingerprint;
DROP INDEX IF EXISTS idx_members_registration_ip;

ALTER TABLE members_lookup
    DROP COLUMN IF EXISTS registration_ip,
    DROP COLUMN IF EXISTS registration_fingerprint,
    DROP COLUMN IF EXISTS registration_user_agent,
    DROP COLUMN IF EXISTS last_login_at,
    DROP COLUMN IF EXISTS last_login_ip,
    DROP COLUMN IF EXISTS last_login_fingerprint;

-- +goose StatementEnd
