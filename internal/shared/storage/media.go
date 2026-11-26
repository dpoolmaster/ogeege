package storage

import (
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// MediaHandler отдаёт изображение по разрешённому серверному ключу.
func MediaHandler(store Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := strings.TrimPrefix(c.Param("key"), "/")
		if !ValidImageKey(key) {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		object, contentType, err := store.Get(c.Request.Context(), key)
		if err != nil {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		defer func() {
			_ = object.Close()
		}()

		c.Header("Content-Type", contentType)
		c.Header("Cache-Control", "private, max-age=3600")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Status(http.StatusOK)
		if _, err := io.Copy(c.Writer, object); err != nil {
			c.Abort()
		}
	}
}
