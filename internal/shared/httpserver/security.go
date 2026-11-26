package httpserver

import "github.com/gin-gonic/gin"

const contentSecurityPolicy = "default-src 'self'; base-uri 'self'; frame-ancestors 'none'; form-action 'self'; img-src 'self' data:; font-src 'self' data:; style-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; script-src 'self'"

// SecurityHeaders устанавливает базовые заголовки безопасности.
func SecurityHeaders(env string) gin.HandlerFunc {
	return func(c *gin.Context) {
		headers := c.Writer.Header()
		headers.Set("X-Content-Type-Options", "nosniff")
		headers.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		headers.Set("X-Frame-Options", "DENY")
		headers.Set("Content-Security-Policy", contentSecurityPolicy)

		if env == "production" && (c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https") {
			headers.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		c.Next()
	}
}
