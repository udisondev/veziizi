package httputil

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type eventMetaKey struct{}

// EventMeta содержит метаданные для аудита событий.
// Передаётся через context из HTTP layer в infrastructure.
type EventMeta struct {
	MemberID       uuid.UUID // ID члена организации (actor)
	OrganizationID uuid.UUID // ID организации (actor's org)
	IP             string    // Client IP
	UserAgent      string    // User-Agent
	Fingerprint    string    // Browser fingerprint
}

// WithEventMeta добавляет метаданные события в контекст.
func WithEventMeta(ctx context.Context, meta EventMeta) context.Context {
	return context.WithValue(ctx, eventMetaKey{}, meta)
}

// EventMetaFromCtx извлекает метаданные события из контекста.
// Возвращает пустой EventMeta если метаданные не найдены.
func EventMetaFromCtx(ctx context.Context) (EventMeta, bool) {
	meta, ok := ctx.Value(eventMetaKey{}).(EventMeta)
	return meta, ok
}

// ToMap конвертирует EventMeta в map[string]string для event envelope.
func (m EventMeta) ToMap() map[string]string {
	result := make(map[string]string)

	if m.MemberID != uuid.Nil {
		result["actor_member_id"] = m.MemberID.String()
	}
	if m.OrganizationID != uuid.Nil {
		result["actor_org_id"] = m.OrganizationID.String()
	}
	if m.IP != "" {
		result["client_ip"] = m.IP
	}
	if m.UserAgent != "" {
		result["user_agent"] = m.UserAgent
	}
	if m.Fingerprint != "" {
		result["fingerprint"] = m.Fingerprint
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

// EventMetaFromRequest создаёт EventMeta из HTTP request.
// memberID и orgID должны быть получены из session отдельно.
func EventMetaFromRequest(r *http.Request, memberID, orgID uuid.UUID) EventMeta {
	return EventMeta{
		MemberID:       memberID,
		OrganizationID: orgID,
		IP:             GetClientIP(r),
		UserAgent:      GetUserAgent(r),
		Fingerprint:    GetFingerprint(r),
	}
}
