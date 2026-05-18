package messaging

import (
	"encoding/json"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
)

// EventEnvelopeMarshaler адаптирует cqrs.CommandEventMarshaler к нашему формату
// EventEnvelope. Один экземпляр используется одновременно EventProcessor'ом (для
// распаковки входящих watermill-сообщений) и опционально EventBus'ом (для публикации
// через cqrs-API). Формат wire-сообщения 1:1 совпадает с тем, что пишет
// EventPublisher.Publish — поэтому новые подписчики могут читать топики,
// публикуемые старым кодом, и наоборот.
type EventEnvelopeMarshaler struct{}

// Marshal оборачивает доменное событие в EventEnvelope и кладёт ключевые поля
// в metadata watermill-сообщения, чтобы NameFromMessage не делал лишний Unmarshal.
func (m EventEnvelopeMarshaler) Marshal(v any) (*message.Message, error) {
	event, ok := v.(eventstore.Event)
	if !ok {
		return nil, fmt.Errorf("messaging: %T does not implement eventstore.Event", v)
	}

	envelope, err := eventstore.NewEventEnvelope(event, nil)
	if err != nil {
		return nil, fmt.Errorf("messaging: create envelope: %w", err)
	}

	payload, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("messaging: marshal envelope: %w", err)
	}

	msg := message.NewMessage(uuid.New().String(), payload)
	msg.Metadata.Set("aggregate_id", event.AggregateID().String())
	msg.Metadata.Set("aggregate_type", event.AggregateType())
	msg.Metadata.Set("event_type", event.EventType())
	return msg, nil
}

// Unmarshal извлекает payload из EventEnvelope и распаковывает его в указатель v
// (cqrs передаёт сюда new(T) — указатель на пустую структуру конкретного события).
func (m EventEnvelopeMarshaler) Unmarshal(msg *message.Message, v any) error {
	var envelope eventstore.EventEnvelope
	if err := json.Unmarshal(msg.Payload, &envelope); err != nil {
		return fmt.Errorf("messaging: unmarshal envelope: %w", err)
	}
	if err := json.Unmarshal(envelope.Payload, v); err != nil {
		return fmt.Errorf("messaging: unmarshal payload into %T: %w", v, err)
	}
	return nil
}

// Name возвращает event_type, вызывая EventType() на пустой структуре события.
// Все наши события реализуют EventType() с возвратом константы, не зависящей
// от значений полей, поэтому zero-value корректно даёт имя типа.
func (m EventEnvelopeMarshaler) Name(v any) string {
	event, ok := v.(eventstore.Event)
	if !ok {
		return ""
	}
	return event.EventType()
}

// NameFromMessage читает event_type из metadata без распаковки payload.
// EventPublisher.Publish и Marshal выше кладут это поле всегда.
func (m EventEnvelopeMarshaler) NameFromMessage(msg *message.Message) string {
	return msg.Metadata.Get("event_type")
}
