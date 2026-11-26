package httpserver

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	csrfCookieName    = "csrf_token"
	csrfHeaderName    = "X-CSRF-Token"
	csrfFormField     = "csrf_token"
	csrfAnonymous     = "anonymous"
	csrfTokenLifetime = 30 * 24 * time.Hour
	csrfTokenKey      = "csrf_token_value"
)

// CSRF реализует double-submit-защиту с HMAC-подписью токена.
type CSRF struct {
	secret  []byte
	secure  bool
	subject func(*gin.Context) string
}

// NewCSRF создаёт middleware CSRF.
// subject возвращает идентификатор сессии; пустой результат означает анонимный запрос.
func NewCSRF(secret string, secure bool, subject func(*gin.Context) string) *CSRF {
	return &CSRF{
		secret:  []byte(secret),
		secure:  secure,
		subject: subject,
	}
}

// Middleware проверяет токен для изменяющих запросов и выдаёт его для безопасных.
func (m *CSRF) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		subject := m.subjectOf(c)

		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			m.Token(c)
			c.Next()
			return
		}

		cookieToken, err := c.Cookie(csrfCookieName)
		if err != nil || cookieToken == "" {
			AbortWithError(c, http.StatusForbidden, "forbidden", "Недействительный CSRF-токен")
			return
		}

		submitted := strings.TrimSpace(c.GetHeader(csrfHeaderName))
		if submitted == "" {
			_ = c.Request.ParseForm()
			submitted = strings.TrimSpace(c.Request.PostForm.Get(csrfFormField))
		}

		if submitted == "" ||
			subtle.ConstantTimeCompare([]byte(cookieToken), []byte(submitted)) != 1 ||
			!m.valid(cookieToken, subject) {
			AbortWithError(c, http.StatusForbidden, "forbidden", "Недействительный CSRF-токен")
			return
		}

		c.Set(csrfTokenKey, cookieToken)
		c.Next()
	}
}

// Token возвращает действующий CSRF-токен и при необходимости устанавливает cookie.
func (m *CSRF) Token(c *gin.Context) string {
	if token, ok := c.Get(csrfTokenKey); ok {
		if value, ok := token.(string); ok && value != "" {
			return value
		}
	}

	subject := m.subjectOf(c)

	if cookieToken, err := c.Cookie(csrfCookieName); err == nil && m.valid(cookieToken, subject) {
		c.Set(csrfTokenKey, cookieToken)
		return cookieToken
	}

	token := m.generate(subject)
	c.SetCookie(csrfCookieName, token, int(csrfTokenLifetime.Seconds()), "/", "", m.secure, false)
	c.Set(csrfTokenKey, token)

	return token
}

// Rotate выдаёт новый токен; вызывается после смены сессии.
func (m *CSRF) Rotate(c *gin.Context) string {
	return m.RotateFor(c, m.subjectOf(c))
}

// RotateFor выдаёт токен, привязанный к указанному субъекту.
func (m *CSRF) RotateFor(c *gin.Context, subject string) string {
	if strings.TrimSpace(subject) == "" {
		subject = csrfAnonymous
	}

	token := m.generate(subject)
	c.SetCookie(csrfCookieName, token, int(csrfTokenLifetime.Seconds()), "/", "", m.secure, false)
	c.Set(csrfTokenKey, token)

	return token
}

func (m *CSRF) subjectOf(c *gin.Context) string {
	if m.subject == nil {
		return csrfAnonymous
	}
	if value := strings.TrimSpace(m.subject(c)); value != "" {
		return value
	}

	return csrfAnonymous
}

func (m *CSRF) generate(subject string) string {
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		// crypto/rand не должен отказывать; при ошибке токен всё равно остаётся непредсказуемым
		// за счёт HMAC и будет отклонён при следующей проверке.
		return ""
	}

	nonce := base64.RawURLEncoding.EncodeToString(random)

	return nonce + "." + m.sign(subject, nonce)
}

func (m *CSRF) sign(subject, nonce string) string {
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(subject))
	mac.Write([]byte("|"))
	mac.Write([]byte(nonce))

	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (m *CSRF) valid(token, subject string) bool {
	nonce, signature, found := strings.Cut(token, ".")
	if !found || nonce == "" || signature == "" {
		return false
	}

	expected := m.sign(subject, nonce)

	return subtle.ConstantTimeCompare([]byte(signature), []byte(expected)) == 1
}
