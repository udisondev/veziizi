// dlq-redrive переотправляет сообщения из deadletter-стрима обратно в их
// исходные топики. Запускать ПОСЛЕ устранения причины отравления (фикс
// хендлера, регистрация типа события и т.п.) — иначе сообщения снова уедут
// в DLQ после исчерпания ретраев.
//
// Топик-источник берётся из метаданных PoisonQueue middleware
// (middleware.PoisonedTopicKey). Сообщения без него ack'аются с warning'ом.
// Инструмент завершается сам, когда стрим иссяк (idle-таймаут).
package main

import (
	"context"
	"log/slog"
	"maps"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"

	"github.com/udisondev/veziizi/backend/internal/infrastructure/messaging"
	"github.com/udisondev/veziizi/backend/internal/pkg/config"
)

const (
	consumerGroup = "dlq_redrive"
	idleTimeout   = 5 * time.Second
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}
	if cfg.Worker.DeadLetterTopic == "" {
		slog.Error("WORKER_DEADLETTER_TOPIC is empty, nothing to redrive")
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	redisClient, err := messaging.NewRedisClient(cfg.Redis)
	if err != nil {
		slog.Error("failed to create redis client", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			slog.Error("failed to close redis client", slog.String("error", err.Error()))
		}
	}()

	wmLogger := watermill.NewSlogLogger(slog.Default())

	sub, err := messaging.NewRedisSubscriber(redisClient, consumerGroup, "dlq-redrive", cfg.Redis, wmLogger)
	if err != nil {
		slog.Error("failed to create subscriber", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Publisher пишет напрямую в Redis-стримы (как forwarder): redrive должен
	// работать и когда Postgres outbox-путь не при чём.
	pub, err := messaging.NewRedisPublisher(redisClient, cfg.Redis, wmLogger)
	if err != nil {
		slog.Error("failed to create publisher", slog.String("error", err.Error()))
		os.Exit(1)
	}

	msgs, err := sub.Subscribe(ctx, cfg.Worker.DeadLetterTopic)
	if err != nil {
		slog.Error("failed to subscribe to deadletter", slog.String("error", err.Error()))
		os.Exit(1)
	}

	var redriven, skipped int
	idle := time.NewTimer(idleTimeout)
	defer idle.Stop()

loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case <-idle.C:
			slog.Info("deadletter stream drained (idle timeout)")
			break loop
		case msg, ok := <-msgs:
			if !ok {
				break loop
			}
			idle.Reset(idleTimeout)

			originalTopic := msg.Metadata.Get(middleware.PoisonedTopicKey)
			if originalTopic == "" {
				slog.Warn("message without poisoned topic metadata, skipping",
					slog.String("uuid", msg.UUID))
				skipped++
				msg.Ack()
				continue
			}

			redrivenMsg := message.NewMessage(msg.UUID, msg.Payload)
			// Клонируем metadata: алиас мутировал бы оригинал — после Nack
			// redisstream ресендит тот же in-memory msg, и без PoisonedTopicKey
			// он ушёл бы в ветку "skipping" с Ack вместо повторного publish.
			redrivenMsg.Metadata = maps.Clone(msg.Metadata)
			// Снимаем poison-метки, чтобы redriven-сообщение не выглядело отравленным.
			delete(redrivenMsg.Metadata, middleware.PoisonedTopicKey)
			delete(redrivenMsg.Metadata, middleware.ReasonForPoisonedKey)
			delete(redrivenMsg.Metadata, middleware.PoisonedHandlerKey)
			delete(redrivenMsg.Metadata, middleware.PoisonedSubscriberKey)

			if err := pub.Publish(originalTopic, redrivenMsg); err != nil {
				slog.Error("failed to republish message, nacking",
					slog.String("uuid", msg.UUID),
					slog.String("topic", originalTopic),
					slog.String("error", err.Error()))
				msg.Nack()
				continue
			}

			slog.Info("message redriven",
				slog.String("uuid", msg.UUID),
				slog.String("topic", originalTopic))
			redriven++
			msg.Ack()
		}
	}

	slog.Info("dlq-redrive finished",
		slog.Int("redriven", redriven),
		slog.Int("skipped", skipped))
}
