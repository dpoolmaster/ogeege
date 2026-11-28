package learning

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"oge-ege/internal/auth"
	"oge-ege/internal/shared/httpserver"
	"oge-ege/internal/shared/render/ui"
)

type fakeService struct {
	todayData      TodayData
	todayErr       error
	goalPageData   GoalPageData
	goalPageErr    error
	setupGoalErr   error
	prepareData    DiagnosticStartData
	prepareErr     error
	startSession   *Session
	startErr       error
	practiceData   PracticeStartData
	practiceErr    error
	reviewData     ReviewStartData
	reviewErr      error
	sessionKind    string
	sessionKindErr error
	sessionView    SessionView
	sessionErr     error
	answerResult   AnswerResult
	answerErr      error
	completeResult *SessionResult
	completeErr    error
	resultView     *ResultView
	resultErr      error
	gapsView       GapMapView
	gapsErr        error
	planView       PlanView
	planErr        error
	replanPlan     *Plan
	replanErr      error
	markErr        error
}

func (f *fakeService) Today(context.Context, uuid.UUID) (TodayData, error) {
	return f.todayData, f.todayErr
}

func (f *fakeService) GoalPageData(context.Context, uuid.UUID) (GoalPageData, error) {
	return f.goalPageData, f.goalPageErr
}

func (f *fakeService) SetupGoal(context.Context, uuid.UUID, GoalInput) (*Plan, error) {
	return &Plan{}, f.setupGoalErr
}

func (f *fakeService) PrepareDiagnostic(context.Context, uuid.UUID, string) (DiagnosticStartData, error) {
	return f.prepareData, f.prepareErr
}

func (f *fakeService) StartDiagnostic(context.Context, uuid.UUID, string) (*Session, error) {
	return f.startSession, f.startErr
}

func (f *fakeService) PracticeStartData(context.Context, uuid.UUID) (PracticeStartData, error) {
	return f.practiceData, f.practiceErr
}

func (f *fakeService) StartPractice(context.Context, uuid.UUID, PracticeInput) (*Session, error) {
	return f.startSession, f.startErr
}

func (f *fakeService) ReviewStartData(context.Context, uuid.UUID) (ReviewStartData, error) {
	return f.reviewData, f.reviewErr
}

func (f *fakeService) StartReview(context.Context, uuid.UUID) (*Session, error) {
	return f.startSession, f.startErr
}

func (f *fakeService) SessionKind(context.Context, uuid.UUID, uuid.UUID) (string, error) {
	return f.sessionKind, f.sessionKindErr
}

func (f *fakeService) SessionView(context.Context, uuid.UUID, uuid.UUID) (SessionView, error) {
	return f.sessionView, f.sessionErr
}

func (f *fakeService) SaveAnswer(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string) (AnswerResult, error) {
	return f.answerResult, f.answerErr
}

func (f *fakeService) CompleteSession(context.Context, uuid.UUID, uuid.UUID) (*SessionResult, error) {
	return f.completeResult, f.completeErr
}

func (f *fakeService) Result(context.Context, uuid.UUID, uuid.UUID) (*ResultView, error) {
	return f.resultView, f.resultErr
}

func (f *fakeService) Gaps(context.Context, uuid.UUID) (GapMapView, error) {
	return f.gapsView, f.gapsErr
}

func (f *fakeService) PlanView(context.Context, uuid.UUID) (PlanView, error) {
	return f.planView, f.planErr
}

func (f *fakeService) Replan(context.Context, uuid.UUID) (*Plan, error) {
	return f.replanPlan, f.replanErr
}

func (f *fakeService) MarkItemDone(context.Context, uuid.UUID, uuid.UUID) error {
	return f.markErr
}

func newTestHandler(service ServiceAPI) *Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	return NewHandler(service, nil, logger, "Верно")
}

func testContext(method, target string, body io.Reader) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest(method, target, body)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	context.Request = request

	return context, recorder
}

func withUser(context *gin.Context) {
	auth.WithUser(context, &auth.User{ID: uuid.New(), Role: auth.RoleStudent, Status: auth.StatusActive})
}

func TestShowTodayRedirectsAnonymous(t *testing.T) {
	handler := newTestHandler(&fakeService{})
	context, recorder := testContext(http.MethodGet, "/app", nil)

	handler.ShowToday(context)

	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("статус = %d, ожидался 303", recorder.Code)
	}
	if recorder.Header().Get("Location") != "/login" {
		t.Fatalf("редирект = %q", recorder.Header().Get("Location"))
	}
}

func TestShowTodayRendersSingleHeading(t *testing.T) {
	handler := newTestHandler(&fakeService{todayData: TodayData{
		HasGoal:  true,
		NextStep: "Изучите теорию",
		NextHref: "/app/plan",
		Goal:     "Цель: 70 баллов",
		GoalMax:  100,
	}})
	context, recorder := testContext(http.MethodGet, "/app", nil)
	withUser(context)

	handler.ShowToday(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("статус = %d", recorder.Code)
	}
	body := recorder.Body.String()
	if strings.Count(body, "<h1") != 1 {
		t.Fatalf("на странице должен быть один h1")
	}
	if !strings.Contains(body, "Сегодня") || !strings.Contains(body, "Изучите теорию") {
		t.Fatal("страница не содержит ожидаемых данных")
	}
}

func TestSubmitGoalValidationError(t *testing.T) {
	service := &fakeService{
		setupGoalErr: ErrInvalidExamDate,
		goalPageData: GoalPageData{Exams: []ExamOption{{Code: "oge_math", Title: "ОГЭ математика"}}},
	}
	handler := newTestHandler(service)
	body := strings.NewReader("exam_code=oge_math&exam_date=2020-01-01&target_score=50&weekly_minutes=300")
	context, recorder := testContext(http.MethodPost, "/app/goal", body)
	withUser(context)

	handler.SubmitGoal(context)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("статус = %d, ожидался 422", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "Укажите будущую дату экзамена") {
		t.Fatal("ошибка должна выводиться рядом с полем")
	}
}

func TestSubmitGoalRequiresCSRF(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &fakeService{}
	handler := newTestHandler(service)
	csrf := httpserver.NewCSRF("secret", false, func(*gin.Context) string { return "" })
	router := gin.New()
	router.Use(csrf.Middleware())
	router.POST("/app/goal", handler.SubmitGoal)

	request := httptest.NewRequest(http.MethodPost, "/app/goal", strings.NewReader("exam_code=oge_math"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("статус = %d, ожидался 403", recorder.Code)
	}
}

func TestAPIStartDiagnosticRejectsUnsupportedKind(t *testing.T) {
	handler := newTestHandler(&fakeService{})
	body := strings.NewReader(`{"kind":"practice"}`)
	context, recorder := testContext(http.MethodPost, "/api/diagnostics", body)
	context.Request.Header.Set("Content-Type", "application/json")
	withUser(context)

	handler.APIStartDiagnostic(context)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("статус = %d, ожидался 422", recorder.Code)
	}
	var envelope struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("разобрать конверт ошибки: %v", err)
	}
	if envelope.Error.Code != "unsupported_kind" {
		t.Fatalf("код ошибки = %q", envelope.Error.Code)
	}
}

func TestAPIPlanNoGoalEnvelope(t *testing.T) {
	handler := newTestHandler(&fakeService{planErr: ErrNoGoal})
	context, recorder := testContext(http.MethodGet, "/api/plan", nil)
	withUser(context)

	handler.APIPlan(context)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("статус = %d, ожидался 409", recorder.Code)
	}
	if !bytes.Contains(recorder.Body.Bytes(), []byte(`"code":"no_goal"`)) {
		t.Fatalf("тело ошибки = %s", recorder.Body.String())
	}
}

func TestAPIPlanRequiresAuthentication(t *testing.T) {
	handler := newTestHandler(&fakeService{})
	context, recorder := testContext(http.MethodGet, "/api/plan", nil)

	handler.APIPlan(context)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("статус = %d, ожидался 401", recorder.Code)
	}
}

func TestShowSessionForeignNotFound(t *testing.T) {
	handler := newTestHandler(&fakeService{sessionErr: ErrNotFound})
	context, recorder := testContext(http.MethodGet, "/app/diagnostics/"+uuid.New().String(), nil)
	withUser(context)
	context.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	handler.ShowSession(context)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("статус = %d, ожидался 404", recorder.Code)
	}
}

func TestShowSessionRendersCheckedState(t *testing.T) {
	sessionID := uuid.New()
	itemID := uuid.New()
	snapshot, _ := json.Marshal(TaskSnapshot{StatementMD: "Сколько будет 2+2?", AnswerSpec: json.RawMessage(`{"kind":"number","accepted":["4"]}`), MaxPoints: 1})
	answer := &Answer{RawAnswer: "4", IsCorrect: true, AwardedPoints: 1}
	handler := newTestHandler(&fakeService{sessionView: SessionView{
		Session: Session{ID: sessionID, Status: SessionInProgress},
		Items: []SessionItemView{{
			Item:     SessionItem{ID: itemID, Position: 1, TaskSnapshot: snapshot},
			Snapshot: TaskSnapshot{StatementMD: "Сколько будет 2+2?", AnswerSpec: json.RawMessage(`{"kind":"number","accepted":["4"]}`), MaxPoints: 1},
			Answer:   answer,
		}},
	}})
	context, recorder := testContext(http.MethodGet, "/app/diagnostics/"+sessionID.String(), nil)
	withUser(context)
	context.Params = gin.Params{{Key: "id", Value: sessionID.String()}}

	handler.ShowSession(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("статус = %d", recorder.Code)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `role="status"`) {
		t.Fatal("результат проверки должен иметь role=status")
	}
	if !strings.Contains(body, "/items/"+itemID.String()) {
		t.Fatal("форма ответа должна указывать на задание")
	}
	if !strings.Contains(body, "Сколько будет 2+2?") {
		t.Fatal("условие задания не отображено")
	}
}

func TestShowGapsRendersTextState(t *testing.T) {
	handler := newTestHandler(&fakeService{gapsView: GapMapView{
		Topics:     []string{"Тема 1"},
		KimNumbers: []int{1},
		Cells:      [][]ui.GapCell{{{State: ui.GapNone, Label: "нет связи"}}},
		HasData:    true,
	}})
	context, recorder := testContext(http.MethodGet, "/app/gaps", nil)
	withUser(context)

	handler.ShowGaps(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("статус = %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "нет связи") {
		t.Fatal("карта пробелов должна содержать текстовое состояние ячейки")
	}
}

func TestShowPlanRendersWeeks(t *testing.T) {
	planID := uuid.New()
	handler := newTestHandler(&fakeService{planView: PlanView{
		HasGoal: true,
		Plan: &Plan{
			ID:            planID,
			ExamDate:      time.Date(2027, 6, 1, 0, 0, 0, 0, time.UTC),
			TargetScore:   70,
			WeeklyMinutes: 300,
			Version:       1,
		},
		Weeks: []WeekView{{
			Week: Week{WeekStart: time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC), Phase: PhaseBase},
			Items: []PlanItemView{{
				Item: PlanItem{DayOfWeek: 1, Kind: KindTheory, TaskCount: 10, Status: ItemPending},
			}},
		}},
	}})
	context, recorder := testContext(http.MethodGet, "/app/plan", nil)
	withUser(context)

	handler.ShowPlan(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("статус = %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "Неделя с 16.11.2026") {
		t.Fatal("страница плана не содержит неделю")
	}
}

func TestShowPracticeRendersForm(t *testing.T) {
	handler := newTestHandler(&fakeService{practiceData: PracticeStartData{
		Available: true,
		Choices:   []PracticeChoice{{Value: "topic:" + uuid.New().String(), Label: "Тема: Алгебра"}},
	}})
	context, recorder := testContext(http.MethodGet, "/app/practice", nil)
	withUser(context)

	handler.ShowPractice(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("статус = %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "Начать практику") {
		t.Fatal("экран должен содержать кнопку старта практики")
	}
}

func TestShowPracticeSessionRejectsOtherKind(t *testing.T) {
	handler := newTestHandler(&fakeService{sessionView: SessionView{
		Session: Session{ID: uuid.New(), Kind: SessionKindDiagnostic},
	}})
	context, recorder := testContext(http.MethodGet, "/app/practice/"+uuid.New().String(), nil)
	withUser(context)
	context.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	handler.ShowPracticeSession(context)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("статус = %d, ожидался 404", recorder.Code)
	}
}

func TestSubmitStartPracticeRedirects(t *testing.T) {
	sessionID := uuid.New()
	handler := newTestHandler(&fakeService{startSession: &Session{ID: sessionID, Kind: SessionKindPractice}})
	body := strings.NewReader("choice=topic:" + uuid.New().String())
	context, recorder := testContext(http.MethodPost, "/app/practice", body)
	withUser(context)

	handler.SubmitStartPractice(context)
	context.Writer.WriteHeaderNow()

	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("статус = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if recorder.Header().Get("Location") != "/app/practice/"+sessionID.String() {
		t.Fatalf("редирект = %q", recorder.Header().Get("Location"))
	}
}

func TestSubmitStartReviewEmptyState(t *testing.T) {
	handler := newTestHandler(&fakeService{
		startErr:   ErrNoReviewTasks,
		reviewData: ReviewStartData{Available: true},
	})
	context, recorder := testContext(http.MethodPost, "/app/review", strings.NewReader(""))
	withUser(context)

	handler.SubmitStartReview(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("статус = %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "Нет заданий для повторения") {
		t.Fatal("должно отображаться пустое состояние")
	}
}

func TestAPICreateSessionUnsupportedKind(t *testing.T) {
	handler := newTestHandler(&fakeService{})
	context, recorder := testContext(http.MethodPost, "/api/sessions", strings.NewReader(`{"kind":"mock"}`))
	withUser(context)

	handler.APICreateSession(context)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("статус = %d", recorder.Code)
	}
}

func TestSubmitStartPracticeNoContentShowsReason(t *testing.T) {
	handler := newTestHandler(&fakeService{startErr: ErrContentUnavailable})
	body := strings.NewReader("choice=topic:" + uuid.New().String())
	context, recorder := testContext(http.MethodPost, "/app/practice", body)
	withUser(context)

	handler.SubmitStartPractice(context)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("статус = %d, ожидался 422", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "нет опубликованных заданий") {
		t.Fatal("должно отображаться объяснение причины")
	}
}
