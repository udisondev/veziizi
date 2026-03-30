-- +goose Up
-- +goose StatementBegin

-- Email templates table for notification emails
CREATE TABLE email_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Unique identifier for template (e.g., "password-reset", "offer-received")
    slug VARCHAR(100) NOT NULL UNIQUE,

    -- Display name for admin UI
    name VARCHAR(255) NOT NULL,

    -- Email subject (supports Go template variables)
    subject VARCHAR(500) NOT NULL,

    -- HTML body (Go template)
    body_html TEXT NOT NULL,

    -- Plain text body (Go template, fallback)
    body_text TEXT NOT NULL,

    -- Template category
    category VARCHAR(20) NOT NULL DEFAULT 'transactional'
        CHECK (category IN ('transactional', 'marketing')),

    -- Variables schema: describes available template variables
    -- Example: {"organization_name": {"type": "string", "required": true, "description": "Name of organization"}}
    variables_schema JSONB NOT NULL DEFAULT '{}',

    -- System templates cannot be deleted (password-reset, email-verification, etc.)
    is_system BOOLEAN NOT NULL DEFAULT FALSE,

    -- Active templates are used for sending
    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for fast lookup by slug (main access pattern)
CREATE INDEX idx_email_templates_slug ON email_templates(slug) WHERE is_active = TRUE;

-- Index for listing by category
CREATE INDEX idx_email_templates_category ON email_templates(category, is_active);

-- Comments
COMMENT ON TABLE email_templates IS 'Email templates for notification system';
COMMENT ON COLUMN email_templates.slug IS 'Unique template identifier (e.g., password-reset)';
COMMENT ON COLUMN email_templates.subject IS 'Email subject line, supports Go template syntax';
COMMENT ON COLUMN email_templates.body_html IS 'HTML email body, supports Go template syntax';
COMMENT ON COLUMN email_templates.body_text IS 'Plain text email body (fallback), supports Go template syntax';
COMMENT ON COLUMN email_templates.category IS 'transactional (high priority, no tracking) or marketing (with tracking, requires opt-in)';
COMMENT ON COLUMN email_templates.variables_schema IS 'JSON schema describing available template variables';
COMMENT ON COLUMN email_templates.is_system IS 'System templates cannot be deleted';
COMMENT ON COLUMN email_templates.is_active IS 'Only active templates are used for sending';

-- Insert default system templates
INSERT INTO email_templates (slug, name, subject, body_html, body_text, category, variables_schema, is_system) VALUES
(
    'password-reset',
    'Password Reset',
    'Сброс пароля в Veziizi',
    '<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="font-family: -apple-system, BlinkMacSystemFont, ''Segoe UI'', Roboto, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background-color: #f8f9fa; border-radius: 8px; padding: 30px;">
        <h1 style="color: #1a1a1a; margin-top: 0; font-size: 24px;">Сброс пароля</h1>
        <p style="color: #4a4a4a; font-size: 16px;">
            Вы запросили сброс пароля для вашего аккаунта в Veziizi.
        </p>
        <p style="color: #4a4a4a; font-size: 16px;">
            Нажмите кнопку ниже, чтобы установить новый пароль:
        </p>
        <p style="margin-top: 20px;">
            <a href="{{.ResetLink}}" style="background-color: #4F46E5; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; display: inline-block;">
                Сбросить пароль
            </a>
        </p>
        <p style="color: #999; font-size: 14px; margin-top: 20px;">
            Ссылка действительна в течение 1 часа.
        </p>
        <p style="color: #999; font-size: 14px;">
            Если вы не запрашивали сброс пароля, просто проигнорируйте это письмо.
        </p>
    </div>
    <p style="color: #999; font-size: 12px; margin-top: 20px; text-align: center;">
        Это письмо отправлено автоматически. Пожалуйста, не отвечайте на него.
    </p>
</body>
</html>',
    'Сброс пароля в Veziizi

Вы запросили сброс пароля для вашего аккаунта в Veziizi.

Перейдите по ссылке, чтобы установить новый пароль:
{{.ResetLink}}

Ссылка действительна в течение 1 часа.

Если вы не запрашивали сброс пароля, просто проигнорируйте это письмо.

---
Это письмо отправлено автоматически.',
    'transactional',
    '{"ResetLink": {"type": "string", "required": true, "description": "Password reset link"}}',
    TRUE
),
(
    'email-verification',
    'Email Verification',
    'Подтверждение email в Veziizi',
    '<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="font-family: -apple-system, BlinkMacSystemFont, ''Segoe UI'', Roboto, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background-color: #f8f9fa; border-radius: 8px; padding: 30px;">
        <h1 style="color: #1a1a1a; margin-top: 0; font-size: 24px;">Подтверждение email</h1>
        <p style="color: #4a4a4a; font-size: 16px;">
            Подтвердите ваш email адрес для получения уведомлений от Veziizi.
        </p>
        <p style="margin-top: 20px;">
            <a href="{{.VerificationLink}}" style="background-color: #4F46E5; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; display: inline-block;">
                Подтвердить email
            </a>
        </p>
        <p style="color: #999; font-size: 14px; margin-top: 20px;">
            Ссылка действительна в течение 24 часов.
        </p>
    </div>
    <p style="color: #999; font-size: 12px; margin-top: 20px; text-align: center;">
        Это письмо отправлено автоматически. Пожалуйста, не отвечайте на него.
    </p>
</body>
</html>',
    'Подтверждение email в Veziizi

Подтвердите ваш email адрес для получения уведомлений от Veziizi.

Перейдите по ссылке:
{{.VerificationLink}}

Ссылка действительна в течение 24 часов.

---
Это письмо отправлено автоматически.',
    'transactional',
    '{"VerificationLink": {"type": "string", "required": true, "description": "Email verification link"}}',
    TRUE
),
(
    'offer-received',
    'New Offer Received',
    'Новое предложение на заявку #{{.RequestNumber}}',
    '<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="font-family: -apple-system, BlinkMacSystemFont, ''Segoe UI'', Roboto, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background-color: #f8f9fa; border-radius: 8px; padding: 30px;">
        <h1 style="color: #1a1a1a; margin-top: 0; font-size: 24px;">Новое предложение</h1>
        <p style="color: #4a4a4a; font-size: 16px;">
            На вашу заявку <strong>#{{.RequestNumber}}</strong> поступило новое предложение от <strong>{{.CarrierName}}</strong>.
        </p>
        <p style="color: #4a4a4a; font-size: 16px;">
            <strong>Цена:</strong> {{.Price}} ₽
        </p>
        <p style="margin-top: 20px;">
            <a href="{{.RequestLink}}" style="background-color: #4F46E5; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; display: inline-block;">
                Посмотреть предложение
            </a>
        </p>
    </div>
    <p style="color: #999; font-size: 12px; margin-top: 20px; text-align: center;">
        <a href="{{.SettingsLink}}" style="color: #666;">Настроить уведомления</a>
    </p>
</body>
</html>',
    'Новое предложение на заявку #{{.RequestNumber}}

На вашу заявку #{{.RequestNumber}} поступило новое предложение от {{.CarrierName}}.

Цена: {{.Price}} ₽

Посмотреть предложение: {{.RequestLink}}

---
Настроить уведомления: {{.SettingsLink}}',
    'transactional',
    '{"RequestNumber": {"type": "string", "required": true}, "CarrierName": {"type": "string", "required": true}, "Price": {"type": "string", "required": true}, "RequestLink": {"type": "string", "required": true}, "SettingsLink": {"type": "string", "required": true}}',
    TRUE
),
(
    'offer-selected',
    'Offer Selected',
    'Ваше предложение выбрано!',
    '<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="font-family: -apple-system, BlinkMacSystemFont, ''Segoe UI'', Roboto, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background-color: #f8f9fa; border-radius: 8px; padding: 30px;">
        <h1 style="color: #1a1a1a; margin-top: 0; font-size: 24px;">Ваше предложение выбрано!</h1>
        <p style="color: #4a4a4a; font-size: 16px;">
            Заказчик <strong>{{.CustomerName}}</strong> выбрал ваше предложение на заявку <strong>#{{.RequestNumber}}</strong>.
        </p>
        <p style="color: #4a4a4a; font-size: 16px;">
            Пожалуйста, подтвердите готовность к выполнению заказа.
        </p>
        <p style="margin-top: 20px;">
            <a href="{{.RequestLink}}" style="background-color: #10B981; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; display: inline-block;">
                Подтвердить
            </a>
        </p>
    </div>
    <p style="color: #999; font-size: 12px; margin-top: 20px; text-align: center;">
        <a href="{{.SettingsLink}}" style="color: #666;">Настроить уведомления</a>
    </p>
</body>
</html>',
    'Ваше предложение выбрано!

Заказчик {{.CustomerName}} выбрал ваше предложение на заявку #{{.RequestNumber}}.

Пожалуйста, подтвердите готовность к выполнению заказа.

Подтвердить: {{.RequestLink}}

---
Настроить уведомления: {{.SettingsLink}}',
    'transactional',
    '{"CustomerName": {"type": "string", "required": true}, "RequestNumber": {"type": "string", "required": true}, "RequestLink": {"type": "string", "required": true}, "SettingsLink": {"type": "string", "required": true}}',
    TRUE
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS email_templates;

-- +goose StatementEnd
