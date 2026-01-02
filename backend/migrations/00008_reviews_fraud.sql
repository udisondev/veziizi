-- +goose Up
-- +goose StatementBegin

-- ==========================================================
-- Reviews with weights and statuses
-- ==========================================================
CREATE TABLE reviews_lookup (
    id UUID PRIMARY KEY,
    order_id UUID NOT NULL,
    reviewer_org_id UUID NOT NULL,
    reviewed_org_id UUID NOT NULL,
    rating INT NOT NULL CHECK (rating >= 1 AND rating <= 5),
    comment TEXT,

    -- Order data for weight calculation
    order_amount BIGINT NOT NULL,
    order_currency VARCHAR(3) NOT NULL,
    order_created_at TIMESTAMPTZ NOT NULL,
    order_completed_at TIMESTAMPTZ NOT NULL,

    -- Weight coefficients
    raw_weight NUMERIC(5,4) NOT NULL DEFAULT 1.0,
    final_weight NUMERIC(5,4) NOT NULL DEFAULT 1.0,

    -- Fraud detection
    fraud_score NUMERIC(5,4) NOT NULL DEFAULT 0.0,
    requires_moderation BOOLEAN NOT NULL DEFAULT FALSE,

    -- Status and timeline
    status VARCHAR(30) NOT NULL DEFAULT 'pending_analysis',
    activation_date TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL,
    analyzed_at TIMESTAMPTZ,
    moderated_at TIMESTAMPTZ,
    moderated_by UUID,
    activated_at TIMESTAMPTZ,

    CONSTRAINT reviews_valid_status CHECK (status IN (
        'pending_analysis', 'pending_moderation', 'approved',
        'rejected', 'active', 'deactivated'
    ))
);

CREATE INDEX idx_reviews_reviewed_org ON reviews_lookup(reviewed_org_id, status, created_at DESC);
CREATE INDEX idx_reviews_reviewer_org ON reviews_lookup(reviewer_org_id, created_at DESC);
CREATE INDEX idx_reviews_pending_moderation ON reviews_lookup(status, fraud_score DESC)
    WHERE status = 'pending_moderation';
CREATE INDEX idx_reviews_pending_activation ON reviews_lookup(activation_date)
    WHERE status = 'approved' AND activated_at IS NULL;
CREATE INDEX idx_reviews_order ON reviews_lookup(order_id);
CREATE INDEX idx_reviews_active ON reviews_lookup(reviewed_org_id)
    WHERE status = 'active';

-- ==========================================================
-- Fraud signals for each review (details)
-- ==========================================================
CREATE TABLE review_fraud_signals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    review_id UUID NOT NULL REFERENCES reviews_lookup(id) ON DELETE CASCADE,
    signal_type VARCHAR(50) NOT NULL,
    severity VARCHAR(10) NOT NULL CHECK (severity IN ('low', 'medium', 'high')),
    description TEXT NOT NULL,
    score_impact NUMERIC(5,4) NOT NULL DEFAULT 0.0,
    evidence JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_fraud_signals_review ON review_fraud_signals(review_id);
CREATE INDEX idx_fraud_signals_type ON review_fraud_signals(signal_type);

-- ==========================================================
-- Interaction statistics between organizations
-- ==========================================================
CREATE TABLE org_interaction_stats (
    org_a UUID NOT NULL,
    org_b UUID NOT NULL,

    -- Order statistics
    total_orders INT NOT NULL DEFAULT 0,
    completed_orders INT NOT NULL DEFAULT 0,
    cancelled_orders INT NOT NULL DEFAULT 0,

    -- Review statistics (directional)
    reviews_a_to_b INT NOT NULL DEFAULT 0,
    reviews_b_to_a INT NOT NULL DEFAULT 0,
    sum_rating_a_to_b INT NOT NULL DEFAULT 0,
    sum_rating_b_to_a INT NOT NULL DEFAULT 0,

    -- Average metrics
    avg_order_amount BIGINT,
    avg_completion_hours INT,

    first_interaction_at TIMESTAMPTZ,
    last_interaction_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (org_a, org_b),
    CHECK (org_a < org_b)
);

CREATE INDEX idx_org_interaction_org_a ON org_interaction_stats(org_a);
CREATE INDEX idx_org_interaction_org_b ON org_interaction_stats(org_b);

-- ==========================================================
-- Organization reputation as reviewer
-- ==========================================================
CREATE TABLE org_reviewer_reputation (
    org_id UUID PRIMARY KEY,

    -- Statistics of left reviews
    total_reviews_left INT NOT NULL DEFAULT 0,
    active_reviews_left INT NOT NULL DEFAULT 0,
    rejected_reviews INT NOT NULL DEFAULT 0,
    deactivated_reviews INT NOT NULL DEFAULT 0,

    -- Reputation score (0.0-1.0)
    reputation_score NUMERIC(5,4) NOT NULL DEFAULT 1.0,

    -- Fraudster flags
    is_suspected_fraudster BOOLEAN NOT NULL DEFAULT FALSE,
    is_confirmed_fraudster BOOLEAN NOT NULL DEFAULT FALSE,
    fraudster_marked_at TIMESTAMPTZ,
    fraudster_marked_by UUID,
    fraudster_reason TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ==========================================================
-- Aggregated organization rating (weighted)
-- ==========================================================
CREATE TABLE organization_ratings (
    org_id UUID PRIMARY KEY,

    -- Simple rating (for API backward compatibility)
    total_reviews INT NOT NULL DEFAULT 0,
    sum_rating INT NOT NULL DEFAULT 0,
    average_rating NUMERIC(3,2) NOT NULL DEFAULT 0,

    -- Weighted rating
    weighted_sum NUMERIC(10,4) NOT NULL DEFAULT 0,
    weight_total NUMERIC(10,4) NOT NULL DEFAULT 0,
    weighted_average NUMERIC(3,2) NOT NULL DEFAULT 0,

    -- Status counters
    pending_reviews INT NOT NULL DEFAULT 0,
    rejected_reviews INT NOT NULL DEFAULT 0,

    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ==========================================================
-- Registration metadata for sock puppet detection
-- ==========================================================
CREATE TABLE org_registration_metadata (
    org_id UUID PRIMARY KEY,
    registration_ip INET,
    registration_user_agent TEXT,
    registration_fingerprint VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_reg_metadata_ip ON org_registration_metadata(registration_ip);
CREATE INDEX idx_reg_metadata_fingerprint ON org_registration_metadata(registration_fingerprint)
    WHERE registration_fingerprint IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS org_registration_metadata;
DROP TABLE IF EXISTS organization_ratings;
DROP TABLE IF EXISTS org_reviewer_reputation;
DROP TABLE IF EXISTS org_interaction_stats;
DROP TABLE IF EXISTS review_fraud_signals;
DROP TABLE IF EXISTS reviews_lookup;

-- +goose StatementEnd
