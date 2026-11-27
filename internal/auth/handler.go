package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"oge-ege/internal/shared/httpserver"
)

// ConsentRecorder сохраняет согласия, предоставленные при регистрации.
type ConsentRecorder interface {
	RecordRegistrationConsents(ctx context.Context, userID uuid.UUID, grants []ConsentGrant, ip string) error
}

// Handler обслуживает маршруты аутентификации: JSON API и HTML-страницы.
type Handler struct {
	service  *Service
	csrf     *httpserver.CSRF
	consents ConsentRecorder
	logger   *slog.Logger
	brand    string
}

// NewHandler создаёт HTTP-обработчик auth.
func NewHandler(service *Service, csrf *httpserver.CSRF, consents ConsentRecorder, logger *slog.Logger, brand string) *Handler {
	return &Handler{
		service:  service,
		csrf:     csrf,
		consents: consents,
		logger:   logger,
		brand:    brand,
	}
}

// Register обрабатывает POST /api/auth/register.
func (h *Handler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBind(&req); err != nil {
		httpserver.AbortWithError(c, http.StatusUnprocessableEntity, "validation_failed", "Проверьте заполнение полей")
		return
	}

	user, err := h.register(c.Request.Context(), req, c.ClientIP())
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"user_id": user.ID,
		"email":   user.Email,
		"role":    user.Role,
		"status":  user.Status,
	})
}

// VerifyEmail обрабатывает POST /api/auth/verify-email.
func (h *Handler) VerifyEmail(c *gin.Context) {
	var req verifyEmailRequest
	if err := c.ShouldBind(&req); err != nil {
		httpserver.AbortWithError(c, http.StatusUnprocessableEntity, "validation_failed", "Проверьте заполнение полей")
		return
	}

	if err := h.service.VerifyEmail(c.Request.Context(), req.Token); err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "verified"})
}

// ResendVerification обрабатывает POST /api/auth/resend-verification.
func (h *Handler) ResendVerification(c *gin.Context) {
	var req resendVerificationRequest
	if err := c.ShouldBind(&req); err != nil {
		httpserver.AbortWithError(c, http.StatusUnprocessableEntity, "validation_failed", "Проверьте заполнение полей")
		return
	}

	if err := h.service.ResendVerification(c.Request.Context(), req.Email); err != nil {
		h.logger.Error("не удалось повторно отправить письмо подтверждения", "error", err)
	}

	c.JSON(http.StatusOK, gin.H{"status": "accepted"})
}

// Login обрабатывает POST /api/auth/login.
func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBind(&req); err != nil {
		httpserver.AbortWithError(c, http.StatusUnprocessableEntity, "validation_failed", "Проверьте заполнение полей")
		return
	}

	user, token, err := h.service.Login(c.Request.Context(), req.Email, req.Password, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		h.writeError(c, err)
		return
	}

	SetSessionCookie(c, token)
	if h.csrf != nil {
		h.csrf.RotateFor(c, token)
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id": user.ID,
		"email":   user.Email,
		"role":    user.Role,
	})
}

// Logout обрабатывает POST /api/auth/logout.
func (h *Handler) Logout(c *gin.Context) {
	session, ok := CurrentSession(c)
	if !ok {
		httpserver.AbortWithError(c, http.StatusUnauthorized, "unauthorized", "Требуется вход в систему")
		return
	}

	if err := h.service.Logout(c.Request.Context(), session.ID); err != nil {
		h.writeError(c, err)
		return
	}

	ClearSessionCookie(c)
	if h.csrf != nil {
		h.csrf.RotateFor(c, "")
	}

	c.JSON(http.StatusOK, gin.H{"status": "logged_out"})
}

// RequestPasswordReset обрабатывает POST /api/auth/password/reset-request.
func (h *Handler) RequestPasswordReset(c *gin.Context) {
	var req passwordResetRequest
	if err := c.ShouldBind(&req); err != nil {
		httpserver.AbortWithError(c, http.StatusUnprocessableEntity, "validation_failed", "Проверьте заполнение полей")
		return
	}

	if err := h.service.RequestPasswordReset(c.Request.Context(), req.Email); err != nil {
		h.logger.Error("не удалось отправить письмо сброса пароля", "error", err)
	}

	c.JSON(http.StatusOK, gin.H{"status": "accepted"})
}

// ResetPassword обрабатывает POST /api/auth/password/reset.
func (h *Handler) ResetPassword(c *gin.Context) {
	var req passwordResetApplyRequest
	if err := c.ShouldBind(&req); err != nil {
		httpserver.AbortWithError(c, http.StatusUnprocessableEntity, "validation_failed", "Проверьте заполнение полей")
		return
	}

	if err := h.service.ResetPassword(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "password_reset"})
}

// ChangePassword обрабатывает POST /api/auth/password/change.
func (h *Handler) ChangePassword(c *gin.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		httpserver.AbortWithError(c, http.StatusUnauthorized, "unauthorized", "Требуется вход в систему")
		return
	}
	session, ok := CurrentSession(c)
	if !ok {
		httpserver.AbortWithError(c, http.StatusUnauthorized, "unauthorized", "Требуется вход в систему")
		return
	}

	var req passwordChangeRequest
	if err := c.ShouldBind(&req); err != nil {
		httpserver.AbortWithError(c, http.StatusUnprocessableEntity, "validation_failed", "Проверьте заполнение полей")
		return
	}

	if err := h.service.ChangePassword(c.Request.Context(), user.ID, session.ID, req.CurrentPassword, req.NewPassword); err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "password_changed"})
}

func (h *Handler) register(ctx context.Context, req registerRequest, ip string) (*User, error) {
	grants, err := NormalizeConsents(req.grants())
	if err != nil {
		return nil, err
	}

	user, err := h.service.Register(ctx, RegisterInput{
		Email:    req.Email,
		Password: req.Password,
		Role:     Role(req.Role),
		Consents: grants,
		IP:       ip,
	})
	if err != nil {
		return nil, err
	}

	if h.consents != nil {
		if err := h.consents.RecordRegistrationConsents(ctx, user.ID, grants, ip); err != nil {
			h.logger.Error("не удалось сохранить согласия при регистрации", "user_id", user.ID, "error", err)
		}
	}

	return user, nil
}

func (h *Handler) writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidEmail):
		httpserver.AbortWithFields(c, http.StatusUnprocessableEntity, "validation_failed", "Некорректный email", map[string]string{"email": "Некорректный адрес"})
	case errors.Is(err, ErrWeakPassword):
		httpserver.AbortWithFields(c, http.StatusUnprocessableEntity, "validation_failed", ErrWeakPassword.Error(), map[string]string{"password": ErrWeakPassword.Error()})
	case errors.Is(err, ErrInvalidRole):
		httpserver.AbortWithFields(c, http.StatusUnprocessableEntity, "validation_failed", "Роль недоступна при регистрации", map[string]string{"role": "Выберите ученика или родителя"})
	case errors.Is(err, ErrConsentRequired):
		httpserver.AbortWithFields(c, http.StatusUnprocessableEntity, "validation_failed", ErrConsentRequired.Error(), map[string]string{"consents": ErrConsentRequired.Error()})
	case errors.Is(err, ErrEmailTaken):
		httpserver.AbortWithFields(c, http.StatusConflict, "conflict", "Email уже зарегистрирован", map[string]string{"email": "Email уже зарегистрирован"})
	case errors.Is(err, ErrInvalidCredentials):
		httpserver.AbortWithError(c, http.StatusUnauthorized, "unauthorized", "Неверный email или пароль")
	case errors.Is(err, ErrEmailNotVerified):
		httpserver.AbortWithError(c, http.StatusForbidden, "email_not_verified", "Подтвердите адрес электронной почты")
	case errors.Is(err, ErrAccountBlocked):
		httpserver.AbortWithError(c, http.StatusForbidden, "forbidden", "Учётная запись заблокирована")
	case errors.Is(err, ErrTooManyAttempts):
		httpserver.AbortWithError(c, http.StatusTooManyRequests, "rate_limited", "Слишком много неудачных попыток входа")
	case errors.Is(err, ErrInvalidToken):
		httpserver.AbortWithFields(c, http.StatusUnprocessableEntity, "validation_failed", "Ссылка недействительна", map[string]string{"token": "Ссылка недействительна"})
	case errors.Is(err, ErrTokenExpired):
		httpserver.AbortWithFields(c, http.StatusUnprocessableEntity, "validation_failed", "Срок действия ссылки истёк", map[string]string{"token": "Срок действия ссылки истёк"})
	case errors.Is(err, ErrTokenUsed):
		httpserver.AbortWithFields(c, http.StatusUnprocessableEntity, "validation_failed", "Ссылка уже использована", map[string]string{"token": "Ссылка уже использована"})
	case errors.Is(err, ErrInvalidCurrentPassword):
		httpserver.AbortWithFields(c, http.StatusUnprocessableEntity, "validation_failed", "Неверный текущий пароль", map[string]string{"current_password": "Неверный текущий пароль"})
	default:
		h.logger.Error("ошибка обработки запроса аутентификации", "error", err)
		httpserver.AbortWithError(c, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка")
	}
}
