-- +goose Up
-- +goose StatementBegin

-- Session events for tracking login/API activity
CREATE TABLE session_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    member_id UUID NOT NULL,
    organization_id UUID NOT NULL,
    event_type VARCHAR(30) NOT NULL, -- login, logout, api_call
    ip_address INET,
    fingerprint VARCHAR(64),
    user_agent TEXT,
    geo_country VARCHAR(2),
    geo_city VARCHAR(100),
    geo_lat NUMERIC(10, 7),
    geo_lon NUMERIC(10, 7),
    endpoint VARCHAR(255), -- for api_call events
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_session_events_member ON session_events(member_id, created_at DESC);
CREATE INDEX idx_session_events_org ON session_events(organization_id, created_at DESC);
CREATE INDEX idx_session_events_ip ON session_events(ip_address, created_at DESC);
CREATE INDEX idx_session_events_type ON session_events(event_type, created_at DESC);

-- Session fraud signals
CREATE TABLE session_fraud_signals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    member_id UUID NOT NULL,
    organization_id UUID NOT NULL,
    signal_type VARCHAR(50) NOT NULL, -- login_geo_jump, session_anomaly, api_abuse
    severity VARCHAR(10) NOT NULL, -- low, medium, high
    description TEXT NOT NULL,
    score_impact NUMERIC(5,4) NOT NULL,
    evidence JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_session_fraud_signals_member ON session_fraud_signals(member_id, created_at DESC);
CREATE INDEX idx_session_fraud_signals_org ON session_fraud_signals(organization_id, created_at DESC);
CREATE INDEX idx_session_fraud_signals_type ON session_fraud_signals(signal_type);

-- Member session behavior (for anomaly detection)
CREATE TABLE member_session_behavior (
    member_id UUID PRIMARY KEY,
    typical_login_hours JSONB, -- histogram of login hours [0-23]
    typical_countries TEXT[], -- list of countries
    typical_ips TEXT[], -- list of known IPs (last 10)
    last_login_at TIMESTAMPTZ,
    last_login_ip INET,
    last_login_country VARCHAR(2),
    last_login_lat NUMERIC(10, 7),
    last_login_lon NUMERIC(10, 7),
    total_logins INT NOT NULL DEFAULT 0,
    suspicious_logins INT NOT NULL DEFAULT 0,
    is_suspicious BOOLEAN NOT NULL DEFAULT FALSE,
    suspicious_reason TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Rate limiting table (for api_abuse without Redis)
CREATE TABLE api_rate_limits (
    key VARCHAR(255) PRIMARY KEY, -- member_id or ip:endpoint
    request_count INT NOT NULL DEFAULT 0,
    window_start TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    blocked_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_api_rate_limits_window ON api_rate_limits(window_start);
CREATE INDEX idx_api_rate_limits_blocked ON api_rate_limits(blocked_until) WHERE blocked_until IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS api_rate_limits;
DROP TABLE IF EXISTS member_session_behavior;
DROP TABLE IF EXISTS session_fraud_signals;
DROP TABLE IF EXISTS session_events;
-- +goose StatementEnd
