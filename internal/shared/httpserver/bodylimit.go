package httpserver

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// BodyLimit ограничивает размер тела запроса.
func BodyLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if maxBytes <= 0 {
			c.Next()
			return
		}

		if c.Request.ContentLength > maxBytes {
			AbortWithError(c, http.StatusRequestEntityTooLarge, "bad_request", "Размер тела запроса превышает допустимый")
			return
		}

		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}
