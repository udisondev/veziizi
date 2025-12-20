package middleware

import (
	"log/slog"
	"net/http"
)

// SEC-005: CSRF защита для SPA
// Проверяем заголовок X-Requested-With для state-changing запросов.
// Браузеры не позволяют установить этот заголовок cross-origin без CORS preflight,
// что делает CSRF атаки невозможными.

// CSRFProtection creates middleware that protects against CSRF attacks
// by requiring X-Requested-With header on state-changing requests.
// Combined with SameSite cookies, this provides robust CSRF protection for SPAs.
func CSRFProtection() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip for safe methods (GET, HEAD, OPTIONS)
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			// Skip for public paths (login, registration don't need CSRF - no session yet)
			if isPublicPath(r.URL.Path, r.Method) {
				next.ServeHTTP(w, r)
				return
			}

			// Check for X-Requested-With header
			// XMLHttpRequest / fetch with custom headers triggers CORS preflight,
			// which browsers block for cross-origin requests without explicit CORS config
			xRequestedWith := r.Header.Get("X-Requested-With")
			if xRequestedWith == "" {
				slog.Warn("SEC-005: CSRF protection - missing X-Requested-With header",
					slog.String("path", r.URL.Path),
					slog.String("method", r.Method),
					slog.String("origin", r.Header.Get("Origin")),
					slog.String("referer", r.Header.Get("Referer")),
				)
				writeError(w, http.StatusForbidden, "CSRF validation failed")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
