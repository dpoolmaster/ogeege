package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestWriteErrorMapping(t *testing.T) {
	env := newTestEnv()
	handler := NewHandler(env.service, nil, nil, discardLogger(), "")

	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "email", err: ErrInvalidEmail, want: http.StatusUnprocessableEntity},
		{name: "пароль", err: ErrWeakPassword, want: http.StatusUnprocessableEntity},
		{name: "роль", err: ErrInvalidRole, want: http.StatusUnprocessableEntity},
		{name: "согласие", err: ErrConsentRequired, want: http.StatusUnprocessableEntity},
		{name: "занятый email", err: ErrEmailTaken, want: http.StatusConflict},
		{name: "учётные данные", err: ErrInvalidCredentials, want: http.StatusUnauthorized},
		{name: "не подтверждён", err: ErrEmailNotVerified, want: http.StatusForbidden},
		{name: "заблокирован", err: ErrAccountBlocked, want: http.StatusForbidden},
		{name: "лимит", err: ErrTooManyAttempts, want: http.StatusTooManyRequests},
		{name: "токен", err: ErrInvalidToken, want: http.StatusUnprocessableEntity},
		{name: "срок токена", err: ErrTokenExpired, want: http.StatusUnprocessableEntity},
		{name: "токен использован", err: ErrTokenUsed, want: http.StatusUnprocessableEntity},
		{name: "текущий пароль", err: ErrInvalidCurrentPassword, want: http.StatusUnprocessableEntity},
		{name: "внутренняя", err: errors.New("boom"), want: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, rec := newTestContext(http.MethodPost, "/", "")
			handler.writeError(c, tt.err)
			if rec.Code != tt.want {
				t.Fatalf("статус = %d, ожидался %d", rec.Code, tt.want)
			}
		})
	}
}

func TestHandlerBindErrors(t *testing.T) {
	env := newTestEnv()
	handler := NewHandler(env.service, nil, nil, discardLogger(), "")
	user := &User{ID: uuid.Must(uuid.NewV7()), Role: RoleStudent, Status: StatusActive}
	session := &Session{ID: uuid.Must(uuid.NewV7()), UserID: user.ID}

	tests := []struct {
		name string
		call func(c *gin.Context)
	}{
		{name: "регистрация", call: handler.Register},
		{name: "подтверждение", call: handler.VerifyEmail},
		{name: "повторная отправка", call: handler.ResendVerification},
		{name: "вход", call: handler.Login},
		{name: "запрос сброса", call: handler.RequestPasswordReset},
		{name: "сброс", call: handler.ResetPassword},
		{
			name: "смена пароля",
			call: func(c *gin.Context) {
				WithUser(c, user)
				WithSession(c, session)
				handler.ChangePassword(c)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, rec := newTestContext(http.MethodPost, "/", "{")
			tt.call(c)
			if rec.Code != http.StatusUnprocessableEntity {
				t.Fatalf("статус = %d, тело %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestRegisterHandlerRejections(t *testing.T) {
	env := newTestEnv()
	router := newTestRouter(env)

	tests := []struct {
		name string
		body string
		want int
	}{
		{
			name: "короткий пароль",
			body: `{"email":"student@example.ru","password":"short","role":"student","consents":[{"kind":"personal_data","granted":true}]}`,
			want: http.StatusUnprocessableEntity,
		},
		{
			name: "служебная роль",
			body: `{"email":"student@example.ru","password":"strong-password","role":"admin","consents":[{"kind":"personal_data","granted":true}]}`,
			want: http.StatusUnprocessableEntity,
		},
		{
			name: "без согласия",
			body: `{"email":"student@example.ru","password":"strong-password","role":"student"}`,
			want: http.StatusUnprocessableEntity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := performJSON(t, router, http.MethodPost, "/api/auth/register", tt.body)
			if rec.Code != tt.want {
				t.Fatalf("статус = %d, тело %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestVerifyEmailHandlerInvalidToken(t *testing.T) {
	env := newTestEnv()
	router := newTestRouter(env)

	rec := performJSON(t, router, http.MethodPost, "/api/auth/verify-email", `{"token":"invalid"}`)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("статус = %d", rec.Code)
	}
}

func TestResendVerificationHandlerBindError(t *testing.T) {
	env := newTestEnv()
	router := newTestRouter(env)

	rec := performJSON(t, router, http.MethodPost, "/api/auth/resend-verification", `{`)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("статус = %d", rec.Code)
	}
}

func TestLoginHandlerTooManyAttempts(t *testing.T) {
	env := newTestEnv()
	router := newTestRouter(env)
	user := env.register(t)
	env.verify(t, user)

	for i := 0; i < MaxLoginAttempts; i++ {
		rec := performJSON(t, router, http.MethodPost, "/api/auth/login", `{"email":"student@example.ru","password":"wrong-password"}`)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("попытка %d: статус %d", i+1, rec.Code)
		}
	}

	rec := performJSON(t, router, http.MethodPost, "/api/auth/login", `{"email":"student@example.ru","password":"strong-password"}`)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("статус = %d, тело %s", rec.Code, rec.Body.String())
	}
}

func TestLogoutWithoutSession(t *testing.T) {
	env := newTestEnv()
	handler := NewHandler(env.service, nil, nil, discardLogger(), "")

	c, rec := newTestContext(http.MethodPost, "/api/auth/logout", "{}")
	handler.Logout(c)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("статус = %d", rec.Code)
	}
}

func TestChangePasswordWithoutSession(t *testing.T) {
	env := newTestEnv()
	handler := NewHandler(env.service, nil, nil, discardLogger(), "")

	c, rec := newTestContext(http.MethodPost, "/api/auth/password/change", `{"current_password":"a","new_password":"b"}`)
	handler.ChangePassword(c)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("без пользователя: статус = %d", rec.Code)
	}

	c, rec = newTestContext(http.MethodPost, "/api/auth/password/change", `{"current_password":"a","new_password":"b"}`)
	WithUser(c, &User{ID: uuid.Must(uuid.NewV7()), Role: RoleStudent})
	handler.ChangePassword(c)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("без сессии: статус = %d", rec.Code)
	}
}

func TestChangePasswordWrongCurrent(t *testing.T) {
	env := newTestEnv()
	router := newTestRouter(env)
	user := env.register(t)
	env.verify(t, user)

	_, token, err := env.service.Login(t.Context(), "student@example.ru", "strong-password", "127.0.0.1", "agent")
	if err != nil {
		t.Fatalf("вход: %v", err)
	}

	rec := performJSONWithSession(t, router, http.MethodPost, "/api/auth/password/change", `{"current_password":"wrong-password","new_password":"new-strong-password"}`, token)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("статус = %d, тело %s", rec.Code, rec.Body.String())
	}
}

func TestEntityAndPrincipal(t *testing.T) {
	for _, role := range []Role{RoleStudent, RoleParent, RoleMethodist, RoleAdmin} {
		if !role.Valid() {
			t.Fatalf("роль %s должна быть валидной", role)
		}
	}
	if Role("unknown").Valid() {
		t.Fatal("неизвестная роль признана валидной")
	}
	if RoleMethodist.IsPublic() || RoleAdmin.IsPublic() {
		t.Fatal("служебная роль признана публичной")
	}

	c, _ := newTestContext(http.MethodGet, "/", "")
	if _, ok := PrincipalFrom(c); ok {
		t.Fatal("principal найден без пользователя")
	}
	if _, ok := CurrentSession(c); ok {
		t.Fatal("сессия найдена без контекста")
	}

	user := &User{ID: uuid.Must(uuid.NewV7()), Email: "student@example.ru", Role: RoleStudent, Status: StatusActive}
	WithUser(c, user)
	principal, ok := PrincipalFrom(c)
	if !ok || principal.ID != user.ID || principal.Role != RoleStudent {
		t.Fatalf("principal = %+v", principal)
	}
}

func TestTokenHelpers(t *testing.T) {
	token, hash, err := NewToken()
	if err != nil {
		t.Fatalf("токен: %v", err)
	}
	if token == "" || len(hash) != 32 {
		t.Fatal("некорректный токен")
	}
	if string(HashToken(token)) != string(hash) {
		t.Fatal("хеш токена не совпадает")
	}
}

func newTestContext(method, path, body string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(method, path, strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.RemoteAddr = "127.0.0.1:1234"

	return c, rec
}

type failingConsentRecorder struct{}

func (failingConsentRecorder) RecordRegistrationConsents(context.Context, uuid.UUID, []ConsentGrant, string) error {
	return errors.New("БД недоступна")
}

func TestRegisterWithConsentRecorderFailure(t *testing.T) {
	env := newTestEnv()
	handler := NewHandler(env.service, nil, failingConsentRecorder{}, discardLogger(), "")

	c, rec := newTestContext(http.MethodPost, "/api/auth/register", `{
		"email": "student@example.ru",
		"password": "strong-password",
		"role": "student",
		"consents": [{"kind": "personal_data", "granted": true}]
	}`)
	handler.Register(c)

	if rec.Code != http.StatusCreated {
		t.Fatalf("статус = %d, тело %s", rec.Code, rec.Body.String())
	}
}

func TestLogoutHandlerServiceError(t *testing.T) {
	env := newTestEnv()
	env.repo.revokeSessionErr = errors.New("БД недоступна")
	handler := NewHandler(env.service, nil, nil, discardLogger(), "")

	user := &User{ID: uuid.Must(uuid.NewV7()), Role: RoleStudent}
	session := &Session{ID: uuid.Must(uuid.NewV7()), UserID: user.ID}

	c, rec := newTestContext(http.MethodPost, "/api/auth/logout", "{}")
	WithUser(c, user)
	WithSession(c, session)
	handler.Logout(c)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("статус = %d, тело %s", rec.Code, rec.Body.String())
	}
}
