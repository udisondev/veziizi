-- +goose Up
-- +goose StatementBegin

-- Sequence for ticket numbers
CREATE SEQUENCE ticket_number_seq START WITH 1;

-- Lookup table for support tickets
CREATE TABLE support_tickets_lookup (
    id UUID PRIMARY KEY,
    ticket_number BIGINT NOT NULL,
    member_id UUID NOT NULL,
    org_id UUID NOT NULL,
    subject VARCHAR(255) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'open',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at TIMESTAMPTZ
);

-- Indexes for efficient queries
CREATE UNIQUE INDEX idx_tickets_number ON support_tickets_lookup (ticket_number);
CREATE INDEX idx_tickets_member ON support_tickets_lookup (member_id);
CREATE INDEX idx_tickets_org ON support_tickets_lookup (org_id);
CREATE INDEX idx_tickets_status ON support_tickets_lookup (status);
CREATE INDEX idx_tickets_created ON support_tickets_lookup (created_at DESC);
CREATE INDEX idx_tickets_updated ON support_tickets_lookup (updated_at DESC);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS support_tickets_lookup;
DROP SEQUENCE IF EXISTS ticket_number_seq;

-- +goose StatementEnd
