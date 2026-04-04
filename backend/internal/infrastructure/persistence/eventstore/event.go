package eventstore

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Event interface {
	AggregateID() uuid.UUID
	AggregateType() string
	EventType() string
	Version() int64
	OccurredAt() time.Time
}

type BaseEvent struct {
	AggregateIDField   uuid.UUID `json:"aggregate_id"`
	AggregateTypeField string    `json:"aggregate_type"`
	VersionField       int64     `json:"version"`
	OccurredAtField    time.Time `json:"occurred_at"`
}

func NewBaseEvent(aggregateID uuid.UUID, aggregateType string, version int64) BaseEvent {
	return BaseEvent{
		AggregateIDField:   aggregateID,
		AggregateTypeField: aggregateType,
		VersionField:       version,
		OccurredAtField:    time.Now().UTC(),
	}
}

func (e BaseEvent) AggregateID() uuid.UUID { return e.AggregateIDField }
func (e BaseEvent) AggregateType() string  { return e.AggregateTypeField }
func (e BaseEvent) Version() int64         { return e.VersionField }
func (e BaseEvent) OccurredAt() time.Time  { return e.OccurredAtField }

type EventEnvelope struct {
	ID            uuid.UUID         `json:"id"`
	AggregateID   uuid.UUID         `json:"aggregate_id"`
	AggregateType string            `json:"aggregate_type"`
	EventType     string            `json:"event_type"`
	Version       int64             `json:"version"`
	Payload       json.RawMessage   `json:"payload"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	OccurredAt    time.Time         `json:"occurred_at"`
}

func NewEventEnvelope(event Event, metadata map[string]string) (EventEnvelope, error) {
	payload, err := json.Marshal(event)
	if err != nil {
		return EventEnvelope{}, fmt.Errorf("failed to marshal event payload: %w", err)
	}

	return EventEnvelope{
		ID:            uuid.New(),
		AggregateID:   event.AggregateID(),
		AggregateType: event.AggregateType(),
		EventType:     event.EventType(),
		Version:       event.Version(),
		Payload:       payload,
		Metadata:      metadata,
		OccurredAt:    event.OccurredAt(),
	}, nil
}

type EventFactory func(data []byte) (Event, error)

var eventRegistry = make(map[string]EventFactory)

func RegisterEvent(eventType string, factory EventFactory) {
	eventRegistry[eventType] = factory
}

func RegisterEventType[T Event](eventType string) {
	RegisterEvent(eventType, func(data []byte) (Event, error) {
		var v T
		return v, json.Unmarshal(data, &v)
	})
}

func (e EventEnvelope) UnmarshalEvent() (Event, error) {
	factory, ok := eventRegistry[e.EventType]
	if !ok {
		return nil, fmt.Errorf("unknown event type: %s", e.EventType)
	}
	return factory(e.Payload)
}
