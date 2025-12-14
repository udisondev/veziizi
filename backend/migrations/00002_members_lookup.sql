-- +goose Up
-- +goose StatementBegin

-- Members lookup (for auth)
CREATE TABLE members_lookup (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    phone VARCHAR(50),
    role VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_members_organization ON members_lookup (organization_id);
CREATE INDEX idx_members_status ON members_lookup (status);

-- Invitations lookup (for token search)
CREATE TABLE invitations_lookup (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL,
    email VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL,
    token VARCHAR(255) NOT NULL UNIQUE,
    status VARCHAR(20) NOT NULL,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_invitations_token ON invitations_lookup (token);
CREATE INDEX idx_invitations_status ON invitations_lookup (status);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS invitations_lookup;
DROP TABLE IF EXISTS members_lookup;

-- +goose StatementEnd
