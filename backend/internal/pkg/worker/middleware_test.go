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

// TestApplyStandardMiddleware_RecovererTurnsPanicIntoError проверяет, что паника
// в хендлере не убивает воркер: Recoverer ловит её, превращает в ошибку, дальше
// Retry организует повторные попытки.
func TestApplyStandardMiddleware_RecovererTurnsPanicIntoError(t *testing.T) {
	logger := watermill.NewSlogLogger(nil)
	pubsub := gochannel.NewGoChannel(gochannel.Config{Persistent: true}, logger)

	router, err := message.NewRouter(message.RouterConfig{}, logger)
	require.NoError(t, err)
	applyStandardMiddleware(router, fastTestConfig(2), logger)

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
	applyStandardMiddleware(router, fastTestConfig(2), logger)

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
