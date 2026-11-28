package learning

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"oge-ege/internal/auth"
	"oge-ege/internal/profile"
	"oge-ege/internal/shared/httpserver"
)

// ServiceAPI описывает сценарии learning, нужные HTTP-слою.
type ServiceAPI interface {
	Today(ctx context.Context, userID uuid.UUID) (TodayData, error)
	GoalPageData(ctx context.Context, userID uuid.UUID) (GoalPageData, error)
	SetupGoal(ctx context.Context, userID uuid.UUID, in GoalInput) (*Plan, error)
	PrepareDiagnostic(ctx context.Context, userID uuid.UUID, examCode string) (DiagnosticStartData, error)
	StartDiagnostic(ctx context.Context, userID uuid.UUID, examCode string) (*Session, error)
	PracticeStartData(ctx context.Context, userID uuid.UUID) (PracticeStartData, error)
	StartPractice(ctx context.Context, userID uuid.UUID, in PracticeInput) (*Session, error)
	ReviewStartData(ctx context.Context, userID uuid.UUID) (ReviewStartData, error)
	StartReview(ctx context.Context, userID uuid.UUID) (*Session, error)
	SessionKind(ctx context.Context, userID, sessionID uuid.UUID) (string, error)
	SessionView(ctx context.Context, userID, sessionID uuid.UUID) (SessionView, error)
	SaveAnswer(ctx context.Context, userID, sessionID, itemID uuid.UUID, raw string) (AnswerResult, error)
	CompleteSession(ctx context.Context, userID, sessionID uuid.UUID) (*SessionResult, error)
	Result(ctx context.Context, userID, sessionID uuid.UUID) (*ResultView, error)
	Gaps(ctx context.Context, userID uuid.UUID) (GapMapView, error)
	PlanView(ctx context.Context, userID uuid.UUID) (PlanView, error)
	Replan(ctx context.Context, userID uuid.UUID) (*Plan, error)
	MarkItemDone(ctx context.Context, userID, itemID uuid.UUID) error
}

// Handler обслуживает маршруты кабинета ученика.
type Handler struct {
	service ServiceAPI
	csrf    *httpserver.CSRF
	logger  *slog.Logger
	brand   string
}

// NewHandler создаёт HTTP-обработчик learning.
func NewHandler(service ServiceAPI, csrf *httpserver.CSRF, logger *slog.Logger, brand string) *Handler {
	return &Handler{service: service, csrf: csrf, logger: logger, brand: brand}
}

// RegisterHTML регистрирует HTML-маршруты кабинета.
func (h *Handler) RegisterHTML(router gin.IRoutes) {
	router.GET("/app", h.ShowToday)
	router.GET("/app/goal", h.ShowGoal)
	router.POST("/app/goal", h.SubmitGoal)
	router.GET("/app/diagnostics", h.ShowDiagnostics)
	router.POST("/app/diagnostics", h.SubmitStartDiagnostics)
	router.GET("/app/diagnostics/:id", h.ShowSession)
	router.POST("/app/diagnostics/:id/items/:itemId", h.SubmitAnswer)
	router.POST("/app/diagnostics/:id/submit", h.SubmitSession)
	router.GET("/app/diagnostics/:id/result", h.ShowResult)
	router.GET("/app/practice", h.ShowPractice)
	router.POST("/app/practice", h.SubmitStartPractice)
	router.GET("/app/practice/:id", h.ShowPracticeSession)
	router.POST("/app/practice/:id/items/:itemId", h.SubmitPracticeAnswer)
	router.POST("/app/practice/:id/submit", h.SubmitPracticeSession)
	router.GET("/app/practice/:id/result", h.ShowPracticeResult)
	router.GET("/app/review", h.ShowReview)
	router.POST("/app/review", h.SubmitStartReview)
	router.GET("/app/review/:id", h.ShowReviewSession)
	router.POST("/app/review/:id/items/:itemId", h.SubmitReviewAnswer)
	router.POST("/app/review/:id/submit", h.SubmitReviewSession)
	router.GET("/app/review/:id/result", h.ShowReviewResult)
	router.GET("/app/gaps", h.ShowGaps)
	router.GET("/app/plan", h.ShowPlan)
	router.POST("/app/plan/replan", h.SubmitReplan)
	router.POST("/app/plan/items/:id/complete", h.SubmitCompleteItem)
}

// RegisterJSON регистрирует JSON-маршруты плана и диагностики.
func (h *Handler) RegisterJSON(router gin.IRoutes) {
	router.GET("/plan", h.APIPlan)
	router.POST("/plan/replan", h.APIReplan)
	router.PATCH("/plan/items/:id", h.APICompleteItem)
	router.POST("/diagnostics", h.APIStartDiagnostic)
	router.POST("/sessions", h.APICreateSession)
	router.GET("/sessions/:id", h.APISession)
	router.PATCH("/sessions/:id/items/:itemId", h.APIAnswer)
	router.POST("/sessions/:id/submit", h.APISubmit)
	router.GET("/sessions/:id/result", h.APIResult)
}

// ShowToday обрабатывает GET /app.
func (h *Handler) ShowToday(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}
	data, err := h.service.Today(c.Request.Context(), user.ID)
	if err != nil {
		h.renderPageError(c, err)
		return
	}
	h.render(c, http.StatusOK, TodayPage(h.brand, data))
}

// ShowGoal обрабатывает GET /app/goal.
func (h *Handler) ShowGoal(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}
	data, err := h.service.GoalPageData(c.Request.Context(), user.ID)
	if err != nil {
		h.renderPageError(c, err)
		return
	}
	h.render(c, http.StatusOK, GoalPage(h.brand, data, h.csrfToken(c)))
}

// SubmitGoal обрабатывает POST /app/goal.
func (h *Handler) SubmitGoal(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	input := GoalInput{
		ExamCode:      c.PostForm("exam_code"),
		ExamDate:      c.PostForm("exam_date"),
		TargetScore:   parseFormInt(c.PostForm("target_score")),
		WeeklyMinutes: parseFormInt(c.PostForm("weekly_minutes")),
	}
	if _, err := h.service.SetupGoal(c.Request.Context(), user.ID, input); err != nil {
		data, dataErr := h.service.GoalPageData(c.Request.Context(), user.ID)
		if dataErr != nil {
			h.renderPageError(c, dataErr)
			return
		}
		data.Input = input
		data.FieldErrors = goalFieldErrors(err)
		data.FormError = goalFormError(err)
		h.render(c, http.StatusUnprocessableEntity, GoalPage(h.brand, data, h.csrfToken(c)))
		return
	}

	c.Redirect(http.StatusSeeOther, "/app/plan")
}

// ShowDiagnostics обрабатывает GET /app/diagnostics.
func (h *Handler) ShowDiagnostics(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}
	data, err := h.service.PrepareDiagnostic(c.Request.Context(), user.ID, c.Query("exam"))
	if err != nil {
		h.renderPageError(c, err)
		return
	}
	h.render(c, http.StatusOK, DiagnosticsPage(h.brand, data, h.csrfToken(c)))
}

// SubmitStartDiagnostics обрабатывает POST /app/diagnostics.
func (h *Handler) SubmitStartDiagnostics(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}
	session, err := h.service.StartDiagnostic(c.Request.Context(), user.ID, c.PostForm("exam_code"))
	if err != nil {
		h.renderPageError(c, err)
		return
	}
	c.Redirect(http.StatusSeeOther, "/app/diagnostics/"+session.ID.String())
}

// ShowSession обрабатывает GET /app/diagnostics/:id.
func (h *Handler) ShowSession(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.renderError(c, http.StatusNotFound, "Диагностика не найдена")
		return
	}
	view, err := h.service.SessionView(c.Request.Context(), user.ID, sessionID)
	if err != nil {
		h.renderPageError(c, err)
		return
	}
	if itemID := strings.TrimSpace(c.Query("item")); itemID != "" {
		if parsed, err := uuid.Parse(itemID); err == nil {
			for index, item := range view.Items {
				if item.Item.ID == parsed {
					view.CurrentIndex = index
					break
				}
			}
		}
	}
	if view.Completed {
		c.Redirect(http.StatusSeeOther, "/app/diagnostics/"+sessionID.String()+"/result")
		return
	}
	h.render(c, http.StatusOK, SessionPage(h.brand, view, h.csrfToken(c)))
}

// SubmitAnswer обрабатывает POST /app/diagnostics/:id/items/:itemId.
func (h *Handler) SubmitAnswer(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.renderError(c, http.StatusNotFound, "Диагностика не найдена")
		return
	}
	itemID, err := uuid.Parse(c.Param("itemId"))
	if err != nil {
		h.renderError(c, http.StatusNotFound, "Задание не найдено")
		return
	}
	result, err := h.service.SaveAnswer(c.Request.Context(), user.ID, sessionID, itemID, c.PostForm("answer"))
	if err != nil {
		h.renderPageError(c, err)
		return
	}
	c.Redirect(http.StatusSeeOther, result.NextHref)
}

// SubmitSession обрабатывает POST /app/diagnostics/:id/submit.
func (h *Handler) SubmitSession(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.renderError(c, http.StatusNotFound, "Диагностика не найдена")
		return
	}
	if _, err := h.service.CompleteSession(c.Request.Context(), user.ID, sessionID); err != nil {
		h.renderPageError(c, err)
		return
	}
	c.Redirect(http.StatusSeeOther, "/app/diagnostics/"+sessionID.String()+"/result")
}

// ShowResult обрабатывает GET /app/diagnostics/:id/result.
func (h *Handler) ShowResult(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.renderError(c, http.StatusNotFound, "Диагностика не найдена")
		return
	}
	view, err := h.service.Result(c.Request.Context(), user.ID, sessionID)
	if errors.Is(err, ErrResultNotReady) {
		c.Redirect(http.StatusSeeOther, "/app/diagnostics/"+sessionID.String())
		return
	}
	if err != nil {
		h.renderPageError(c, err)
		return
	}
	h.render(c, http.StatusOK, ResultPage(h.brand, *view))
}

// ShowGaps обрабатывает GET /app/gaps.
func (h *Handler) ShowGaps(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}
	view, err := h.service.Gaps(c.Request.Context(), user.ID)
	if errors.Is(err, ErrNoGoal) {
		h.render(c, http.StatusOK, GapsPage(h.brand, GapMapView{}, false))
		return
	}
	if err != nil {
		h.renderPageError(c, err)
		return
	}
	h.render(c, http.StatusOK, GapsPage(h.brand, view, true))
}

// ShowPlan обрабатывает GET /app/plan.
func (h *Handler) ShowPlan(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}
	view, err := h.service.PlanView(c.Request.Context(), user.ID)
	if err != nil {
		h.renderPageError(c, err)
		return
	}
	h.render(c, http.StatusOK, PlanPage(h.brand, view, h.csrfToken(c)))
}

// SubmitReplan обрабатывает POST /app/plan/replan.
func (h *Handler) SubmitReplan(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}
	if _, err := h.service.Replan(c.Request.Context(), user.ID); err != nil {
		h.renderPageError(c, err)
		return
	}
	c.Redirect(http.StatusSeeOther, "/app/plan")
}

// SubmitCompleteItem обрабатывает POST /app/plan/items/:id/complete.
func (h *Handler) SubmitCompleteItem(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}
	itemID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.renderError(c, http.StatusNotFound, "Пункт плана не найден")
		return
	}
	if err := h.service.MarkItemDone(c.Request.Context(), user.ID, itemID); err != nil {
		h.renderPageError(c, err)
		return
	}
	c.Redirect(http.StatusSeeOther, "/app/plan")
}

// APIPlan обрабатывает GET /api/plan.
func (h *Handler) APIPlan(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		httpserver.AbortWithError(c, http.StatusUnauthorized, "unauthorized", "Требуется вход в систему")
		return
	}
	view, err := h.service.PlanView(c.Request.Context(), user.ID)
	if err != nil {
		h.apiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"plan": planResponse(view)})
}

// APIReplan обрабатывает POST /api/plan/replan.
func (h *Handler) APIReplan(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		httpserver.AbortWithError(c, http.StatusUnauthorized, "unauthorized", "Требуется вход в систему")
		return
	}
	plan, err := h.service.Replan(c.Request.Context(), user.ID)
	if err != nil {
		h.apiError(c, err)
		return
	}
	view, err := h.service.PlanView(c.Request.Context(), user.ID)
	if err != nil {
		h.apiError(c, err)
		return
	}
	view.Plan = plan
	c.JSON(http.StatusOK, gin.H{"plan": planResponse(view)})
}

// APICompleteItem обрабатывает PATCH /api/plan/items/:id.
func (h *Handler) APICompleteItem(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		httpserver.AbortWithError(c, http.StatusUnauthorized, "unauthorized", "Требуется вход в систему")
		return
	}
	itemID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpserver.AbortWithError(c, http.StatusNotFound, "not_found", "Пункт плана не найден")
		return
	}
	if err := h.service.MarkItemDone(c.Request.Context(), user.ID, itemID); err != nil {
		h.apiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "done"})
}

// APIStartDiagnostic обрабатывает POST /api/diagnostics.
func (h *Handler) APIStartDiagnostic(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		httpserver.AbortWithError(c, http.StatusUnauthorized, "unauthorized", "Требуется вход в систему")
		return
	}
	var req struct {
		Kind     string `json:"kind"`
		ExamCode string `json:"exam_code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpserver.AbortWithError(c, http.StatusUnprocessableEntity, "validation_failed", "Проверьте тело запроса")
		return
	}
	if strings.TrimSpace(req.Kind) != SessionKindDiagnostic {
		httpserver.AbortWithError(c, http.StatusUnprocessableEntity, "unsupported_kind", "Поддерживается только диагностика")
		return
	}
	session, err := h.service.StartDiagnostic(c.Request.Context(), user.ID, req.ExamCode)
	if err != nil {
		h.apiError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"session": gin.H{"id": session.ID, "kind": session.Kind, "status": session.Status}})
}

// APISession обрабатывает GET /api/sessions/:id.
func (h *Handler) APISession(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		httpserver.AbortWithError(c, http.StatusUnauthorized, "unauthorized", "Требуется вход в систему")
		return
	}
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpserver.AbortWithError(c, http.StatusNotFound, "not_found", "Сессия не найдена")
		return
	}
	view, err := h.service.SessionView(c.Request.Context(), user.ID, sessionID)
	if err != nil {
		h.apiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"session": sessionResponse(view)})
}

// APIAnswer обрабатывает PATCH /api/sessions/:id/items/:itemId.
func (h *Handler) APIAnswer(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		httpserver.AbortWithError(c, http.StatusUnauthorized, "unauthorized", "Требуется вход в систему")
		return
	}
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpserver.AbortWithError(c, http.StatusNotFound, "not_found", "Сессия не найдена")
		return
	}
	itemID, err := uuid.Parse(c.Param("itemId"))
	if err != nil {
		httpserver.AbortWithError(c, http.StatusNotFound, "not_found", "Задание не найдено")
		return
	}
	var req struct {
		Answer string `json:"answer"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpserver.AbortWithError(c, http.StatusUnprocessableEntity, "validation_failed", "Проверьте тело запроса")
		return
	}
	result, err := h.service.SaveAnswer(c.Request.Context(), user.ID, sessionID, itemID, req.Answer)
	if err != nil {
		h.apiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"result": gin.H{
		"correct":    result.Correct,
		"reason":     result.Reason,
		"normalized": result.Normalized,
		"expected":   result.Expected,
	}})
}

// APISubmit обрабатывает POST /api/sessions/:id/submit.
func (h *Handler) APISubmit(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		httpserver.AbortWithError(c, http.StatusUnauthorized, "unauthorized", "Требуется вход в систему")
		return
	}
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpserver.AbortWithError(c, http.StatusNotFound, "not_found", "Сессия не найдена")
		return
	}
	result, err := h.service.CompleteSession(c.Request.Context(), user.ID, sessionID)
	if err != nil {
		h.apiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"result": resultResponse(result)})
}

// APIResult обрабатывает GET /api/sessions/:id/result.
func (h *Handler) APIResult(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		httpserver.AbortWithError(c, http.StatusUnauthorized, "unauthorized", "Требуется вход в систему")
		return
	}
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpserver.AbortWithError(c, http.StatusNotFound, "not_found", "Сессия не найдена")
		return
	}
	view, err := h.service.Result(c.Request.Context(), user.ID, sessionID)
	if err != nil {
		h.apiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"result": resultResponse(&view.Result)})
}

// ShowPractice обрабатывает GET /app/practice.
func (h *Handler) ShowPractice(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}
	data, err := h.service.PracticeStartData(c.Request.Context(), user.ID)
	if errors.Is(err, ErrNoGoal) {
		data = PracticeStartData{Available: false, Reason: "Задайте учебную цель, чтобы начать практику."}
		err = nil
	}
	if err != nil {
		h.renderPageError(c, err)
		return
	}
	if plan := strings.TrimSpace(c.Query("plan")); plan != "" {
		if _, err := uuid.Parse(plan); err == nil {
			data.Selected = "plan:" + plan
		}
	}
	h.render(c, http.StatusOK, PracticePage(h.brand, data, h.csrfToken(c)))
}

// SubmitStartPractice обрабатывает POST /app/practice.
func (h *Handler) SubmitStartPractice(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}
	input, err := parsePracticeChoice(c.PostForm("choice"))
	if err != nil {
		data, dataErr := h.service.PracticeStartData(c.Request.Context(), user.ID)
		if dataErr != nil {
			h.renderPageError(c, dataErr)
			return
		}
		data.Selected = c.PostForm("choice")
		h.render(c, http.StatusUnprocessableEntity, PracticePage(h.brand, data, h.csrfToken(c)))
		return
	}
	session, err := h.service.StartPractice(c.Request.Context(), user.ID, input)
	if err != nil {
		h.renderPageError(c, err)
		return
	}
	c.Redirect(http.StatusSeeOther, "/app/practice/"+session.ID.String())
}

// ShowPracticeSession обрабатывает GET /app/practice/:id.
func (h *Handler) ShowPracticeSession(c *gin.Context) {
	h.showSession(c, SessionKindPractice, "/app/practice")
}

// SubmitPracticeAnswer обрабатывает POST /app/practice/:id/items/:itemId.
func (h *Handler) SubmitPracticeAnswer(c *gin.Context) {
	h.submitAnswer(c, SessionKindPractice, "/app/practice")
}

// SubmitPracticeSession обрабатывает POST /app/practice/:id/submit.
func (h *Handler) SubmitPracticeSession(c *gin.Context) {
	h.submitSession(c, SessionKindPractice, "/app/practice")
}

// ShowPracticeResult обрабатывает GET /app/practice/:id/result.
func (h *Handler) ShowPracticeResult(c *gin.Context) {
	h.showResult(c, SessionKindPractice, "/app/practice")
}

// ShowReview обрабатывает GET /app/review.
func (h *Handler) ShowReview(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}
	data, err := h.service.ReviewStartData(c.Request.Context(), user.ID)
	if err != nil {
		h.renderPageError(c, err)
		return
	}
	h.render(c, http.StatusOK, ReviewPage(h.brand, data, h.csrfToken(c)))
}

// SubmitStartReview обрабатывает POST /app/review.
func (h *Handler) SubmitStartReview(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}
	session, err := h.service.StartReview(c.Request.Context(), user.ID)
	if errors.Is(err, ErrNoReviewTasks) {
		data, dataErr := h.service.ReviewStartData(c.Request.Context(), user.ID)
		if dataErr != nil {
			h.renderPageError(c, dataErr)
			return
		}
		data.Available = false
		data.Reason = "Нет заданий для повторения. Перейдите к практике или диагностике."
		h.render(c, http.StatusOK, ReviewPage(h.brand, data, h.csrfToken(c)))
		return
	}
	if err != nil {
		h.renderPageError(c, err)
		return
	}
	c.Redirect(http.StatusSeeOther, "/app/review/"+session.ID.String())
}

// ShowReviewSession обрабатывает GET /app/review/:id.
func (h *Handler) ShowReviewSession(c *gin.Context) {
	h.showSession(c, SessionKindReview, "/app/review")
}

// SubmitReviewAnswer обрабатывает POST /app/review/:id/items/:itemId.
func (h *Handler) SubmitReviewAnswer(c *gin.Context) {
	h.submitAnswer(c, SessionKindReview, "/app/review")
}

// SubmitReviewSession обрабатывает POST /app/review/:id/submit.
func (h *Handler) SubmitReviewSession(c *gin.Context) {
	h.submitSession(c, SessionKindReview, "/app/review")
}

// ShowReviewResult обрабатывает GET /app/review/:id/result.
func (h *Handler) ShowReviewResult(c *gin.Context) {
	h.showResult(c, SessionKindReview, "/app/review")
}

func (h *Handler) showSession(c *gin.Context, kind, base string) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.renderError(c, http.StatusNotFound, "Сессия не найдена")
		return
	}
	view, err := h.service.SessionView(c.Request.Context(), user.ID, sessionID)
	if err != nil {
		h.renderPageError(c, err)
		return
	}
	if view.Session.Kind != kind {
		h.renderError(c, http.StatusNotFound, "Сессия не найдена")
		return
	}
	if itemID := strings.TrimSpace(c.Query("item")); itemID != "" {
		if parsed, err := uuid.Parse(itemID); err == nil {
			for index, item := range view.Items {
				if item.Item.ID == parsed {
					view.CurrentIndex = index
					break
				}
			}
		}
	}
	if view.Completed {
		c.Redirect(http.StatusSeeOther, base+"/"+sessionID.String()+"/result")
		return
	}
	h.render(c, http.StatusOK, SessionPage(h.brand, view, h.csrfToken(c)))
}

func (h *Handler) submitAnswer(c *gin.Context, kind, base string) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.renderError(c, http.StatusNotFound, "Сессия не найдена")
		return
	}
	itemID, err := uuid.Parse(c.Param("itemId"))
	if err != nil {
		h.renderError(c, http.StatusNotFound, "Задание не найдено")
		return
	}
	if !h.requireKind(c, user.ID, sessionID, kind) {
		return
	}
	answer := strings.Join(c.PostFormArray("answer"), ",")
	result, err := h.service.SaveAnswer(c.Request.Context(), user.ID, sessionID, itemID, answer)
	if err != nil {
		h.renderPageError(c, err)
		return
	}
	if result.NextHref == "" {
		result.NextHref = base + "/" + sessionID.String()
	}
	c.Redirect(http.StatusSeeOther, result.NextHref)
}

func (h *Handler) submitSession(c *gin.Context, kind, base string) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.renderError(c, http.StatusNotFound, "Сессия не найдена")
		return
	}
	if !h.requireKind(c, user.ID, sessionID, kind) {
		return
	}
	if _, err := h.service.CompleteSession(c.Request.Context(), user.ID, sessionID); err != nil {
		h.renderPageError(c, err)
		return
	}
	c.Redirect(http.StatusSeeOther, base+"/"+sessionID.String()+"/result")
}

func (h *Handler) showResult(c *gin.Context, kind, base string) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.renderError(c, http.StatusNotFound, "Сессия не найдена")
		return
	}
	view, err := h.service.Result(c.Request.Context(), user.ID, sessionID)
	if errors.Is(err, ErrResultNotReady) {
		c.Redirect(http.StatusSeeOther, base+"/"+sessionID.String())
		return
	}
	if err != nil {
		h.renderPageError(c, err)
		return
	}
	if view.Session.Kind != kind {
		h.renderError(c, http.StatusNotFound, "Сессия не найдена")
		return
	}
	h.render(c, http.StatusOK, ResultPage(h.brand, *view))
}

func (h *Handler) requireKind(c *gin.Context, userID, sessionID uuid.UUID, kind string) bool {
	actual, err := h.service.SessionKind(c.Request.Context(), userID, sessionID)
	if err != nil {
		h.renderPageError(c, err)
		return false
	}
	if actual != kind {
		h.renderError(c, http.StatusNotFound, "Сессия не найдена")
		return false
	}

	return true
}

func parsePracticeChoice(value string) (PracticeInput, error) {
	value = strings.TrimSpace(value)
	prefix, raw, found := strings.Cut(value, ":")
	if !found {
		return PracticeInput{}, invalid("choice", "выберите тему, номер КИМ или пункт плана")
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return PracticeInput{}, invalid("choice", "некорректный идентификатор")
	}

	switch prefix {
	case "topic":
		return PracticeInput{TopicID: &id}, nil
	case "kim":
		return PracticeInput{KimNumberID: &id}, nil
	case "plan":
		return PracticeInput{PlanItemID: &id}, nil
	default:
		return PracticeInput{}, invalid("choice", "выберите тему, номер КИМ или пункт плана")
	}
}

// APICreateSession обрабатывает POST /api/sessions.
func (h *Handler) APICreateSession(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		httpserver.AbortWithError(c, http.StatusUnauthorized, "unauthorized", "Требуется вход в систему")
		return
	}
	var req struct {
		Kind        string  `json:"kind"`
		TopicID     *string `json:"topic_id"`
		KimNumberID *string `json:"kim_number_id"`
		PlanItemID  *string `json:"plan_item_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpserver.AbortWithError(c, http.StatusUnprocessableEntity, "validation_failed", "Проверьте тело запроса")
		return
	}

	var (
		session *Session
		err     error
	)
	switch strings.TrimSpace(req.Kind) {
	case SessionKindPractice:
		input := PracticeInput{}
		if input.TopicID, err = parseOptionalUUID(req.TopicID); err != nil {
			httpserver.AbortWithError(c, http.StatusUnprocessableEntity, "validation_failed", "Некорректный topic_id")
			return
		}
		if input.KimNumberID, err = parseOptionalUUID(req.KimNumberID); err != nil {
			httpserver.AbortWithError(c, http.StatusUnprocessableEntity, "validation_failed", "Некорректный kim_number_id")
			return
		}
		if input.PlanItemID, err = parseOptionalUUID(req.PlanItemID); err != nil {
			httpserver.AbortWithError(c, http.StatusUnprocessableEntity, "validation_failed", "Некорректный plan_item_id")
			return
		}
		session, err = h.service.StartPractice(c.Request.Context(), user.ID, input)
	case SessionKindReview:
		session, err = h.service.StartReview(c.Request.Context(), user.ID)
	default:
		httpserver.AbortWithError(c, http.StatusUnprocessableEntity, "unsupported_kind", "Поддерживаются виды practice и review")
		return
	}
	if err != nil {
		h.apiError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"session": gin.H{"id": session.ID, "kind": session.Kind, "status": session.Status}})
}

func parseOptionalUUID(value *string) (*uuid.UUID, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, nil
	}
	parsed, err := uuid.Parse(strings.TrimSpace(*value))
	if err != nil {
		return nil, err
	}

	return &parsed, nil
}

func (h *Handler) apiError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpserver.AbortWithError(c, http.StatusNotFound, "not_found", "Данные не найдены")
	case errors.Is(err, ErrForbidden):
		httpserver.AbortWithError(c, http.StatusForbidden, "forbidden", "Доступ запрещён")
	case errors.Is(err, ErrNoGoal):
		httpserver.AbortWithError(c, http.StatusConflict, "no_goal", "Сначала задайте учебную цель")
	case errors.Is(err, ErrNoPlan):
		httpserver.AbortWithError(c, http.StatusNotFound, "no_plan", "Учебный план не найден")
	case errors.Is(err, ErrExamPassed), errors.Is(err, ErrInvalidExamDate), errors.Is(err, ErrInvalidTargetScore),
		errors.Is(err, ErrInvalidWeeklyMinutes), errors.Is(err, ErrInvalidExamCode):
		httpserver.AbortWithError(c, http.StatusUnprocessableEntity, "validation_failed", err.Error())
	case errors.Is(err, ErrExamUnavailable):
		httpserver.AbortWithError(c, http.StatusUnprocessableEntity, "exam_unavailable", "Трек не найден или не опубликован")
	case errors.Is(err, ErrInsufficientContent):
		httpserver.AbortWithError(c, http.StatusUnprocessableEntity, "insufficient_content", "Недостаточно опубликованных заданий для диагностики")
	case errors.Is(err, ErrContentUnavailable):
		httpserver.AbortWithError(c, http.StatusUnprocessableEntity, "insufficient_content", "Для выбранного направления нет опубликованных заданий")
	case errors.Is(err, ErrNoReviewTasks):
		httpserver.AbortWithError(c, http.StatusUnprocessableEntity, "no_review_tasks", "Нет заданий для повторения")
	case errors.Is(err, ErrSessionCompleted):
		httpserver.AbortWithError(c, http.StatusConflict, "session_completed", "Сессия уже завершена")
	case errors.Is(err, ErrResultNotReady):
		httpserver.AbortWithError(c, http.StatusConflict, "result_not_ready", "Результат ещё не готов")
	case errors.Is(err, ErrUnsupportedKind), errors.Is(err, ErrInvalidSessionKind):
		httpserver.AbortWithError(c, http.StatusUnprocessableEntity, "unsupported_kind", err.Error())
	case isValidationError(err):
		httpserver.AbortWithError(c, http.StatusUnprocessableEntity, "validation_failed", err.Error())
	case errors.Is(err, profile.ErrExamUnavailable):
		httpserver.AbortWithError(c, http.StatusUnprocessableEntity, "exam_unavailable", "Трек не найден или не опубликован")
	case errors.Is(err, profile.ErrInvalidExamDate):
		httpserver.AbortWithError(c, http.StatusUnprocessableEntity, "validation_failed", "Дата экзамена должна быть в будущем")
	case errors.Is(err, profile.ErrInvalidTargetScore):
		httpserver.AbortWithError(c, http.StatusUnprocessableEntity, "validation_failed", "Целевой балл должен быть от 0 до 100")
	case errors.Is(err, profile.ErrInvalidWeeklyMinutes):
		httpserver.AbortWithError(c, http.StatusUnprocessableEntity, "validation_failed", "Недельное время должно быть от 1 до 10080 минут")
	default:
		h.logger.Error("ошибка учебного цикла", "path", c.Request.URL.Path, "error", err)
		httpserver.AbortWithError(c, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка")
	}
}

func isValidationError(err error) bool {
	var validationErr *ValidationError

	return errors.As(err, &validationErr)
}

func (h *Handler) renderPageError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		h.renderError(c, http.StatusNotFound, "Данные не найдены")
	case errors.Is(err, ErrNoGoal):
		h.render(c, http.StatusOK, GoalPage(h.brand, GoalPageData{FormError: "Сначала задайте учебную цель."}, h.csrfToken(c)))
	case errors.Is(err, ErrInsufficientContent):
		h.render(c, http.StatusUnprocessableEntity, DiagnosticsPage(h.brand, DiagnosticStartData{
			Available: false,
			Reason:    "Недостаточно опубликованных заданий, чтобы покрыть все номера КИМ.",
		}, h.csrfToken(c)))
	case errors.Is(err, ErrContentUnavailable):
		h.render(c, http.StatusUnprocessableEntity, PracticePage(h.brand, PracticeStartData{
			Available: false,
			Reason:    "Для выбранной темы или номера КИМ нет опубликованных заданий с автоматической проверкой. Выберите другое направление.",
		}, h.csrfToken(c)))
	case errors.Is(err, ErrNoReviewTasks):
		h.render(c, http.StatusOK, ReviewPage(h.brand, ReviewStartData{
			Available: false,
			Reason:    "Нет заданий для повторения. Перейдите к практике или диагностике.",
		}, h.csrfToken(c)))
	case errors.Is(err, ErrExamUnavailable):
		h.renderError(c, http.StatusUnprocessableEntity, "Трек не найден или не опубликован")
	default:
		h.logger.Error("ошибка страницы кабинета", "path", c.Request.URL.Path, "error", err)
		h.renderError(c, http.StatusInternalServerError, "Внутренняя ошибка")
	}
}

func (h *Handler) renderError(c *gin.Context, status int, message string) {
	h.render(c, status, ErrorPage(h.brand, status, message))
}

func (h *Handler) render(c *gin.Context, status int, component templ.Component) {
	var buf bytes.Buffer
	if err := component.Render(c.Request.Context(), &buf); err != nil {
		h.logger.Error("не удалось отрендерить страницу кабинета", "path", c.Request.URL.Path, "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Data(status, "text/html; charset=utf-8", buf.Bytes())
}

func (h *Handler) csrfToken(c *gin.Context) string {
	if h.csrf == nil {
		return ""
	}

	return h.csrf.Token(c)
}

func goalFieldErrors(err error) map[string]string {
	fields := map[string]string{}
	switch {
	case errors.Is(err, ErrInvalidExamCode), errors.Is(err, ErrExamUnavailable), errors.Is(err, profile.ErrExamUnavailable):
		fields["exam_code"] = "Выберите опубликованный трек."
	case errors.Is(err, ErrInvalidExamDate), errors.Is(err, ErrExamPassed), errors.Is(err, profile.ErrInvalidExamDate):
		fields["exam_date"] = "Укажите будущую дату экзамена."
	case errors.Is(err, ErrInvalidTargetScore), errors.Is(err, profile.ErrInvalidTargetScore):
		fields["target_score"] = "Целевой балл должен быть от 0 до 100."
	case errors.Is(err, ErrInvalidWeeklyMinutes), errors.Is(err, profile.ErrInvalidWeeklyMinutes):
		fields["weekly_minutes"] = "Недельное время должно быть от 1 до 10080 минут."
	}

	return fields
}

func goalFormError(err error) string {
	if len(goalFieldErrors(err)) > 0 {
		return "Проверьте выделенные поля."
	}

	return "Не удалось сохранить цель. Попробуйте ещё раз."
}

func planResponse(view PlanView) any {
	if view.Plan == nil {
		return nil
	}
	plan := view.Plan
	response := PlanResponse{
		ID:            plan.ID,
		ExamID:        plan.ExamID,
		ExamDate:      plan.ExamDate.Format(dateLayout),
		TargetScore:   plan.TargetScore,
		WeeklyMinutes: plan.WeeklyMinutes,
		Status:        string(plan.Status),
		Version:       plan.Version,
		Weeks:         make([]WeekResponse, 0, len(view.Weeks)),
	}
	for _, week := range view.Weeks {
		weekResponse := WeekResponse{
			WeekStart: week.Week.WeekStart.Format(dateLayout),
			Phase:     string(week.Week.Phase),
			Items:     make([]ItemResponse, 0, len(week.Items)),
		}
		for _, item := range week.Items {
			weekResponse.Items = append(weekResponse.Items, ItemResponse{
				ID:          item.Item.ID,
				DayOfWeek:   item.Item.DayOfWeek,
				Kind:        string(item.Item.Kind),
				TopicID:     item.Item.TopicID,
				KimNumberID: item.Item.KimNumberID,
				TaskCount:   item.Item.TaskCount,
				Status:      string(item.Item.Status),
			})
		}
		response.Weeks = append(response.Weeks, weekResponse)
	}

	return response
}

func sessionResponse(view SessionView) SessionResponse {
	response := SessionResponse{
		ID:        view.Session.ID,
		Kind:      view.Session.Kind,
		Status:    string(view.Session.Status),
		StartedAt: view.Session.StartedAt,
		Items:     make([]SessionItemResponse, 0, len(view.Items)),
	}
	for _, item := range view.Items {
		entry := SessionItemResponse{
			ID:        item.Item.ID,
			Position:  item.Item.Position,
			Status:    string(item.Item.Status),
			Statement: item.Snapshot.StatementMD,
			Options:   make([]SessionOptionResponse, 0, len(item.Snapshot.Options)),
		}
		for _, option := range item.Snapshot.Options {
			entry.Options = append(entry.Options, SessionOptionResponse(option))
		}
		if item.Answer != nil {
			answer := item.Answer.RawAnswer
			correct := item.Answer.IsCorrect
			entry.Answer = &answer
			entry.IsCorrect = &correct
		}
		response.Items = append(response.Items, entry)
	}

	return response
}

func resultResponse(result *SessionResult) ResultResponse {
	return ResultResponse{
		SessionID:    result.SessionID,
		PrimaryScore: result.PrimaryScore,
		MaxScore:     result.MaxScore,
		Breakdown:    result.Breakdown,
	}
}
