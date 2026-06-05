package tests

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/udisondev/veziizi/backend/e2e/client"
	"github.com/udisondev/veziizi/backend/e2e/fixtures"
	"github.com/udisondev/veziizi/backend/e2e/setup"
)

// SSESuite покрывает SSE-шлюз целиком: HTTP → event store → outbox →
// forwarder → Redis Streams → sse.Tailer (XREAD без consumer group) →
// sse.Hub → text/event-stream клиенту. Плюс публикацию InAppCreated /
// InAppBatchRead из NotificationService (без неё стрим notification.events
// пуст и SSE-уведомлений не существует).
type SSESuite struct {
	suite.Suite
	ts  *setup.Suite
	ctx *fixtures.TestContext
}

func TestSSESuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(SSESuite))
}

func (s *SSESuite) SetupSuite() {
	s.ts = getSuite(s.T())
	s.ctx = fixtures.NewTestContext(s.T(), s.ts.BaseURL)
}

// sseEvent — распарсенное событие text/event-stream.
type sseEvent struct {
	Name string
	Data struct {
		EntityID  uuid.UUID `json:"entity_id"`
		EventType string    `json:"event_type"`
	}
}

// sseStream — живое SSE-соединение тестового клиента. Полученные события
// буферизуются: порядок доставки между стримами не гарантирован (например,
// freight_request с offer.made обгоняет notification от диспетчера), поэтому
// waitFor не имеет права выбрасывать «не свои» события — их ждёт следующий
// waitFor.
type sseStream struct {
	events <-chan sseEvent
	cancel context.CancelFunc

	mu  sync.Mutex
	buf []sseEvent
}

func (st *sseStream) Close() { st.cancel() }

// takeBuffered забирает первое подходящее событие из буфера.
func (st *sseStream) takeBuffered(name string, entityID *uuid.UUID) (sseEvent, bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	for i, e := range st.buf {
		if e.Name != name {
			continue
		}
		if entityID != nil && e.Data.EntityID != *entityID {
			continue
		}
		st.buf = append(st.buf[:i], st.buf[i+1:]...)
		return e, true
	}
	return sseEvent{}, false
}

// waitFor ждёт событие с именем name (опционально — с конкретным entity_id);
// остальные события складываются в буфер для последующих waitFor.
func (st *sseStream) waitFor(t *testing.T, name string, entityID *uuid.UUID, timeout time.Duration) sseEvent {
	t.Helper()
	deadline := time.After(timeout)
	for {
		if e, ok := st.takeBuffered(name, entityID); ok {
			return e
		}
		select {
		case e, ok := <-st.events:
			if !ok {
				t.Fatalf("SSE stream closed while waiting for %q", name)
			}
			st.mu.Lock()
			st.buf = append(st.buf, e)
			st.mu.Unlock()
		case <-deadline:
			t.Fatalf("timed out waiting for SSE event %q", name)
		}
	}
}

// openStream открывает /api/v1/events/stream с cookie-сессией клиента c и
// возвращается только после первой строки потока (`retry:`) — она пишется
// после hub.Subscribe, т.е. соединение зарегистрировано до того, как тест
// начнёт генерировать события.
func (s *SSESuite) openStream(c *client.Client) *sseStream {
	s.T().Helper()

	ctx, cancel := context.WithCancel(context.Background())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.ts.BaseURL+"/api/v1/events/stream", nil)
	s.Require().NoError(err)

	// Отдельный http.Client без Timeout (клиентский 10s убил бы стрим), но с
	// тем же cookie jar — сессия залогиненного клиента.
	httpClient := &http.Client{Jar: c.HTTPClient.Jar}
	resp, err := httpClient.Do(req) //nolint:bodyclose // закрывается в reader-горутине ниже
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode)
	s.Require().Equal("text/event-stream", resp.Header.Get("Content-Type"))

	events := make(chan sseEvent, 64)
	opened := make(chan struct{})

	go func() {
		defer close(events)
		defer func() { _ = resp.Body.Close() }()

		var (
			cur        sseEvent
			hasData    bool
			openedOnce bool
		)
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			// Первая же строка (`retry: 3000`) пишется после hub.Subscribe —
			// значит соединение зарегистрировано и события не потеряются.
			if !openedOnce && line != "" {
				openedOnce = true
				close(opened)
			}
			switch {
			case strings.HasPrefix(line, "event: "):
				cur.Name = strings.TrimPrefix(line, "event: ")
			case strings.HasPrefix(line, "data: "):
				if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &cur.Data); err == nil {
					hasData = true
				}
			case line == "": // конец события
				// ping (heartbeat) не буферизуем — не интересен waitFor'ам.
				if cur.Name != "" && cur.Name != "ping" && hasData {
					events <- cur
				}
				cur, hasData = sseEvent{}, false
			}
		}
	}()

	select {
	case <-opened:
	case <-time.After(5 * time.Second):
		cancel()
		s.T().Fatal("SSE stream did not deliver first heartbeat")
	}

	st := &sseStream{events: events, cancel: cancel}
	s.T().Cleanup(st.Close)
	return st
}

// TestSSE001_RequiresAuth: стрим без сессии закрыт авторизацией.
func (s *SSESuite) TestSSE001_RequiresAuth() {
	httpClient := &http.Client{Timeout: 5 * time.Second}
	resp, err := httpClient.Get(s.ts.BaseURL + "/api/v1/events/stream")
	s.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()
	s.Assert().Equal(http.StatusUnauthorized, resp.StatusCode)
}

// TestSSE002_OfferMadePushesEvents: оффер перевозчика доезжает до заказчика
// как notification-пинок (notification-dispatcher → InAppCreated →
// notification.events) и до обеих сторон как freight_request-пинок
// (freightrequest.events, роутинг по организациям через lookup'ы).
func (s *SSESuite) TestSSE002_OfferMadePushesEvents() {
	fr := fixtures.NewFreightRequest(s.T(), s.ctx.Customer.Client).Create()
	// Роутинг и правило OfferMade читают freight_requests_lookup — дожидаемся
	// проекции до создания оффера.
	s.ts.Sync()

	customerStream := s.openStream(s.ctx.Customer.Client)
	carrierStream := s.openStream(s.ctx.Carrier.Client)

	fixtures.NewOffer(s.T(), s.ctx.Carrier.Client, fr.ID).Create()

	const timeout = 10 * time.Second
	notif := customerStream.waitFor(s.T(), "notification", nil, timeout)
	s.Assert().Equal("notification.inapp_created", notif.Data.EventType)

	frEvent := customerStream.waitFor(s.T(), "freight_request", &fr.ID, timeout)
	s.Assert().Equal("offer.made", frEvent.Data.EventType)

	carrierStream.waitFor(s.T(), "freight_request", &fr.ID, timeout)
}

// TestSSE003_MarkAllReadPushesUnread: read-all публикует InAppBatchRead, и
// другие вкладки member'а получают unread-пинок для пересчёта счётчика.
func (s *SSESuite) TestSSE003_MarkAllReadPushesUnread() {
	stream := s.openStream(s.ctx.Customer.Client)

	resp, err := s.ctx.Customer.Client.MarkAllNotificationsRead()
	s.Require().NoError(err)
	s.Require().Less(resp.StatusCode, 300, string(resp.RawBody))

	e := stream.waitFor(s.T(), "unread", nil, 10*time.Second)
	s.Assert().Equal("notification.inapp_batch_read", e.Data.EventType)
}
