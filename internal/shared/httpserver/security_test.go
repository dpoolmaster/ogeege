package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSecurityHeadersCSP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(SecurityHeaders("test"))
	router.GET("/", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	policy := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(policy, "font-src 'self' data:") {
		t.Fatalf("в CSP нет font-src: %q", policy)
	}
	if !strings.Contains(policy, "script-src 'self'") {
		t.Fatalf("в CSP нет script-src 'self': %q", policy)
	}
}
