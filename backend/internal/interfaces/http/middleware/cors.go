package middleware

import (
	"net/http"
	"strings"

	"github.com/udisondev/veziizi/backend/internal/pkg/config"
)

func CORS(cfg *config.Config) func(http.Handler) http.Handler {
	allowedOrigins := map[string]bool{
		"http://localhost:5173":  true,
		"http://localhost:3000":  true,
		"http://127.0.0.1:5173": true,
		"http://127.0.0.1:3000": true,
	}

	if cfg.IsProduction() {
		allowedOrigins = map[string]bool{
			"https://везиизи.рф":               true,
			"https://xn--e1aebcghhi.xn--p1acf": true,
		}
	}

	if cfg.HTTP.CORSOrigins != "" {
		for _, origin := range strings.Split(cfg.HTTP.CORSOrigins, ",") {
			allowedOrigins[strings.TrimSpace(origin)] = true
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			if origin != "" {
				if allowedOrigins[origin] || (cfg.IsDevelopment() && isLocalhostOrigin(origin)) {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Credentials", "true")
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
					w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Requested-With, Authorization")
					w.Header().Set("Access-Control-Max-Age", "86400")
				}
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func isLocalhostOrigin(origin string) bool {
	return strings.HasPrefix(origin, "http://localhost:") ||
		strings.HasPrefix(origin, "http://127.0.0.1:")
}
