package learning

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"oge-ege/internal/shared/render/ui"
)

// GoalInput содержит данные формы учебной цели.
type GoalInput struct {
	ExamCode      string
	ExamDate      string
	TargetScore   int
	WeeklyMinutes int
}

// AnswerInput содержит ответ ученика.
type AnswerInput struct {
	Answer string
}

// ExamOption — трек для выбора в учебной цели.
type ExamOption struct {
	Code  string
	Title string
}

// TodayData описывает экран «Сегодня».
type TodayData struct {
	HasGoal     bool
	HasPlan     bool
	NextStep    string
	NextHref    string
	Gaps        []ui.GapSummary
	Goal        string
	GoalCurrent int
	GoalMax     int
	Empty       bool
}

// GoalPageData описывает экран учебной цели.
type GoalPageData struct {
	Exams          []ExamOption
	Input          GoalInput
	FieldErrors    map[string]string
	FormError      string
	AlreadyHasGoal bool
}

// DiagnosticStartData описывает экран старта диагностики.
type DiagnosticStartData struct {
	Available       bool
	DurationMinutes int
	TaskCount       int
	Reason          string
	HasActive       bool
	ActiveSessionID uuid.UUID
}

// PracticeInput содержит параметры запуска практики.
type PracticeInput struct {
	TopicID     *uuid.UUID
	KimNumberID *uuid.UUID
	PlanItemID  *uuid.UUID
	TargetCount int
}

// PracticeChoice — вариант выбора на экране старта практики.
type PracticeChoice struct {
	Value string
	Label string
}

// PracticeStartData описывает экран старта практики.
type PracticeStartData struct {
	Available       bool
	Reason          string
	Choices         []PracticeChoice
	Selected        string
	HasActive       bool
	ActiveSessionID uuid.UUID
}

// ReviewStartData описывает экран работы над ошибками.
type ReviewStartData struct {
	Available       bool
	Reason          string
	HasActive       bool
	ActiveSessionID uuid.UUID
}

// SessionItemView объединяет элемент сессии, снимок и ответ.
type SessionItemView struct {
	Item     SessionItem
	Snapshot TaskSnapshot
	Answer   *Answer
}

// SessionView описывает прохождение диагностики.
type SessionView struct {
	Session      Session
	Items        []SessionItemView
	CurrentIndex int
	Completed    bool
	Result       *SessionResult
}

// AnswerResult описывает результат проверки ответа.
type AnswerResult struct {
	Correct    bool
	Reason     string
	Normalized string
	Expected   []string
	NextHref   string
	Finished   bool
}

// TopicBreakdown — разбивка результата по теме.
type TopicBreakdown struct {
	TopicID uuid.UUID
	Title   string
	Correct int
	Total   int
}

// KimBreakdown — разбивка результата по номеру КИМ.
type KimBreakdown struct {
	KimNumberID uuid.UUID
	Number      int
	Correct     int
	Total       int
}

// ResultView описывает экран результата диагностики.
type ResultView struct {
	Session Session
	Result  SessionResult
	Items   []BreakdownItem
	Topics  []TopicBreakdown
	Kims    []KimBreakdown
}

// BreakdownItem — строка разбивки результата.
type BreakdownItem struct {
	Position   int
	KimNumber  int
	TopicTitle string
	Correct    bool
	Awarded    float64
	MaxPoints  float64
	Reason     string
	RawAnswer  string
}

// GapMapView описывает карту пробелов.
type GapMapView struct {
	Topics          []string
	KimNumbers      []int
	Cells           [][]ui.GapCell
	HasData         bool
	Recommendations []ui.GapSummary
}

// PlanItemView объединяет пункт плана и подписи.
type PlanItemView struct {
	Item       PlanItem
	TopicTitle string
	KimNumber  int
	Done       bool
}

// WeekView описывает неделю плана.
type WeekView struct {
	Week      Week
	Items     []PlanItemView
	TotalLoad int
}

// PlanView описывает экран плана.
type PlanView struct {
	HasGoal bool
	Reason  string
	Plan    *Plan
	Weeks   []WeekView
	Empty   bool
}

// breakdownPayload — структура breakdown_json результата.
type breakdownPayload struct {
	Items  []breakdownEntry `json:"items"`
	Topics []topicEntry     `json:"topics"`
	Kims   []kimEntry       `json:"kims"`
}

type breakdownEntry struct {
	Position    int        `json:"position"`
	TaskID      uuid.UUID  `json:"task_id"`
	KimNumberID uuid.UUID  `json:"kim_number_id"`
	TopicID     *uuid.UUID `json:"topic_id,omitempty"`
	Correct     bool       `json:"correct"`
	Awarded     float64    `json:"awarded"`
	MaxPoints   float64    `json:"max_points"`
	Reason      string     `json:"reason"`
	RawAnswer   string     `json:"raw_answer"`
}

type topicEntry struct {
	TopicID uuid.UUID `json:"topic_id"`
	Correct int       `json:"correct"`
	Total   int       `json:"total"`
}

type kimEntry struct {
	KimNumberID uuid.UUID `json:"kim_number_id"`
	Correct     int       `json:"correct"`
	Total       int       `json:"total"`
}

// PlanSnapshot — снимок плана для JSON-ответов.
type PlanResponse struct {
	ID            uuid.UUID      `json:"id"`
	ExamID        uuid.UUID      `json:"exam_id"`
	ExamDate      string         `json:"exam_date"`
	TargetScore   int            `json:"target_score"`
	WeeklyMinutes int            `json:"weekly_minutes"`
	Status        string         `json:"status"`
	Version       int            `json:"version"`
	Weeks         []WeekResponse `json:"weeks"`
}

// WeekResponse — неделя в JSON-ответе.
type WeekResponse struct {
	WeekStart string         `json:"week_start"`
	Phase     string         `json:"phase"`
	Items     []ItemResponse `json:"items"`
}

// ItemResponse — пункт плана в JSON-ответе.
type ItemResponse struct {
	ID          uuid.UUID  `json:"id"`
	DayOfWeek   int        `json:"day_of_week"`
	Kind        string     `json:"kind"`
	TopicID     *uuid.UUID `json:"topic_id,omitempty"`
	KimNumberID *uuid.UUID `json:"kim_number_id,omitempty"`
	TaskCount   int        `json:"task_count"`
	Status      string     `json:"status"`
}

// SessionResponse — сессия в JSON-ответе.
type SessionResponse struct {
	ID        uuid.UUID             `json:"id"`
	Kind      string                `json:"kind"`
	Status    string                `json:"status"`
	StartedAt time.Time             `json:"started_at"`
	Items     []SessionItemResponse `json:"items"`
}

// SessionItemResponse — элемент сессии в JSON-ответе.
type SessionItemResponse struct {
	ID        uuid.UUID               `json:"id"`
	Position  int                     `json:"position"`
	Status    string                  `json:"status"`
	Statement string                  `json:"statement_md"`
	Answer    *string                 `json:"answer,omitempty"`
	IsCorrect *bool                   `json:"is_correct,omitempty"`
	Options   []SessionOptionResponse `json:"options,omitempty"`
}

// SessionOptionResponse — вариант ответа в JSON-сессии без признака правильности.
type SessionOptionResponse struct {
	ID        uuid.UUID `json:"id"`
	Position  int       `json:"position"`
	ContentMD string    `json:"content_md"`
}

// ResultResponse — результат сессии в JSON-ответе.
type ResultResponse struct {
	SessionID    uuid.UUID       `json:"session_id"`
	PrimaryScore float64         `json:"primary_score"`
	MaxScore     float64         `json:"max_score"`
	Breakdown    json.RawMessage `json:"breakdown"`
}
