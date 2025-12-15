-- +goose Up
-- +goose StatementBegin

-- Platform admins table (simple table, not event sourced)
CREATE TABLE platform_admins (
    id UUID PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Pending organizations (для админ-панели модерации)
-- Добавляется при OrganizationCreated, удаляется при Approved/Rejected
CREATE TABLE pending_organizations (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    inn VARCHAR(20) NOT NULL,
    legal_name VARCHAR(255) NOT NULL,
    country VARCHAR(10) NOT NULL,
    email VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_pending_organizations_created_at ON pending_organizations (created_at DESC);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS pending_organizations;
DROP TABLE IF EXISTS platform_admins;

-- +goose StatementEnd
