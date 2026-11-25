package httpserver

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// Recovery перехватывает паники и возвращает безопасный ответ об ошибке.
func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Error(
					"паника в обработчике",
					"request_id", RequestIDFromGin(c),
					"panic", fmt.Sprint(recovered),
					"stack", string(debug.Stack()),
				)
				AbortWithError(c, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка")
			}
		}()

		c.Next()
	}
}
