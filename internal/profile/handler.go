package profile

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"oge-ege/internal/auth"
	"oge-ege/internal/shared/httpserver"
)

// Handler обслуживает маршруты /api/me.
type Handler struct {
	service *Service
	logger  *slog.Logger
}

// NewHandler создаёт HTTP-обработчик profile.
func NewHandler(service *Service, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

// Me обрабатывает GET /api/me.
func (h *Handler) Me(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		httpserver.AbortWithError(c, http.StatusUnauthorized, "unauthorized", "Требуется вход в систему")
		return
	}

	profile, err := h.service.GetProfile(c.Request.Context(), user.ID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		h.writeError(c, err)
		return
	}

	response := gin.H{
		"user_id": user.ID,
		"email":   user.Email,
		"role":    user.Role,
		"status":  user.Status,
		"profile": nil,
	}
	if profile != nil {
		response["profile"] = profileResponse(profile)
	}

	c.JSON(http.StatusOK, response)
}

// GetProfile обрабатывает GET /api/me/profile.
func (h *Handler) GetProfile(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		httpserver.AbortWithError(c, http.StatusUnauthorized, "unauthorized", "Требуется вход в систему")
		return
	}

	profile, err := h.service.GetProfile(c.Request.Context(), user.ID)
	if errors.Is(err, ErrNotFound) {
		c.JSON(http.StatusOK, gin.H{"profile": nil})
		return
	}
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"profile": profileResponse(profile)})
}

// UpdateProfile обрабатывает PATCH /api/me/profile.
func (h *Handler) UpdateProfile(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		httpserver.AbortWithError(c, http.StatusUnauthorized, "unauthorized", "Требуется вход в систему")
		return
	}

	var req profileRequest
	if err := c.ShouldBind(&req); err != nil {
		httpserver.AbortWithError(c, http.StatusUnprocessableEntity, "validation_failed", "Проверьте заполнение полей")
		return
	}

	profile, err := h.service.SaveProfile(c.Request.Context(), user.ID, user.Role, ProfileInput(req))
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"profile": profileResponse(profile)})
}

// GetGoals обрабатывает GET /api/me/goals.
func (h *Handler) GetGoals(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		httpserver.AbortWithError(c, http.StatusUnauthorized, "unauthorized", "Требуется вход в систему")
		return
	}

	goal, err := h.service.GetCurrentGoal(c.Request.Context(), user.ID)
	if errors.Is(err, ErrNotFound) {
		c.JSON(http.StatusOK, gin.H{"goal": nil})
		return
	}
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"goal": goalResponse(goal)})
}

// SaveGoals обрабатывает PUT /api/me/goals.
func (h *Handler) SaveGoals(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		httpserver.AbortWithError(c, http.StatusUnauthorized, "unauthorized", "Требуется вход в систему")
		return
	}

	var req goalRequest
	if err := c.ShouldBind(&req); err != nil {
		httpserver.AbortWithError(c, http.StatusUnprocessableEntity, "validation_failed", "Проверьте заполнение полей")
		return
	}

	goal, err := h.service.SaveGoal(c.Request.Context(), user.ID, GoalInput(req))
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"goal": goalResponse(goal)})
}

// GetSettings обрабатывает GET /api/me/settings.
func (h *Handler) GetSettings(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		httpserver.AbortWithError(c, http.StatusUnauthorized, "unauthorized", "Требуется вход в систему")
		return
	}

	settings, err := h.service.GetSettings(c.Request.Context(), user.ID)
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"settings": settingsResponse(settings)})
}

// UpdateSettings обрабатывает PATCH /api/me/settings.
func (h *Handler) UpdateSettings(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		httpserver.AbortWithError(c, http.StatusUnauthorized, "unauthorized", "Требуется вход в систему")
		return
	}

	var req settingsRequest
	if err := c.ShouldBind(&req); err != nil {
		httpserver.AbortWithError(c, http.StatusUnprocessableEntity, "validation_failed", "Проверьте заполнение полей")
		return
	}

	current, err := h.service.GetSettings(c.Request.Context(), user.ID)
	if err != nil {
		h.writeError(c, err)
		return
	}

	emailNotifications := current.EmailNotifications
	if req.EmailNotifications != nil {
		emailNotifications = *req.EmailNotifications
	}
	weeklyReport := current.WeeklyReport
	if req.WeeklyReport != nil {
		weeklyReport = *req.WeeklyReport
	}

	settings, err := h.service.SaveSettings(c.Request.Context(), user.ID, emailNotifications, weeklyReport)
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"settings": settingsResponse(settings)})
}

// DeleteRequest обрабатывает POST /api/me/delete-request.
func (h *Handler) DeleteRequest(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		httpserver.AbortWithError(c, http.StatusUnauthorized, "unauthorized", "Требуется вход в систему")
		return
	}

	if err := h.service.DeleteAccount(c.Request.Context(), user.ID); err != nil {
		h.writeError(c, err)
		return
	}

	auth.ClearSessionCookie(c)
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (h *Handler) writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrBirthDateRequired):
		httpserver.AbortWithFields(c, http.StatusUnprocessableEntity, "validation_failed", err.Error(), map[string]string{"birth_date": err.Error()})
	case errors.Is(err, ErrInvalidBirthDate):
		httpserver.AbortWithFields(c, http.StatusUnprocessableEntity, "validation_failed", err.Error(), map[string]string{"birth_date": err.Error()})
	case errors.Is(err, ErrInvalidTimezone):
		httpserver.AbortWithFields(c, http.StatusUnprocessableEntity, "validation_failed", err.Error(), map[string]string{"timezone": err.Error()})
	case errors.Is(err, ErrInvalidGrade):
		httpserver.AbortWithFields(c, http.StatusUnprocessableEntity, "validation_failed", err.Error(), map[string]string{"grade": err.Error()})
	case errors.Is(err, ErrInvalidExamID):
		httpserver.AbortWithFields(c, http.StatusUnprocessableEntity, "validation_failed", err.Error(), map[string]string{"exam_id": err.Error()})
	case errors.Is(err, ErrExamUnavailable):
		httpserver.AbortWithFields(c, http.StatusUnprocessableEntity, "validation_failed", err.Error(), map[string]string{"exam_id": err.Error()})
	case errors.Is(err, ErrInvalidExamDate):
		httpserver.AbortWithFields(c, http.StatusUnprocessableEntity, "validation_failed", err.Error(), map[string]string{"exam_date": err.Error()})
	case errors.Is(err, ErrInvalidTargetScore):
		httpserver.AbortWithFields(c, http.StatusUnprocessableEntity, "validation_failed", err.Error(), map[string]string{"target_score": err.Error()})
	case errors.Is(err, ErrInvalidWeeklyMinutes):
		httpserver.AbortWithFields(c, http.StatusUnprocessableEntity, "validation_failed", err.Error(), map[string]string{"weekly_minutes": err.Error()})
	default:
		h.logger.Error("ошибка обработки запроса профиля", "error", err)
		httpserver.AbortWithError(c, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка")
	}
}

func profileResponse(profile *Profile) gin.H {
	var birthDate any
	if profile.BirthDate != nil {
		birthDate = profile.BirthDate.Format(dateLayout)
	}

	return gin.H{
		"first_name": profile.FirstName,
		"last_name":  profile.LastName,
		"birth_date": birthDate,
		"grade":      profile.Grade,
		"timezone":   profile.Timezone,
		"contacts":   profile.Contacts,
	}
}

func goalResponse(goal *StudyGoal) gin.H {
	return gin.H{
		"exam_id":        goal.ExamID,
		"exam_date":      goal.ExamDate.Format(dateLayout),
		"target_score":   goal.TargetScore,
		"weekly_minutes": goal.WeeklyMinutes,
	}
}

func settingsResponse(settings *Settings) gin.H {
	return gin.H{
		"email_notifications": settings.EmailNotifications,
		"weekly_report":       settings.WeeklyReport,
	}
}
