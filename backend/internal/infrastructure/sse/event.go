// Package sse — SSE-шлюз API-процесса: hub держит открытые соединения
// браузеров, tailer хвостом читает Redis-стримы (XREAD без consumer group),
// router маппит доменные события на получателей. Семантика push-to-refetch:
// клиенту уходит лёгкий «пинок» {entity_id, event_type}, данные он перечитывает
// существующими GET-endpoint'ами. Потеря пинка некритична — фронт делает
// refetch при каждом (пере)подключении.
package sse

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// Имена SSE-событий (поле "event:"), на которые подписывается фронт.
const (
	EventNotification   = "notification"   // новое in-app уведомление
	EventUnread         = "unread"         // изменился счётчик непрочитанных
	EventFreightRequest = "freight_request" // изменилась заявка/офферы
	EventSupportTicket  = "support_ticket" // изменился тикет поддержки
)

// Event — пуш-событие для клиента.
type Event struct {
	Type      string    `json:"-"`
	EntityID  uuid.UUID `json:"entity_id"`
	EventType string    `json:"event_type"`
}

// Encode сериализует событие в wire-формат SSE. data всегда одна строка
// (json.Marshal не вставляет переводы строк), поэтому multiline-экранирование
// не требуется.
func (e Event) Encode() []byte {
	data, err := json.Marshal(e)
	if err != nil {
		// Event состоит из UUID и строки — Marshal не падает; ветка на случай
		// будущих полей.
		data = []byte("{}")
	}
	return fmt.Appendf(nil, "event: %s\ndata: %s\n\n", e.Type, data)
}
