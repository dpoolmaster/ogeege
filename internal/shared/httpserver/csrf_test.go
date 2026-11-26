package httpserver

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCSRFMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	csrf := NewCSRF("test-secret", false, func(c *gin.Context) string {
		return c.GetHeader("X-Subject")
	})

	router := gin.New()
	router.Use(csrf.Middleware())
	router.GET("/token", func(c *gin.Context) {
		c.String(http.StatusOK, csrf.Token(c))
	})
	router.POST("/submit", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	token, cookie := fetchCSRFToken(t, router, "session-1")

	t.Run("действующий токен", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(""))
		req.Header.Set("X-Subject", "session-1")
		req.Header.Set(csrfHeaderName, token)
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("статус = %d, тело %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("без токена", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/submit", nil)
		req.Header.Set("X-Subject", "session-1")
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("статус = %d", rec.Code)
		}
	})

	t.Run("чужой токен", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/submit", nil)
		req.Header.Set("X-Subject", "session-1")
		req.Header.Set(csrfHeaderName, "foreign")
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("статус = %d", rec.Code)
		}
	})

	t.Run("токен другой сессии", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/submit", nil)
		req.Header.Set("X-Subject", "session-2")
		req.Header.Set(csrfHeaderName, token)
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("статус = %d", rec.Code)
		}
	})
}

func fetchCSRFToken(t *testing.T, router *gin.Engine, subject string) (string, *http.Cookie) {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/token", nil)
	req.Header.Set("X-Subject", subject)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("получение токена: статус %d", rec.Code)
	}

	body, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatalf("прочитать тело: %v", err)
	}
	token := strings.TrimSpace(string(body))
	if token == "" {
		t.Fatal("пустой CSRF-токен")
	}

	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == csrfCookieName {
			return token, cookie
		}
	}

	t.Fatal("cookie CSRF не установлена")

	return "", nil
}
