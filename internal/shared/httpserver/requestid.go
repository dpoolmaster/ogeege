package httpserver

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"oge-ege/internal/shared/ids"
)

const requestIDHeader = "X-Request-Id"

type requestIDContextKey struct{}

// RequestID назначает или принимает идентификатор запроса.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := strings.TrimSpace(c.GetHeader(requestIDHeader))
		if _, err := uuid.Parse(id); err != nil {
			generated, genErr := ids.New()
			if genErr != nil {
				// UUID v4 нужен только как аварийный запас, если системный генератор недоступен.
				generated = uuid.New()
			}
			id = generated.String()
		}

		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), requestIDContextKey{}, id))
		c.Set("request_id", id)
		c.Header(requestIDHeader, id)
		c.Next()
	}
}

// RequestIDFromContext возвращает идентификатор запроса из контекста.
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDContextKey{}).(string)

	return id
}

// RequestIDFromGin возвращает идентификатор запроса из Gin-контекста.
func RequestIDFromGin(c *gin.Context) string {
	return c.GetString("request_id")
}
