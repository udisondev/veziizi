package middleware

import (
	"net/http"
	"strings"

	"github.com/udisondev/veziizi/backend/internal/pkg/config"
)

// SEC-015: Ограничение размера request body
// Защита от DoS атак через большие запросы

// BodyLimit creates middleware that limits request body size
func BodyLimit(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			contentType := r.Header.Get("Content-Type")
			var maxSize int64

			if strings.HasPrefix(contentType, "multipart/form-data") {
				maxSize = cfg.Security.MaxFileUploadSize
			} else {
				maxSize = cfg.Security.MaxJSONBodySize
			}

			r.Body = http.MaxBytesReader(w, r.Body, maxSize)

			next.ServeHTTP(w, r)
		})
	}
}
