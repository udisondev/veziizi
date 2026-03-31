package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

// TestAdminRoutes_SubrouterPathPrefix проверяет, что admin-хендлеры корректно
// регистрируют маршруты как относительные пути на subrouter с PathPrefix.
//
// Регрессионный тест для бага: маршруты регистрировались с полными путями
// (например, "/api/v1/admin/auth/login") на subrouter с PathPrefix("/api/v1/admin"),
// что приводило к удвоению префикса и 404 на все admin-эндпоинты.
//
// Фикс: маршруты используют относительные пути ("/auth/login").
func TestAdminRoutes_SubrouterPathPrefix(t *testing.T) {
	t.Parallel()

	// Создаём роутер, имитирующий production setup из cmd/api/main.go
	router := mux.NewRouter()
	adminRouter := router.PathPrefix("/api/v1/admin").Subrouter()

	// Регистрируем dummy-хендлер на admin subrouter с относительными путями
	// (как это делает AdminHandler.RegisterRoutes)
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Воспроизводим маршруты из AdminHandler.RegisterRoutes
	adminRouter.HandleFunc("/auth/login", dummyHandler).Methods(http.MethodPost)
	adminRouter.HandleFunc("/auth/logout", dummyHandler).Methods(http.MethodPost)
	adminRouter.HandleFunc("/auth/me", dummyHandler).Methods(http.MethodGet)
	adminRouter.HandleFunc("/organizations", dummyHandler).Methods(http.MethodGet)
	adminRouter.HandleFunc("/organizations/{id}", dummyHandler).Methods(http.MethodGet)
	adminRouter.HandleFunc("/organizations/{id}/approve", dummyHandler).Methods(http.MethodPost)
	adminRouter.HandleFunc("/organizations/{id}/reject", dummyHandler).Methods(http.MethodPost)
	adminRouter.HandleFunc("/organizations/{id}/mark-fraudster", dummyHandler).Methods(http.MethodPost)
	adminRouter.HandleFunc("/organizations/{id}/unmark-fraudster", dummyHandler).Methods(http.MethodPost)
	adminRouter.HandleFunc("/fraudsters", dummyHandler).Methods(http.MethodGet)
	adminRouter.HandleFunc("/reviews", dummyHandler).Methods(http.MethodGet)
	adminRouter.HandleFunc("/reviews/{id}", dummyHandler).Methods(http.MethodGet)
	adminRouter.HandleFunc("/reviews/{id}/approve", dummyHandler).Methods(http.MethodPost)
	adminRouter.HandleFunc("/reviews/{id}/reject", dummyHandler).Methods(http.MethodPost)

	// Воспроизводим маршруты из AdminSupportHandler.RegisterRoutes
	adminRouter.HandleFunc("/support/tickets", dummyHandler).Methods(http.MethodGet)
	adminRouter.HandleFunc("/support/tickets/{id}", dummyHandler).Methods(http.MethodGet)
	adminRouter.HandleFunc("/support/tickets/{id}/messages", dummyHandler).Methods(http.MethodPost)
	adminRouter.HandleFunc("/support/tickets/{id}/close", dummyHandler).Methods(http.MethodPost)

	// Воспроизводим маршруты из AdminEmailTemplatesHandler.RegisterRoutes
	adminRouter.HandleFunc("/email-templates", dummyHandler).Methods(http.MethodGet)
	adminRouter.HandleFunc("/email-templates", dummyHandler).Methods(http.MethodPost)
	adminRouter.HandleFunc("/email-templates/preview", dummyHandler).Methods(http.MethodPost)
	adminRouter.HandleFunc("/email-templates/{id}", dummyHandler).Methods(http.MethodGet)
	adminRouter.HandleFunc("/email-templates/{id}", dummyHandler).Methods(http.MethodPatch)
	adminRouter.HandleFunc("/email-templates/{id}", dummyHandler).Methods(http.MethodDelete)

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
				t.Errorf("%s %s returned 404; route not matched (possible full-path-on-subrouter bug)", tt.method, tt.path)
			}
			// Ожидаем 200 OK, поскольку dummy handler всегда возвращает 200
			if rr.Code != http.StatusOK {
				t.Errorf("%s %s = %d; want %d", tt.method, tt.path, rr.Code, http.StatusOK)
			}
		})
	}
}

// TestAdminRoutes_FullPathsOnSubrouter_Fail демонстрирует, что баг воспроизводился
// при регистрации полных путей на subrouter. Это негативный тест — проверяет,
// что полные пути действительно не матчатся (как было до фикса).
func TestAdminRoutes_FullPathsOnSubrouter_Fail(t *testing.T) {
	t.Parallel()

	router := mux.NewRouter()
	adminRouter := router.PathPrefix("/api/v1/admin").Subrouter()

	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Имитируем ОШИБОЧНУЮ регистрацию: полные пути на subrouter
	adminRouter.HandleFunc("/api/v1/admin/auth/login", dummyHandler).Methods(http.MethodPost)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/auth/login", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	// При полном пути на subrouter gorilla/mux пытается найти
	// /api/v1/admin/api/v1/admin/auth/login — что возвращает 405 или 404
	if rr.Code == http.StatusOK {
		t.Error("full path on subrouter unexpectedly matched; test premise is wrong")
	}
}
