package middleware

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/udisondev/veziizi/backend/internal/interfaces/http/session"
	"github.com/udisondev/veziizi/backend/internal/pkg/httputil"
)

// EventMetaEnricher добавляет метаданные для аудита событий в контекст.
// Должен вызываться после auth middleware для получения member_id и org_id.
func EventMetaEnricher(sessionManager *session.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var memberID, orgID uuid.UUID

			// Получаем данные из session (если авторизован)
			if sessionManager != nil {
				memberID, _ = sessionManager.GetMemberID(r)
				orgID, _ = sessionManager.GetOrganizationID(r)
			}

			// Создаём metadata и добавляем в context
			meta := httputil.EventMetaFromRequest(r, memberID, orgID)
			ctx := httputil.WithEventMeta(r.Context(), meta)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
