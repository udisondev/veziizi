package handlers

import (
	"context"
	"errors"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/google/uuid"
)

// eventIDFromCtx достаёт UUID события из msg.Metadata. Marshaler кладёт его
// под ключом "event_id" — это EventEnvelope.ID, стабильный через retry'и.
func eventIDFromCtx(ctx context.Context) (uuid.UUID, error) {
	msg := cqrs.OriginalMessageFromCtx(ctx)
	if msg == nil {
		return uuid.Nil, errors.New("no original message in ctx")
	}
	raw := msg.Metadata.Get("event_id")
	if raw == "" {
		return uuid.Nil, errors.New("event_id missing in message metadata")
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse event_id: %w", err)
	}
	return id, nil
}
