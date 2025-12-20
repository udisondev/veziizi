package httputil

import (
	"net/http"
	"strconv"
)

// SEC-016: Валидация pagination параметров
// Защита от DoS через большие limit и отрицательные offset

const (
	DefaultLimit = 20
	MaxLimit     = 100
	DefaultOffset = 0
)

// PaginationParams содержит валидированные параметры пагинации
type PaginationParams struct {
	Limit  int
	Offset int
}

// ParsePagination извлекает и валидирует limit и offset из query string
func ParsePagination(r *http.Request) PaginationParams {
	limit := DefaultLimit
	offset := DefaultOffset

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			offset = o
		}
	}

	// SEC-016: Валидация диапазонов
	if limit <= 0 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}
	if offset < 0 {
		offset = 0
	}

	return PaginationParams{
		Limit:  limit,
		Offset: offset,
	}
}
