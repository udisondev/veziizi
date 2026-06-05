package sse

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"

	frevents "github.com/udisondev/veziizi/backend/internal/domain/freightrequest/events"
	notifevents "github.com/udisondev/veziizi/backend/internal/domain/notification/events"
	supportevents "github.com/udisondev/veziizi/backend/internal/domain/support/events"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/projections"
)

type fakeFreightLookup struct {
	request *projections.FreightRequestListItem
	offers  []projections.OfferListItem
	err     error
	calls   int
}

func (f *fakeFreightLookup) GetByID(context.Context, uuid.UUID) (*projections.FreightRequestListItem, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return f.request, nil
}

func (f *fakeFreightLookup) ListOffers(context.Context, ...projections.OfferFilterOption) ([]projections.OfferListItem, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return f.offers, nil
}

type fakeSupportLookup struct {
	ticket *projections.TicketListItem
	err    error
}

func (f *fakeSupportLookup) GetByID(context.Context, uuid.UUID) (*projections.TicketListItem, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.ticket, nil
}

func envelope(t *testing.T, aggregateType, eventType string, aggregateID uuid.UUID, payload any) eventstore.EventEnvelope {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return eventstore.EventEnvelope{
		ID:            uuid.New(),
		AggregateID:   aggregateID,
		AggregateType: aggregateType,
		EventType:     eventType,
		Payload:       raw,
	}
}

func TestRouteNotification(t *testing.T) {
	hub := NewHub(8, 100)
	member, org := uuid.New(), uuid.New()
	c := mustSubscribe(t, hub, member, org)

	r := NewRouter(hub, &fakeFreightLookup{}, &fakeSupportLookup{})
	notifID := uuid.New()

	r.Route(t.Context(), envelope(t, notifevents.AggregateType, notifevents.TypeInAppCreated, notifID,
		map[string]any{"member_id": member}))

	got := recvEvent(t, c)
	if got.Type != EventNotification || got.EntityID != notifID {
		t.Errorf("got %+v, want type=%s entity=%s", got, EventNotification, notifID)
	}

	r.Route(t.Context(), envelope(t, notifevents.AggregateType, notifevents.TypeInAppBatchRead, notifID,
		map[string]any{"member_id": member}))
	if got := recvEvent(t, c); got.Type != EventUnread {
		t.Errorf("got type %q, want %q", got.Type, EventUnread)
	}

	// Чужой member ничего не получает.
	r.Route(t.Context(), envelope(t, notifevents.AggregateType, notifevents.TypeInAppCreated, notifID,
		map[string]any{"member_id": uuid.New()}))
	select {
	case e := <-c.Events():
		t.Errorf("unexpected event %+v", e)
	default:
	}
}

func TestRouteFreightRequestMergesPayloadAndLookups(t *testing.T) {
	hub := NewHub(8, 100)
	customerOrg, carrierOrg, offerOrg := uuid.New(), uuid.New(), uuid.New()
	customer := mustSubscribe(t, hub, uuid.New(), customerOrg)
	carrier := mustSubscribe(t, hub, uuid.New(), carrierOrg)
	offerer := mustSubscribe(t, hub, uuid.New(), offerOrg)
	stranger := mustSubscribe(t, hub, uuid.New(), uuid.New())

	frID := uuid.New()
	freight := &fakeFreightLookup{
		request: &projections.FreightRequestListItem{ID: frID, CustomerOrgID: customerOrg},
		offers:  []projections.OfferListItem{{FreightRequestID: frID, CarrierOrgID: offerOrg}},
	}
	r := NewRouter(hub, freight, &fakeSupportLookup{})

	// carrier_org_id приходит из payload (offer.made), customer и второй
	// оффер — из lookup'ов.
	r.Route(t.Context(), envelope(t, frevents.AggregateType, frevents.TypeOfferMade, frID,
		map[string]any{"carrier_org_id": carrierOrg}))

	for name, c := range map[string]*Conn{"customer": customer, "carrier": carrier, "offerer": offerer} {
		got := recvEvent(t, c)
		if got.Type != EventFreightRequest || got.EntityID != frID {
			t.Errorf("%s got %+v, want type=%s entity=%s", name, got, EventFreightRequest, frID)
		}
	}
	select {
	case e := <-stranger.Events():
		t.Errorf("stranger unexpectedly got %+v", e)
	default:
	}
}

func TestRouteFreightRequestLookupErrorFallsBackToPayload(t *testing.T) {
	hub := NewHub(8, 100)
	customerOrg := uuid.New()
	customer := mustSubscribe(t, hub, uuid.New(), customerOrg)

	r := NewRouter(hub, &fakeFreightLookup{err: errors.New("projection lag")}, &fakeSupportLookup{})
	frID := uuid.New()

	// Проекция ещё не догнала created-событие — org id берётся из payload.
	r.Route(t.Context(), envelope(t, frevents.AggregateType, frevents.TypeFreightRequestCreated, frID,
		map[string]any{"customer_org_id": customerOrg}))

	if got := recvEvent(t, customer); got.EntityID != frID {
		t.Errorf("got %+v, want entity=%s", got, frID)
	}
}

func TestRouteSupportTicket(t *testing.T) {
	hub := NewHub(8, 100)
	owner, org := uuid.New(), uuid.New()
	c := mustSubscribe(t, hub, owner, org)

	ticketID := uuid.New()
	support := &fakeSupportLookup{ticket: &projections.TicketListItem{ID: ticketID, MemberID: owner}}
	r := NewRouter(hub, &fakeFreightLookup{}, support)

	// Сообщение админа — пушим владельцу.
	r.Route(t.Context(), envelope(t, supportevents.AggregateType, supportevents.TypeMessageAdded, ticketID,
		map[string]any{"sender_type": "admin"}))
	if got := recvEvent(t, c); got.Type != EventSupportTicket || got.EntityID != ticketID {
		t.Errorf("got %+v, want type=%s entity=%s", got, EventSupportTicket, ticketID)
	}

	// Сообщение самого member'а — не пушим.
	r.Route(t.Context(), envelope(t, supportevents.AggregateType, supportevents.TypeMessageAdded, ticketID,
		map[string]any{"sender_type": "member"}))
	select {
	case e := <-c.Events():
		t.Errorf("unexpected event %+v", e)
	default:
	}

	// Закрытие тикета — пушим.
	r.Route(t.Context(), envelope(t, supportevents.AggregateType, supportevents.TypeTicketClosed, ticketID,
		map[string]any{}))
	if got := recvEvent(t, c); got.EventType != supportevents.TypeTicketClosed {
		t.Errorf("got %+v, want event_type=%s", got, supportevents.TypeTicketClosed)
	}
}

func TestRouteSupportTicketLookupErrorDoesNotPanic(t *testing.T) {
	hub := NewHub(8, 100)
	// Подписчик нужен, чтобы пройти guard «нет клиентов — нет lookup'ов» и
	// реально дойти до ошибки lookup'а.
	mustSubscribe(t, hub, uuid.New(), uuid.New())
	r := NewRouter(hub, &fakeFreightLookup{}, &fakeSupportLookup{err: errors.New("boom")})

	r.Route(t.Context(), envelope(t, supportevents.AggregateType, supportevents.TypeTicketClosed, uuid.New(),
		map[string]any{}))
}

func TestRouteSkipsLookupsWhenNoClients(t *testing.T) {
	hub := NewHub(8, 100)
	freight := &fakeFreightLookup{err: errors.New("must not be called")}
	r := NewRouter(hub, freight, &fakeSupportLookup{})

	// Пустой хаб — Route выходит до lookup'ов (err-мок не должен дергаться;
	// если бы дернулся, это лишь slog.Debug, поэтому проверяем счётчиком).
	r.Route(t.Context(), envelope(t, frevents.AggregateType, frevents.TypeOfferMade, uuid.New(),
		map[string]any{}))
	if freight.calls != 0 {
		t.Errorf("lookups called %d times with empty hub, want 0", freight.calls)
	}
}

func TestEventEncode(t *testing.T) {
	id := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	got := string(Event{Type: EventNotification, EntityID: id, EventType: "notification.inapp_created"}.Encode())
	want := "event: notification\ndata: {\"entity_id\":\"11111111-2222-3333-4444-555555555555\",\"event_type\":\"notification.inapp_created\"}\n\n"
	if got != want {
		t.Errorf("Encode() = %q, want %q", got, want)
	}
}
