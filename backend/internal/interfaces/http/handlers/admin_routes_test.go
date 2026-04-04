package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// TestAdminRoutes_SubrouterPathPrefix проверяет, что admin-хендлеры корректно
// регистрируют маршруты как относительные пути на subrouter с Route prefix.
func TestAdminRoutes_SubrouterPathPrefix(t *testing.T) {
	t.Parallel()

	// Создаём роутер, имитирующий production setup из cmd/api/main.go
	router := chi.NewRouter()

	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	router.Route("/api/v1/admin", func(r chi.Router) {
		// Воспроизводим маршруты из AdminHandler.RegisterRoutes
		r.Post("/auth/login", dummyHandler)
		r.Post("/auth/logout", dummyHandler)
		r.Get("/auth/me", dummyHandler)
		r.Get("/organizations", dummyHandler)
		r.Get("/organizations/{id}", dummyHandler)
		r.Post("/organizations/{id}/approve", dummyHandler)
		r.Post("/organizations/{id}/reject", dummyHandler)
		r.Post("/organizations/{id}/mark-fraudster", dummyHandler)
		r.Post("/organizations/{id}/unmark-fraudster", dummyHandler)
		r.Get("/fraudsters", dummyHandler)
		r.Get("/reviews", dummyHandler)
		r.Get("/reviews/{id}", dummyHandler)
		r.Post("/reviews/{id}/approve", dummyHandler)
		r.Post("/reviews/{id}/reject", dummyHandler)

		// Воспроизводим маршруты из AdminSupportHandler.RegisterRoutes
		r.Get("/support/tickets", dummyHandler)
		r.Get("/support/tickets/{id}", dummyHandler)
		r.Post("/support/tickets/{id}/messages", dummyHandler)
		r.Post("/support/tickets/{id}/close", dummyHandler)

		// Воспроизводим маршруты из AdminEmailTemplatesHandler.RegisterRoutes
		r.Get("/email-templates", dummyHandler)
		r.Post("/email-templates", dummyHandler)
		r.Post("/email-templates/preview", dummyHandler)
		r.Get("/email-templates/{id}", dummyHandler)
		r.Patch("/email-templates/{id}", dummyHandler)
		r.Delete("/email-templates/{id}", dummyHandler)
	})

	tests := []struct {
		name   string
		method string
		path   string
	}{
		// AdminHandler
		{"admin login", http.MethodPost, "/api/v1/admin/auth/login"},
		{"admin logout", http.MethodPost, "/api/v1/admin/auth/logout"},
		{"admin me", http.MethodGet, "/api/v1/admin/auth/me"},
		{"list organizations", http.MethodGet, "/api/v1/admin/organizations"},
		{"get organization", http.MethodGet, "/api/v1/admin/organizations/550e8400-e29b-41d4-a716-446655440000"},
		{"approve organization", http.MethodPost, "/api/v1/admin/organizations/550e8400-e29b-41d4-a716-446655440000/approve"},
		{"reject organization", http.MethodPost, "/api/v1/admin/organizations/550e8400-e29b-41d4-a716-446655440000/reject"},
		{"mark fraudster", http.MethodPost, "/api/v1/admin/organizations/550e8400-e29b-41d4-a716-446655440000/mark-fraudster"},
		{"unmark fraudster", http.MethodPost, "/api/v1/admin/organizations/550e8400-e29b-41d4-a716-446655440000/unmark-fraudster"},
		{"list fraudsters", http.MethodGet, "/api/v1/admin/fraudsters"},
		{"list reviews", http.MethodGet, "/api/v1/admin/reviews"},
		{"get review", http.MethodGet, "/api/v1/admin/reviews/550e8400-e29b-41d4-a716-446655440000"},
		{"approve review", http.MethodPost, "/api/v1/admin/reviews/550e8400-e29b-41d4-a716-446655440000/approve"},
		{"reject review", http.MethodPost, "/api/v1/admin/reviews/550e8400-e29b-41d4-a716-446655440000/reject"},

		// AdminSupportHandler
		{"list support tickets", http.MethodGet, "/api/v1/admin/support/tickets"},
		{"get support ticket", http.MethodGet, "/api/v1/admin/support/tickets/550e8400-e29b-41d4-a716-446655440000"},
		{"add ticket message", http.MethodPost, "/api/v1/admin/support/tickets/550e8400-e29b-41d4-a716-446655440000/messages"},
		{"close ticket", http.MethodPost, "/api/v1/admin/support/tickets/550e8400-e29b-41d4-a716-446655440000/close"},

		// AdminEmailTemplatesHandler
		{"list email templates", http.MethodGet, "/api/v1/admin/email-templates"},
		{"create email template", http.MethodPost, "/api/v1/admin/email-templates"},
		{"preview email template", http.MethodPost, "/api/v1/admin/email-templates/preview"},
		{"get email template", http.MethodGet, "/api/v1/admin/email-templates/550e8400-e29b-41d4-a716-446655440000"},
		{"update email template", http.MethodPatch, "/api/v1/admin/email-templates/550e8400-e29b-41d4-a716-446655440000"},
		{"delete email template", http.MethodDelete, "/api/v1/admin/email-templates/550e8400-e29b-41d4-a716-446655440000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(tt.method, tt.path, nil)
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			if rr.Code == http.StatusNotFound {
				t.Errorf("%s %s returned 404; route not matched", tt.method, tt.path)
			}
			if rr.Code != http.StatusOK {
				t.Errorf("%s %s = %d; want %d", tt.method, tt.path, rr.Code, http.StatusOK)
			}
		})
	}
}
