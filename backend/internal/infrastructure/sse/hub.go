package sse

import (
	"errors"
	"sync"

	"github.com/google/uuid"
)

var (
	// ErrTooManyConnections — превышен лимит соединений (на member или общий).
	ErrTooManyConnections = errors.New("sse: too many connections")
	// ErrShuttingDown — хаб закрывается, новые подписки не принимаются.
	ErrShuttingDown = errors.New("sse: hub is shutting down")
)

// connBufferSize — буфер канала событий одного соединения. Переполнение
// означает, что клиент не вычитывает поток — такое соединение закрываем
// (EventSource переподключится и сделает refetch).
const connBufferSize = 16

// Conn — одно SSE-соединение браузера.
type Conn struct {
	memberID uuid.UUID
	orgID    uuid.UUID

	events    chan Event
	done      chan struct{}
	closeOnce sync.Once
}

// Events — канал пуш-событий для записи в ResponseWriter. Канал никогда не
// закрывается со стороны хаба (конкурентная запись + close = panic) — сигнал
// завершения только через Done.
func (c *Conn) Events() <-chan Event { return c.events }

// Done закрывается, когда хаб принудительно завершает соединение
// (переполнение буфера, shutdown).
func (c *Conn) Done() <-chan struct{} { return c.done }

func (c *Conn) close() {
	c.closeOnce.Do(func() { close(c.done) })
}

// Hub — реестр активных SSE-соединений с роутингом по member и organization.
type Hub struct {
	mu           sync.RWMutex
	byMember     map[uuid.UUID]map[*Conn]struct{}
	byOrg        map[uuid.UUID]map[*Conn]struct{}
	total        int
	shuttingDown bool

	maxPerMember int
	maxTotal     int
}

// NewHub создает хаб с лимитами соединений.
func NewHub(maxPerMember, maxTotal int) *Hub {
	return &Hub{
		byMember:     make(map[uuid.UUID]map[*Conn]struct{}),
		byOrg:        make(map[uuid.UUID]map[*Conn]struct{}),
		maxPerMember: maxPerMember,
		maxTotal:     maxTotal,
	}
}

// Subscribe регистрирует новое соединение member'а.
func (h *Hub) Subscribe(memberID, orgID uuid.UUID) (*Conn, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.shuttingDown {
		return nil, ErrShuttingDown
	}
	if h.total >= h.maxTotal || len(h.byMember[memberID]) >= h.maxPerMember {
		return nil, ErrTooManyConnections
	}

	c := &Conn{
		memberID: memberID,
		orgID:    orgID,
		events:   make(chan Event, connBufferSize),
		done:     make(chan struct{}),
	}

	if h.byMember[memberID] == nil {
		h.byMember[memberID] = make(map[*Conn]struct{})
	}
	h.byMember[memberID][c] = struct{}{}

	if h.byOrg[orgID] == nil {
		h.byOrg[orgID] = make(map[*Conn]struct{})
	}
	h.byOrg[orgID][c] = struct{}{}

	h.total++
	return c, nil
}

// Unsubscribe удаляет соединение из реестра. Вызывается из defer
// SSE-хендлера — единственное место, где мутируются мапы по живому Conn.
func (h *Hub) Unsubscribe(c *Conn) {
	c.close()

	h.mu.Lock()
	defer h.mu.Unlock()

	if conns, ok := h.byMember[c.memberID]; ok {
		if _, ok := conns[c]; ok {
			delete(conns, c)
			h.total--
			if len(conns) == 0 {
				delete(h.byMember, c.memberID)
			}
		}
	}
	if conns, ok := h.byOrg[c.orgID]; ok {
		delete(conns, c)
		if len(conns) == 0 {
			delete(h.byOrg, c.orgID)
		}
	}
}

// PublishToMember доставляет событие всем соединениям member'а.
func (h *Hub) PublishToMember(memberID uuid.UUID, e Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.byMember[memberID] {
		deliver(c, e)
	}
}

// PublishToOrg доставляет событие всем соединениям членов организации.
func (h *Hub) PublishToOrg(orgID uuid.UUID, e Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.byOrg[orgID] {
		deliver(c, e)
	}
}

// deliver выполняется под RLock и потому не трогает мапы хаба: медленному
// клиенту просто закрываем done — его хендлер завершится и сам вызовет
// Unsubscribe.
func deliver(c *Conn, e Event) {
	select {
	case c.events <- e:
	default:
		c.close()
	}
}

// Shutdown закрывает все соединения и запрещает новые подписки. Вызывать ДО
// http.Server.Shutdown — иначе graceful shutdown повиснет на живых SSE-потоках
// до таймаута.
func (h *Hub) Shutdown() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.shuttingDown = true
	for _, conns := range h.byMember {
		for c := range conns {
			c.close()
		}
	}
}

// Len возвращает число активных соединений (для метрик/тестов).
func (h *Hub) Len() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.total
}
