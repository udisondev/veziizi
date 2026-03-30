-- +goose Up
-- +goose StatementBegin

-- Add email fields to notification_preferences
ALTER TABLE notification_preferences
    ADD COLUMN email VARCHAR(255),
    ADD COLUMN email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN email_verified_at TIMESTAMPTZ,
    ADD COLUMN email_marketing_consent BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN email_marketing_consent_at TIMESTAMPTZ;

-- Index for email lookups (find member by email)
CREATE INDEX idx_notification_preferences_email ON notification_preferences(email) WHERE email IS NOT NULL;

-- Update default for enabled_categories to include email: false
-- Note: existing rows will keep their current JSONB values, new rows will get email: false
ALTER TABLE notification_preferences
    ALTER COLUMN enabled_categories SET DEFAULT '{
        "freight_requests": {"in_app": true, "telegram": false, "email": false},
        "offers": {"in_app": true, "telegram": false, "email": false},
        "orders": {"in_app": true, "telegram": false, "email": false},
        "reviews": {"in_app": true, "telegram": false, "email": false},
        "organization": {"in_app": true, "telegram": false, "email": false}
    }'::jsonb;

-- Add email channel to notification_delivery_log check constraint (if exists)
-- Note: channel is VARCHAR(20), no constraint exists, just documenting email as valid value

COMMENT ON COLUMN notification_preferences.email IS 'Email address for notifications (can differ from auth email)';
COMMENT ON COLUMN notification_preferences.email_verified IS 'Whether email is verified for notifications';
COMMENT ON COLUMN notification_preferences.email_verified_at IS 'When email was verified';
COMMENT ON COLUMN notification_preferences.email_marketing_consent IS 'Explicit opt-in for marketing emails (GDPR)';
COMMENT ON COLUMN notification_preferences.email_marketing_consent_at IS 'When marketing consent was given';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Remove email fields from notification_preferences
DROP INDEX IF EXISTS idx_notification_preferences_email;

ALTER TABLE notification_preferences
    DROP COLUMN IF EXISTS email_marketing_consent_at,
    DROP COLUMN IF EXISTS email_marketing_consent,
    DROP COLUMN IF EXISTS email_verified_at,
    DROP COLUMN IF EXISTS email_verified,
    DROP COLUMN IF EXISTS email;

-- Restore original default without email
ALTER TABLE notification_preferences
    ALTER COLUMN enabled_categories SET DEFAULT '{
        "offers": {"in_app": true, "telegram": false},
        "orders": {"in_app": true, "telegram": false},
        "reviews": {"in_app": true, "telegram": false},
        "organization": {"in_app": true, "telegram": false}
    }'::jsonb;

-- +goose StatementEnd
