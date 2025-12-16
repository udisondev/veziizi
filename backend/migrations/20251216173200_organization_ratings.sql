-- +goose Up
-- +goose StatementBegin

-- Отзывы на организации (денормализация из Order events)
CREATE TABLE organization_reviews_lookup (
    id UUID PRIMARY KEY,
    order_id UUID NOT NULL,
    reviewer_org_id UUID NOT NULL,
    reviewer_org_name VARCHAR(255) NOT NULL,
    reviewed_org_id UUID NOT NULL,
    rating INT NOT NULL CHECK (rating >= 1 AND rating <= 5),
    comment TEXT,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_org_reviews_reviewed_org ON organization_reviews_lookup (reviewed_org_id, created_at DESC);
CREATE INDEX idx_org_reviews_reviewer_org ON organization_reviews_lookup (reviewer_org_id);
CREATE INDEX idx_org_reviews_order ON organization_reviews_lookup (order_id);

-- Агрегированный рейтинг организации
CREATE TABLE organization_ratings (
    org_id UUID PRIMARY KEY,
    total_reviews INT NOT NULL DEFAULT 0,
    sum_rating INT NOT NULL DEFAULT 0,
    average_rating NUMERIC(3,2) NOT NULL DEFAULT 0
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS organization_ratings;
DROP TABLE IF EXISTS organization_reviews_lookup;

-- +goose StatementEnd
