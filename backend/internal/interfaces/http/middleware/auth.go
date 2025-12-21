package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"codeberg.org/udison/veziizi/backend/internal/interfaces/http/session"
)

// Public paths that don't require authentication
var publicPaths = map[string]map[string]bool{
	"/api/v1/auth/login":    {http.MethodPost: true},
	"/api/v1/organizations": {http.MethodPost: true}, // registration
}

// Public path prefixes
var publicPrefixes = []struct {
	prefix string
	method string
}{
	{"/api/v1/invitations/", http.MethodGet},  // get invitation by token
	{"/api/v1/invitations/", http.MethodPost}, // accept invitation
	{"/api/v1/admin/auth/", http.MethodPost},  // admin login/logout
	// DEV ENDPOINTS REMOVED - SEC-001: они защищены через DevOnly middleware и проверку IsDevelopment()
}

// isPublicPath checks if the path and method combination is public
func isPublicPath(path, method string) bool {
	// Check exact matches
	if methods, ok := publicPaths[path]; ok {
		if methods[method] {
			return true
		}
	}

	// Check prefixes
	for _, p := range publicPrefixes {
		if strings.HasPrefix(path, p.prefix) && method == p.method {
			return true
		}
	}

	return false
}

// RequireAuth creates middleware that checks if user is authenticated
// Skips authentication for public paths
func RequireAuth(sessionManager *session.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for public paths
			if isPublicPath(r.URL.Path, r.Method) {
				next.ServeHTTP(w, r)
				return
			}

			// Skip auth for admin paths (they have their own auth)
			if strings.HasPrefix(r.URL.Path, "/api/v1/admin/") {
				next.ServeHTTP(w, r)
				return
			}

			// Skip auth for dev paths (protected by DevOnly middleware in main.go)
			if strings.HasPrefix(r.URL.Path, "/api/v1/dev/") {
				next.ServeHTTP(w, r)
				return
			}

			if _, ok := sessionManager.GetMemberID(r); !ok {
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAdminAuth creates middleware that checks if admin is authenticated
func RequireAdminAuth(adminSession *session.AdminManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip for login endpoint
			if r.URL.Path == "/api/v1/admin/auth/login" && r.Method == http.MethodPost {
				next.ServeHTTP(w, r)
				return
			}

			if _, ok := adminSession.GetAdminID(r); !ok {
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": message}); err != nil {
		slog.Error("failed to encode error response", slog.String("error", err.Error()))
	}
}
