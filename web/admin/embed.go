// Package webadmin встраивает собранное Vue-приложение админки.
package webadmin

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// FS содержит собранные файлы SPA.
//
//go:embed dist
var FS embed.FS

// Handler отдаёт статические файлы SPA и возвращает index.html для клиентских маршрутов.
func Handler() gin.HandlerFunc {
	sub, err := fs.Sub(FS, "dist")
	if err != nil {
		return func(c *gin.Context) {
			c.Status(http.StatusInternalServerError)
		}
	}

	fileServer := http.FileServer(http.FS(sub))

	return func(c *gin.Context) {
		path := strings.TrimPrefix(c.Param("path"), "/")
		if path != "" {
			if info, err := fs.Stat(sub, path); err == nil && !info.IsDir() {
				c.Request.URL.Path = "/" + path
				fileServer.ServeHTTP(c.Writer, c.Request)
				return
			}
		}

		index, err := fs.ReadFile(sub, "index.html")
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		c.Data(http.StatusOK, "text/html; charset=utf-8", index)
	}
}
