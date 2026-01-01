package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	sessionApp "codeberg.org/udison/veziizi/backend/internal/application/session"
	"codeberg.org/udison/veziizi/backend/internal/interfaces/http/session"
	"codeberg.org/udison/veziizi/backend/internal/pkg/httputil"
	"github.com/google/uuid"
)

// SEC-003: IP-based rate limiting для public endpoints
// Простой in-memory rate limiter для защиты от брутфорса
type ipRateLimiter struct {
	mu       sync.RWMutex
	requests map[string]*ipRequestInfo
}

type ipRequestInfo struct {
	count     int
	firstSeen time.Time
	blocked   bool
	blockUntil time.Time
}

var (
	// Глобальный rate limiter для public endpoints
	publicRateLimiter = &ipRateLimiter{
		requests: make(map[string]*ipRequestInfo),
	}
	// Rate limiter для geo endpoints (более высокий лимит)
	geoRateLimiter = &ipRateLimiter{
		requests: make(map[string]*ipRequestInfo),
	}
	// Конфигурация rate limiting
	maxRequestsPerWindow      = 10               // Максимум запросов за окно (public)
	maxGeoRequestsPerWindow   = 200              // Максимум запросов за окно (geo)
	maxAdminRequestsPerWindow = 50               // Максимум запросов за окно (admin)
	windowDuration            = 1 * time.Minute  // Окно в 1 минуту
	blockDuration             = 15 * time.Minute // Блокировка на 15 минут

	// Канал для graceful shutdown cleanup горутины
	cleanupStop     chan struct{}
	cleanupStopOnce sync.Once
)

// checkIPRateLimitWithMax проверяет rate limit по IP с настраиваемым лимитом
// Возвращает true если запрос разрешён, false если заблокирован
func (l *ipRateLimiter) checkIPRateLimitWithMax(ip, endpoint string, maxRequests int) (bool, string) {
	key := fmt.Sprintf("%s:%s", ip, endpoint)
	now := time.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	info, exists := l.requests[key]
	if !exists {
		l.requests[key] = &ipRequestInfo{
			count:     1,
			firstSeen: now,
		}
		return true, ""
	}

	// Проверить блокировку
	if info.blocked {
		if now.Before(info.blockUntil) {
			remaining := info.blockUntil.Sub(now).Round(time.Second)
			return false, fmt.Sprintf("too many requests, try again in %s", remaining)
		}
		// Блокировка истекла — сбросить счётчик
		info.blocked = false
		info.count = 1
		info.firstSeen = now
		return true, ""
	}

	// Сбросить счётчик если окно истекло
	if now.Sub(info.firstSeen) > windowDuration {
		info.count = 1
		info.firstSeen = now
		return true, ""
	}

	// Увеличить счётчик
	info.count++

	// Проверить лимит
	if info.count > maxRequests {
		info.blocked = true
		info.blockUntil = now.Add(blockDuration)
		slog.Warn("SEC-003: IP rate limited",
			slog.String("ip", ip),
			slog.String("endpoint", endpoint),
			slog.Int("count", info.count),
			slog.Int("max", maxRequests),
		)
		return false, fmt.Sprintf("too many requests, try again in %s", blockDuration)
	}

	return true, ""
}

// checkIPRateLimit проверяет rate limit по IP для public endpoints (лимит по умолчанию)
func (l *ipRateLimiter) checkIPRateLimit(ip, endpoint string) (bool, string) {
	return l.checkIPRateLimitWithMax(ip, endpoint, maxRequestsPerWindow)
}

// cleanupOldEntries периодически очищает старые записи (вызывать в фоне)
func (l *ipRateLimiter) cleanupOldEntries() {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	for key, info := range l.requests {
		// Удалить записи старше 1 часа
		if now.Sub(info.firstSeen) > time.Hour && !info.blocked {
			delete(l.requests, key)
		}
		// Удалить истёкшие блокировки старше 1 часа
		if info.blocked && now.Sub(info.blockUntil) > time.Hour {
			delete(l.requests, key)
		}
	}
}

// SetRateLimits allows configuring rate limits for testing.
func SetRateLimits(maxPublic, maxGeo int) {
	maxRequestsPerWindow = maxPublic
	maxGeoRequestsPerWindow = maxGeo
	maxAdminRequestsPerWindow = maxPublic // Use same limit for admin in tests
}

func init() {
	cleanupStop = make(chan struct{})
	// Запустить периодическую очистку с поддержкой graceful shutdown
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				publicRateLimiter.cleanupOldEntries()
				geoRateLimiter.cleanupOldEntries()
			case <-cleanupStop:
				slog.Debug("rate limiter cleanup goroutine stopped")
				return
			}
		}
	}()
}

// StopRateLimiterCleanup останавливает фоновую горутину очистки rate limiter.
// Вызывается при graceful shutdown сервера.
func StopRateLimiterCleanup() {
	cleanupStopOnce.Do(func() {
		close(cleanupStop)
	})
}

// RateLimiter creates middleware that limits API request rate
// Uses PostgreSQL-based rate limiting (no Redis required)
// SEC-003: Добавлен IP-based rate limiting для public endpoints
func RateLimiter(
	sessionManager *session.Manager,
	sessionAnalyzer *sessionApp.SessionAnalyzer,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Geo endpoints: higher rate limit (200/min) for autocomplete
			if strings.HasPrefix(r.URL.Path, "/api/v1/geo/") {
				clientIP := httputil.GetClientIP(r)
				allowed, reason := geoRateLimiter.checkIPRateLimitWithMax(clientIP, "geo", maxGeoRequestsPerWindow)
				if !allowed {
					writeError(w, http.StatusTooManyRequests, reason)
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			// SEC-003: IP-based rate limiting для public paths (login, register, invitations)
			if isPublicPath(r.URL.Path, r.Method) {
				clientIP := httputil.GetClientIP(r)
				endpoint := r.URL.Path

				allowed, reason := publicRateLimiter.checkIPRateLimit(clientIP, endpoint)
				if !allowed {
					writeError(w, http.StatusTooManyRequests, reason)
					return
				}

				next.ServeHTTP(w, r)
				return
			}

			// SEC-003: Admin paths rate limiting (более высокий лимит, но всё равно ограничен)
			if strings.HasPrefix(r.URL.Path, "/api/v1/admin/") {
				clientIP := httputil.GetClientIP(r)
				// maxAdminRequestsPerWindow requests per minute для admin - защита от brute force
				allowed, reason := publicRateLimiter.checkIPRateLimitWithMax(clientIP, "admin", maxAdminRequestsPerWindow)
				if !allowed {
					slog.Warn("SEC-003: Admin endpoint rate limited",
						slog.String("ip", clientIP),
						slog.String("path", r.URL.Path),
					)
					writeError(w, http.StatusTooManyRequests, reason)
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			// Get member ID from session
			memberID, ok := sessionManager.GetMemberID(r)
			if !ok {
				// Not authenticated, skip rate limiting (auth middleware will handle)
				next.ServeHTTP(w, r)
				return
			}

			// Get org ID from session
			orgID, _ := sessionManager.GetOrganizationID(r)
			if orgID == uuid.Nil {
				orgID = memberID // fallback
			}

			// Build endpoint key for tracking
			endpoint := fmt.Sprintf("%s %s", r.Method, r.URL.Path)

			// Check for API abuse
			result, err := sessionAnalyzer.CheckAPIAbuse(r.Context(), memberID, orgID, endpoint)
			if err != nil {
				slog.Error("rate limiter: failed to check API abuse",
					slog.String("error", err.Error()),
					slog.String("member_id", memberID.String()),
				)
				// Don't block on error, just log
				next.ServeHTTP(w, r)
				return
			}

			// Block if rate limited
			if result.BlockLogin {
				slog.Warn("rate limiter: blocking request",
					slog.String("member_id", memberID.String()),
					slog.String("reason", result.BlockReason),
				)
				writeError(w, http.StatusTooManyRequests, result.BlockReason)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
