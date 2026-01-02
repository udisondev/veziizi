-- +goose Up
-- +goose StatementBegin

-- Members lookup (for auth)
-- Consolidated from: 00002, 20251219121809
CREATE TABLE members_lookup (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    phone VARCHAR(50),
    telegram_id BIGINT,
    role VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL,
    -- Fraud detection metadata
    registration_ip INET,
    registration_fingerprint VARCHAR(64),
    registration_user_agent TEXT,
    last_login_at TIMESTAMPTZ,
    last_login_ip INET,
    last_login_fingerprint VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_members_organization ON members_lookup (organization_id);
CREATE INDEX idx_members_status ON members_lookup (status);
CREATE INDEX idx_members_registration_ip ON members_lookup(registration_ip);
CREATE INDEX idx_members_registration_fingerprint ON members_lookup(registration_fingerprint)
    WHERE registration_fingerprint IS NOT NULL;
CREATE INDEX idx_members_last_login_ip ON members_lookup(last_login_ip);

-- Invitations lookup (for token search)
-- Consolidated from: 00002, 20251216180000
CREATE TABLE invitations_lookup (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL,
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    phone VARCHAR(50),
    role VARCHAR(20) NOT NULL,
    token VARCHAR(255) NOT NULL UNIQUE,
    status VARCHAR(20) NOT NULL,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_invitations_token ON invitations_lookup (token);
CREATE INDEX idx_invitations_status ON invitations_lookup (status);
CREATE INDEX idx_invitations_organization ON invitations_lookup (organization_id);

-- Login history for fraud detection and audit
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
DROP TABLE IF EXISTS invitations_lookup;
DROP TABLE IF EXISTS members_lookup;

-- +goose StatementEnd
