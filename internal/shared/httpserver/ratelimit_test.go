package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRateLimiterAllow(t *testing.T) {
	now := time.Date(2026, time.November, 26, 12, 0, 0, 0, time.UTC)
	limiter := NewRateLimiter(2, time.Minute)
	limiter.now = func() time.Time { return now }

	first := limiter.Allow("ip")
	second := limiter.Allow("ip")
	if !first || !second {
		t.Fatal("первые два запроса должны быть разрешены")
	}
	if limiter.Allow("ip") {
		t.Fatal("третий запрос должен быть отклонён")
	}

	now = now.Add(time.Minute)
	if !limiter.Allow("ip") {
		t.Fatal("после окна запрос должен быть разрешён")
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, time.November, 26, 12, 0, 0, 0, time.UTC)
	limiter := NewRateLimiter(1, time.Minute)
	limiter.now = func() time.Time { return now }

	router := gin.New()
	router.Use(limiter.Middleware(ClientIPKey))
	router.GET("/limited", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	first := httptest.NewRecorder()
	router.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/limited", nil))
	if first.Code != http.StatusOK {
		t.Fatalf("первый запрос: статус %d", first.Code)
	}

	second := httptest.NewRecorder()
	router.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/limited", nil))
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("второй запрос: статус %d", second.Code)
	}
	if !strings.Contains(second.Body.String(), "rate_limited") {
		t.Fatalf("тело ответа = %s", second.Body.String())
	}

	now = now.Add(time.Minute)
	third := httptest.NewRecorder()
	router.ServeHTTP(third, httptest.NewRequest(http.MethodGet, "/limited", nil))
	if third.Code != http.StatusOK {
		t.Fatalf("после окна: статус %d", third.Code)
	}
}
