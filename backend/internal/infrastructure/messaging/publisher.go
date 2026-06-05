package messaging

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/components/forwarder"
	"github.com/ThreeDotsLabs/watermill/message"
	wmMiddleware "github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"github.com/udisondev/veziizi/backend/internal/pkg/httputil"
)

// OutboxTopic — единственный Postgres-топик, в который пишутся ВСЕ исходящие
// сообщения (доменные события и команды уведомлений), завёрнутые в
// forwarder-envelope с целевым топиком внутри. Forwarder-воркер — единственный
// его потребитель: разворачивает envelope и публикует в Redis-стрим
// DestinationTopic. Один топик = один последовательный читатель = порядок
// публикации сохраняется при перекладке в Redis.
const OutboxTopic = "events_to_forward"

// EventPublisher — фасад поверх cqrs.EventBus (канонический publish-путь
// watermill, как в threedotlabs/edriven: продюсеры публикуют через EventBus,
// а не через message.Publisher напрямую). Доменные сервисы зовут
// Publish(ctx, events...) и не знают ни про watermill, ни про tx-aware outbox,
// ни про marshaler — всё это конфигурация EventBus:
//
//	Marshaler            — EventEnvelopeMarshaler (wire-формат доменных событий)
//	GeneratePublishTopic — aggregateTopics по AggregateType
//	OnPublish            — enrichFromContext (audit-меты + correlation_id)
//
// EventBus.Publish делает msg.SetContext(ctx) ДО publisher.Publish (проверено
// в watermill v1.5.x), поэтому tx-aware семантика txAwarePublisher сохраняется:
// события попадают в outbox атомарно с транзакцией event store.
//
// Каждое событие — отдельный INSERT в outbox (EventBus публикует по одному).
// Прежний hand-rolled batch по топикам экономил round-trip'ы при
// multi-event saveAndPublish, но дублировал весь пайплайн EventBus руками;
// при типичных 1-3 событиях на транзакцию выигрыш не стоил второй
// параллельной реализации publish-пути.
type EventPublisher struct {
	bus   *cqrs.EventBus
	txPub *txAwarePublisher
	pub   message.Publisher // forwarder.Publisher поверх txPub: envelope + публикация в OutboxTopic
}

func NewEventPublisher(pool *pgxpool.Pool, logger watermill.LoggerAdapter) (*EventPublisher, error) {
	txPub, err := newTxAwarePublisher(pool, logger)
	if err != nil {
		return nil, err
	}

	// forwarder.Publisher заворачивает каждое сообщение в envelope с
	// DestinationTopic=topic и публикует его в OutboxTopic через нижележащий
	// txPub. Ключевая деталь (проверено в watermill v1.5.1):
	// wrapMessageInEnvelope переносит msg.Context() в обёрнутое сообщение,
	// поэтому tx из контекста доезжает до txAwarePublisher.
	fwdPub := forwarder.NewPublisher(txPub, forwarder.PublisherConfig{
		ForwarderTopic: OutboxTopic,
	})

	bus, err := cqrs.NewEventBusWithConfig(fwdPub, cqrs.EventBusConfig{
		Marshaler:            EventEnvelopeMarshaler{},
		GeneratePublishTopic: generatePublishTopic,
		OnPublish:            enrichFromContext,
		Logger:               logger,
	})
	if err != nil {
		return nil, fmt.Errorf("create event bus: %w", err)
	}

	return &EventPublisher{bus: bus, txPub: txPub, pub: fwdPub}, nil
}

// Publish публикует доменные события через cqrs.EventBus. Все события одной tx
// (если ctx содержит tx) попадут в БД атомарно: txAwarePublisher читает tx из
// msg.Context() и использует sql.TxFromPgx; при ошибке любого Publish
// возвращаем error, и dbtx.InTx делает Rollback всего набора.
func (p *EventPublisher) Publish(ctx context.Context, events ...eventstore.Event) error {
	for _, ev := range events {
		if err := p.bus.Publish(ctx, ev); err != nil {
			return fmt.Errorf("publish %s: %w", ev.EventType(), err)
		}
	}
	return nil
}

// ForwarderPublisher возвращает forwarder-обёрнутый tx-aware publisher.
// Любая публикация через него уезжает в Postgres outbox (envelope с целевым
// топиком) и доставляется forwarder-воркером в Redis. Используется
// NotificationBus: команды уведомлений идут тем же outbox-путём, что и
// доменные события, — это даёт атомарность «обработал событие + поставил
// команду» при публикации внутри tx (dedupGuard в dispatcher'е).
func (p *EventPublisher) ForwarderPublisher() message.Publisher {
	return p.pub
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
	"organization":    TopicOrganizationEvents,
	"freight_request": TopicFreightRequestEvents,
	"review":          TopicReviewEvents,
	"support_ticket":  TopicSupportEvents,
	"notification":    TopicNotificationEvents,
}

// RedisStreamTopics возвращает все Redis-стримы, в которые forwarder публикует
// сообщения (доменные топики + каналы уведомлений). Используется метриками
// forwarder'а: лаг consumer group'ов считается по известному списку стримов.
func RedisStreamTopics() []string {
	seen := make(map[string]struct{}, len(aggregateTopics)+len(notificationTopics))
	topics := make([]string, 0, len(aggregateTopics)+len(notificationTopics))
	for _, t := range aggregateTopics {
		if _, ok := seen[t]; !ok {
			seen[t] = struct{}{}
			topics = append(topics, t)
		}
	}
	for _, t := range notificationTopics {
		if _, ok := seen[t]; !ok {
			seen[t] = struct{}{}
			topics = append(topics, t)
		}
	}
	return topics
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

// enrichFromContext — OnPublish-хук cqrs.EventBus: запускается после Marshal,
// до самого Publish. msg.Context() уже содержит ctx, переданный в
// bus.Publish(ctx, ev). Здесь мы достаём из ctx audit-метаданные (actor, IP,
// fingerprint, correlation_id) и кладём их в msg.Metadata — где их прочитают
// watermill_messages.metadata колонкой и subscriber-side хендлеры через
// wmMiddleware.MessageCorrelationID.
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
