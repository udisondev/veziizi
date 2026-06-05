package wmtest_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/udisondev/veziizi/backend/internal/domain/organization/events"
	"github.com/udisondev/veziizi/backend/internal/domain/organization/values"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"github.com/udisondev/veziizi/backend/internal/pkg/testkit/wmtest"
)

// TestPipeline_RoutesByEventType — базовый smoke-test самого testkit'а: проверяет
// что pipeline корректно диспатчит разные типы событий в разные handler'ы.
// Этот же паттерн пользователь копирует для своих handler'ов.
func TestPipeline_RoutesByEventType(t *testing.T) {
	addedCh := make(chan *events.MemberAdded, 4)
	removedCh := make(chan *events.MemberRemoved, 4)

	pipe := wmtest.NewPipeline(t, "test-group",
		cqrs.NewGroupEventHandler(func(_ context.Context, e *events.MemberAdded) error {
			addedCh <- e
			return nil
		}),
		cqrs.NewGroupEventHandler(func(_ context.Context, e *events.MemberRemoved) error {
			removedCh <- e
			return nil
		}),
	)

	orgID := uuid.New()
	memberID := uuid.New()

	require.NoError(t, pipe.Publish(events.MemberAdded{
		BaseEvent: eventstore.NewBaseEvent(orgID, events.AggregateType, 1),
		MemberID:  memberID,
		Email:     "alice@example.com",
		Role:      values.MemberRoleOwner,
	}))
	require.NoError(t, pipe.Publish(events.MemberRemoved{
		BaseEvent: eventstore.NewBaseEvent(orgID, events.AggregateType, 2),
		MemberID:  memberID,
	}))

	select {
	case got := <-addedCh:
		require.Equal(t, memberID, got.MemberID)
		require.Equal(t, "alice@example.com", got.Email)
	case <-time.After(time.Second):
		t.Fatal("MemberAdded handler timed out")
	}
	select {
	case got := <-removedCh:
		require.Equal(t, memberID, got.MemberID)
	case <-time.After(time.Second):
		t.Fatal("MemberRemoved handler timed out")
	}
}

// TestPipeline_HandlerErrorTriggersRetry — gochannel при handler-error делает
// немедленный resend (без backoff). Это означает: testkit видит retry, но не
// видит backoff из middleware.Retry — тот живёт в реальном router'е воркера.
//
// Поведенческое следствие для пользователей testkit: можно проверить, что
// handler возвращает error на «плохом» событии, но проверка стратегии retry/DLQ
// должна идти в pkg/worker/middleware_test.go.
func TestPipeline_HandlerErrorTriggersRetry(t *testing.T) {
	var calls atomic.Int32
	done := make(chan struct{})

	pipe := wmtest.NewPipeline(t, "retry-group",
		cqrs.NewGroupEventHandler(func(_ context.Context, _ *events.MemberAdded) error {
			n := calls.Add(1)
			if n < 3 {
				return errors.New("transient")
			}
			select {
			case <-done:
			default:
				close(done)
			}
			return nil
		}),
	)

	require.NoError(t, pipe.Publish(events.MemberAdded{
		BaseEvent: eventstore.NewBaseEvent(uuid.New(), events.AggregateType, 1),
		MemberID:  uuid.New(),
		Role:      values.MemberRoleOwner, // обязательно: enum-валидация в Unmarshal
	}))

	select {
	case <-done:
		require.GreaterOrEqual(t, calls.Load(), int32(3),
			"handler должен был быть вызван минимум 3 раза до успеха")
	case <-time.After(2 * time.Second):
		t.Fatalf("retry не отработал, calls=%d", calls.Load())
	}
}
