-- +goose Up
-- +goose StatementBegin

-- Добавляем опциональные поля для предзаполнения профиля приглашённого
ALTER TABLE invitations_lookup ADD COLUMN name VARCHAR(255);
ALTER TABLE invitations_lookup ADD COLUMN phone VARCHAR(50);

-- Индекс по organization_id для списка приглашений
CREATE INDEX idx_invitations_organization ON invitations_lookup (organization_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_invitations_organization;
ALTER TABLE invitations_lookup DROP COLUMN IF EXISTS phone;
ALTER TABLE invitations_lookup DROP COLUMN IF EXISTS name;

-- +goose StatementEnd
