-- +goose Up
-- +goose StatementBegin

-- Улучшенный индекс для pending_moderation запросов
-- Добавляем created_at для оптимизации ORDER BY fraud_score DESC, created_at ASC
DROP INDEX IF EXISTS idx_reviews_pending_moderation;
CREATE INDEX idx_reviews_pending_moderation ON reviews_lookup(status, fraud_score DESC, created_at ASC)
    WHERE status = 'pending_moderation';

-- Индекс для ListReviewsForActivation (approved + activation_date)
CREATE INDEX IF NOT EXISTS idx_reviews_for_activation ON reviews_lookup(status, activation_date ASC)
    WHERE status = 'approved';

-- Индекс для ListActiveReviewsByReviewer
CREATE INDEX IF NOT EXISTS idx_reviews_active_by_reviewer ON reviews_lookup(reviewer_org_id, status)
    WHERE status = 'active';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_reviews_active_by_reviewer;
DROP INDEX IF EXISTS idx_reviews_for_activation;
DROP INDEX IF EXISTS idx_reviews_pending_moderation;

-- Восстанавливаем старый индекс
CREATE INDEX idx_reviews_pending_moderation ON reviews_lookup(status, fraud_score DESC)
    WHERE status = 'pending_moderation';

-- +goose StatementEnd
