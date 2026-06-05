package sse

import (
	"testing"

	"github.com/google/uuid"
)

func mustSubscribe(t *testing.T, h *Hub, memberID, orgID uuid.UUID) *Conn {
	t.Helper()
	c, err := h.Subscribe(memberID, orgID)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	return c
}

func recvEvent(t *testing.T, c *Conn) Event {
	t.Helper()
	select {
	case e := <-c.Events():
		return e
	default:
		t.Fatal("expected event, channel is empty")
		return Event{}
	}
}

func TestHubPublishToMember(t *testing.T) {
	h := NewHub(8, 100)
	member1, member2 := uuid.New(), uuid.New()
	org := uuid.New()

	c1 := mustSubscribe(t, h, member1, org)
	c1b := mustSubscribe(t, h, member1, org)
	c2 := mustSubscribe(t, h, member2, org)

	e := Event{Type: EventNotification, EntityID: uuid.New(), EventType: "notification.inapp_created"}
	h.PublishToMember(member1, e)

	if got := recvEvent(t, c1); got != e {
		t.Errorf("c1 got %+v, want %+v", got, e)
	}
	if got := recvEvent(t, c1b); got != e {
		t.Errorf("c1b got %+v, want %+v", got, e)
	}
	select {
	case got := <-c2.Events():
		t.Errorf("c2 unexpectedly got %+v", got)
	default:
	}
}

func TestHubPublishToOrg(t *testing.T) {
	h := NewHub(8, 100)
	org1, org2 := uuid.New(), uuid.New()

	c1 := mustSubscribe(t, h, uuid.New(), org1)
	c2 := mustSubscribe(t, h, uuid.New(), org1)
	c3 := mustSubscribe(t, h, uuid.New(), org2)

	e := Event{Type: EventFreightRequest, EntityID: uuid.New(), EventType: "offer.made"}
	h.PublishToOrg(org1, e)

	if got := recvEvent(t, c1); got != e {
		t.Errorf("c1 got %+v, want %+v", got, e)
	}
	if got := recvEvent(t, c2); got != e {
		t.Errorf("c2 got %+v, want %+v", got, e)
	}
	select {
	case got := <-c3.Events():
		t.Errorf("c3 unexpectedly got %+v", got)
	default:
	}
}

func TestHubSlowClientClosed(t *testing.T) {
	h := NewHub(8, 100)
	member, org := uuid.New(), uuid.New()
	c := mustSubscribe(t, h, member, org)

	e := Event{Type: EventUnread}
	for range connBufferSize {
		h.PublishToMember(member, e)
	}
	select {
	case <-c.Done():
		t.Fatal("conn closed before buffer overflow")
	default:
	}

	// Переполнение буфера — соединение закрывается, событие дропается.
	h.PublishToMember(member, e)
	select {
	case <-c.Done():
	default:
		t.Fatal("conn not closed after buffer overflow")
	}
}

func TestHubLimits(t *testing.T) {
	h := NewHub(2, 3)
	member, org := uuid.New(), uuid.New()

	mustSubscribe(t, h, member, org)
	mustSubscribe(t, h, member, org)
	if _, err := h.Subscribe(member, org); err != ErrTooManyConnections {
		t.Errorf("per-member limit: got %v, want ErrTooManyConnections", err)
	}

	mustSubscribe(t, h, uuid.New(), org)
	if _, err := h.Subscribe(uuid.New(), org); err != ErrTooManyConnections {
		t.Errorf("total limit: got %v, want ErrTooManyConnections", err)
	}
}

func TestHubUnsubscribe(t *testing.T) {
	h := NewHub(8, 100)
	member, org := uuid.New(), uuid.New()

	c := mustSubscribe(t, h, member, org)
	h.Unsubscribe(c)

	if h.Len() != 0 {
		t.Errorf("Len after Unsubscribe = %d, want 0", h.Len())
	}
	select {
	case <-c.Done():
	default:
		t.Error("conn not closed after Unsubscribe")
	}

	// Повторный Unsubscribe (хендлер + overflow) не паникует и не ломает счётчик.
	h.Unsubscribe(c)
	if h.Len() != 0 {
		t.Errorf("Len after double Unsubscribe = %d, want 0", h.Len())
	}

	h.PublishToMember(member, Event{Type: EventUnread}) // не паникует
}

func TestHubShutdown(t *testing.T) {
	h := NewHub(8, 100)
	c1 := mustSubscribe(t, h, uuid.New(), uuid.New())
	c2 := mustSubscribe(t, h, uuid.New(), uuid.New())

	h.Shutdown()

	for i, c := range []*Conn{c1, c2} {
		select {
		case <-c.Done():
		default:
			t.Errorf("conn %d not closed after Shutdown", i+1)
		}
	}

	if _, err := h.Subscribe(uuid.New(), uuid.New()); err != ErrShuttingDown {
		t.Errorf("Subscribe after Shutdown: got %v, want ErrShuttingDown", err)
	}
}
