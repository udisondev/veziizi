-- +goose Up
-- +goose StatementBegin

-- Tokens for email verification functionality
CREATE TABLE email_verification_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    member_id UUID NOT NULL,

    -- Cryptographically secure token (32 bytes = 64 hex chars)
    token VARCHAR(64) NOT NULL UNIQUE,

    -- Email being verified (stored to handle email changes during verification)
    email VARCHAR(255) NOT NULL,

    -- Token expiration (default 24 hours from creation)
    expires_at TIMESTAMPTZ NOT NULL,

    -- When the token was used (NULL if not used yet)
    used_at TIMESTAMPTZ,

    -- Audit fields
    ip_address INET,
    user_agent TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for token lookup (main access pattern)
CREATE INDEX idx_email_verification_tokens_token ON email_verification_tokens(token)
    WHERE used_at IS NULL;

-- Index for cleanup of expired tokens
CREATE INDEX idx_email_verification_tokens_expires ON email_verification_tokens(expires_at)
    WHERE used_at IS NULL;

-- Index for rate limiting (count tokens per member in time window)
CREATE INDEX idx_email_verification_tokens_member_created ON email_verification_tokens(member_id, created_at DESC);

-- Comments
COMMENT ON TABLE email_verification_tokens IS 'Tokens for email verification functionality';
COMMENT ON COLUMN email_verification_tokens.token IS 'Cryptographically secure token (hex-encoded 32 bytes)';
COMMENT ON COLUMN email_verification_tokens.email IS 'Email address being verified (snapshot at token creation)';
COMMENT ON COLUMN email_verification_tokens.expires_at IS 'Token expiration time (24 hours from creation)';
COMMENT ON COLUMN email_verification_tokens.used_at IS 'When token was used to verify email (NULL if unused)';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS email_verification_tokens;

-- +goose StatementEnd
