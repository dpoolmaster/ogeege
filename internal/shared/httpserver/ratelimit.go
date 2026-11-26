package httpserver

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const rateLimitCleanupThreshold = 4096

type rateLimitEntry struct {
	windowStart time.Time
	count       int
}

// RateLimiter ограничивает число запросов в фиксированном окне.
type RateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	now     func() time.Time
	entries map[string]rateLimitEntry
}

// NewRateLimiter создаёт ограничитель с лимитом на окно.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		limit:   limit,
		window:  window,
		now:     time.Now,
		entries: make(map[string]rateLimitEntry),
	}
}

// Allow сообщает, разрешён ли запрос с указанным ключом.
func (l *RateLimiter) Allow(key string) bool {
	if l.limit <= 0 || l.window <= 0 {
		return true
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	entry, ok := l.entries[key]
	if !ok || now.Sub(entry.windowStart) >= l.window {
		l.entries[key] = rateLimitEntry{windowStart: now, count: 1}
		l.cleanup(now)
		return true
	}
	if entry.count >= l.limit {
		return false
	}

	entry.count++
	l.entries[key] = entry

	return true
}

// Middleware ограничивает запросы по ключу из keyFunc.
func (l *RateLimiter) Middleware(keyFunc func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !l.Allow(keyFunc(c)) {
			AbortWithError(c, http.StatusTooManyRequests, "rate_limited", "Слишком много запросов, попробуйте позже")
			return
		}

		c.Next()
	}
}

func (l *RateLimiter) cleanup(now time.Time) {
	if len(l.entries) < rateLimitCleanupThreshold {
		return
	}

	for key, entry := range l.entries {
		if now.Sub(entry.windowStart) >= l.window {
			delete(l.entries, key)
		}
	}
}

// ClientIPKey использует IP клиента как ключ ограничения.
func ClientIPKey(c *gin.Context) string {
	return c.ClientIP()
}
