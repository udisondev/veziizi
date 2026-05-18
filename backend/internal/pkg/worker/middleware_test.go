package worker

import (
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/udisondev/veziizi/backend/internal/pkg/config"
)

// fastTestConfig — короткие интервалы, чтобы тест не висел.
func fastTestConfig(maxRetries int) config.WorkerConfig {
	return config.WorkerConfig{
		RetryMaxRetries:      maxRetries,
		RetryInitialInterval: 5 * time.Millisecond,
		RetryMaxInterval:     20 * time.Millisecond,
		RetryMultiplier:      2,
	}
}

// fastTestConfigWithDLQ — то же что fastTestConfig, но включает PoisonQueue.
func fastTestConfigWithDLQ(maxRetries int, topic string) config.WorkerConfig {
	cfg := fastTestConfig(maxRetries)
	cfg.DeadLetterTopic = topic
	return cfg
}

// TestApplyStandardMiddleware_RecovererTurnsPanicIntoError проверяет, что паника
// в хендлере не убивает воркер: Recoverer ловит её, превращает в ошибку, дальше
// Retry организует повторные попытки.
func TestApplyStandardMiddleware_RecovererTurnsPanicIntoError(t *testing.T) {
	logger := watermill.NewSlogLogger(nil)
	pubsub := gochannel.NewGoChannel(gochannel.Config{Persistent: true}, logger)

	router, err := message.NewRouter(message.RouterConfig{}, logger)
	require.NoError(t, err)
	require.NoError(t, applyStandardMiddleware(router, fastTestConfig(2), logger, nil))

	var calls atomic.Int32
	router.AddConsumerHandler("panicking", "topic", pubsub, func(*message.Message) error {
		calls.Add(1)
		panic("boom")
	})

	ctx := t.Context()
	go func() { _ = router.Run(ctx) }()
	<-router.Running()

	require.NoError(t, pubsub.Publish("topic", message.NewMessage(uuid.NewString(), []byte(`{}`))))

	// MaxRetries=2 → всего 3 попытки (initial + 2 retries). Recoverer должен поймать
	// паники во всех трёх и не уронить router.
	require.Eventually(t, func() bool { return calls.Load() >= 3 }, time.Second, 5*time.Millisecond,
		"handler should be invoked at least 3 times")
}

// TestApplyStandardMiddleware_RetryBacksOffOnError проверяет, что между попытками
// действительно есть задержка (а не горячий цикл).
func TestApplyStandardMiddleware_RetryBacksOffOnError(t *testing.T) {
	logger := watermill.NewSlogLogger(nil)
	pubsub := gochannel.NewGoChannel(gochannel.Config{Persistent: true}, logger)

	router, err := message.NewRouter(message.RouterConfig{}, logger)
	require.NoError(t, err)
	require.NoError(t, applyStandardMiddleware(router, fastTestConfig(2), logger, nil))

	type call struct {
		at time.Time
	}
	callsCh := make(chan call, 8)

	router.AddConsumerHandler("failing", "topic", pubsub, func(*message.Message) error {
		callsCh <- call{at: time.Now()}
		return errors.New("nope")
	})

	ctx := t.Context()
	go func() { _ = router.Run(ctx) }()
	<-router.Running()

	require.NoError(t, pubsub.Publish("topic", message.NewMessage(uuid.NewString(), []byte(`{}`))))

	var calls []call
	deadline := time.After(time.Second)
	for len(calls) < 3 {
		select {
		case c := <-callsCh:
			calls = append(calls, c)
		case <-deadline:
			t.Fatalf("got only %d calls, expected at least 3", len(calls))
		}
	}

	gap := calls[1].at.Sub(calls[0].at)
	require.Greater(t, gap, 3*time.Millisecond,
		fmt.Sprintf("expected backoff between retries, got %s", gap))
}

// TestApplyStandardMiddleware_PoisonQueueAfterRetriesExhausted проверяет, что
// после исчерпания Retry-попыток ядовитое сообщение публикуется в DLQ-топик
// с metadata о причине и больше не повторяется.
func TestApplyStandardMiddleware_PoisonQueueAfterRetriesExhausted(t *testing.T) {
	logger := watermill.NewSlogLogger(nil)
	pubsub := gochannel.NewGoChannel(gochannel.Config{Persistent: true}, logger)

	// Subscribe сначала — gochannel доставляет только тем, кто подписан на момент Publish.
	dlqMessages, err := pubsub.Subscribe(t.Context(), "deadletter")
	require.NoError(t, err)

	router, err := message.NewRouter(message.RouterConfig{}, logger)
	require.NoError(t, err)
	require.NoError(t, applyStandardMiddleware(router, fastTestConfigWithDLQ(2, "deadletter"), logger, pubsub))

	var calls atomic.Int32
	router.AddConsumerHandler("always-fails", "topic", pubsub, func(*message.Message) error {
		calls.Add(1)
		return errors.New("permanent failure")
	})

	go func() { _ = router.Run(t.Context()) }()
	<-router.Running()

	msgID := uuid.NewString()
	require.NoError(t, pubsub.Publish("topic", message.NewMessage(msgID, []byte(`{"x":1}`))))

	// Ждём сообщение в DLQ.
	select {
	case dead := <-dlqMessages:
		require.Equal(t, msgID, dead.UUID, "DLQ должен сохранить оригинальный UUID")
		require.JSONEq(t, `{"x":1}`, string(dead.Payload), "payload не должен теряться")
		require.Equal(t, "permanent failure", dead.Metadata.Get("reason_poisoned"),
			"причина должна записаться в metadata")
		require.Equal(t, "topic", dead.Metadata.Get("topic_poisoned"))
		dead.Ack()
	case <-time.After(time.Second):
		t.Fatalf("expected message in DLQ, got %d handler calls", calls.Load())
	}

	// После Ack handler больше не должен вызываться — PoisonQueue обнулил err.
	beforeWait := calls.Load()
	time.Sleep(100 * time.Millisecond)
	require.Equal(t, beforeWait, calls.Load(), "после DLQ-публикации сообщение не должно retry'иться")
}
