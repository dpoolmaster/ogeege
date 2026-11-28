package content

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"oge-ege/internal/shared/db"
)

// Repository описывает хранилище модуля content.
type Repository interface {
	CreateExam(ctx context.Context, q db.Querier, exam *Exam) error
	UpdateExam(ctx context.Context, q db.Querier, exam *Exam) error
	GetExam(ctx context.Context, q db.Querier, id uuid.UUID) (*Exam, error)
	GetExamByCode(ctx context.Context, q db.Querier, code string) (*Exam, error)
	ListExams(ctx context.Context, q db.Querier, filter ExamFilter) ([]Exam, error)
	ListPublishedExams(ctx context.Context, q db.Querier) ([]Exam, error)
	SetExamStatus(ctx context.Context, q db.Querier, id uuid.UUID, status Status, now time.Time) error
	DeleteExam(ctx context.Context, q db.Querier, id uuid.UUID) error

	CreateExamYear(ctx context.Context, q db.Querier, year *ExamYear) error
	UpdateExamYear(ctx context.Context, q db.Querier, year *ExamYear) error
	GetExamYear(ctx context.Context, q db.Querier, id uuid.UUID) (*ExamYear, error)
	GetExamYearByYear(ctx context.Context, q db.Querier, examID uuid.UUID, year int) (*ExamYear, error)
	GetLatestPublishedExamYear(ctx context.Context, q db.Querier, examID uuid.UUID) (*ExamYear, error)
	ListExamYears(ctx context.Context, q db.Querier, filter ExamYearFilter) ([]ExamYear, error)
	SetExamYearStatus(ctx context.Context, q db.Querier, id uuid.UUID, status Status, now time.Time) error
	DeleteExamYear(ctx context.Context, q db.Querier, id uuid.UUID) error

	CreateKimNumber(ctx context.Context, q db.Querier, number *KimNumber) error
	UpdateKimNumber(ctx context.Context, q db.Querier, number *KimNumber) error
	GetKimNumber(ctx context.Context, q db.Querier, id uuid.UUID) (*KimNumber, error)
	GetKimNumberByNumber(ctx context.Context, q db.Querier, examYearID uuid.UUID, number int) (*KimNumber, error)
	ListKimNumbers(ctx context.Context, q db.Querier, filter KimNumberFilter) ([]KimNumber, error)
	DeleteKimNumber(ctx context.Context, q db.Querier, id uuid.UUID) error
	SumMaxPrimaryScore(ctx context.Context, q db.Querier, examYearID uuid.UUID) (int, error)

	CreateTopic(ctx context.Context, q db.Querier, topic *Topic) error
	UpdateTopic(ctx context.Context, q db.Querier, topic *Topic) error
	GetTopic(ctx context.Context, q db.Querier, id uuid.UUID) (*Topic, error)
	GetTopicByCode(ctx context.Context, q db.Querier, examID uuid.UUID, code string) (*Topic, error)
	ListTopics(ctx context.Context, q db.Querier, filter TopicFilter) ([]Topic, error)
	DeleteTopic(ctx context.Context, q db.Querier, id uuid.UUID) error
	ReplaceKimTopics(ctx context.Context, q db.Querier, kimNumberID uuid.UUID, topics []KimTopic) error
	ListKimTopics(ctx context.Context, q db.Querier, kimNumberID uuid.UUID) ([]KimTopic, error)
	ListKimTopicsByExamYear(ctx context.Context, q db.Querier, examYearID uuid.UUID) ([]KimTopic, error)
	GetKimTopic(ctx context.Context, q db.Querier, kimNumberID, topicID uuid.UUID) (*KimTopic, error)

	CreateTheory(ctx context.Context, q db.Querier, theory *Theory) error
	UpdateTheory(ctx context.Context, q db.Querier, theory *Theory) error
	GetTheory(ctx context.Context, q db.Querier, id uuid.UUID) (*Theory, error)
	GetActiveTheory(ctx context.Context, q db.Querier, topicID uuid.UUID) (*Theory, error)
	GetLatestTheoryVersion(ctx context.Context, q db.Querier, topicID uuid.UUID) (int, error)
	ListTheoryVersions(ctx context.Context, q db.Querier, topicID uuid.UUID) ([]Theory, error)
	SetTheoryStatus(ctx context.Context, q db.Querier, id uuid.UUID, status Status, now time.Time) error

	CreateTask(ctx context.Context, q db.Querier, task *Task) error
	UpdateTask(ctx context.Context, q db.Querier, task *Task) error
	GetTask(ctx context.Context, q db.Querier, id uuid.UUID) (*Task, error)
	GetTaskByHash(ctx context.Context, q db.Querier, hash []byte) (*Task, error)
	ListTasks(ctx context.Context, q db.Querier, filter TaskFilter) ([]Task, error)
	ListPublicTasks(ctx context.Context, q db.Querier, filter TaskFilter) ([]Task, error)
	SetTaskStatus(ctx context.Context, q db.Querier, id uuid.UUID, status Status, isPublic bool, now time.Time) error
	DeleteTask(ctx context.Context, q db.Querier, id uuid.UUID) error
	CreateTaskVersion(ctx context.Context, q db.Querier, version *TaskVersion) error
	ListTaskVersions(ctx context.Context, q db.Querier, taskID uuid.UUID) ([]TaskVersion, error)
	ListCandidateTasks(ctx context.Context, q db.Querier, examYearID, kimNumberID uuid.UUID, difficulty *Difficulty, topicID *uuid.UUID) ([]Task, error)
	CreateTaskAttachment(ctx context.Context, q db.Querier, attachment *TaskAttachment) error
	ListTaskAttachments(ctx context.Context, q db.Querier, taskID uuid.UUID) ([]TaskAttachment, error)

	CreateVariant(ctx context.Context, q db.Querier, variant *Variant) error
	GetVariant(ctx context.Context, q db.Querier, id uuid.UUID) (*Variant, error)
	ListVariants(ctx context.Context, q db.Querier, filter VariantFilter) ([]Variant, error)
	SetVariantStatus(ctx context.Context, q db.Querier, id uuid.UUID, status Status, now time.Time) error
	CountVariants(ctx context.Context, q db.Querier, examYearID uuid.UUID) (int, error)

	CreateRubric(ctx context.Context, q db.Querier, rubric *Rubric) error
	GetRubric(ctx context.Context, q db.Querier, id uuid.UUID) (*Rubric, error)
	GetActiveRubric(ctx context.Context, q db.Querier, kimNumberID, examYearID uuid.UUID) (*Rubric, error)
	GetLatestRubricVersion(ctx context.Context, q db.Querier, kimNumberID, examYearID uuid.UUID) (int, error)
	ListRubrics(ctx context.Context, q db.Querier, filter RubricFilter) ([]Rubric, error)
	SetRubricStatus(ctx context.Context, q db.Querier, id uuid.UUID, status Status, now time.Time) error

	ReplaceScoreScales(ctx context.Context, q db.Querier, examYearID uuid.UUID, entries []ScoreScale) error
	ListScoreScales(ctx context.Context, q db.Querier, examYearID uuid.UUID) ([]ScoreScale, error)
}

// PgRepository работает с PostgreSQL.
type PgRepository struct{}

// NewPgRepository создаёт репозиторий content.
func NewPgRepository() *PgRepository {
	return &PgRepository{}
}

const examColumns = `
    id,
    code,
    title,
    subject,
    status,
    created_at,
    updated_at
`

const createExamQuery = `
INSERT INTO content.exams (id, code, title, subject, status, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $6)
`

const updateExamQuery = `
UPDATE content.exams
SET code = $2, title = $3, subject = $4, updated_at = $5
WHERE id = $1
`

const getExamQuery = `
SELECT ` + examColumns + `
FROM content.exams
WHERE id = $1
`

const getExamByCodeQuery = `
SELECT ` + examColumns + `
FROM content.exams
WHERE code = $1
`

const listExamsQuery = `
SELECT ` + examColumns + `
FROM content.exams
WHERE ($1::uuid IS NULL OR id > $1)
  AND ($2::text IS NULL OR status = $2)
  AND ($3::text = '' OR title ILIKE '%' || $3 || '%' OR code ILIKE '%' || $3 || '%')
ORDER BY id
LIMIT $4
`

const listPublishedExamsQuery = `
SELECT ` + examColumns + `
FROM content.exams
WHERE status = 'published'
ORDER BY id
`

const setExamStatusQuery = `
UPDATE content.exams
SET status = $2, updated_at = $3
WHERE id = $1
`

const deleteExamQuery = `DELETE FROM content.exams WHERE id = $1`

// CreateExam сохраняет трек.
func (r *PgRepository) CreateExam(ctx context.Context, q db.Querier, exam *Exam) error {
	if _, err := q.Exec(ctx, createExamQuery,
		exam.ID, exam.Code, exam.Title, exam.Subject, string(exam.Status), exam.CreatedAt,
	); err != nil {
		return wrapWriteError("создать трек", err)
	}

	return nil
}

// UpdateExam обновляет трек.
func (r *PgRepository) UpdateExam(ctx context.Context, q db.Querier, exam *Exam) error {
	if _, err := q.Exec(ctx, updateExamQuery,
		exam.ID, exam.Code, exam.Title, exam.Subject, exam.UpdatedAt,
	); err != nil {
		return wrapWriteError("обновить трек", err)
	}

	return nil
}

// GetExam возвращает трек по идентификатору.
func (r *PgRepository) GetExam(ctx context.Context, q db.Querier, id uuid.UUID) (*Exam, error) {
	return scanExam(q.QueryRow(ctx, getExamQuery, id))
}

// GetExamByCode возвращает трек по коду.
func (r *PgRepository) GetExamByCode(ctx context.Context, q db.Querier, code string) (*Exam, error) {
	return scanExam(q.QueryRow(ctx, getExamByCodeQuery, code))
}

// ListExams возвращает страницу треков.
func (r *PgRepository) ListExams(ctx context.Context, q db.Querier, filter ExamFilter) ([]Exam, error) {
	rows, err := q.Query(ctx, listExamsQuery,
		nullableUUID(filter.Cursor), nullableStatus(filter.Status), filter.Query, normalizeLimit(filter.Limit),
	)
	if err != nil {
		return nil, fmt.Errorf("выбрать треки: %w", err)
	}
	defer rows.Close()

	var exams []Exam
	for rows.Next() {
		exam, err := scanExam(rows)
		if err != nil {
			return nil, err
		}
		exams = append(exams, *exam)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("прочитать треки: %w", err)
	}

	return exams, nil
}

// ListPublishedExams возвращает опубликованные треки.
func (r *PgRepository) ListPublishedExams(ctx context.Context, q db.Querier) ([]Exam, error) {
	rows, err := q.Query(ctx, listPublishedExamsQuery)
	if err != nil {
		return nil, fmt.Errorf("выбрать опубликованные треки: %w", err)
	}
	defer rows.Close()

	var exams []Exam
	for rows.Next() {
		exam, err := scanExam(rows)
		if err != nil {
			return nil, err
		}
		exams = append(exams, *exam)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("прочитать опубликованные треки: %w", err)
	}

	return exams, nil
}

// SetExamStatus меняет статус трека.
func (r *PgRepository) SetExamStatus(ctx context.Context, q db.Querier, id uuid.UUID, status Status, now time.Time) error {
	if _, err := q.Exec(ctx, setExamStatusQuery, id, string(status), now); err != nil {
		return fmt.Errorf("изменить статус трека: %w", err)
	}

	return nil
}

// DeleteExam удаляет трек.
func (r *PgRepository) DeleteExam(ctx context.Context, q db.Querier, id uuid.UUID) error {
	if _, err := q.Exec(ctx, deleteExamQuery, id); err != nil {
		return fmt.Errorf("удалить трек: %w", err)
	}

	return nil
}

const examYearColumns = `
    id,
    exam_id,
    year,
    duration_minutes,
    status,
    created_at,
    updated_at
`

const createExamYearQuery = `
INSERT INTO content.exam_years (id, exam_id, year, duration_minutes, status, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $6)
`

const updateExamYearQuery = `
UPDATE content.exam_years
SET year = $2, duration_minutes = $3, updated_at = $4
WHERE id = $1
`

const getExamYearQuery = `
SELECT ` + examYearColumns + `
FROM content.exam_years
WHERE id = $1
`

const getExamYearByYearQuery = `
SELECT ` + examYearColumns + `
FROM content.exam_years
WHERE exam_id = $1 AND year = $2
`

const getLatestPublishedExamYearQuery = `
SELECT ` + examYearColumns + `
FROM content.exam_years
WHERE exam_id = $1 AND status = 'published'
ORDER BY year DESC
LIMIT 1
`

const listExamYearsQuery = `
SELECT ` + examYearColumns + `
FROM content.exam_years
WHERE ($1::uuid IS NULL OR id > $1)
  AND ($2::uuid IS NULL OR exam_id = $2)
  AND ($3::text IS NULL OR status = $3)
ORDER BY id
LIMIT $4
`

const setExamYearStatusQuery = `
UPDATE content.exam_years
SET status = $2, updated_at = $3
WHERE id = $1
`

const deleteExamYearQuery = `DELETE FROM content.exam_years WHERE id = $1`

// CreateExamYear сохраняет учебный год.
func (r *PgRepository) CreateExamYear(ctx context.Context, q db.Querier, year *ExamYear) error {
	if _, err := q.Exec(ctx, createExamYearQuery,
		year.ID, year.ExamID, year.Year, year.DurationMinutes, string(year.Status), year.CreatedAt,
	); err != nil {
		return wrapWriteError("создать учебный год", err)
	}

	return nil
}

// UpdateExamYear обновляет учебный год.
func (r *PgRepository) UpdateExamYear(ctx context.Context, q db.Querier, year *ExamYear) error {
	if _, err := q.Exec(ctx, updateExamYearQuery,
		year.ID, year.Year, year.DurationMinutes, year.UpdatedAt,
	); err != nil {
		return wrapWriteError("обновить учебный год", err)
	}

	return nil
}

// GetExamYear возвращает учебный год по идентификатору.
func (r *PgRepository) GetExamYear(ctx context.Context, q db.Querier, id uuid.UUID) (*ExamYear, error) {
	return scanExamYear(q.QueryRow(ctx, getExamYearQuery, id))
}

// GetExamYearByYear возвращает учебный год трека.
func (r *PgRepository) GetExamYearByYear(ctx context.Context, q db.Querier, examID uuid.UUID, year int) (*ExamYear, error) {
	return scanExamYear(q.QueryRow(ctx, getExamYearByYearQuery, examID, year))
}

// GetLatestPublishedExamYear возвращает последний опубликованный учебный год трека.
func (r *PgRepository) GetLatestPublishedExamYear(ctx context.Context, q db.Querier, examID uuid.UUID) (*ExamYear, error) {
	return scanExamYear(q.QueryRow(ctx, getLatestPublishedExamYearQuery, examID))
}

// ListExamYears возвращает страницу учебных годов.
func (r *PgRepository) ListExamYears(ctx context.Context, q db.Querier, filter ExamYearFilter) ([]ExamYear, error) {
	rows, err := q.Query(ctx, listExamYearsQuery,
		nullableUUID(filter.Cursor), nullableUUIDPtr(filter.ExamID), nullableStatus(filter.Status), normalizeLimit(filter.Limit),
	)
	if err != nil {
		return nil, fmt.Errorf("выбрать учебные годы: %w", err)
	}
	defer rows.Close()

	var years []ExamYear
	for rows.Next() {
		year, err := scanExamYear(rows)
		if err != nil {
			return nil, err
		}
		years = append(years, *year)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("прочитать учебные годы: %w", err)
	}

	return years, nil
}

// SetExamYearStatus меняет статус учебного года.
func (r *PgRepository) SetExamYearStatus(ctx context.Context, q db.Querier, id uuid.UUID, status Status, now time.Time) error {
	if _, err := q.Exec(ctx, setExamYearStatusQuery, id, string(status), now); err != nil {
		return fmt.Errorf("изменить статус учебного года: %w", err)
	}

	return nil
}

// DeleteExamYear удаляет учебный год.
func (r *PgRepository) DeleteExamYear(ctx context.Context, q db.Querier, id uuid.UUID) error {
	if _, err := q.Exec(ctx, deleteExamYearQuery, id); err != nil {
		return fmt.Errorf("удалить учебный год: %w", err)
	}

	return nil
}

func scanExam(row rowScanner) (*Exam, error) {
	var exam Exam
	err := row.Scan(
		&exam.ID,
		&exam.Code,
		&exam.Title,
		&exam.Subject,
		&exam.Status,
		&exam.CreatedAt,
		&exam.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("прочитать трек: %w", err)
	}

	return &exam, nil
}

func scanExamYear(row rowScanner) (*ExamYear, error) {
	var year ExamYear
	err := row.Scan(
		&year.ID,
		&year.ExamID,
		&year.Year,
		&year.DurationMinutes,
		&year.Status,
		&year.CreatedAt,
		&year.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("прочитать учебный год: %w", err)
	}

	return &year, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func wrapWriteError(action string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return fmt.Errorf("%s: %w", action, ErrConflict)
	}

	return fmt.Errorf("%s: %w", action, err)
}

func normalizeLimit(limit int) int {
	const (
		defaultLimit = 50
		maxLimit     = 200
	)

	if limit <= 0 {
		return defaultLimit
	}
	if limit > maxLimit {
		return maxLimit
	}

	return limit
}

func nullableUUID(id uuid.UUID) any {
	if id == uuid.Nil {
		return nil
	}

	return id
}

func nullableUUIDPtr(id *uuid.UUID) any {
	if id == nil || *id == uuid.Nil {
		return nil
	}

	return *id
}

func nullableStrings(values []string) any {
	if len(values) == 0 {
		return nil
	}

	return values
}

func nullableStatus(status *Status) any {
	if status == nil {
		return nil
	}

	return string(*status)
}

func marshalJSON(value any) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("сериализовать JSON: %w", err)
	}

	return data, nil
}
