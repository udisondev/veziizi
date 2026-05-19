-- +goose Up
-- +goose StatementBegin

-- Дедупликация обработки событий по проекциям. Накопительные операции в
-- проекциях (IncrementPendingReviews, AddWeightedRating и т.д.) не идемпотентны
-- сами по себе: повторная доставка одного и того же события раздувает счётчики.
-- Перед накопительной операцией handler берёт «резерв» по (projection_name,
-- event_id); ON CONFLICT DO NOTHING значит «уже обработано, skip».
--
-- event_id — это EventEnvelope.ID (UUID, сгенерированный при создании envelope),
-- стабильный через retry'и одного сообщения. Берётся из msg.Metadata["event_id"]
-- (кладётся EventEnvelopeMarshaler.Marshal).
CREATE TABLE projection_event_dedup (
    projection_name TEXT NOT NULL,
    event_id        UUID NOT NULL,
    processed_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (projection_name, event_id)
);

CREATE INDEX idx_projection_event_dedup_processed_at ON projection_event_dedup(processed_at);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS projection_event_dedup;
-- +goose StatementEnd
