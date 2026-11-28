// Package learning реализует учебную цель, диагностику, освоение, карту пробелов
// и недельный план подготовки ученика.
package learning

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// PlanStatus — статус версии учебного плана.
type PlanStatus string

const (
	PlanActive     PlanStatus = "active"
	PlanSuperseded PlanStatus = "superseded"
)

// ItemStatus — статус пункта плана.
type ItemStatus string

const (
	ItemPending ItemStatus = "pending"
	ItemDone    ItemStatus = "done"
)

// SessionStatus — статус учебной сессии.
type SessionStatus string

const (
	SessionInProgress SessionStatus = "in_progress"
	SessionCompleted  SessionStatus = "completed"
)

// SessionItemStatus — статус элемента сессии.
type SessionItemStatus string

const (
	SessionItemPending  SessionItemStatus = "pending"
	SessionItemAnswered SessionItemStatus = "answered"
)

// Phase — фаза недели плана.
type Phase string

const (
	PhaseDiagnostics Phase = "diagnostics"
	PhaseBase        Phase = "base"
	PhasePractice    Phase = "practice"
	PhaseReview      Phase = "review"
)

// ItemKind — вид работы пункта плана.
type ItemKind string

const (
	KindDiagnostic ItemKind = "diagnostic"
	KindTheory     ItemKind = "theory"
	KindPractice   ItemKind = "practice"
	KindReview     ItemKind = "review"
)

// SessionKindDiagnostic — поддерживаемый на этапе вид сессии.
const SessionKindDiagnostic = "diagnostic"

// Plan — версия учебного плана.
type Plan struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	ExamID          uuid.UUID
	ExamDate        time.Time
	TargetScore     int
	WeeklyMinutes   int
	Status          PlanStatus
	Version         int
	LastReplannedAt *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Week — неделя плана.
type Week struct {
	ID        uuid.UUID
	PlanID    uuid.UUID
	WeekStart time.Time
	Phase     Phase
	Goals     json.RawMessage
}

// PlanItem — пункт плана.
type PlanItem struct {
	ID          uuid.UUID
	WeekID      uuid.UUID
	DayOfWeek   int
	Kind        ItemKind
	TopicID     *uuid.UUID
	KimNumberID *uuid.UUID
	TaskCount   int
	Status      ItemStatus
}

// Session — учебная сессия.
type Session struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	ExamID       uuid.UUID
	Kind         string
	VariantID    *uuid.UUID
	Status       SessionStatus
	StartedAt    time.Time
	DeadlineAt   *time.Time
	FinishedAt   *time.Time
	ScorePrimary *float64
	CreatedAt    time.Time
}

// SessionItem — задание внутри сессии.
type SessionItem struct {
	ID           uuid.UUID
	SessionID    uuid.UUID
	Position     int
	TaskID       uuid.UUID
	TaskSnapshot json.RawMessage
	MaxPoints    float64
	Status       SessionItemStatus
}

// Answer — сохранённый ответ на элемент сессии.
type Answer struct {
	ID               uuid.UUID
	SessionItemID    uuid.UUID
	RawAnswer        string
	NormalizedAnswer string
	IsCorrect        bool
	AwardedPoints    float64
	CheckKind        string
	CheckedAt        time.Time
	CreatedAt        time.Time
}

// TopicMastery — освоение темы.
type TopicMastery struct {
	UserID          uuid.UUID
	TopicID         uuid.UUID
	Mastery         float64
	Attempts        int
	Correct         int
	IntervalDays    int
	Ease            float64
	LastPracticedAt *time.Time
	NextReviewAt    *time.Time
}

// KimMastery — освоение номера КИМ.
type KimMastery struct {
	UserID          uuid.UUID
	KimNumberID     uuid.UUID
	Mastery         float64
	Attempts        int
	Correct         int
	LastPracticedAt *time.Time
}

// SessionResult — сохранённый результат сессии.
type SessionResult struct {
	SessionID    uuid.UUID
	PrimaryScore float64
	MaxScore     float64
	TestScore    *int
	Grade        *int
	Breakdown    json.RawMessage
}

// TaskSnapshot — неизменяемый снимок задания на момент старта сессии.
type TaskSnapshot struct {
	TaskID        uuid.UUID            `json:"task_id"`
	Version       int                  `json:"version"`
	StatementMD   string               `json:"statement_md"`
	AnswerSpec    json.RawMessage      `json:"answer_spec"`
	AnswerDisplay string               `json:"answer_display"`
	Difficulty    string               `json:"difficulty"`
	KimNumberID   uuid.UUID            `json:"kim_number_id"`
	TopicID       *uuid.UUID           `json:"topic_id,omitempty"`
	MaxPoints     float64              `json:"max_points"`
	Attachments   []AttachmentSnapshot `json:"attachments"`
}

// AttachmentSnapshot — вложение задания в снимке.
type AttachmentSnapshot struct {
	Kind string `json:"kind"`
	URL  string `json:"url"`
	Alt  string `json:"alt"`
}

// Ошибки модуля learning.
var (
	ErrNotFound             = errors.New("данные не найдены")
	ErrConflict             = errors.New("конфликт данных")
	ErrForbidden            = errors.New("доступ запрещён")
	ErrNoGoal               = errors.New("учебная цель не задана")
	ErrExamPassed           = errors.New("дата экзамена уже прошла")
	ErrExamUnavailable      = errors.New("трек не найден или не опубликован")
	ErrInvalidExamDate      = errors.New("дата экзамена должна быть в будущем")
	ErrInvalidTargetScore   = errors.New("целевой балл должен быть от 0 до 100")
	ErrInvalidWeeklyMinutes = errors.New("недельное время должно быть от 1 до 10080 минут")
	ErrInvalidExamCode      = errors.New("выберите учебный трек")
	ErrInsufficientContent  = errors.New("недостаточно опубликованных заданий для диагностики")
	ErrSessionCompleted     = errors.New("сессия уже завершена")
	ErrUnsupportedKind      = errors.New("вид сессии пока не поддерживается")
	ErrNoPlan               = errors.New("учебный план не найден")
	ErrInvalidSessionKind   = errors.New("поддерживается только диагностическая сессия")
	ErrResultNotReady       = errors.New("результат ещё не готов")
)

// ValidationError описывает ошибку проверки поля.
type ValidationError struct {
	Field   string
	Message string
}

// Error возвращает текст ошибки.
func (e *ValidationError) Error() string {
	if e.Field == "" {
		return e.Message
	}

	return e.Field + ": " + e.Message
}

func invalid(field, message string) error {
	return &ValidationError{Field: field, Message: message}
}
