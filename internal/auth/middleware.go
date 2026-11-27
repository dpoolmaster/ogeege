package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"oge-ege/internal/shared/httpserver"
)

const (
	// SessionCookieName — имя cookie сессии.
	SessionCookieName = "session"
	userContextKey    = "auth_user"
	sessionContextKey = "auth_session"
)

// SetSessionCookie устанавливает сессионную cookie.
func SetSessionCookie(c *gin.Context, token string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(SessionDuration.Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearSessionCookie очищает сессионную cookie.
func ClearSessionCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

// SessionTokenFromRequest возвращает токен сессии из cookie.
func SessionTokenFromRequest(c *gin.Context) string {
	token, err := c.Cookie(SessionCookieName)
	if err != nil {
		return ""
	}

	return token
}

// Auth проверяет сессионную cookie и кладёт пользователя в контекст.
func Auth(service *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, session, err := service.Authenticate(c.Request.Context(), SessionTokenFromRequest(c))
		if err != nil {
			httpserver.AbortWithError(c, http.StatusUnauthorized, "unauthorized", "Требуется вход в систему")
			return
		}

		WithUser(c, user)
		WithSession(c, session)
		c.Next()
	}
}

// OptionalAuth распознаёт сессию для публичных страниц, не прерывая анонимный запрос.
func OptionalAuth(service *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, session, err := service.Authenticate(c.Request.Context(), SessionTokenFromRequest(c))
		if err == nil {
			WithUser(c, user)
			WithSession(c, session)
		}

		c.Next()
	}
}

// AuthPage проверяет сессию для HTML-страниц и при её отсутствии перенаправляет на вход.
func AuthPage(service *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, session, err := service.Authenticate(c.Request.Context(), SessionTokenFromRequest(c))
		if err != nil {
			c.Redirect(http.StatusSeeOther, "/login")
			c.Abort()
			return
		}

		WithUser(c, user)
		WithSession(c, session)
		c.Next()
	}
}

// RequireRole проверяет, что роль пользователя входит в допустимый набор.
func RequireRole(roles ...Role) gin.HandlerFunc {
	allowed := make(map[Role]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}

	return func(c *gin.Context) {
		user, ok := CurrentUser(c)
		if !ok {
			httpserver.AbortWithError(c, http.StatusUnauthorized, "unauthorized", "Требуется вход в систему")
			return
		}
		if _, ok := allowed[user.Role]; !ok {
			httpserver.AbortWithError(c, http.StatusForbidden, "forbidden", "Недостаточно прав")
			return
		}

		c.Next()
	}
}

// WithUser кладёт пользователя в контекст запроса.
func WithUser(c *gin.Context, user *User) {
	c.Set(userContextKey, user)
}

// WithSession кладёт сессию в контекст запроса.
func WithSession(c *gin.Context, session *Session) {
	c.Set(sessionContextKey, session)
}

// CurrentUser возвращает пользователя из контекста запроса.
func CurrentUser(c *gin.Context) (*User, bool) {
	value, ok := c.Get(userContextKey)
	if !ok {
		return nil, false
	}

	user, ok := value.(*User)

	return user, ok
}

// CurrentSession возвращает сессию из контекста запроса.
func CurrentSession(c *gin.Context) (*Session, bool) {
	value, ok := c.Get(sessionContextKey)
	if !ok {
		return nil, false
	}

	session, ok := value.(*Session)

	return session, ok
}

// PrincipalFrom возвращает минимальные данные пользователя из контекста.
func PrincipalFrom(c *gin.Context) (Principal, bool) {
	user, ok := CurrentUser(c)
	if !ok {
		return Principal{}, false
	}

	return Principal{
		ID:     user.ID,
		Email:  user.Email,
		Role:   user.Role,
		Status: user.Status,
	}, true
}
