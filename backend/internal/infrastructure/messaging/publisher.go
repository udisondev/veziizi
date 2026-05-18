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

// EventPublisher — фасад поверх cqrs.EventBus. Доменные сервисы зовут
// Publish(ctx, events...) и не знают ни про watermill, ни про tx-aware
// механику outbox'а, ни про маршалер. Топик определяется автоматически
// из event.AggregateType().
//
// Внутри:
//   - cqrs.EventBus делает Marshal через EventEnvelopeMarshaler.
//   - GeneratePublishTopic кладёт events одного aggregate type в один топик
//     (organization → "organization.events" и т.д.).
//   - OnPublish обогащает msg.Metadata audit-полями и correlation_id из ctx.
//   - txAwarePublisher выбирает sql-publisher: tx или default — это и есть
//     outbox-семантика, события в watermill_messages_<topic> попадают в ту же
//     транзакцию, что и записи в event store.
type EventPublisher struct {
	bus    *cqrs.EventBus
	txPub  *txAwarePublisher
	logger watermill.LoggerAdapter
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

	return &EventPublisher{bus: bus, txPub: txPub, logger: logger}, nil
}

// Publish публикует одно или несколько событий через cqrs.EventBus. Топик
// каждого события определяется его AggregateType. Если в ctx есть транзакция
// (dbtx.WithTx) — все события публикуются в её рамках; иначе через autocommit
// publisher на пуле.
func (p *EventPublisher) Publish(ctx context.Context, events ...eventstore.Event) error {
	for _, ev := range events {
		if err := p.bus.Publish(ctx, ev); err != nil {
			return fmt.Errorf("publish %s: %w", ev.EventType(), err)
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
