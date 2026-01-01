-- +goose Up
-- Support tickets system

-- Sequence for ticket numbers
CREATE SEQUENCE IF NOT EXISTS ticket_number_seq START WITH 1;

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

-- Add Telegram columns to platform_admins for notifications
ALTER TABLE platform_admins ADD COLUMN telegram_chat_id BIGINT;
ALTER TABLE platform_admins ADD COLUMN telegram_username VARCHAR(64);

-- +goose Down
ALTER TABLE platform_admins DROP COLUMN IF EXISTS telegram_username;
ALTER TABLE platform_admins DROP COLUMN IF EXISTS telegram_chat_id;

DROP TABLE IF EXISTS support_tickets_lookup;
DROP SEQUENCE IF EXISTS ticket_number_seq;
