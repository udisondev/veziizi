package messaging_test

import (
	"context"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/udisondev/veziizi/backend/internal/domain/organization/events"
	"github.com/udisondev/veziizi/backend/internal/domain/organization/values"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/messaging"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
)

func TestEventEnvelopeMarshaler_RoundTrip(t *testing.T) {
	m := messaging.EventEnvelopeMarshaler{}

	orgID := uuid.New()
	memberID := uuid.New()
	original := events.MemberAdded{
		BaseEvent: eventstore.NewBaseEvent(orgID, events.AggregateType, 1),
		MemberID:  memberID,
		Email:     "test@example.com",
		Name:      "Test Member",
		Role:      values.MemberRoleOwner,
	}

	msg, err := m.Marshal(original)
	require.NoError(t, err)
	require.Equal(t, events.TypeMemberAdded, msg.Metadata.Get("event_type"))
	require.Equal(t, orgID.String(), msg.Metadata.Get("aggregate_id"))
	require.Equal(t, events.AggregateType, msg.Metadata.Get("aggregate_type"))
	require.Equal(t, events.TypeMemberAdded, m.NameFromMessage(msg))

	decoded := &events.MemberAdded{}
	require.NoError(t, m.Unmarshal(msg, decoded))
	require.Equal(t, original.MemberID, decoded.MemberID)
	require.Equal(t, original.Email, decoded.Email)
	require.Equal(t, original.Name, decoded.Name)
	require.Equal(t, original.AggregateID(), decoded.AggregateID())
	require.Equal(t, original.AggregateType(), decoded.AggregateType())
	require.Equal(t, original.Version(), decoded.Version())
}

// TestEventEnvelopeMarshaler_NameOnZeroValue фиксирует контракт, на котором
// держится cqrs.NewGroupEventHandler: имя события должно возвращаться корректно
// от пустой структуры (cqrs дёргает Name(new(T)) до того, как пришло сообщение).
func TestEventEnvelopeMarshaler_NameOnZeroValue(t *testing.T) {
	m := messaging.EventEnvelopeMarshaler{}
	require.Equal(t, events.TypeMemberAdded, m.Name(&events.MemberAdded{}))
	require.Equal(t, events.TypeMemberRemoved, m.Name(&events.MemberRemoved{}))
	require.Equal(t, events.TypeMemberRoleChanged, m.Name(&events.MemberRoleChanged{}))
}

// TestEventGroupProcessor_DispatchesByType — интеграционный тест через gochannel:
// публикуем три разных события, проверяем, что каждый типизированный хендлер
// видит только своё и получает корректно распакованную структуру.
func TestEventGroupProcessor_DispatchesByType(t *testing.T) {
	logger := watermill.NewSlogLogger(nil)
	pubsub := gochannel.NewGoChannel(gochannel.Config{Persistent: true}, logger)

	router, err := message.NewRouter(message.RouterConfig{}, logger)
	require.NoError(t, err)

	addedCh := make(chan *events.MemberAdded, 4)
	removedCh := make(chan *events.MemberRemoved, 4)

	ep, err := cqrs.NewEventGroupProcessorWithConfig(router, cqrs.EventGroupProcessorConfig{
		GenerateSubscribeTopic: func(cqrs.EventGroupProcessorGenerateSubscribeTopicParams) (string, error) {
			return messaging.TopicOrganizationEvents, nil
		},
		SubscriberConstructor: func(cqrs.EventGroupProcessorSubscriberConstructorParams) (message.Subscriber, error) {
			return pubsub, nil
		},
		Marshaler:         messaging.EventEnvelopeMarshaler{},
		Logger:            logger,
		AckOnUnknownEvent: true,
	})
	require.NoError(t, err)

	require.NoError(t, ep.AddHandlersGroup("members_projection",
		cqrs.NewGroupEventHandler(func(_ context.Context, e *events.MemberAdded) error {
			addedCh <- e
			return nil
		}),
		cqrs.NewGroupEventHandler(func(_ context.Context, e *events.MemberRemoved) error {
			removedCh <- e
			return nil
		}),
	))

	ctx := t.Context()
	go func() {
		if err := router.Run(ctx); err != nil {
			t.Logf("router stopped: %v", err)
		}
	}()
	<-router.Running()

	orgID := uuid.New()
	memberID := uuid.New()
	m := messaging.EventEnvelopeMarshaler{}

	addedMsg, err := m.Marshal(events.MemberAdded{
		BaseEvent: eventstore.NewBaseEvent(orgID, events.AggregateType, 1),
		MemberID:  memberID,
		Email:     "alice@example.com",
		Name:      "Alice",
		Role:      values.MemberRoleOwner,
	})
	require.NoError(t, err)
	removedMsg, err := m.Marshal(events.MemberRemoved{
		BaseEvent: eventstore.NewBaseEvent(orgID, events.AggregateType, 2),
		MemberID:  memberID,
	})
	require.NoError(t, err)

	// Также публикуем сообщение неизвестного типа — оно должно быть ack'нуто
	// (AckOnUnknownEvent: true) и не уйти ни в один из каналов.
	unknownMsg := message.NewMessage(uuid.New().String(), []byte(`{}`))
	unknownMsg.Metadata.Set("event_type", "something.irrelevant")

	require.NoError(t, pubsub.Publish(messaging.TopicOrganizationEvents, addedMsg, removedMsg, unknownMsg))

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

	// Перекрёстных доставок быть не должно.
	require.Empty(t, addedCh)
	require.Empty(t, removedCh)
}
