package auth

import (
	"errors"
	"net/http"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"

	"oge-ege/internal/shared/render"
)

// PageData содержит данные Templ-страниц аутентификации.
type PageData struct {
	Brand       string
	CSRF        string
	Error       string
	Message     string
	Email       string
	Token       string
	RoleLabel   string
	StatusLabel string
}

// ShowLogin обрабатывает GET /login.
func (h *Handler) ShowLogin(c *gin.Context) {
	if h.redirectIfAuthenticated(c) {
		return
	}

	data := PageData{Brand: h.brand, CSRF: h.csrfToken(c)}
	if c.Query("verified") != "" {
		data.Message = "Email подтверждён. Войдите в систему."
	}
	if c.Query("reset") != "" {
		data.Message = "Пароль изменён. Войдите в систему."
	}
	h.render(c, http.StatusOK, LoginPage(data))
}

// ShowRegister обрабатывает GET /register.
func (h *Handler) ShowRegister(c *gin.Context) {
	if h.redirectIfAuthenticated(c) {
		return
	}

	h.render(c, http.StatusOK, RegisterPage(PageData{Brand: h.brand, CSRF: h.csrfToken(c)}))
}

// ShowVerifyEmail обрабатывает GET /verify-email.
func (h *Handler) ShowVerifyEmail(c *gin.Context) {
	h.render(c, http.StatusOK, VerifyEmailPage(PageData{
		Brand: h.brand,
		CSRF:  h.csrfToken(c),
		Token: c.Query("token"),
	}))
}

// ShowAccount обрабатывает GET /account.
func (h *Handler) ShowAccount(c *gin.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	h.render(c, http.StatusOK, AccountPage(PageData{
		Brand:       h.brand,
		CSRF:        h.csrfToken(c),
		Email:       user.Email,
		RoleLabel:   roleLabel(user.Role),
		StatusLabel: statusLabel(user.Status),
	}))
}

// ShowLogout обрабатывает GET /logout: показывает подтверждение выхода без изменения состояния.
func (h *Handler) ShowLogout(c *gin.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	h.render(c, http.StatusOK, LogoutPage(PageData{
		Brand: h.brand,
		CSRF:  h.csrfToken(c),
		Email: user.Email,
	}))
}

// ShowPasswordReset обрабатывает GET /password/reset.
func (h *Handler) ShowPasswordReset(c *gin.Context) {
	h.render(c, http.StatusOK, PasswordResetPage(PageData{
		Brand: h.brand,
		CSRF:  h.csrfToken(c),
		Token: c.Query("token"),
	}))
}

func (h *Handler) render(c *gin.Context, status int, component templ.Component) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(status)
	if err := render.Component(c.Request.Context(), c.Writer, component); err != nil {
		h.logger.Error("не удалось отрендерить страницу", "error", err)
	}
}

// redirectIfAuthenticated уводит вошедшего пользователя на страницу аккаунта.
func (h *Handler) redirectIfAuthenticated(c *gin.Context) bool {
	if _, _, err := h.service.Authenticate(c.Request.Context(), SessionTokenFromRequest(c)); err != nil {
		return false
	}

	c.Redirect(http.StatusSeeOther, "/account")

	return true
}

func (h *Handler) csrfToken(c *gin.Context) string {
	if h.csrf == nil {
		return ""
	}

	return h.csrf.Token(c)
}

func roleLabel(role Role) string {
	switch role {
	case RoleStudent:
		return "Ученик"
	case RoleParent:
		return "Родитель"
	case RoleMethodist:
		return "Методист"
	case RoleAdmin:
		return "Администратор"
	default:
		return string(role)
	}
}

func statusLabel(status Status) string {
	switch status {
	case StatusUnverified:
		return "Email не подтверждён"
	case StatusActive:
		return "Активен"
	case StatusBlocked:
		return "Заблокирован"
	case StatusAnonymized:
		return "Удалён"
	default:
		return string(status)
	}
}

func errorMessage(err error) string {
	switch {
	case errors.Is(err, ErrInvalidEmail):
		return "Некорректный адрес электронной почты."
	case errors.Is(err, ErrWeakPassword):
		return "Пароль должен содержать не менее 10 символов."
	case errors.Is(err, ErrEmailTaken):
		return "Email уже зарегистрирован."
	case errors.Is(err, ErrInvalidRole):
		return "Выберите ученика или родителя."
	case errors.Is(err, ErrConsentRequired):
		return "Необходимо согласие на обработку персональных данных."
	case errors.Is(err, ErrInvalidCredentials):
		return "Неверный email или пароль."
	case errors.Is(err, ErrEmailNotVerified):
		return "Подтвердите адрес электронной почты."
	case errors.Is(err, ErrAccountBlocked):
		return "Учётная запись заблокирована."
	case errors.Is(err, ErrTooManyAttempts):
		return "Слишком много попыток. Попробуйте позже."
	case errors.Is(err, ErrInvalidToken):
		return "Ссылка недействительна."
	case errors.Is(err, ErrTokenExpired):
		return "Срок действия ссылки истёк."
	case errors.Is(err, ErrTokenUsed):
		return "Ссылка уже использована."
	default:
		return "Не удалось выполнить действие. Попробуйте позже."
	}
}
