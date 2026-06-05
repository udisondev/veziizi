package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/sse"
	"github.com/udisondev/veziizi/backend/internal/interfaces/http/session"
	"github.com/udisondev/veziizi/backend/internal/pkg/config"
)

// defaultHeartbeat — fallback при незаданном/нулевом SSE_HEARTBEAT_INTERVAL,
// совпадает с envDefault конфига.
const defaultHeartbeat = 25 * time.Second

// EventsHandler — SSE-стрим пуш-«пинков» о доменных событиях. Один стрим на
// приложение: фронт держит единственный EventSource и раздаёт события
// подписчикам по полю event (см. frontend/src/services/eventStream.ts).
type EventsHandler struct {
	hub       *sse.Hub
	session   *session.Manager
	heartbeat time.Duration
}

func NewEventsHandler(hub *sse.Hub, sessionManager *session.Manager, cfg *config.Config) *EventsHandler {
	heartbeat := cfg.SSE.HeartbeatInterval
	// Guard как у tailer'а (block <= 0): иначе SSE_HEARTBEAT_INTERVAL=0
	// уронит time.NewTicker паникой на первом же подключении.
	if heartbeat <= 0 {
		heartbeat = defaultHeartbeat
	}
	return &EventsHandler{
		hub:       hub,
		session:   sessionManager,
		heartbeat: heartbeat,
	}
}

func (h *EventsHandler) RegisterRoutes(r chi.Router) {
	r.Get("/api/v1/events/stream", h.Stream)
}

// Stream держит долгоживущее SSE-соединение. Аутентификация — session cookie
// (EventSource шлёт её сам), CSRF не нужен (GET). Потерянные события клиент
// компенсирует refetch'ем при (пере)подключении, поэтому Last-Event-ID и
// реплей не поддерживаются — только живой хвост.
func (h *EventsHandler) Stream(w http.ResponseWriter, r *http.Request) {
	memberID, ok := h.session.GetMemberID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	orgID, ok := h.session.GetOrganizationID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	conn, err := h.hub.Subscribe(memberID, orgID)
	if err != nil {
		if errors.Is(err, sse.ErrTooManyConnections) {
			writeError(w, http.StatusTooManyRequests, "too many event stream connections")
			return
		}
		writeError(w, http.StatusServiceUnavailable, "event stream unavailable")
		return
	}
	defer h.hub.Unsubscribe(conn)

	rc := http.NewResponseController(w)
	// Снимаем WriteTimeout сервера (15s) для этого соединения — SSE живёт
	// часами. Работает через Unwrap-цепочку обёрток ResponseWriter.
	if err := rc.SetWriteDeadline(time.Time{}); err != nil {
		slog.Error("sse: reset write deadline failed", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no") // nginx: не буферизовать стрим
	w.WriteHeader(http.StatusOK)

	// retry — пауза реконнекта EventSource; первый flush заодно подтверждает
	// клиенту открытие стрима (event 'open').
	if _, err := w.Write([]byte("retry: 3000\n\n")); err != nil {
		return
	}
	if err := rc.Flush(); err != nil {
		slog.Error("sse: flush unsupported", slog.String("error", err.Error()))
		return
	}

	heartbeat := time.NewTicker(h.heartbeat)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-conn.Done():
			return
		case e := <-conn.Events():
			if _, err := w.Write(e.Encode()); err != nil {
				return
			}
			if err := rc.Flush(); err != nil {
				return
			}
		case <-heartbeat.C:
			// Именованное событие, а не SSE-комментарий: комментарии не
			// генерируют события в EventSource, и клиент не смог бы отличить
			// живое соединение от «тихо умершего» (см. eventStream.ts watchdog).
			if _, err := w.Write([]byte("event: ping\ndata: {}\n\n")); err != nil {
				return
			}
			if err := rc.Flush(); err != nil {
				return
			}
		}
	}
}
