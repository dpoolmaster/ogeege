package publicsite

import (
	"embed"
	"mime"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed assets/*
var assetsFS embed.FS

const assetsPrefix = "/static/public/"

// AssetsHandler отдаёт статические ресурсы публичного контура.
func AssetsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		name := strings.TrimPrefix(c.Param("file"), "/")
		name = path.Clean(name)
		if name == "." || strings.HasPrefix(name, "..") {
			c.Status(http.StatusNotFound)
			return
		}

		data, err := assetsFS.ReadFile("assets/" + name)
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}

		contentType := mime.TypeByExtension(path.Ext(name))
		if contentType == "" {
			contentType = http.DetectContentType(data)
		}
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Data(http.StatusOK, contentType, data)
	}
}

// AssetsPrefix возвращает префикс статических ресурсов публичного контура.
func AssetsPrefix() string {
	return assetsPrefix
}
