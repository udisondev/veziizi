package messaging

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
	wmMiddleware "github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"github.com/udisondev/veziizi/backend/internal/pkg/httputil"
)

// EventPublisher — фасад поверх txAwarePublisher с CQRS-маршалером.
// Доменные сервисы зовут Publish(ctx, events...) и не знают ни про watermill,
// ни про tx-aware outbox, ни про marshaler.
//
// Топик каждого события выбирается из aggregateTopics по AggregateType.
// События одного топика отправляются ОДНИМ batch-вызовом publisher.Publish,
// не N отдельных round-trip'ов к БД — это важно для saveAndPublish'ей, которые
// одной транзакцией публикуют несколько событий аггрегата (MemberAdded +
// MemberRoleChanged + …). cqrs.EventBus не используется потому, что его
// Publish(ctx, event) — строго по одному событию.
//
// Marshaler, OnPublish и Generate-логика идентичны cqrs.EventBus — вынесены
// в helper'ы ниже, чтобы поведение оставалось «как у CQRS», но с batch'ингом.
type EventPublisher struct {
	bus       *cqrs.EventBus  // оставлен для Bus() — на случай, если потребуется raw cqrs API.
	txPub     *txAwarePublisher
	marshaler EventEnvelopeMarshaler
	logger    watermill.LoggerAdapter
}

func NewEventPublisher(pool *pgxpool.Pool, logger watermill.LoggerAdapter) (*EventPublisher, error) {
	txPub, err := newTxAwarePublisher(pool, logger)
	if err != nil {
		return nil, err
	}

	bus, err := cqrs.NewEventBusWithConfig(txPub, cqrs.EventBusConfig{
		Marshaler:            EventEnvelopeMarshaler{},
		GeneratePublishTopic: generatePublishTopic,
		OnPublish:            enrichFromContext,
		Logger:               logger,
	})
	if err != nil {
		return nil, fmt.Errorf("create event bus: %w", err)
	}

	return &EventPublisher{bus: bus, txPub: txPub, marshaler: EventEnvelopeMarshaler{}, logger: logger}, nil
}

// Publish маршалит все события, группирует по топику, и публикует каждую
// группу одним вызовом publisher.Publish — это даёт batch INSERT в
// watermill_messages_<topic> вместо N отдельных. Все события одной tx
// (если ctx содержит tx) попадут в БД атомарно: txAwarePublisher читает tx
// из msg.Context() и использует sql.TxFromPgx; при ошибке любого Publish
// возвращаем error, и dbtx.InTx делает Rollback всего набора.
func (p *EventPublisher) Publish(ctx context.Context, events ...eventstore.Event) error {
	if len(events) == 0 {
		return nil
	}

	byTopic := make(map[string][]*message.Message, len(events))
	for _, ev := range events {
		msg, err := p.marshaler.Marshal(ev)
		if err != nil {
			return fmt.Errorf("marshal %s: %w", ev.EventType(), err)
		}
		msg.SetContext(ctx)

		// Тот же enrichFromContext, что и в cqrs.EventBus OnPublish: audit-меты
		// из EventMeta + correlation_id в msg.Metadata. Держим в одном месте,
		// чтобы wire-формат CQRS-пути и batch-пути был идентичен.
		if err := enrichFromContext(cqrs.OnEventSendParams{
			EventName: p.marshaler.Name(ev),
			Event:     ev,
			Message:   msg,
		}); err != nil {
			return fmt.Errorf("enrich %s: %w", ev.EventType(), err)
		}

		topic, err := generatePublishTopic(cqrs.GenerateEventPublishTopicParams{
			EventName: p.marshaler.Name(ev),
			Event:     ev,
		})
		if err != nil {
			return err
		}
		byTopic[topic] = append(byTopic[topic], msg)
	}

	for topic, msgs := range byTopic {
		if err := p.txPub.Publish(topic, msgs...); err != nil {
			return fmt.Errorf("publish to %s: %w", topic, err)
		}
	}
	return nil
}

// Bus возвращает underlying cqrs.EventBus для редких случаев, когда нужен
// прямой доступ (например, тесты, которые хотят опубликовать non-domain event).
// В обычном коде используй Publish.
func (p *EventPublisher) Bus() *cqrs.EventBus {
	return p.bus
}

// RawPublisher возвращает sql-publisher на пуле в autocommit. Используется
// PoisonQueue middleware (DLQ всегда вне tx) и для non-event-store топиков
// типа notification.email — там нужен прямой message.Publisher с raw msg.
func (p *EventPublisher) RawPublisher() message.Publisher {
	return p.txPub.DefaultPublisher()
}

func (p *EventPublisher) Close() error {
	return p.txPub.Close()
}

// aggregateTopics — маппинг AggregateType -> watermill topic. Имена сложились
// исторически и не подчиняются единому шаблону (organization → organization.events,
// но support_ticket → support.events, freight_request → freightrequest.events).
// Менять AggregateType или имена топиков нельзя: первое сломает Load aggregate'ов
// из БД, второе — оставит ранее опубликованные сообщения в orphan-таблицах.
var aggregateTopics = map[string]string{
	"organization":    "organization.events",
	"freight_request": "freightrequest.events",
	"review":          "review.events",
	"support_ticket":  "support.events",
	"notification":    "notification.events",
}

func generatePublishTopic(p cqrs.GenerateEventPublishTopicParams) (string, error) {
	ev, ok := p.Event.(eventstore.Event)
	if !ok {
		return "", fmt.Errorf("messaging: %T does not implement eventstore.Event", p.Event)
	}
	topic, ok := aggregateTopics[ev.AggregateType()]
	if !ok {
		return "", fmt.Errorf("messaging: no topic mapping for aggregate type %q", ev.AggregateType())
	}
	return topic, nil
}

// enrichFromContext запускается cqrs.EventBus после Marshal, до самого Publish.
// msg.Context() уже содержит ctx, переданный в bus.Publish(ctx, ev). Здесь
// мы достаём из ctx audit-метаданные (actor, IP, fingerprint, correlation_id)
// и кладём их в msg.Metadata — где их прочитают watermill_messages.metadata
// колонкой и subscriber-side хендлеры через wmMiddleware.MessageCorrelationID.
func enrichFromContext(params cqrs.OnEventSendParams) error {
	meta, ok := httputil.EventMetaFromCtx(params.Message.Context())
	if !ok {
		return nil
	}
	for k, v := range meta.ToMap() {
		params.Message.Metadata.Set(k, v)
	}
	if meta.CorrelationID != "" {
		wmMiddleware.SetCorrelationID(meta.CorrelationID, params.Message)
	}
	return nil
}
