package httputil

import (
	"net"
	"net/http"
	"strings"
)

// GetClientIP извлекает реальный IP клиента из HTTP request.
// Учитывает заголовки X-Forwarded-For, X-Real-IP и RemoteAddr.
func GetClientIP(r *http.Request) string {
	// 1. X-Forwarded-For (первый IP в цепочке — клиент)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		clientIP := strings.TrimSpace(ips[0])
		if clientIP != "" {
			return clientIP
		}
	}

	// 2. X-Real-IP (устанавливается nginx/reverse proxy)
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// 3. RemoteAddr (прямое соединение, убираем порт)
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// Если не удалось разделить — вернуть как есть
		return r.RemoteAddr
	}
	return host
}

// GetUserAgent возвращает User-Agent из HTTP request.
func GetUserAgent(r *http.Request) string {
	return r.Header.Get("User-Agent")
}

// GetFingerprint возвращает fingerprint из заголовка X-Fingerprint.
func GetFingerprint(r *http.Request) string {
	return r.Header.Get("X-Fingerprint")
}

// ClientMetadata содержит metadata клиента из HTTP request.
type ClientMetadata struct {
	IP          string
	UserAgent   string
	Fingerprint string
	// Geo fields (populated by GeoIP service)
	GeoCountry string
	GeoCity    string
	GeoLat     float64
	GeoLon     float64
}

// GetClientMetadata извлекает все metadata клиента из request.
// Для geo-данных используйте EnrichWithGeo после получения.
func GetClientMetadata(r *http.Request) ClientMetadata {
	return ClientMetadata{
		IP:          GetClientIP(r),
		UserAgent:   GetUserAgent(r),
		Fingerprint: GetFingerprint(r),
	}
}
