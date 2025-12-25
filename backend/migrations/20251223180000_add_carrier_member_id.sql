-- +goose Up
-- Добавляем carrier_member_id в offers_lookup для уведомлений перевозчику
ALTER TABLE offers_lookup ADD COLUMN carrier_member_id UUID;
CREATE INDEX idx_offers_carrier_member ON offers_lookup(carrier_member_id);

-- +goose Down
DROP INDEX IF EXISTS idx_offers_carrier_member;
ALTER TABLE offers_lookup DROP COLUMN IF EXISTS carrier_member_id;
