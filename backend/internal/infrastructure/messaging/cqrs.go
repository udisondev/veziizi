package messaging

import (
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-sql/v4/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/udisondev/veziizi/backend/internal/pkg/config"
)

// NewSQLSubscriber собирает watermill-sql subscriber с дефолтными для проекта
// схемой и offsets adapter'ом и сразу инициализирует таблицы для указанного топика.
//
// После переезда воркеров на Redis Streams единственный потребитель этой
// функции — forwarder-воркер: он читает Postgres outbox-топик (OutboxTopic)
// и перекладывает сообщения в Redis. SQL-подписка с одним offset'ом на группу
// под FOR UPDATE здесь уместна: forwarder работает в одном инстансе.
// pollInterval — период опроса outbox-таблицы. Дефолт watermill-sql (1s)
// добавлял бы секунду базовой латентности всему событийному пайплайну.
func NewSQLSubscriber(
	pool *pgxpool.Pool,
	topic string,
	consumerGroup string,
	pollInterval time.Duration,
	logger watermill.LoggerAdapter,
) (message.Subscriber, error) {
	sub, err := sql.NewSubscriber(
		sql.BeginnerFromPgx(pool),
		sql.SubscriberConfig{
			SchemaAdapter:  sql.DefaultPostgreSQLSchema{},
			OffsetsAdapter: sql.DefaultPostgreSQLOffsetsAdapter{},
			ConsumerGroup:  consumerGroup,
			PollInterval:   pollInterval,
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
// зарегистрированные хендлеры группы шарят один Redis-подписчик (consumer group).
//
// Один воркер = один Redis-стрим + один consumer group, обрабатывает все типы
// событий из этого стрима (AckOnUnknownEvent ack'ает чужие). В отличие от
// прежней SQL-подписки, инстансы одного воркера (уникальные consumerName)
// делят поток между собой — воркер масштабируется горизонтально.
//
// marshaler — формат wire-сообщений; обычно EventEnvelopeMarshaler для доменных
// событий, либо NotificationMarshaler() (JSONMarshaler) для команд на каналы
// уведомлений.
func NewEventGroupProcessor(
	router *message.Router,
	redisClient redis.UniversalClient,
	redisCfg config.RedisConfig,
	topic string,
	consumerGroup string,
	consumerName string,
	marshaler cqrs.CommandEventMarshaler,
	logger watermill.LoggerAdapter,
) (*cqrs.EventGroupProcessor, error) {
	return cqrs.NewEventGroupProcessorWithConfig(router, cqrs.EventGroupProcessorConfig{
		GenerateSubscribeTopic: func(cqrs.EventGroupProcessorGenerateSubscribeTopicParams) (string, error) {
			return topic, nil
		},
		SubscriberConstructor: func(cqrs.EventGroupProcessorSubscriberConstructorParams) (message.Subscriber, error) {
			return NewRedisSubscriber(redisClient, consumerGroup, consumerName, redisCfg, logger)
		},
		Marshaler:         marshaler,
		Logger:            logger,
		AckOnUnknownEvent: true,
	})
}
