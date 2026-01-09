package httputil

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// EncodeCursor кодирует данные в base64 URL-safe строку для cursor pagination.
func EncodeCursor(data any) (string, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("marshal cursor: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// DecodeCursor декодирует base64 cursor в структуру.
func DecodeCursor[T any](cursor string) (*T, error) {
	b, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, fmt.Errorf("decode base64: %w", err)
	}
	var result T
	if err := json.Unmarshal(b, &result); err != nil {
		return nil, fmt.Errorf("unmarshal cursor: %w", err)
	}
	return &result, nil
}
