-- +goose Up
-- +goose StatementBegin

CREATE TABLE organizations_lookup (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    legal_name TEXT NOT NULL,
    inn TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_organizations_lookup_status ON organizations_lookup(status);
CREATE INDEX idx_organizations_lookup_inn ON organizations_lookup(inn);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS organizations_lookup;

-- +goose StatementEnd
