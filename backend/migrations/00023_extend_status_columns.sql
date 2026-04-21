-- +goose Up
-- +goose StatementBegin

-- freight_requests_lookup.status must fit "cancelled_after_confirmed" (25 chars).
-- VARCHAR(20) causes SQLSTATE 22001 and the watermill consumer retries forever,
-- blocking freight_requests consumer group and starving downstream projections.
ALTER TABLE freight_requests_lookup
    ALTER COLUMN status TYPE VARCHAR(32);

ALTER TABLE offers_lookup
    ALTER COLUMN status TYPE VARCHAR(32);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE freight_requests_lookup
    ALTER COLUMN status TYPE VARCHAR(20);

ALTER TABLE offers_lookup
    ALTER COLUMN status TYPE VARCHAR(20);

-- +goose StatementEnd
