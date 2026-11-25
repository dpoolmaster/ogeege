package httpserver

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const readinessTimeout = 3 * time.Second

// Pinger проверяет доступность базы данных.
type Pinger interface {
	Ping(ctx context.Context) error
}

// ReadinessChecker проверяет готовность зависимости.
type ReadinessChecker interface {
	Check(ctx context.Context) error
}

// HealthHandler отвечает успехом, пока процесс работает.
func HealthHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}

// ReadyHandler проверяет БД и состояние миграций.
func ReadyHandler(pinger Pinger, checker ReadinessChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		if pinger == nil || checker == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not_ready",
				"reason": "проверки готовности не настроены",
			})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), readinessTimeout)
		defer cancel()

		if err := pinger.Ping(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not_ready",
				"reason": "база данных недоступна: " + err.Error(),
			})
			return
		}

		if err := checker.Check(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not_ready",
				"reason": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	}
}
