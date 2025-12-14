-- +goose Up
-- +goose StatementBegin

-- Events table (append-only event store)
CREATE TABLE events (
    id UUID PRIMARY KEY,
    aggregate_id UUID NOT NULL,
    aggregate_type VARCHAR(100) NOT NULL,
    event_type VARCHAR(200) NOT NULL,
    version BIGINT NOT NULL,
    data JSONB NOT NULL,
    metadata JSONB,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT events_aggregate_version_unique UNIQUE (aggregate_id, version)
);

CREATE INDEX idx_events_aggregate ON events (aggregate_id, aggregate_type);
CREATE INDEX idx_events_occurred_at ON events (occurred_at);
CREATE INDEX idx_events_event_type ON events (event_type);

-- Snapshots table
CREATE TABLE snapshots (
    aggregate_id UUID PRIMARY KEY,
    aggregate_type VARCHAR(100) NOT NULL,
    version BIGINT NOT NULL,
    data JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS snapshots;
DROP TABLE IF EXISTS events;

-- +goose StatementEnd
