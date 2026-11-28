// Package content реализует учебный контент: треки, структуру КИМ, темы,
// теорию, банк заданий, варианты, рубрики и шкалы перевода.
package content

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// SubjectOption — допустимый предмет трека: код и отображаемое название.
type SubjectOption struct {
	Code  string
	Title string
}

var subjects = []SubjectOption{
	{Code: "math", Title: "Математика"},
	{Code: "physics", Title: "Физика"},
	{Code: "russian", Title: "Русский язык"},
	{Code: "informatics", Title: "Информатика"},
}

// Subjects возвращает допустимые предметы в порядке отображения.
func Subjects() []SubjectOption {
	result := make([]SubjectOption, len(subjects))
	copy(result, subjects)

	return result
}

// ValidSubject сообщает, входит ли код предмета в список допустимых.
func ValidSubject(code string) bool {
	for _, subject := range subjects {
		if subject.Code == code {
			return true
		}
	}

	return false
}

// Status — статус публикации сущности контента.
type Status string

const (
	StatusDraft     Status = "draft"
	StatusPublished Status = "published"
	StatusArchived  Status = "archived"
)

// Valid сообщает, известен ли статус.
func (s Status) Valid() bool {
	switch s {
	case StatusDraft, StatusPublished, StatusArchived:
		return true
	default:
		return false
	}
}

// AnswerType — тип ответа задания.
type AnswerType string

const (
	AnswerTypeShort    AnswerType = "short"
	AnswerTypeExtended AnswerType = "extended"
)

// Valid сообщает, известен ли тип ответа.
func (a AnswerType) Valid() bool {
	return a == AnswerTypeShort || a == AnswerTypeExtended
}

// Difficulty — сложность задания.
type Difficulty string

const (
	DifficultyEasy   Difficulty = "easy"
	DifficultyMedium Difficulty = "medium"
	DifficultyHard   Difficulty = "hard"
)

// Valid сообщает, известна ли сложность.
func (d Difficulty) Valid() bool {
	switch d {
	case DifficultyEasy, DifficultyMedium, DifficultyHard:
		return true
	default:
		return false
	}
}

// VariantKind — назначение варианта.
type VariantKind string

const (
	VariantDiagnostic VariantKind = "diagnostic"
	VariantPractice   VariantKind = "practice"
	VariantMock       VariantKind = "mock"
)

// Valid сообщает, известно ли назначение варианта.
func (k VariantKind) Valid() bool {
	switch k {
	case VariantDiagnostic, VariantPractice, VariantMock:
		return true
	default:
		return false
	}
}

// Exam — учебный трек.
type Exam struct {
	ID        uuid.UUID
	Code      string
	Title     string
	Subject   string
	Status    Status
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ExamYear — учебный год трека.
type ExamYear struct {
	ID              uuid.UUID
	ExamID          uuid.UUID
	Year            int
	DurationMinutes int
	Status          Status
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// KimNumber — номер КИМ в структуре варианта.
type KimNumber struct {
	ID              uuid.UUID
	ExamYearID      uuid.UUID
	Number          int
	Part            int
	AnswerType      AnswerType
	MaxPrimaryScore int
	Title           string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Topic — тема трека.
type Topic struct {
	ID         uuid.UUID
	ExamID     uuid.UUID
	ParentID   *uuid.UUID
	Code       string
	Title      string
	OrderIndex int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// KimTopic — связь номера КИМ с темой.
type KimTopic struct {
	KimNumberID uuid.UUID
	TopicID     uuid.UUID
	Weight      float64
}

// Theory — версия теоретического материала темы.
type Theory struct {
	ID        uuid.UUID
	TopicID   uuid.UUID
	ContentMD string
	Version   int
	Status    Status
	UpdatedBy *uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Task — активная версия задания.
type Task struct {
	ID            uuid.UUID
	ExamYearID    uuid.UUID
	KimNumberID   uuid.UUID
	TopicID       *uuid.UUID
	Difficulty    Difficulty
	StatementMD   string
	AnswerSpec    json.RawMessage
	AnswerDisplay string
	Source        string
	SourceRef     string
	Status        Status
	IsPublic      bool
	ContentHash   []byte
	Version       int
	CreatedBy     *uuid.UUID
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Options       []TaskOption
}

// TaskOption — вариант ответа задания.
type TaskOption struct {
	ID        uuid.UUID
	TaskID    uuid.UUID
	Position  int
	ContentMD string
	IsCorrect bool
}

// TaskVersion — сохранённая версия задания.
type TaskVersion struct {
	ID            uuid.UUID
	TaskID        uuid.UUID
	Version       int
	StatementMD   string
	AnswerSpec    json.RawMessage
	AnswerDisplay string
	Difficulty    Difficulty
	KimNumberID   uuid.UUID
	TopicID       *uuid.UUID
	Source        string
	SourceRef     string
	CreatedBy     *uuid.UUID
	CreatedAt     time.Time
}

// TaskAttachment — изображение задания.
type TaskAttachment struct {
	ID         uuid.UUID
	TaskID     uuid.UUID
	Kind       string
	StorageKey string
	URL        string
	Alt        string
	CreatedAt  time.Time
}

// Variant — вариант заданий.
type Variant struct {
	ID              uuid.UUID
	ExamYearID      uuid.UUID
	Title           string
	Kind            VariantKind
	DurationMinutes int
	Status          Status
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Tasks           []VariantTask
}

// VariantTask — позиция задания в варианте.
type VariantTask struct {
	ID        uuid.UUID
	VariantID uuid.UUID
	Position  int
	TaskID    uuid.UUID
	Points    int
}

// RubricCriterion — критерий оценивания части 2.
type RubricCriterion struct {
	Code        string `json:"code"`
	Description string `json:"description"`
	MaxPoints   int    `json:"max_points"`
}

// RubricCriteria — набор критериев рубрики.
type RubricCriteria []RubricCriterion

// Rubric — версия рубрики части 2.
type Rubric struct {
	ID          uuid.UUID
	KimNumberID uuid.UUID
	ExamYearID  uuid.UUID
	Version     int
	Criteria    RubricCriteria
	Status      Status
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ScoreScale — запись шкалы перевода баллов.
type ScoreScale struct {
	ID           uuid.UUID
	ExamYearID   uuid.UUID
	PrimaryScore int
	TestScore    int
	Grade        *int
	MinThreshold *int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ExamInput содержит данные трека.
type ExamInput struct {
	Code    string
	Title   string
	Subject string
}

// ExamYearInput содержит данные учебного года.
type ExamYearInput struct {
	ExamID          uuid.UUID
	Year            int
	DurationMinutes int
}

// KimNumberInput содержит данные номера КИМ.
type KimNumberInput struct {
	ExamYearID      uuid.UUID
	Number          int
	Part            int
	AnswerType      AnswerType
	MaxPrimaryScore int
	Title           string
}

// TopicInput содержит данные темы.
type TopicInput struct {
	ExamID     uuid.UUID
	ParentID   *uuid.UUID
	Code       string
	Title      string
	OrderIndex int
}

// KimTopicInput содержит связь темы с номером КИМ.
type KimTopicInput struct {
	TopicID uuid.UUID
	Weight  float64
}

// TheoryInput содержит новую версию теории.
type TheoryInput struct {
	TopicID   uuid.UUID
	ContentMD string
	UpdatedBy *uuid.UUID
}

// TaskInput содержит данные задания.
type TaskInput struct {
	ExamYearID    uuid.UUID
	KimNumberID   uuid.UUID
	TopicID       *uuid.UUID
	Difficulty    Difficulty
	StatementMD   string
	AnswerSpec    json.RawMessage
	AnswerDisplay string
	Source        string
	SourceRef     string
	CreatedBy     *uuid.UUID
	Options       []TaskOptionInput
}

// TaskOptionInput содержит данные варианта ответа.
type TaskOptionInput struct {
	ID        uuid.UUID
	Position  int
	ContentMD string
	IsCorrect bool
}

// TaskAttachmentInput содержит данные изображения задания.
type TaskAttachmentInput struct {
	Kind       string
	StorageKey string
	URL        string
	Alt        string
}

// VariantTaskInput содержит позицию ручного варианта.
type VariantTaskInput struct {
	TaskID uuid.UUID
	Points int
}

// VariantInput содержит данные ручного варианта.
type VariantInput struct {
	ExamYearID      uuid.UUID
	Title           string
	Kind            VariantKind
	DurationMinutes int
	Tasks           []VariantTaskInput
}

// GenerationRule описывает правило отбора заданий для генерации варианта.
type GenerationRule struct {
	KimNumberID uuid.UUID
	Count       int
	Difficulty  *Difficulty
	TopicID     *uuid.UUID
}

// GenerateVariantInput содержит данные генерации варианта.
type GenerateVariantInput struct {
	ExamYearID      uuid.UUID
	Title           string
	Kind            VariantKind
	DurationMinutes int
	Rules           []GenerationRule
}

// RubricInput содержит новую версию рубрики.
type RubricInput struct {
	KimNumberID uuid.UUID
	ExamYearID  uuid.UUID
	Criteria    RubricCriteria
}

// ScoreScaleEntry содержит строку шкалы перевода.
type ScoreScaleEntry struct {
	PrimaryScore int
	TestScore    int
	Grade        *int
	MinThreshold *int
}

// ScoreScaleInput содержит шкалу перевода учебного года.
type ScoreScaleInput struct {
	ExamYearID uuid.UUID
	Entries    []ScoreScaleEntry
}

// ImportItem — элемент импорта задания.
type ImportItem struct {
	ExamYearID    uuid.UUID
	KimNumberID   uuid.UUID
	TopicID       *uuid.UUID
	Difficulty    Difficulty
	StatementMD   string
	AnswerSpec    json.RawMessage
	AnswerDisplay string
	Source        string
	SourceRef     string
	CreatedBy     *uuid.UUID
	Options       []TaskOptionInput
}

// ImportCreated описывает созданное задание.
type ImportCreated struct {
	Index  int
	TaskID uuid.UUID
}

// ImportSkipped описывает пропущенный дубликат.
type ImportSkipped struct {
	Index  int
	TaskID uuid.UUID
	Reason string
}

// ImportRejected описывает отклонённый элемент.
type ImportRejected struct {
	Index  int
	Reason string
}

// ImportReport — отчёт импорта.
type ImportReport struct {
	Created  []ImportCreated
	Skipped  []ImportSkipped
	Rejected []ImportRejected
}

// ExamFilter — фильтр списка треков.
type ExamFilter struct {
	Cursor uuid.UUID
	Limit  int
	Status *Status
	Query  string
}

// ExamYearFilter — фильтр списка учебных годов.
type ExamYearFilter struct {
	Cursor uuid.UUID
	Limit  int
	ExamID *uuid.UUID
	Status *Status
}

// KimNumberFilter — фильтр списка номеров КИМ.
type KimNumberFilter struct {
	Cursor     uuid.UUID
	Limit      int
	ExamYearID *uuid.UUID
}

// TopicFilter — фильтр списка тем.
type TopicFilter struct {
	Cursor   uuid.UUID
	Limit    int
	ExamID   *uuid.UUID
	ParentID *uuid.UUID
	Code     string
}

// TaskFilter — фильтр списка заданий.
type TaskFilter struct {
	Cursor               uuid.UUID
	Limit                int
	ExamID               *uuid.UUID
	ExamYearID           *uuid.UUID
	KimNumberID          *uuid.UUID
	TopicID              *uuid.UUID
	Status               *Status
	Query                string
	Kinds                []string
	ExcludePublishedMock bool
}

// VariantFilter — фильтр списка вариантов.
type VariantFilter struct {
	Cursor     uuid.UUID
	Limit      int
	ExamYearID *uuid.UUID
	Status     *Status
}

// RubricFilter — фильтр списка рубрик.
type RubricFilter struct {
	Cursor      uuid.UUID
	Limit       int
	KimNumberID *uuid.UUID
	ExamYearID  *uuid.UUID
	Status      *Status
}

// Ошибки модуля content.
var (
	ErrNotFound     = errors.New("объект контента не найден")
	ErrConflict     = errors.New("конфликт данных")
	ErrUnavailable  = errors.New("трек недоступен")
	ErrInsufficient = errors.New("недостаточно заданий для генерации варианта")
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

	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

func invalid(field, message string) error {
	return &ValidationError{Field: field, Message: message}
}
