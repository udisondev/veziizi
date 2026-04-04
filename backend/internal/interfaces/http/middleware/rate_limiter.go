package middleware

import (
	"fmt"
	"hash/fnv"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	sessionApp "github.com/udisondev/veziizi/backend/internal/application/session"
	"github.com/udisondev/veziizi/backend/internal/interfaces/http/session"
	"github.com/udisondev/veziizi/backend/internal/pkg/config"
	"github.com/udisondev/veziizi/backend/internal/pkg/httputil"
)

const (
	// Количество шардов для снижения contention
	numShards = 32
)

// SEC-003: IP-based rate limiting для public endpoints
// Sharded rate limiter для снижения mutex contention
type shardedRateLimiter struct {
	shards [numShards]*rateLimiterShard
}

type rateLimiterShard struct {
	mu       sync.RWMutex
	requests map[string]*ipRequestInfo
}

type ipRequestInfo struct {
	count      int
	firstSeen  time.Time
	blocked    bool
	blockUntil time.Time
}

// rateLimitResult содержит результат проверки для логирования вне lock
type rateLimitResult struct {
	allowed    bool
	reason     string
	shouldLog  bool
	ip         string
	endpoint   string
	count      int
	maxRequest int
}

var (
	// Глобальные rate limiter'ы (sharded)
	publicRateLimiter *shardedRateLimiter
	geoRateLimiter    *shardedRateLimiter

	// Конфигурация rate limiting (заполняется из config)
	maxRequestsPerWindow      = 10               // Максимум запросов за окно (public)
	maxGeoRequestsPerWindow   = 200              // Максимум запросов за окно (geo)
	maxAdminRequestsPerWindow = 50               // Максимум запросов за окно (admin)
	windowDuration            = 1 * time.Minute  // Окно в 1 минуту
	blockDuration             = 15 * time.Minute // Блокировка на 15 минут

	// Управление cleanup горутиной
	cleanupStop     chan struct{}
	cleanupStopOnce sync.Once
	cleanupStarted  atomic.Bool
)

// newShardedRateLimiter создаёт новый sharded rate limiter
func newShardedRateLimiter() *shardedRateLimiter {
	l := &shardedRateLimiter{}
	for i := range numShards {
		l.shards[i] = &rateLimiterShard{
			requests: make(map[string]*ipRequestInfo),
		}
	}
	return l
}

// getShard возвращает шард для данного ключа
func (l *shardedRateLimiter) getShard(key string) *rateLimiterShard {
	h := fnv.New32a()
	h.Write([]byte(key))
	return l.shards[h.Sum32()%numShards]
}

// checkIPRateLimitWithMax проверяет rate limit по IP с настраиваемым лимитом
// Возвращает результат для логирования вне lock
func (l *shardedRateLimiter) checkIPRateLimitWithMax(ip, endpoint string, maxRequests int) rateLimitResult {
	// Используем IP как ключ шарда для равномерного распределения
	shard := l.getShard(ip)
	key := ip + ":" + endpoint // Простая конкатенация вместо fmt.Sprintf
	now := time.Now()

	shard.mu.Lock()
	info, exists := shard.requests[key]
	if !exists {
		shard.requests[key] = &ipRequestInfo{
			count:     1,
			firstSeen: now,
		}
		shard.mu.Unlock()
		return rateLimitResult{allowed: true}
	}

	// Проверить блокировку
	if info.blocked {
		if now.Before(info.blockUntil) {
			remaining := info.blockUntil.Sub(now).Round(time.Second)
			shard.mu.Unlock()
			return rateLimitResult{
				allowed: false,
				reason:  fmt.Sprintf("too many requests, try again in %s", remaining),
			}
		}
		// Блокировка истекла — сбросить счётчик
		info.blocked = false
		info.count = 1
		info.firstSeen = now
		shard.mu.Unlock()
		return rateLimitResult{allowed: true}
	}

	// Сбросить счётчик если окно истекло
	if now.Sub(info.firstSeen) > windowDuration {
		info.count = 1
		info.firstSeen = now
		shard.mu.Unlock()
		return rateLimitResult{allowed: true}
	}

	// Увеличить счётчик
	info.count++
	currentCount := info.count

	// Проверить лимит
	if currentCount > maxRequests {
		info.blocked = true
		info.blockUntil = now.Add(blockDuration)
		shard.mu.Unlock()
		// Возвращаем данные для логирования ВНЕ lock
		return rateLimitResult{
			allowed:    false,
			reason:     fmt.Sprintf("too many requests, try again in %s", blockDuration),
			shouldLog:  true,
			ip:         ip,
			endpoint:   endpoint,
			count:      currentCount,
			maxRequest: maxRequests,
		}
	}

	shard.mu.Unlock()
	return rateLimitResult{allowed: true}
}

// checkIPRateLimit проверяет rate limit по IP для public endpoints (лимит по умолчанию)
func (l *shardedRateLimiter) checkIPRateLimit(ip, endpoint string) rateLimitResult {
	return l.checkIPRateLimitWithMax(ip, endpoint, maxRequestsPerWindow)
}

// cleanupOldEntries периодически очищает старые записи
func (l *shardedRateLimiter) cleanupOldEntries() {
	now := time.Now()
	for _, shard := range l.shards {
		shard.mu.Lock()
		for key, info := range shard.requests {
			// Удалить записи старше 1 часа
			if now.Sub(info.firstSeen) > time.Hour && !info.blocked {
				delete(shard.requests, key)
				continue
			}
			// Удалить истёкшие блокировки старше 1 часа
			if info.blocked && now.Sub(info.blockUntil) > time.Hour {
				delete(shard.requests, key)
			}
		}
		shard.mu.Unlock()
	}
}

// SetRateLimits allows configuring rate limits for testing.
func SetRateLimits(maxPublic, maxGeo int) {
	maxRequestsPerWindow = maxPublic
	maxGeoRequestsPerWindow = maxGeo
	maxAdminRequestsPerWindow = maxPublic // Use same limit for admin in tests
}

// InitRateLimiter инициализирует rate limiter'ы и запускает cleanup горутину.
// Должен вызываться из main() перед использованием middleware.
// Если cfg равен nil, используются значения по умолчанию.
func InitRateLimiter(cfg *config.RateLimitConfig) {
	if cleanupStarted.Swap(true) {
		// Уже инициализирован
		return
	}

	// Применяем конфигурацию если передана
	if cfg != nil {
		if cfg.PublicMaxRequests > 0 {
			maxRequestsPerWindow = cfg.PublicMaxRequests
		}
		if cfg.GeoMaxRequests > 0 {
			maxGeoRequestsPerWindow = cfg.GeoMaxRequests
		}
		if cfg.AdminMaxRequests > 0 {
			maxAdminRequestsPerWindow = cfg.AdminMaxRequests
		}
		if cfg.WindowDuration > 0 {
			windowDuration = cfg.WindowDuration
		}
		if cfg.BlockDuration > 0 {
			blockDuration = cfg.BlockDuration
		}
	}

	publicRateLimiter = newShardedRateLimiter()
	geoRateLimiter = newShardedRateLimiter()
	cleanupStop = make(chan struct{})

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
		if cleanupStop != nil {
			close(cleanupStop)
		}
	})
}

// RateLimiter creates middleware that limits API request rate
// Uses PostgreSQL-based rate limiting (no Redis required)
// SEC-003: Добавлен IP-based rate limiting для public endpoints
func RateLimiter(
	sessionManager *session.Manager,
	sessionAnalyzer *sessionApp.SessionAnalyzer,
) func(http.Handler) http.Handler {
	// Убеждаемся что rate limiter инициализирован (с дефолтами если не инициализирован)
	if !cleanupStarted.Load() {
		InitRateLimiter(nil)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Geo endpoints: higher rate limit (200/min) for autocomplete
			if strings.HasPrefix(r.URL.Path, "/api/v1/geo/") {
				clientIP := httputil.GetClientIP(r)
				result := geoRateLimiter.checkIPRateLimitWithMax(clientIP, "geo", maxGeoRequestsPerWindow)
				if !result.allowed {
					writeError(w, http.StatusTooManyRequests, result.reason)
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			// SEC-003: IP-based rate limiting для public paths (login, register, invitations)
			if isPublicPath(r.URL.Path, r.Method) {
				clientIP := httputil.GetClientIP(r)
				endpoint := r.URL.Path

				result := publicRateLimiter.checkIPRateLimit(clientIP, endpoint)
				// Логируем ВНЕ lock
				if result.shouldLog {
					slog.Warn("SEC-003: IP rate limited",
						slog.String("ip", result.ip),
						slog.String("endpoint", result.endpoint),
						slog.Int("count", result.count),
						slog.Int("max", result.maxRequest),
					)
				}
				if !result.allowed {
					writeError(w, http.StatusTooManyRequests, result.reason)
					return
				}

				next.ServeHTTP(w, r)
				return
			}

			// SEC-003: Admin paths rate limiting (более высокий лимит, но всё равно ограничен)
			if strings.HasPrefix(r.URL.Path, "/api/v1/admin/") {
				clientIP := httputil.GetClientIP(r)
				result := publicRateLimiter.checkIPRateLimitWithMax(clientIP, "admin", maxAdminRequestsPerWindow)
				// Логируем ВНЕ lock
				if result.shouldLog {
					slog.Warn("SEC-003: Admin endpoint rate limited",
						slog.String("ip", result.ip),
						slog.String("path", r.URL.Path),
					)
				}
				if !result.allowed {
					writeError(w, http.StatusTooManyRequests, result.reason)
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

			// Build endpoint key for tracking (простая конкатенация)
			endpoint := r.Method + " " + r.URL.Path

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
