-- +goose Up
-- +goose StatementBegin

-- Fleet projection for organization vehicles
CREATE TABLE vehicles_lookup (
    id                  UUID PRIMARY KEY,
    org_id              UUID NOT NULL,
    registration_number VARCHAR(32) NOT NULL,
    brand               VARCHAR(64),
    model               VARCHAR(64),
    vehicle_type        VARCHAR(30) NOT NULL,
    vehicle_subtype     VARCHAR(30) NOT NULL,
    loading_types       TEXT[] NOT NULL DEFAULT '{}',
    capacity            NUMERIC(12, 2),
    volume              NUMERIC(12, 2),
    length              NUMERIC(8, 2),
    width               NUMERIC(8, 2),
    height              NUMERIC(8, 2),
    requires_adr        BOOLEAN NOT NULL DEFAULT FALSE,
    has_temperature     BOOLEAN NOT NULL DEFAULT FALSE,
    temp_min            NUMERIC(5, 2),
    temp_max            NUMERIC(5, 2),
    thermograph         BOOLEAN NOT NULL DEFAULT FALSE,
    status              VARCHAR(20) NOT NULL,
    rejection_reason    TEXT,
    created_at          TIMESTAMPTZ NOT NULL,
    updated_at          TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_vehicles_org_id ON vehicles_lookup(org_id);
CREATE INDEX idx_vehicles_status ON vehicles_lookup(status);
CREATE INDEX idx_vehicles_type ON vehicles_lookup(vehicle_type);
CREATE INDEX idx_vehicles_subtype ON vehicles_lookup(vehicle_subtype);
CREATE INDEX idx_vehicles_capacity ON vehicles_lookup(capacity);
CREATE INDEX idx_vehicles_volume ON vehicles_lookup(volume);
CREATE INDEX idx_vehicles_requires_adr ON vehicles_lookup(requires_adr);
CREATE INDEX idx_vehicles_loading_types ON vehicles_lookup USING GIN (loading_types);
CREATE INDEX idx_vehicles_created_at ON vehicles_lookup(created_at DESC);

-- Pending vehicles fast-list for admin moderation panel
CREATE TABLE pending_vehicles (
    id                  UUID PRIMARY KEY,
    org_id              UUID NOT NULL,
    registration_number VARCHAR(32) NOT NULL,
    brand               VARCHAR(64),
    model               VARCHAR(64),
    vehicle_type        VARCHAR(30) NOT NULL,
    vehicle_subtype     VARCHAR(30) NOT NULL,
    submitted_at        TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_pending_vehicles_submitted ON pending_vehicles(submitted_at DESC);
CREATE INDEX idx_pending_vehicles_org_id ON pending_vehicles(org_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS pending_vehicles;
DROP TABLE IF EXISTS vehicles_lookup;

-- +goose StatementEnd
