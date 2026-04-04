package httputil

import (
	"net"
	"net/http"
	"strings"
	"sync"
)

var (
	trustedProxies   = map[string]struct{}{"127.0.0.1": {}, "::1": {}}
	trustedProxiesMu sync.RWMutex
)

// SetTrustedProxies устанавливает список доверенных прокси-серверов.
// Вызывается при инициализации приложения из конфигурации.
// SEC-004: Только запросы от trusted proxy доверяют заголовкам X-Forwarded-For и X-Real-IP.
func SetTrustedProxies(proxies []string) {
	trustedProxiesMu.Lock()
	defer trustedProxiesMu.Unlock()
	trustedProxies = make(map[string]struct{}, len(proxies))
	for _, p := range proxies {
		trustedProxies[strings.TrimSpace(p)] = struct{}{}
	}
}

// isTrustedProxy проверяет, является ли IP доверенным прокси.
func isTrustedProxy(ip string) bool {
	trustedProxiesMu.RLock()
	defer trustedProxiesMu.RUnlock()
	_, ok := trustedProxies[ip]
	return ok
}

// GetClientIP извлекает реальный IP клиента из HTTP request.
// SEC-004: Заголовки X-Forwarded-For и X-Real-IP доверяются только от trusted proxy.
func GetClientIP(r *http.Request) string {
	remoteIP := getRemoteIP(r.RemoteAddr)

	// SEC-004: Доверяем proxy headers только от trusted proxies
	if isTrustedProxy(remoteIP) {
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
	}

	// 3. RemoteAddr (прямое соединение или недоверенный proxy)
	return remoteIP
}

// getRemoteIP извлекает IP из RemoteAddr (убирает порт).
func getRemoteIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
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
