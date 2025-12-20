-- +goose Up
-- +goose StatementBegin

-- ==========================================================
-- Fraud сигналы для заказов
-- ==========================================================
CREATE TABLE order_fraud_signals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL,
    org_id UUID NOT NULL,
    signal_type VARCHAR(50) NOT NULL,
    severity VARCHAR(10) NOT NULL CHECK (severity IN ('low', 'medium', 'high')),
    description TEXT NOT NULL,
    score_impact NUMERIC(5,4) NOT NULL DEFAULT 0.0,
    evidence JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_order_fraud_signals_order ON order_fraud_signals(order_id);
CREATE INDEX idx_order_fraud_signals_org ON order_fraud_signals(org_id);
CREATE INDEX idx_order_fraud_signals_type ON order_fraud_signals(signal_type);

-- ==========================================================
-- Поведение организации в заказах (для fraud detection)
-- ==========================================================
CREATE TABLE org_order_behavior (
    org_id UUID PRIMARY KEY,

    -- Статистика как заказчик
    total_orders_as_customer INT NOT NULL DEFAULT 0,
    completed_as_customer INT NOT NULL DEFAULT 0,
    cancelled_as_customer INT NOT NULL DEFAULT 0,

    -- Статистика как перевозчик
    total_orders_as_carrier INT NOT NULL DEFAULT 0,
    completed_as_carrier INT NOT NULL DEFAULT 0,
    cancelled_as_carrier INT NOT NULL DEFAULT 0,

    -- Метрики выполнения
    avg_completion_hours NUMERIC(10,2),
    min_completion_hours NUMERIC(10,2),

    -- Fraud флаги
    is_suspicious BOOLEAN NOT NULL DEFAULT FALSE,
    suspicious_reason TEXT,
    suspicious_marked_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ==========================================================
-- Circular orders detection (цепочки заказов между орг)
-- ==========================================================
CREATE TABLE org_order_chains (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Участники цепочки (JSON массив org_id)
    chain_orgs JSONB NOT NULL,
    chain_length INT NOT NULL,

    -- Детали
    order_ids JSONB NOT NULL,
    total_amount BIGINT NOT NULL DEFAULT 0,

    -- Временное окно
    first_order_at TIMESTAMPTZ NOT NULL,
    last_order_at TIMESTAMPTZ NOT NULL,

    -- Статус
    is_suspicious BOOLEAN NOT NULL DEFAULT FALSE,
    reviewed_at TIMESTAMPTZ,
    reviewed_by UUID,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_org_order_chains_orgs ON org_order_chains USING GIN (chain_orgs);
CREATE INDEX idx_org_order_chains_suspicious ON org_order_chains(is_suspicious) WHERE is_suspicious = TRUE;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS org_order_chains;
DROP TABLE IF EXISTS org_order_behavior;
DROP TABLE IF EXISTS order_fraud_signals;

-- +goose StatementEnd
