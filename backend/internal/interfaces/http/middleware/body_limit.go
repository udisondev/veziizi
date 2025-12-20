package middleware

import (
	"net/http"
)

// SEC-015: Ограничение размера request body
// Защита от DoS атак через большие запросы

const (
	// Максимальный размер JSON body (1 MB)
	maxJSONBodySize = 1 << 20 // 1 MB

	// Максимальный размер файла (10 MB)
	maxFileUploadSize = 10 << 20 // 10 MB
)

// BodyLimit creates middleware that limits request body size
func BodyLimit() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Определяем лимит по Content-Type
			contentType := r.Header.Get("Content-Type")
			var maxSize int64

			// Для multipart/form-data (загрузка файлов) разрешаем больший размер
			if len(contentType) >= 19 && contentType[:19] == "multipart/form-data" {
				maxSize = maxFileUploadSize
			} else {
				maxSize = maxJSONBodySize
			}

			// SEC-015: Ограничиваем размер body
			r.Body = http.MaxBytesReader(w, r.Body, maxSize)

			next.ServeHTTP(w, r)
		})
	}
}
