package httpserver

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestLogger пишет одну структурированную запись на каждый запрос.
func RequestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		logger.Info(
			"http-запрос",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", RequestIDFromGin(c),
			"bytes", c.Writer.Size(),
		)
	}
}
