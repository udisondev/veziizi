package messaging

import (
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/redis/go-redis/v9"

	"github.com/udisondev/veziizi/backend/internal/pkg/config"
)

// NewRedisClient создаёт go-redis клиент из REDIS_URL.
func NewRedisClient(cfg config.RedisConfig) (redis.UniversalClient, error) {
	opts, err := redis.ParseURL(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse REDIS_URL: %w", err)
	}
	return redis.NewClient(opts), nil
}

// NewRedisPublisher — publisher в Redis Streams. Используется ТОЛЬКО
// forwarder-воркером (перекладывает outbox → стримы) и PoisonQueue middleware
// (DLQ-стрим). Доменный код в Redis напрямую не публикует — все события и
// команды идут через Postgres outbox (см. EventPublisher), чтобы публикация
// оставалась атомарной с транзакцией event store.
func NewRedisPublisher(
	client redis.UniversalClient,
	cfg config.RedisConfig,
	logger watermill.LoggerAdapter,
) (message.Publisher, error) {
	pub, err := redisstream.NewPublisher(redisstream.PublisherConfig{
		Client:        client,
		DefaultMaxlen: cfg.MaxLen,
	}, logger)
	if err != nil {
		return nil, fmt.Errorf("create redis publisher: %w", err)
	}
	return pub, nil
}

// NewRedisSubscriber — consumer-group подписчик одного воркера на один стрим.
// В отличие от watermill-sql (один offset на группу под FOR UPDATE), Redis
// Streams раздаёт сообщения консьюмерам группы конкурентно — инстансы одного
// воркера делят поток между собой (горизонтальное масштабирование).
//
// consumerName обязан быть уникальным на инстанс: pending-список ведётся per
// consumer, и две реплики под одним именем будут невидимо красть сообщения
// друг у друга. Зависшие pending упавшего инстанса перехватываются живыми
// через XAUTOCLAIM (MaxIdleTime/ClaimInterval).
//
// OldestId "0" — новая consumer group читает стрим с начала, а не с "$":
// без этого события, опубликованные между созданием стрима forwarder'ом и
// первым стартом воркера, были бы потеряны.
func NewRedisSubscriber(
	client redis.UniversalClient,
	consumerGroup string,
	consumerName string,
	cfg config.RedisConfig,
	logger watermill.LoggerAdapter,
) (message.Subscriber, error) {
	if consumerName == "" {
		return nil, fmt.Errorf("redis subscriber for group %s: empty consumer name", consumerGroup)
	}
	sub, err := redisstream.NewSubscriber(redisstream.SubscriberConfig{
		Client:          client,
		ConsumerGroup:   consumerGroup,
		Consumer:        consumerName,
		MaxIdleTime:     cfg.MaxIdleTime,
		ClaimInterval:   cfg.ClaimInterval,
		NackResendSleep: cfg.NackResendSleep,
		BlockTime:       cfg.BlockTime,
		OldestId:        "0",
	}, logger)
	if err != nil {
		return nil, fmt.Errorf("create redis subscriber for group %s: %w", consumerGroup, err)
	}
	return sub, nil
}
