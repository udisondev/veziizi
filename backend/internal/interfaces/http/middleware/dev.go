package middleware

import (
	"net/http"

	"codeberg.org/udison/veziizi/backend/internal/pkg/config"
)

// DevOnly blocks access in production mode
func DevOnly(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.IsProduction() {
				writeError(w, http.StatusNotFound, "not found")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
