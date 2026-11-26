package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newTestRouter(env *testEnv) *gin.Engine {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(env.service, nil, nil, discardLogger())
	router := gin.New()
	router.POST("/api/auth/register", handler.Register)
	router.POST("/api/auth/verify-email", handler.VerifyEmail)
	router.POST("/api/auth/resend-verification", handler.ResendVerification)
	router.POST("/api/auth/login", handler.Login)
	router.POST("/api/auth/password/reset-request", handler.RequestPasswordReset)
	router.POST("/api/auth/password/reset", handler.ResetPassword)

	protected := router.Group("/api/auth", Auth(env.service))
	protected.POST("/logout", handler.Logout)
	protected.POST("/password/change", handler.ChangePassword)

	return router
}

func TestRegisterHandler(t *testing.T) {
	env := newTestEnv()
	router := newTestRouter(env)

	t.Run("успех", func(t *testing.T) {
		rec := performJSON(t, router, http.MethodPost, "/api/auth/register", `{
			"email": "student@example.ru",
			"password": "strong-password",
			"role": "student",
			"consents": [{"kind": "personal_data", "granted": true}]
		}`)
		if rec.Code != http.StatusCreated {
			t.Fatalf("статус = %d, тело %s", rec.Code, rec.Body.String())
		}

		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("ответ не JSON: %v", err)
		}
		if body["status"] != string(StatusUnverified) {
			t.Fatalf("статус в ответе = %v", body["status"])
		}
	})

	t.Run("некорректный email", func(t *testing.T) {
		rec := performJSON(t, router, http.MethodPost, "/api/auth/register", `{
			"email": "bad",
			"password": "strong-password",
			"role": "student",
			"consents": [{"kind": "personal_data", "granted": true}]
		}`)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("статус = %d", rec.Code)
		}
	})

	t.Run("занятый email", func(t *testing.T) {
		rec := performJSON(t, router, http.MethodPost, "/api/auth/register", `{
			"email": "student@example.ru",
			"password": "strong-password",
			"role": "student",
			"consents": [{"kind": "personal_data", "granted": true}]
		}`)
		if rec.Code != http.StatusConflict {
			t.Fatalf("статус = %d", rec.Code)
		}
	})
}

func TestVerifyEmailHandler(t *testing.T) {
	env := newTestEnv()
	router := newTestRouter(env)
	env.register(t)
	token := tokenFromLastMessage(t, env.enqueuer)

	rec := performJSON(t, router, http.MethodPost, "/api/auth/verify-email", `{"token":"`+token+`"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("статус = %d, тело %s", rec.Code, rec.Body.String())
	}
}

func TestResendVerificationHandlerIsNeutral(t *testing.T) {
	env := newTestEnv()
	router := newTestRouter(env)

	for _, email := range []string{"student@example.ru", "nobody@example.ru"} {
		rec := performJSON(t, router, http.MethodPost, "/api/auth/resend-verification", `{"email":"`+email+`"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("статус для %s = %d", email, rec.Code)
		}
		if rec.Body.String() != `{"status":"accepted"}` {
			t.Fatalf("ответ для %s = %s", email, rec.Body.String())
		}
	}
}

func TestLoginHandler(t *testing.T) {
	env := newTestEnv()
	router := newTestRouter(env)
	user := env.register(t)

	t.Run("неподтверждённый", func(t *testing.T) {
		rec := performJSON(t, router, http.MethodPost, "/api/auth/login", `{"email":"student@example.ru","password":"strong-password"}`)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("статус = %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "email_not_verified") {
			t.Fatalf("тело = %s", rec.Body.String())
		}
	})

	env.verify(t, user)

	t.Run("успех", func(t *testing.T) {
		rec := performJSON(t, router, http.MethodPost, "/api/auth/login", `{"email":"student@example.ru","password":"strong-password"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("статус = %d, тело %s", rec.Code, rec.Body.String())
		}

		cookies := rec.Result().Cookies()
		if len(cookies) == 0 {
			t.Fatal("сессионная cookie не установлена")
		}
		cookie := cookies[0]
		if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode {
			t.Fatalf("параметры cookie: HttpOnly=%t Secure=%t SameSite=%v", cookie.HttpOnly, cookie.Secure, cookie.SameSite)
		}
	})

	t.Run("неверные данные", func(t *testing.T) {
		rec := performJSON(t, router, http.MethodPost, "/api/auth/login", `{"email":"student@example.ru","password":"wrong-password"}`)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("статус = %d", rec.Code)
		}
		if strings.Contains(rec.Body.String(), "не существует") {
			t.Fatal("ответ раскрывает существование аккаунта")
		}
	})
}

func TestPasswordResetHandlers(t *testing.T) {
	env := newTestEnv()
	router := newTestRouter(env)
	user := env.register(t)
	env.verify(t, user)

	rec := performJSON(t, router, http.MethodPost, "/api/auth/password/reset-request", `{"email":"student@example.ru"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("запрос сброса: статус %d", rec.Code)
	}
	token := tokenFromLastMessage(t, env.enqueuer)

	rec = performJSON(t, router, http.MethodPost, "/api/auth/password/reset", `{"token":"`+token+`","new_password":"new-strong-password"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("сброс: статус %d, тело %s", rec.Code, rec.Body.String())
	}
}

func TestLogoutAndChangePasswordHandlers(t *testing.T) {
	env := newTestEnv()
	router := newTestRouter(env)
	user := env.register(t)
	env.verify(t, user)

	_, token, err := env.service.Login(t.Context(), "student@example.ru", "strong-password", "127.0.0.1", "agent")
	if err != nil {
		t.Fatalf("вход: %v", err)
	}

	rec := performJSONWithSession(t, router, http.MethodPost, "/api/auth/password/change", `{"current_password":"strong-password","new_password":"new-strong-password"}`, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("смена пароля: статус %d, тело %s", rec.Code, rec.Body.String())
	}

	rec = performJSONWithSession(t, router, http.MethodPost, "/api/auth/logout", `{}`, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("выход: статус %d, тело %s", rec.Code, rec.Body.String())
	}

	if _, _, err := env.service.Authenticate(t.Context(), token); err == nil {
		t.Fatal("сессия не отозвана после выхода")
	}
}

func performJSON(t *testing.T, router *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	return rec
}

func performJSONWithSession(t *testing.T, router *gin.Engine, method, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: token})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	return rec
}
