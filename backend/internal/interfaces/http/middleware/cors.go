package middleware

import (
	"net/http"
	"strings"

	"codeberg.org/udison/veziizi/backend/internal/pkg/config"
)

// SEC-010: CORS middleware
// Ограничивает cross-origin запросы только разрешёнными origins.
// В development разрешаем localhost, в production — только production домен.

// CORS creates middleware that adds CORS headers
func CORS(cfg *config.Config) func(http.Handler) http.Handler {
	// Разрешённые origins в зависимости от окружения
	allowedOrigins := map[string]bool{
		"http://localhost:5173": true, // Vite dev server
		"http://localhost:3000": true, // Alternative dev port
		"http://127.0.0.1:5173": true,
		"http://127.0.0.1:3000": true,
	}

	// В production добавляем production origin
	if cfg.IsProduction() {
		// Очищаем dev origins в production
		allowedOrigins = map[string]bool{
			// TODO: добавить production домен
			// "https://veziizi.com": true,
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Проверяем origin
			if origin != "" {
				if allowedOrigins[origin] || (cfg.IsDevelopment() && isLocalhostOrigin(origin)) {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Credentials", "true")
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
					w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Requested-With, Authorization")
					w.Header().Set("Access-Control-Max-Age", "86400") // 24 hours
				}
				// Если origin не разрешён, просто не добавляем CORS headers
				// Браузер заблокирует запрос
			}

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isLocalhostOrigin проверяет что origin это localhost (для development)
func isLocalhostOrigin(origin string) bool {
	return strings.HasPrefix(origin, "http://localhost:") ||
		strings.HasPrefix(origin, "http://127.0.0.1:")
}
