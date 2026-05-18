package messaging

import (
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-sql/v4/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewSQLSubscriber собирает watermill-sql subscriber с дефолтными для проекта
// схемой и offsets adapter'ом и сразу инициализирует таблицы для указанного топика.
func NewSQLSubscriber(
	pool *pgxpool.Pool,
	topic string,
	consumerGroup string,
	logger watermill.LoggerAdapter,
) (message.Subscriber, error) {
	sub, err := sql.NewSubscriber(
		sql.BeginnerFromPgx(pool),
		sql.SubscriberConfig{
			SchemaAdapter:  sql.DefaultPostgreSQLSchema{},
			OffsetsAdapter: sql.DefaultPostgreSQLOffsetsAdapter{},
			ConsumerGroup:  consumerGroup,
		},
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("create sql subscriber: %w", err)
	}
	if err := sub.SubscribeInitialize(topic); err != nil {
		return nil, fmt.Errorf("initialize subscriber for topic %s: %w", topic, err)
	}
	return sub, nil
}

// NewEventGroupProcessor создаёт CQRS EventGroupProcessor, в котором все
// зарегистрированные хендлеры группы шарят один subscriber и общий offset.
//
// Это структурный эквивалент текущей схемы worker.Run: один воркер = один топик +
// один consumer group, обрабатывает все типы событий из этого топика. Разница в
// том, что вместо ручного type switch'а в Handle хендлеры регистрируются
// типизированно через cqrs.NewGroupEventHandler[T].
func NewEventGroupProcessor(
	router *message.Router,
	pool *pgxpool.Pool,
	topic string,
	consumerGroup string,
	logger watermill.LoggerAdapter,
) (*cqrs.EventGroupProcessor, error) {
	return cqrs.NewEventGroupProcessorWithConfig(router, cqrs.EventGroupProcessorConfig{
		GenerateSubscribeTopic: func(cqrs.EventGroupProcessorGenerateSubscribeTopicParams) (string, error) {
			return topic, nil
		},
		SubscriberConstructor: func(cqrs.EventGroupProcessorSubscriberConstructorParams) (message.Subscriber, error) {
			return NewSQLSubscriber(pool, topic, consumerGroup, logger)
		},
		Marshaler:         EventEnvelopeMarshaler{},
		Logger:            logger,
		AckOnUnknownEvent: true,
	})
}
