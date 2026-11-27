package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newHTMLRouter(env *testEnv) *gin.Engine {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(env.service, nil, nil, discardLogger(), "")
	router := gin.New()
	router.GET("/login", handler.ShowLogin)
	router.GET("/register", handler.ShowRegister)
	router.GET("/verify-email", handler.ShowVerifyEmail)
	router.GET("/password/reset", handler.ShowPasswordReset)
	router.GET("/account", AuthPage(env.service), handler.ShowAccount)
	router.GET("/logout", AuthPage(env.service), handler.ShowLogout)
	router.POST("/login", handler.SubmitLogin)
	router.POST("/register", handler.SubmitRegister)
	router.POST("/verify-email", handler.SubmitVerifyEmail)
	router.POST("/resend-verification", handler.SubmitResendVerification)
	router.POST("/password/reset-request", handler.SubmitPasswordResetRequest)
	router.POST("/password/reset", handler.SubmitPasswordReset)
	router.POST("/logout", AuthPage(env.service), handler.SubmitLogout)

	return router
}

func TestAuthPagesRender(t *testing.T) {
	env := newTestEnv()
	router := newHTMLRouter(env)

	tests := []struct {
		path string
		want string
	}{
		{path: "/login", want: "Вход"},
		{path: "/login?verified=1", want: "Email подтверждён"},
		{path: "/login?reset=1", want: "Пароль изменён"},
		{path: "/register", want: "Регистрация"},
		{path: "/verify-email?token=abc", want: "Подтвердить email"},
		{path: "/verify-email", want: "Отправить письмо повторно"},
		{path: "/password/reset?token=abc", want: "Сохранить пароль"},
		{path: "/password/reset", want: "Отправить ссылку"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if rec.Code != http.StatusOK {
				t.Fatalf("статус = %d", rec.Code)
			}
			if !strings.Contains(rec.Body.String(), tt.want) {
				t.Fatalf("страница не содержит %q: %s", tt.want, rec.Body.String())
			}
		})
	}
}

func TestRegisterForm(t *testing.T) {
	env := newTestEnv()
	router := newHTMLRouter(env)

	t.Run("успех", func(t *testing.T) {
		rec := performForm(t, router, "/register", url.Values{
			"email":         {"student@example.ru"},
			"password":      {"strong-password"},
			"role":          {"student"},
			"personal_data": {"true"},
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("статус = %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "Аккаунт создан") {
			t.Fatalf("тело = %s", rec.Body.String())
		}
	})

	t.Run("ошибка", func(t *testing.T) {
		rec := performForm(t, router, "/register", url.Values{
			"email":         {"bad"},
			"password":      {"strong-password"},
			"role":          {"student"},
			"personal_data": {"true"},
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("статус = %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "Некорректный адрес") {
			t.Fatalf("тело = %s", rec.Body.String())
		}
	})
}

func TestLoginForm(t *testing.T) {
	env := newTestEnv()
	router := newHTMLRouter(env)
	user := env.register(t)
	env.verify(t, user)

	t.Run("успех", func(t *testing.T) {
		rec := performForm(t, router, "/login", url.Values{
			"email":    {"student@example.ru"},
			"password": {"strong-password"},
		})
		if rec.Code != http.StatusSeeOther {
			t.Fatalf("статус = %d", rec.Code)
		}
		if rec.Header().Get("Location") != "/account" {
			t.Fatalf("Location = %q", rec.Header().Get("Location"))
		}
	})

	t.Run("ошибка", func(t *testing.T) {
		rec := performForm(t, router, "/login", url.Values{
			"email":    {"student@example.ru"},
			"password": {"wrong-password"},
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("статус = %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "Неверный email или пароль") {
			t.Fatalf("тело = %s", rec.Body.String())
		}
	})
}

func TestVerifyAndResetForms(t *testing.T) {
	env := newTestEnv()
	router := newHTMLRouter(env)
	env.register(t)
	verifyToken := tokenFromLastMessage(t, env.enqueuer)

	t.Run("подтверждение", func(t *testing.T) {
		rec := performForm(t, router, "/verify-email", url.Values{"token": {verifyToken}})
		if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/login?verified=1" {
			t.Fatalf("статус = %d, Location = %q", rec.Code, rec.Header().Get("Location"))
		}
	})

	t.Run("повторная отправка", func(t *testing.T) {
		rec := performForm(t, router, "/resend-verification", url.Values{"email": {"student@example.ru"}})
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Если аккаунт существует") {
			t.Fatalf("статус = %d, тело = %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("запрос сброса", func(t *testing.T) {
		rec := performForm(t, router, "/password/reset-request", url.Values{"email": {"student@example.ru"}})
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Если аккаунт существует") {
			t.Fatalf("статус = %d, тело = %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("применение сброса", func(t *testing.T) {
		resetToken := tokenFromLastMessage(t, env.enqueuer)
		rec := performForm(t, router, "/password/reset", url.Values{
			"token":        {resetToken},
			"new_password": {"new-strong-password"},
		})
		if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/login?reset=1" {
			t.Fatalf("статус = %d, Location = %q", rec.Code, rec.Header().Get("Location"))
		}
	})

	t.Run("ошибка сброса", func(t *testing.T) {
		rec := performForm(t, router, "/password/reset", url.Values{
			"token":        {"invalid"},
			"new_password": {"new-strong-password"},
		})
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Ссылка недействительна") {
			t.Fatalf("статус = %d, тело = %s", rec.Code, rec.Body.String())
		}
	})
}

func TestErrorMessage(t *testing.T) {
	tests := []struct {
		err  error
		want string
	}{
		{err: ErrInvalidEmail, want: "Некорректный"},
		{err: ErrWeakPassword, want: "10 символов"},
		{err: ErrEmailTaken, want: "уже зарегистрирован"},
		{err: ErrInvalidRole, want: "ученика или родителя"},
		{err: ErrConsentRequired, want: "согласие"},
		{err: ErrInvalidCredentials, want: "Неверный email"},
		{err: ErrEmailNotVerified, want: "Подтвердите"},
		{err: ErrAccountBlocked, want: "заблокирована"},
		{err: ErrTooManyAttempts, want: "Слишком много"},
		{err: ErrInvalidToken, want: "недействительна"},
		{err: ErrTokenExpired, want: "истёк"},
		{err: ErrTokenUsed, want: "использована"},
		{err: errors.New("другая"), want: "Попробуйте позже"},
	}

	for _, tt := range tests {
		if got := errorMessage(tt.err); !strings.Contains(got, tt.want) {
			t.Fatalf("errorMessage(%v) = %q, ожидалась подстрока %q", tt.err, got, tt.want)
		}
	}
}

func performForm(t *testing.T, router *gin.Engine, path string, values url.Values) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	return rec
}

func TestAccountPage(t *testing.T) {
	env := newTestEnv()
	router := newHTMLRouter(env)
	user := env.register(t)
	env.verify(t, user)

	t.Run("без сессии редирект на вход", func(t *testing.T) {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/account", nil))
		if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/login" {
			t.Fatalf("статус = %d, Location = %q", rec.Code, rec.Header().Get("Location"))
		}
	})

	t.Run("с сессией показывает данные и кнопку выхода", func(t *testing.T) {
		_, token, err := env.service.Login(t.Context(), "student@example.ru", "strong-password", "127.0.0.1", "agent")
		if err != nil {
			t.Fatalf("вход: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/account", nil)
		req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: token})
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("статус = %d", rec.Code)
		}
		body := rec.Body.String()
		for _, part := range []string{"student@example.ru", "Ученик", "Активен", "Выйти", `action="/logout"`} {
			if !strings.Contains(body, part) {
				t.Fatalf("страница не содержит %q: %s", part, body)
			}
		}
	})
}

func TestShowLogout(t *testing.T) {
	env := newTestEnv()
	router := newHTMLRouter(env)
	user := env.register(t)
	env.verify(t, user)

	t.Run("без сессии редирект на вход", func(t *testing.T) {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/logout", nil))
		if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/login" {
			t.Fatalf("статус = %d, Location = %q", rec.Code, rec.Header().Get("Location"))
		}
	})

	t.Run("с сессией показывает подтверждение", func(t *testing.T) {
		_, token, err := env.service.Login(t.Context(), "student@example.ru", "strong-password", "127.0.0.1", "agent")
		if err != nil {
			t.Fatalf("вход: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/logout", nil)
		req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: token})
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("статус = %d", rec.Code)
		}
		body := rec.Body.String()
		for _, part := range []string{"Выход", `action="/logout"`, "Выйти"} {
			if !strings.Contains(body, part) {
				t.Fatalf("страница не содержит %q: %s", part, body)
			}
		}
		if _, _, err := env.service.Authenticate(t.Context(), token); err != nil {
			t.Fatalf("GET /logout не должен отзывать сессию: %v", err)
		}
	})
}

func TestSubmitLogout(t *testing.T) {
	env := newTestEnv()
	router := newHTMLRouter(env)
	user := env.register(t)
	env.verify(t, user)

	_, token, err := env.service.Login(t.Context(), "student@example.ru", "strong-password", "127.0.0.1", "agent")
	if err != nil {
		t.Fatalf("вход: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/logout", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: token})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/login" {
		t.Fatalf("статус = %d, Location = %q", rec.Code, rec.Header().Get("Location"))
	}
	if _, _, err := env.service.Authenticate(t.Context(), token); err == nil {
		t.Fatal("сессия не отозвана после выхода")
	}
}

func TestAPIStaysJSON(t *testing.T) {
	env := newTestEnv()
	router := newTestRouter(env)
	user := env.register(t)
	env.verify(t, user)

	rec := performForm(t, router, "/api/auth/login", url.Values{
		"email":    {"student@example.ru"},
		"password": {"strong-password"},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("статус = %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
		t.Fatalf("API вернул Content-Type = %q", got)
	}
	if rec.Header().Get("Location") != "" {
		t.Fatalf("API выполнил редирект на %q", rec.Header().Get("Location"))
	}
}

func TestLoginAndRegisterRedirectWhenAuthenticated(t *testing.T) {
	env := newTestEnv()
	router := newHTMLRouter(env)
	user := env.register(t)
	env.verify(t, user)

	_, token, err := env.service.Login(t.Context(), "student@example.ru", "strong-password", "127.0.0.1", "agent")
	if err != nil {
		t.Fatalf("вход: %v", err)
	}

	for _, path := range []string{"/login", "/register"} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: token})
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/account" {
				t.Fatalf("статус = %d, Location = %q", rec.Code, rec.Header().Get("Location"))
			}
		})
	}

	t.Run("без сессии страницы доступны", func(t *testing.T) {
		for _, path := range []string{"/login", "/register"} {
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
			if rec.Code != http.StatusOK {
				t.Fatalf("%s: статус = %d", path, rec.Code)
			}
		}
	})
}
