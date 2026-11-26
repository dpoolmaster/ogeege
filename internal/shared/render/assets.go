package render

import (
	_ "embed"
	"net/http"

	"github.com/gin-gonic/gin"
)

const stylesPath = "/static/app.css"

//go:embed assets/app.css
var appCSS []byte

// StylesHandler отдаёт общий CSS приложения.
func StylesHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "public, max-age=86400")
		c.Data(http.StatusOK, "text/css; charset=utf-8", appCSS)
	}
}

// StylesPath возвращает путь к общему CSS для подключения в layout.
func StylesPath() string {
	return stylesPath
}
