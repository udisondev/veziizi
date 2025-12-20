package middleware

import (
	"net/http"

	"codeberg.org/udison/veziizi/backend/internal/pkg/config"
)

// SEC-011: Security headers middleware
// Добавляет важные security headers для защиты от различных атак.

// SecurityHeaders creates middleware that adds security headers to all responses
func SecurityHeaders(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// X-Content-Type-Options: предотвращает MIME-sniffing
			w.Header().Set("X-Content-Type-Options", "nosniff")

			// X-Frame-Options: защита от clickjacking
			w.Header().Set("X-Frame-Options", "DENY")

			// X-XSS-Protection: включает XSS фильтр браузера (legacy, но всё ещё полезно)
			w.Header().Set("X-XSS-Protection", "1; mode=block")

			// Referrer-Policy: контролирует информацию в Referer header
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

			// Permissions-Policy: ограничивает возможности браузера
			w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

			// Content-Security-Policy: защита от XSS и injection атак
			// Базовая политика для API - разрешаем только self
			w.Header().Set("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none'")

			// Strict-Transport-Security: принудительный HTTPS (только в production)
			if cfg.IsProduction() {
				// max-age=31536000 (1 год), includeSubDomains
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}

			next.ServeHTTP(w, r)
		})
	}
}
