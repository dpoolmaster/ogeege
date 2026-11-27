package render

import (
	"embed"
	"mime"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	stylesPath   = "/static/app.css"
	fontsPrefix  = "/static/fonts/"
	faviconPath  = "/favicon.svg"
	fontsDirName = "fonts"
)

//go:embed assets
var assetsFS embed.FS

// StylesHandler отдаёт общий CSS приложения.
func StylesHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		data, err := assetsFS.ReadFile("assets/app.css")
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.Header("Cache-Control", "public, max-age=86400")
		c.Data(http.StatusOK, "text/css; charset=utf-8", data)
	}
}

// FontsHandler отдаёт self-hosted шрифты.
func FontsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		name := path.Clean(strings.TrimPrefix(c.Param("file"), "/"))
		if name == "." || strings.HasPrefix(name, "..") || strings.Contains(name, "/") {
			c.Status(http.StatusNotFound)
			return
		}

		data, err := assetsFS.ReadFile("assets/" + fontsDirName + "/" + name)
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}

		contentType := mime.TypeByExtension(path.Ext(name))
		if contentType == "" {
			contentType = "font/woff2"
		}
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Data(http.StatusOK, contentType, data)
	}
}

// FaviconHandler отдаёт favicon в формате SVG.
func FaviconHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		data, err := assetsFS.ReadFile("assets/favicon.svg")
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.Header("Cache-Control", "public, max-age=86400")
		c.Data(http.StatusOK, "image/svg+xml", data)
	}
}

// StylesPath возвращает путь к общему CSS для подключения в layout.
func StylesPath() string {
	return stylesPath
}

// FontsPath возвращает путь к self-hosted шрифту.
func FontsPath(name string) string {
	return fontsPrefix + name
}

// FaviconPath возвращает путь к favicon.
func FaviconPath() string {
	return faviconPath
}
