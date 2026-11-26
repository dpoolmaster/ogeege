package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SubmitLogin обрабатывает POST /login.
func (h *Handler) SubmitLogin(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBind(&req); err != nil {
		h.render(c, http.StatusOK, LoginPage(PageData{CSRF: h.csrfToken(c), Email: req.Email, Error: "Проверьте заполнение полей."}))
		return
	}

	_, token, err := h.service.Login(c.Request.Context(), req.Email, req.Password, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		h.render(c, http.StatusOK, LoginPage(PageData{CSRF: h.csrfToken(c), Email: req.Email, Error: errorMessage(err)}))
		return
	}

	SetSessionCookie(c, token)
	if h.csrf != nil {
		h.csrf.RotateFor(c, token)
	}

	c.Redirect(http.StatusSeeOther, "/account")
}

// SubmitRegister обрабатывает POST /register.
func (h *Handler) SubmitRegister(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBind(&req); err != nil {
		h.render(c, http.StatusOK, RegisterPage(PageData{CSRF: h.csrfToken(c), Email: req.Email, Error: "Проверьте заполнение полей."}))
		return
	}

	user, err := h.register(c.Request.Context(), req, c.ClientIP())
	if err != nil {
		h.render(c, http.StatusOK, RegisterPage(PageData{CSRF: h.csrfToken(c), Email: req.Email, Error: errorMessage(err)}))
		return
	}

	h.render(c, http.StatusOK, VerifyEmailPage(PageData{
		CSRF:    h.csrfToken(c),
		Email:   user.Email,
		Message: "Аккаунт создан. Проверьте почту и подтвердите email.",
	}))
}

// SubmitVerifyEmail обрабатывает POST /verify-email.
func (h *Handler) SubmitVerifyEmail(c *gin.Context) {
	var req verifyEmailRequest
	if err := c.ShouldBind(&req); err != nil {
		h.render(c, http.StatusOK, VerifyEmailPage(PageData{CSRF: h.csrfToken(c), Token: req.Token, Error: "Проверьте заполнение полей."}))
		return
	}

	if err := h.service.VerifyEmail(c.Request.Context(), req.Token); err != nil {
		h.render(c, http.StatusOK, VerifyEmailPage(PageData{CSRF: h.csrfToken(c), Token: req.Token, Error: errorMessage(err)}))
		return
	}

	c.Redirect(http.StatusSeeOther, "/login?verified=1")
}

// SubmitResendVerification обрабатывает POST /resend-verification.
func (h *Handler) SubmitResendVerification(c *gin.Context) {
	var req resendVerificationRequest
	if err := c.ShouldBind(&req); err != nil {
		h.render(c, http.StatusOK, VerifyEmailPage(PageData{CSRF: h.csrfToken(c), Error: "Проверьте заполнение полей."}))
		return
	}

	if err := h.service.ResendVerification(c.Request.Context(), req.Email); err != nil {
		h.logger.Error("не удалось повторно отправить письмо подтверждения", "error", err)
	}

	h.render(c, http.StatusOK, VerifyEmailPage(PageData{
		CSRF:    h.csrfToken(c),
		Email:   req.Email,
		Message: "Если аккаунт существует, письмо отправлено.",
	}))
}

// SubmitLogout обрабатывает POST /logout.
func (h *Handler) SubmitLogout(c *gin.Context) {
	session, ok := CurrentSession(c)
	if !ok {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	if err := h.service.Logout(c.Request.Context(), session.ID); err != nil {
		h.logger.Error("не удалось отозвать сессию", "error", err)
		user, _ := CurrentUser(c)
		data := PageData{
			CSRF:  h.csrfToken(c),
			Error: "Не удалось выйти. Попробуйте ещё раз.",
		}
		if user != nil {
			data.Email = user.Email
			data.RoleLabel = roleLabel(user.Role)
			data.StatusLabel = statusLabel(user.Status)
		}
		h.render(c, http.StatusInternalServerError, AccountPage(data))
		return
	}

	ClearSessionCookie(c)
	if h.csrf != nil {
		h.csrf.RotateFor(c, "")
	}

	c.Redirect(http.StatusSeeOther, "/login")
}

// SubmitPasswordResetRequest обрабатывает POST /password/reset-request.
func (h *Handler) SubmitPasswordResetRequest(c *gin.Context) {
	var req passwordResetRequest
	if err := c.ShouldBind(&req); err != nil {
		h.render(c, http.StatusOK, PasswordResetPage(PageData{CSRF: h.csrfToken(c), Error: "Проверьте заполнение полей."}))
		return
	}

	if err := h.service.RequestPasswordReset(c.Request.Context(), req.Email); err != nil {
		h.logger.Error("не удалось отправить письмо сброса пароля", "error", err)
	}

	h.render(c, http.StatusOK, PasswordResetPage(PageData{
		CSRF:    h.csrfToken(c),
		Email:   req.Email,
		Message: "Если аккаунт существует, письмо со ссылкой отправлено.",
	}))
}

// SubmitPasswordReset обрабатывает POST /password/reset.
func (h *Handler) SubmitPasswordReset(c *gin.Context) {
	var req passwordResetApplyRequest
	if err := c.ShouldBind(&req); err != nil {
		h.render(c, http.StatusOK, PasswordResetPage(PageData{CSRF: h.csrfToken(c), Token: req.Token, Error: "Проверьте заполнение полей."}))
		return
	}

	if err := h.service.ResetPassword(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		h.render(c, http.StatusOK, PasswordResetPage(PageData{CSRF: h.csrfToken(c), Token: req.Token, Error: errorMessage(err)}))
		return
	}

	c.Redirect(http.StatusSeeOther, "/login?reset=1")
}
