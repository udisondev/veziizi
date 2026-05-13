-- +goose Up
-- +goose StatementBegin

-- Log of "customer pinged carrier about a freight request" events.
-- Anti-spam constraint: a customer cannot ping the same carrier_org twice on the
-- same freight_request. Records are append-only; they are not deleted when the
-- request changes status.
CREATE TABLE freight_request_invites_log (
    id                 UUID PRIMARY KEY,
    freight_request_id UUID NOT NULL,
    carrier_org_id     UUID NOT NULL,
    vehicle_id         UUID NOT NULL,
    invited_by         UUID NOT NULL,
    invited_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_invites_unique_pair
    ON freight_request_invites_log(freight_request_id, carrier_org_id);
CREATE INDEX idx_invites_freight ON freight_request_invites_log(freight_request_id);
CREATE INDEX idx_invites_carrier_org ON freight_request_invites_log(carrier_org_id);

-- Per-member view history for freight requests. Lets carriers come back to
-- previously seen requests they were notified about.
CREATE TABLE freight_request_views (
    member_id          UUID NOT NULL,
    freight_request_id UUID NOT NULL,
    first_viewed_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_viewed_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    view_count         INT NOT NULL DEFAULT 1,
    PRIMARY KEY (member_id, freight_request_id)
);

CREATE INDEX idx_views_member_last_viewed
    ON freight_request_views(member_id, last_viewed_at DESC);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS freight_request_views;
DROP TABLE IF EXISTS freight_request_invites_log;

-- +goose StatementEnd
