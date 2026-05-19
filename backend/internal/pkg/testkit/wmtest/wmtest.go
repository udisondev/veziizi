// Package wmtest даёт in-memory pipeline (gochannel + cqrs.EventGroupProcessor +
// EventEnvelopeMarshaler) для unit-тестов хендлеров.
//
// Использование:
//
//	pipe := wmtest.NewPipeline(t, "members-test",
//	    cqrs.NewGroupEventHandler(handler.OnMemberAdded),
//	    cqrs.NewGroupEventHandler(handler.OnMemberRemoved),
//	)
//	require.NoError(t, pipe.Publish(events.MemberAdded{...}))
//	// ассертим побочные эффекты handler'а (на mock'ах его зависимостей)
//
// Что покрывается:
//   - Сериализация/десериализация через тот же EventEnvelopeMarshaler, что и в продакшене.
//   - Диспатч по event_type на типизированные cqrs.GroupEventHandler'ы.
//   - Контракт handler'а: что он принимает event нужного типа и не падает на корректном payload.
//
// Что НЕ покрывается (важные расхождения с продакшеном):
//   - Backoff между retry-попытками: gochannel при Nack пересылает сообщение
//     немедленно, поэтому при handler-ошибке тест попадает в горячий цикл.
//     В продакшене Retry middleware даёт exponential backoff. Если хочешь
//     проверять retry-логику с backoff/DLQ — пиши тест в pkg/worker
//     (middleware_test.go) с реальным router'ом и middleware-stack'ом.
//   - Watermill_offsets, FOR UPDATE locking, гарантия порядка между репликами.
//   - Бизнес-логика хендлера с записью в БД — для этого e2e/setup.NewSuite(t).
package wmtest

import (
	"context"
	"testing"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/stretchr/testify/require"

	"github.com/udisondev/veziizi/backend/internal/infrastructure/messaging"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
)

// Pipeline — собранный в памяти конвейер: gochannel pubsub, router,
// EventGroupProcessor с настоящим EventEnvelopeMarshaler.
type Pipeline struct {
	t      *testing.T
	pubsub *gochannel.GoChannel
	topic  string
}

// NewPipeline поднимает in-memory CQRS pipeline и регистрирует хендлеры
// в указанной группе. Router запускается в фоне; завершение привязано к t.Cleanup.
//
// topic выбираем сами — для in-memory он не пишется в БД, просто ключ канала.
// Если хендлер чувствителен к msg.Context (например, читает correlation_id),
// pubsub сохраняет тот ctx, с которым вызвали Publish.
func NewPipeline(t *testing.T, groupName string, handlers ...cqrs.GroupEventHandler) *Pipeline {
	t.Helper()

	logger := watermill.NewSlogLogger(nil)
	pubsub := gochannel.NewGoChannel(gochannel.Config{Persistent: true}, logger)

	router, err := message.NewRouter(message.RouterConfig{}, logger)
	require.NoError(t, err)

	const topic = "wmtest.events"

	ep, err := cqrs.NewEventGroupProcessorWithConfig(router, cqrs.EventGroupProcessorConfig{
		GenerateSubscribeTopic: func(cqrs.EventGroupProcessorGenerateSubscribeTopicParams) (string, error) {
			return topic, nil
		},
		SubscriberConstructor: func(cqrs.EventGroupProcessorSubscriberConstructorParams) (message.Subscriber, error) {
			return pubsub, nil
		},
		Marshaler:         messaging.EventEnvelopeMarshaler{},
		Logger:            logger,
		AckOnUnknownEvent: true,
	})
	require.NoError(t, err)
	require.NoError(t, ep.AddHandlersGroup(groupName, handlers...))

	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		if err := router.Run(ctx); err != nil {
			t.Logf("router stopped: %v", err)
		}
	}()
	<-router.Running()

	t.Cleanup(func() {
		cancel()
		_ = router.Close()
		_ = pubsub.Close()
	})

	return &Pipeline{t: t, pubsub: pubsub, topic: topic}
}

// Publish сериализует событие через продакшен-маршалер и кидает его в канал.
// Если хендлер вернёт ошибку, gochannel сделает Nack и повторно доставит
// сообщение — это эквивалент behaviour'а sql-subscriber'а, но без backoff'а.
func (p *Pipeline) Publish(event eventstore.Event) error {
	p.t.Helper()
	m := messaging.EventEnvelopeMarshaler{}
	msg, err := m.Marshal(event)
	if err != nil {
		return err
	}
	return p.pubsub.Publish(p.topic, msg)
}
