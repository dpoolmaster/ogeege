package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRequireRole(t *testing.T) {
	tests := []struct {
		name       string
		role       Role
		wantStatus int
	}{
		{name: "ученик", role: RoleStudent, wantStatus: http.StatusForbidden},
		{name: "родитель", role: RoleParent, wantStatus: http.StatusForbidden},
		{name: "методист", role: RoleMethodist, wantStatus: http.StatusForbidden},
		{name: "администратор", role: RoleAdmin, wantStatus: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.GET("/admin",
				func(c *gin.Context) {
					c.Set(userContextKey, &User{Role: tt.role})
					c.Next()
				},
				RequireRole(RoleAdmin),
				func(c *gin.Context) { c.Status(http.StatusOK) },
			)

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin", nil))

			if rec.Code != tt.wantStatus {
				t.Fatalf("статус = %d, ожидался %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestRequireRoleWithoutAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/admin", RequireRole(RoleAdmin), func(c *gin.Context) { c.Status(http.StatusOK) })

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("статус = %d", rec.Code)
	}
}

func TestAuthMiddleware(t *testing.T) {
	env := newTestEnv()
	user := env.register(t)
	env.verify(t, user)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/protected", Auth(env.service), func(c *gin.Context) { c.Status(http.StatusOK) })

	_, token, err := env.service.Login(t.Context(), "student@example.ru", "strong-password", "127.0.0.1", "agent")
	if err != nil {
		t.Fatalf("вход: %v", err)
	}

	t.Run("действующая сессия", func(t *testing.T) {
		rec := performWithCookie(t, router, token)
		if rec.Code != http.StatusOK {
			t.Fatalf("статус = %d", rec.Code)
		}
	})

	t.Run("истёкшая сессия", func(t *testing.T) {
		env.clock.Advance(SessionDuration + time.Minute)
		rec := performWithCookie(t, router, token)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("статус = %d", rec.Code)
		}
		env.clock.Advance(-(SessionDuration + time.Minute))
	})

	t.Run("отозванная сессия", func(t *testing.T) {
		session, err := env.repo.GetSessionByTokenHash(t.Context(), nil, HashToken(token))
		if err != nil {
			t.Fatalf("сессия: %v", err)
		}
		if err := env.service.Logout(t.Context(), session.ID); err != nil {
			t.Fatalf("выход: %v", err)
		}
		rec := performWithCookie(t, router, token)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("статус = %d", rec.Code)
		}
	})

	t.Run("без cookie", func(t *testing.T) {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/protected", nil))
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("статус = %d", rec.Code)
		}
	})
}

func TestSessionCookieAttributes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	SetSessionCookie(c, "session-token")

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookie = %d", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Name != SessionCookieName {
		t.Fatalf("имя = %q", cookie.Name)
	}
	if !cookie.HttpOnly {
		t.Fatal("cookie не HttpOnly")
	}
	if !cookie.Secure {
		t.Fatal("cookie не Secure")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("SameSite = %v", cookie.SameSite)
	}
	if cookie.Path != "/" {
		t.Fatalf("Path = %q", cookie.Path)
	}
}

func performWithCookie(t *testing.T, router *gin.Engine, token string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: token})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	return rec
}

func TestMethodistDeniedAdminCapabilities(t *testing.T) {
	gin.SetMode(gin.TestMode)

	routes := []string{"/admin/users/roles", "/admin/billing"}
	for _, route := range routes {
		t.Run(route, func(t *testing.T) {
			router := gin.New()
			router.GET(route,
				func(c *gin.Context) {
					c.Set(userContextKey, &User{Role: RoleMethodist})
					c.Next()
				},
				RequireRole(RoleAdmin),
				func(c *gin.Context) { c.Status(http.StatusOK) },
			)

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, route, nil))
			if rec.Code != http.StatusForbidden {
				t.Fatalf("методист получил статус %d", rec.Code)
			}
		})
	}
}

func TestOptionalAuth(t *testing.T) {
	env := newTestEnv()
	user := env.register(t)
	env.verify(t, user)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/public", OptionalAuth(env.service), func(c *gin.Context) {
		if _, ok := CurrentUser(c); ok {
			c.String(http.StatusOK, "user")
			return
		}
		c.String(http.StatusOK, "anonymous")
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/public", nil))
	if rec.Code != http.StatusOK || rec.Body.String() != "anonymous" {
		t.Fatalf("аноним: %d %q", rec.Code, rec.Body.String())
	}

	_, token, err := env.service.Login(t.Context(), "student@example.ru", "strong-password", "127.0.0.1", "agent")
	if err != nil {
		t.Fatalf("вход: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/public", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: token})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Body.String() != "user" {
		t.Fatalf("аутентификация: %d %q", rec.Code, rec.Body.String())
	}
}
