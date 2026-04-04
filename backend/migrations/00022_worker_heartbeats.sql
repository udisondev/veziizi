-- +goose Up
CREATE TABLE worker_heartbeats (
    name             TEXT PRIMARY KEY,
    worker_type      TEXT NOT NULL DEFAULT 'event',
    started_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_heartbeat   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status           TEXT NOT NULL DEFAULT 'running'
);

CREATE INDEX idx_worker_heartbeats_status ON worker_heartbeats(status);

-- +goose Down
DROP TABLE IF EXISTS worker_heartbeats;
