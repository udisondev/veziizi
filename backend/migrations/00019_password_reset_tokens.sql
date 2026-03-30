-- +goose Up
-- +goose StatementBegin

-- Tokens for password reset functionality
CREATE TABLE password_reset_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    member_id UUID NOT NULL REFERENCES members_lookup(id) ON DELETE CASCADE,

    -- Cryptographically secure token (32 bytes = 64 hex chars)
    token VARCHAR(64) NOT NULL UNIQUE,

    -- Token expiration (default 1 hour from creation)
    expires_at TIMESTAMPTZ NOT NULL,

    -- When the token was used (NULL if not used yet)
    used_at TIMESTAMPTZ,

    -- Audit fields
    ip_address INET,
    user_agent TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for token lookup (main access pattern)
CREATE INDEX idx_password_reset_tokens_token ON password_reset_tokens(token)
    WHERE used_at IS NULL;

-- Index for cleanup of expired tokens
CREATE INDEX idx_password_reset_tokens_expires ON password_reset_tokens(expires_at)
    WHERE used_at IS NULL;

-- Index for rate limiting (count tokens per member in time window)
CREATE INDEX idx_password_reset_tokens_member_created ON password_reset_tokens(member_id, created_at DESC);

-- Comments
COMMENT ON TABLE password_reset_tokens IS 'Tokens for password reset functionality';
COMMENT ON COLUMN password_reset_tokens.token IS 'Cryptographically secure token (hex-encoded 32 bytes)';
COMMENT ON COLUMN password_reset_tokens.expires_at IS 'Token expiration time (1 hour from creation)';
COMMENT ON COLUMN password_reset_tokens.used_at IS 'When token was used to reset password (NULL if unused)';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS password_reset_tokens;

-- +goose StatementEnd
