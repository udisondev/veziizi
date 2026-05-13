-- +goose Up
-- +goose StatementBegin

-- freight_requests_lookup.status was VARCHAR(20), but the FreightRequestStatus
-- enum contains "cancelled_after_confirmed" (25 chars), which made the
-- CancelledAfterConfirmed handler crash with SQLSTATE 22001 (value too long
-- for type character varying).
ALTER TABLE freight_requests_lookup ALTER COLUMN status TYPE VARCHAR(40);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE freight_requests_lookup ALTER COLUMN status TYPE VARCHAR(20);

-- +goose StatementEnd
