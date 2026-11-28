package learning

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

// Repository описывает доступ к таблицам схемы learning.
type Repository interface {
	CreatePlan(ctx context.Context, q db.Querier, plan *Plan) error
	GetActivePlan(ctx context.Context, q db.Querier, userID uuid.UUID) (*Plan, error)
	SupersedeActivePlans(ctx context.Context, q db.Querier, userID uuid.UUID, now time.Time) error
	UpdatePlanLastReplanned(ctx context.Context, q db.Querier, planID uuid.UUID, now time.Time) error

	CreateWeeks(ctx context.Context, q db.Querier, weeks []Week) error
	ListWeeks(ctx context.Context, q db.Querier, planID uuid.UUID) ([]Week, error)
	CreateItems(ctx context.Context, q db.Querier, items []PlanItem) error
	ListItemsByPlan(ctx context.Context, q db.Querier, planID uuid.UUID) ([]PlanItem, error)
	MarkItemDone(ctx context.Context, q db.Querier, itemID, userID uuid.UUID) error

	CreateSession(ctx context.Context, q db.Querier, session *Session) error
	CreateSessionItems(ctx context.Context, q db.Querier, items []SessionItem) error
	GetSession(ctx context.Context, q db.Querier, id uuid.UUID) (*Session, error)
	GetSessionByUser(ctx context.Context, q db.Querier, id, userID uuid.UUID) (*Session, error)
	GetInProgressSession(ctx context.Context, q db.Querier, userID, examID uuid.UUID, kind string) (*Session, error)
	GetPlanItemForUser(ctx context.Context, q db.Querier, itemID, userID uuid.UUID) (*PlanItem, uuid.UUID, error)
	ListDueTopicIDs(ctx context.Context, q db.Querier, userID uuid.UUID, now time.Time) ([]uuid.UUID, error)
	ListIncorrectTaskIDs(ctx context.Context, q db.Querier, userID uuid.UUID) ([]uuid.UUID, error)
	ListSessionItems(ctx context.Context, q db.Querier, sessionID uuid.UUID) ([]SessionItem, error)
	GetSessionItem(ctx context.Context, q db.Querier, sessionID, itemID uuid.UUID) (*SessionItem, error)
	ListAnswers(ctx context.Context, q db.Querier, sessionID uuid.UUID) ([]Answer, error)
	UpsertAnswer(ctx context.Context, q db.Querier, answer *Answer) error
	MarkSessionItemAnswered(ctx context.Context, q db.Querier, itemID uuid.UUID) error
	CompleteSession(ctx context.Context, q db.Querier, sessionID uuid.UUID, now time.Time, score float64) error

	UpsertTopicMastery(ctx context.Context, q db.Querier, mastery *TopicMastery) error
	ListTopicMastery(ctx context.Context, q db.Querier, userID uuid.UUID) ([]TopicMastery, error)
	UpsertKimMastery(ctx context.Context, q db.Querier, mastery *KimMastery) error
	ListKimMastery(ctx context.Context, q db.Querier, userID uuid.UUID) ([]KimMastery, error)

	UpsertSessionResult(ctx context.Context, q db.Querier, result *SessionResult) error
	GetSessionResult(ctx context.Context, q db.Querier, sessionID uuid.UUID) (*SessionResult, error)
	HasCompletedDiagnostic(ctx context.Context, q db.Querier, userID, examID uuid.UUID) (bool, error)
}

// PgRepository реализует Repository поверх PostgreSQL.
type PgRepository struct{}

// NewPgRepository создаёт репозиторий learning.
func NewPgRepository() *PgRepository {
	return &PgRepository{}
}

const planColumns = `
    id,
    user_id,
    exam_id,
    exam_date,
    target_score,
    weekly_minutes,
    status,
    version,
    last_replanned_at,
    created_at,
    updated_at
`

const createPlanQuery = `
INSERT INTO learning.study_plans (
    id, user_id, exam_id, exam_date, target_score, weekly_minutes, status, version,
    last_replanned_at, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)
`

const getActivePlanQuery = `
SELECT ` + planColumns + `
FROM learning.study_plans
WHERE user_id = $1 AND status = 'active'
`

const supersedeActivePlansQuery = `
UPDATE learning.study_plans
SET status = 'superseded', updated_at = $2
WHERE user_id = $1 AND status = 'active'
`

const updatePlanLastReplannedQuery = `
UPDATE learning.study_plans
SET last_replanned_at = $2, updated_at = $2
WHERE id = $1
`

const createWeekQuery = `
INSERT INTO learning.plan_weeks (id, plan_id, week_start, phase, goals_json)
VALUES ($1, $2, $3, $4, $5::jsonb)
`

const listWeeksQuery = `
SELECT id, plan_id, week_start, phase, goals_json
FROM learning.plan_weeks
WHERE plan_id = $1
ORDER BY week_start
`

const createItemQuery = `
INSERT INTO learning.plan_items (id, week_id, day_of_week, kind, topic_id, kim_number_id, task_count, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
`

const listItemsByPlanQuery = `
SELECT pi.id, pi.week_id, pi.day_of_week, pi.kind, pi.topic_id, pi.kim_number_id, pi.task_count, pi.status
FROM learning.plan_items pi
JOIN learning.plan_weeks pw ON pw.id = pi.week_id
WHERE pw.plan_id = $1
ORDER BY pw.week_start, pi.day_of_week
`

const markItemDoneQuery = `
UPDATE learning.plan_items pi
SET status = 'done'
FROM learning.plan_weeks pw
JOIN learning.study_plans sp ON sp.id = pw.plan_id
WHERE pi.week_id = pw.id AND pi.id = $1 AND sp.user_id = $2
`

const sessionColumns = `
    id,
    user_id,
    exam_id,
    kind,
    variant_id,
    status,
    started_at,
    deadline_at,
    finished_at,
    score_primary::float8,
    plan_item_id,
    created_at
`

const createSessionQuery = `
INSERT INTO learning.sessions (
    id, user_id, exam_id, kind, variant_id, status, started_at, deadline_at, finished_at,
    score_primary, plan_item_id, created_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
`

const getSessionQuery = `
SELECT ` + sessionColumns + `
FROM learning.sessions
WHERE id = $1
`

const getSessionByUserQuery = `
SELECT ` + sessionColumns + `
FROM learning.sessions
WHERE id = $1 AND user_id = $2
`

const getInProgressSessionQuery = `
SELECT ` + sessionColumns + `
FROM learning.sessions
WHERE user_id = $1 AND exam_id = $2 AND kind = $3 AND status = 'in_progress'
ORDER BY created_at DESC
LIMIT 1
`

const getPlanItemForUserQuery = `
SELECT pi.id, pi.week_id, pi.day_of_week, pi.kind, pi.topic_id, pi.kim_number_id, pi.task_count, pi.status,
       sp.exam_id
FROM learning.plan_items pi
JOIN learning.plan_weeks pw ON pw.id = pi.week_id
JOIN learning.study_plans sp ON sp.id = pw.plan_id
WHERE pi.id = $1 AND sp.user_id = $2 AND sp.status = 'active'
`

const listDueTopicIDsQuery = `
SELECT topic_id
FROM learning.topic_mastery
WHERE user_id = $1 AND next_review_at IS NOT NULL AND next_review_at <= $2
ORDER BY next_review_at
`

const listIncorrectTaskIDsQuery = `
SELECT DISTINCT si.task_id
FROM learning.answers a
JOIN learning.session_items si ON si.id = a.session_item_id
JOIN learning.sessions s ON s.id = si.session_id
WHERE s.user_id = $1 AND a.is_correct = false
ORDER BY si.task_id
`

const createSessionItemQuery = `
INSERT INTO learning.session_items (id, session_id, position, task_id, task_snapshot, max_points, status)
VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7)
`

const listSessionItemsQuery = `
SELECT id, session_id, position, task_id, task_snapshot, max_points::float8, status
FROM learning.session_items
WHERE session_id = $1
ORDER BY position
`

const getSessionItemQuery = `
SELECT id, session_id, position, task_id, task_snapshot, max_points::float8, status
FROM learning.session_items
WHERE session_id = $1 AND id = $2
`

const listAnswersQuery = `
SELECT a.id, a.session_item_id, a.raw_answer, a.normalized_answer, a.is_correct,
       a.awarded_points::float8, a.check_kind, a.checked_at, a.created_at
FROM learning.answers a
JOIN learning.session_items si ON si.id = a.session_item_id
WHERE si.session_id = $1
ORDER BY si.position
`

const upsertAnswerQuery = `
INSERT INTO learning.answers (
    id, session_item_id, raw_answer, normalized_answer, is_correct, awarded_points,
    check_kind, checked_at, created_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (session_item_id) DO UPDATE SET
    raw_answer = EXCLUDED.raw_answer,
    normalized_answer = EXCLUDED.normalized_answer,
    is_correct = EXCLUDED.is_correct,
    awarded_points = EXCLUDED.awarded_points,
    check_kind = EXCLUDED.check_kind,
    checked_at = EXCLUDED.checked_at
`

const markSessionItemAnsweredQuery = `
UPDATE learning.session_items
SET status = 'answered'
WHERE id = $1
`

const completeSessionQuery = `
UPDATE learning.sessions
SET status = 'completed', finished_at = $2, score_primary = $3
WHERE id = $1
`

const upsertTopicMasteryQuery = `
INSERT INTO learning.topic_mastery (
    user_id, topic_id, mastery, attempts, correct, interval_days, ease,
    last_practiced_at, next_review_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (user_id, topic_id) DO UPDATE SET
    mastery = EXCLUDED.mastery,
    attempts = EXCLUDED.attempts,
    correct = EXCLUDED.correct,
    interval_days = EXCLUDED.interval_days,
    ease = EXCLUDED.ease,
    last_practiced_at = EXCLUDED.last_practiced_at,
    next_review_at = EXCLUDED.next_review_at
`

const listTopicMasteryQuery = `
SELECT user_id, topic_id, mastery::float8, attempts, correct, interval_days, ease::float8,
       last_practiced_at, next_review_at
FROM learning.topic_mastery
WHERE user_id = $1
ORDER BY topic_id
`

const upsertKimMasteryQuery = `
INSERT INTO learning.kim_mastery (user_id, kim_number_id, mastery, attempts, correct, last_practiced_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (user_id, kim_number_id) DO UPDATE SET
    mastery = EXCLUDED.mastery,
    attempts = EXCLUDED.attempts,
    correct = EXCLUDED.correct,
    last_practiced_at = EXCLUDED.last_practiced_at
`

const listKimMasteryQuery = `
SELECT user_id, kim_number_id, mastery::float8, attempts, correct, last_practiced_at
FROM learning.kim_mastery
WHERE user_id = $1
ORDER BY kim_number_id
`

const upsertSessionResultQuery = `
INSERT INTO learning.session_results (
    session_id, primary_score, max_score, test_score, grade, breakdown_json
) VALUES ($1, $2, $3, $4, $5, $6::jsonb)
ON CONFLICT (session_id) DO UPDATE SET
    primary_score = EXCLUDED.primary_score,
    max_score = EXCLUDED.max_score,
    test_score = EXCLUDED.test_score,
    grade = EXCLUDED.grade,
    breakdown_json = EXCLUDED.breakdown_json
`

const getSessionResultQuery = `
SELECT session_id, primary_score::float8, max_score::float8, test_score, grade, breakdown_json
FROM learning.session_results
WHERE session_id = $1
`

const hasCompletedDiagnosticQuery = `
SELECT EXISTS (
    SELECT 1 FROM learning.sessions
    WHERE user_id = $1 AND exam_id = $2 AND kind = 'diagnostic' AND status = 'completed'
)
`

// CreatePlan сохраняет версию плана.
func (r *PgRepository) CreatePlan(ctx context.Context, q db.Querier, plan *Plan) error {
	if _, err := q.Exec(ctx, createPlanQuery,
		plan.ID, plan.UserID, plan.ExamID, plan.ExamDate, plan.TargetScore, plan.WeeklyMinutes,
		string(plan.Status), plan.Version, plan.LastReplannedAt, plan.CreatedAt,
	); err != nil {
		return wrapWriteError("создать учебный план", err)
	}

	return nil
}

// GetActivePlan возвращает активный план ученика.
func (r *PgRepository) GetActivePlan(ctx context.Context, q db.Querier, userID uuid.UUID) (*Plan, error) {
	return scanPlan(q.QueryRow(ctx, getActivePlanQuery, userID))
}

// SupersedeActivePlans переводит активные планы в историю.
func (r *PgRepository) SupersedeActivePlans(ctx context.Context, q db.Querier, userID uuid.UUID, now time.Time) error {
	if _, err := q.Exec(ctx, supersedeActivePlansQuery, userID, now); err != nil {
		return fmt.Errorf("обновить активный план: %w", err)
	}

	return nil
}

// UpdatePlanLastReplanned обновляет время пересчёта.
func (r *PgRepository) UpdatePlanLastReplanned(ctx context.Context, q db.Querier, planID uuid.UUID, now time.Time) error {
	if _, err := q.Exec(ctx, updatePlanLastReplannedQuery, planID, now); err != nil {
		return fmt.Errorf("обновить время пересчёта: %w", err)
	}

	return nil
}

// CreateWeeks сохраняет недели плана.
func (r *PgRepository) CreateWeeks(ctx context.Context, q db.Querier, weeks []Week) error {
	for _, week := range weeks {
		goals := week.Goals
		if len(goals) == 0 {
			goals = json.RawMessage(`{}`)
		}
		if _, err := q.Exec(ctx, createWeekQuery, week.ID, week.PlanID, week.WeekStart, string(week.Phase), []byte(goals)); err != nil {
			return wrapWriteError("создать неделю плана", err)
		}
	}

	return nil
}

// ListWeeks возвращает недели плана.
func (r *PgRepository) ListWeeks(ctx context.Context, q db.Querier, planID uuid.UUID) ([]Week, error) {
	rows, err := q.Query(ctx, listWeeksQuery, planID)
	if err != nil {
		return nil, fmt.Errorf("выбрать недели плана: %w", err)
	}
	defer rows.Close()

	var weeks []Week
	for rows.Next() {
		var (
			week  Week
			goals []byte
		)
		if err := rows.Scan(&week.ID, &week.PlanID, &week.WeekStart, &week.Phase, &goals); err != nil {
			return nil, fmt.Errorf("прочитать неделю плана: %w", err)
		}
		week.Goals = goals
		weeks = append(weeks, week)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("прочитать недели плана: %w", err)
	}

	return weeks, nil
}

// CreateItems сохраняет пункты плана.
func (r *PgRepository) CreateItems(ctx context.Context, q db.Querier, items []PlanItem) error {
	for _, item := range items {
		if _, err := q.Exec(ctx, createItemQuery,
			item.ID, item.WeekID, item.DayOfWeek, string(item.Kind), item.TopicID,
			item.KimNumberID, item.TaskCount, string(item.Status),
		); err != nil {
			return wrapWriteError("создать пункт плана", err)
		}
	}

	return nil
}

// ListItemsByPlan возвращает пункты плана.
func (r *PgRepository) ListItemsByPlan(ctx context.Context, q db.Querier, planID uuid.UUID) ([]PlanItem, error) {
	rows, err := q.Query(ctx, listItemsByPlanQuery, planID)
	if err != nil {
		return nil, fmt.Errorf("выбрать пункты плана: %w", err)
	}
	defer rows.Close()

	var items []PlanItem
	for rows.Next() {
		var item PlanItem
		if err := rows.Scan(
			&item.ID, &item.WeekID, &item.DayOfWeek, &item.Kind, &item.TopicID,
			&item.KimNumberID, &item.TaskCount, &item.Status,
		); err != nil {
			return nil, fmt.Errorf("прочитать пункт плана: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("прочитать пункты плана: %w", err)
	}

	return items, nil
}

// MarkItemDone отмечает пункт выполненным с проверкой владельца.
func (r *PgRepository) MarkItemDone(ctx context.Context, q db.Querier, itemID, userID uuid.UUID) error {
	tag, err := q.Exec(ctx, markItemDoneQuery, itemID, userID)
	if err != nil {
		return fmt.Errorf("отметить пункт выполненным: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// CreateSession сохраняет сессию.
func (r *PgRepository) CreateSession(ctx context.Context, q db.Querier, session *Session) error {
	if _, err := q.Exec(ctx, createSessionQuery,
		session.ID, session.UserID, session.ExamID, session.Kind, session.VariantID,
		string(session.Status), session.StartedAt, session.DeadlineAt, session.FinishedAt,
		session.ScorePrimary, session.PlanItemID, session.CreatedAt,
	); err != nil {
		return wrapWriteError("создать сессию", err)
	}

	return nil
}

// CreateSessionItems сохраняет задания сессии.
func (r *PgRepository) CreateSessionItems(ctx context.Context, q db.Querier, items []SessionItem) error {
	for _, item := range items {
		if _, err := q.Exec(ctx, createSessionItemQuery,
			item.ID, item.SessionID, item.Position, item.TaskID, []byte(item.TaskSnapshot),
			item.MaxPoints, string(item.Status),
		); err != nil {
			return wrapWriteError("создать задание сессии", err)
		}
	}

	return nil
}

// GetSession возвращает сессию.
func (r *PgRepository) GetSession(ctx context.Context, q db.Querier, id uuid.UUID) (*Session, error) {
	return scanSession(q.QueryRow(ctx, getSessionQuery, id))
}

// GetSessionByUser возвращает сессию владельца.
func (r *PgRepository) GetSessionByUser(ctx context.Context, q db.Querier, id, userID uuid.UUID) (*Session, error) {
	return scanSession(q.QueryRow(ctx, getSessionByUserQuery, id, userID))
}

// GetInProgressSession возвращает незавершённую сессию ученика.
func (r *PgRepository) GetInProgressSession(ctx context.Context, q db.Querier, userID, examID uuid.UUID, kind string) (*Session, error) {
	return scanSession(q.QueryRow(ctx, getInProgressSessionQuery, userID, examID, kind))
}

// GetPlanItemForUser возвращает пункт плана владельца и идентификатор трека.
func (r *PgRepository) GetPlanItemForUser(ctx context.Context, q db.Querier, itemID, userID uuid.UUID) (*PlanItem, uuid.UUID, error) {
	var (
		item   PlanItem
		examID uuid.UUID
	)
	err := q.QueryRow(ctx, getPlanItemForUserQuery, itemID, userID).Scan(
		&item.ID, &item.WeekID, &item.DayOfWeek, &item.Kind, &item.TopicID,
		&item.KimNumberID, &item.TaskCount, &item.Status, &examID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, uuid.Nil, ErrNotFound
	}
	if err != nil {
		return nil, uuid.Nil, fmt.Errorf("прочитать пункт плана: %w", err)
	}

	return &item, examID, nil
}

// ListDueTopicIDs возвращает темы с наступившим сроком повторения.
func (r *PgRepository) ListDueTopicIDs(ctx context.Context, q db.Querier, userID uuid.UUID, now time.Time) ([]uuid.UUID, error) {
	rows, err := q.Query(ctx, listDueTopicIDsQuery, userID, now)
	if err != nil {
		return nil, fmt.Errorf("выбрать темы для повторения: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("прочитать тему для повторения: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("прочитать темы для повторения: %w", err)
	}

	return ids, nil
}

// ListIncorrectTaskIDs возвращает ранее неверно решённые задания ученика.
func (r *PgRepository) ListIncorrectTaskIDs(ctx context.Context, q db.Querier, userID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := q.Query(ctx, listIncorrectTaskIDsQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("выбрать неверно решённые задания: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("прочитать неверно решённое задание: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("прочитать неверно решённые задания: %w", err)
	}

	return ids, nil
}

// ListSessionItems возвращает задания сессии.
func (r *PgRepository) ListSessionItems(ctx context.Context, q db.Querier, sessionID uuid.UUID) ([]SessionItem, error) {
	rows, err := q.Query(ctx, listSessionItemsQuery, sessionID)
	if err != nil {
		return nil, fmt.Errorf("выбрать задания сессии: %w", err)
	}
	defer rows.Close()

	var items []SessionItem
	for rows.Next() {
		var (
			item     SessionItem
			snapshot []byte
		)
		if err := rows.Scan(&item.ID, &item.SessionID, &item.Position, &item.TaskID, &snapshot, &item.MaxPoints, &item.Status); err != nil {
			return nil, fmt.Errorf("прочитать задание сессии: %w", err)
		}
		item.TaskSnapshot = snapshot
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("прочитать задания сессии: %w", err)
	}

	return items, nil
}

// GetSessionItem возвращает задание сессии.
func (r *PgRepository) GetSessionItem(ctx context.Context, q db.Querier, sessionID, itemID uuid.UUID) (*SessionItem, error) {
	var (
		item     SessionItem
		snapshot []byte
	)
	err := q.QueryRow(ctx, getSessionItemQuery, sessionID, itemID).Scan(
		&item.ID, &item.SessionID, &item.Position, &item.TaskID, &snapshot, &item.MaxPoints, &item.Status,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("прочитать задание сессии: %w", err)
	}
	item.TaskSnapshot = snapshot

	return &item, nil
}

// ListAnswers возвращает ответы сессии.
func (r *PgRepository) ListAnswers(ctx context.Context, q db.Querier, sessionID uuid.UUID) ([]Answer, error) {
	rows, err := q.Query(ctx, listAnswersQuery, sessionID)
	if err != nil {
		return nil, fmt.Errorf("выбрать ответы сессии: %w", err)
	}
	defer rows.Close()

	var answers []Answer
	for rows.Next() {
		var answer Answer
		if err := rows.Scan(
			&answer.ID, &answer.SessionItemID, &answer.RawAnswer, &answer.NormalizedAnswer,
			&answer.IsCorrect, &answer.AwardedPoints, &answer.CheckKind, &answer.CheckedAt, &answer.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("прочитать ответ сессии: %w", err)
		}
		answers = append(answers, answer)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("прочитать ответы сессии: %w", err)
	}

	return answers, nil
}

// UpsertAnswer сохраняет или обновляет ответ.
func (r *PgRepository) UpsertAnswer(ctx context.Context, q db.Querier, answer *Answer) error {
	if _, err := q.Exec(ctx, upsertAnswerQuery,
		answer.ID, answer.SessionItemID, answer.RawAnswer, answer.NormalizedAnswer,
		answer.IsCorrect, answer.AwardedPoints, answer.CheckKind, answer.CheckedAt, answer.CreatedAt,
	); err != nil {
		return wrapWriteError("сохранить ответ", err)
	}

	return nil
}

// MarkSessionItemAnswered отмечает задание отвеченным.
func (r *PgRepository) MarkSessionItemAnswered(ctx context.Context, q db.Querier, itemID uuid.UUID) error {
	if _, err := q.Exec(ctx, markSessionItemAnsweredQuery, itemID); err != nil {
		return fmt.Errorf("обновить статус задания сессии: %w", err)
	}

	return nil
}

// CompleteSession завершает сессию.
func (r *PgRepository) CompleteSession(ctx context.Context, q db.Querier, sessionID uuid.UUID, now time.Time, score float64) error {
	if _, err := q.Exec(ctx, completeSessionQuery, sessionID, now, score); err != nil {
		return fmt.Errorf("завершить сессию: %w", err)
	}

	return nil
}

// UpsertTopicMastery сохраняет освоение темы.
func (r *PgRepository) UpsertTopicMastery(ctx context.Context, q db.Querier, mastery *TopicMastery) error {
	if _, err := q.Exec(ctx, upsertTopicMasteryQuery,
		mastery.UserID, mastery.TopicID, mastery.Mastery, mastery.Attempts, mastery.Correct,
		mastery.IntervalDays, mastery.Ease, mastery.LastPracticedAt, mastery.NextReviewAt,
	); err != nil {
		return wrapWriteError("сохранить освоение темы", err)
	}

	return nil
}

// ListTopicMastery возвращает освоение тем ученика.
func (r *PgRepository) ListTopicMastery(ctx context.Context, q db.Querier, userID uuid.UUID) ([]TopicMastery, error) {
	rows, err := q.Query(ctx, listTopicMasteryQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("выбрать освоение тем: %w", err)
	}
	defer rows.Close()

	var items []TopicMastery
	for rows.Next() {
		var item TopicMastery
		if err := rows.Scan(
			&item.UserID, &item.TopicID, &item.Mastery, &item.Attempts, &item.Correct,
			&item.IntervalDays, &item.Ease, &item.LastPracticedAt, &item.NextReviewAt,
		); err != nil {
			return nil, fmt.Errorf("прочитать освоение темы: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("прочитать освоение тем: %w", err)
	}

	return items, nil
}

// UpsertKimMastery сохраняет освоение номера КИМ.
func (r *PgRepository) UpsertKimMastery(ctx context.Context, q db.Querier, mastery *KimMastery) error {
	if _, err := q.Exec(ctx, upsertKimMasteryQuery,
		mastery.UserID, mastery.KimNumberID, mastery.Mastery, mastery.Attempts, mastery.Correct,
		mastery.LastPracticedAt,
	); err != nil {
		return wrapWriteError("сохранить освоение номера КИМ", err)
	}

	return nil
}

// ListKimMastery возвращает освоение номеров КИМ ученика.
func (r *PgRepository) ListKimMastery(ctx context.Context, q db.Querier, userID uuid.UUID) ([]KimMastery, error) {
	rows, err := q.Query(ctx, listKimMasteryQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("выбрать освоение номеров КИМ: %w", err)
	}
	defer rows.Close()

	var items []KimMastery
	for rows.Next() {
		var item KimMastery
		if err := rows.Scan(
			&item.UserID, &item.KimNumberID, &item.Mastery, &item.Attempts, &item.Correct,
			&item.LastPracticedAt,
		); err != nil {
			return nil, fmt.Errorf("прочитать освоение номера КИМ: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("прочитать освоение номеров КИМ: %w", err)
	}

	return items, nil
}

// UpsertSessionResult сохраняет результат сессии.
func (r *PgRepository) UpsertSessionResult(ctx context.Context, q db.Querier, result *SessionResult) error {
	breakdown := result.Breakdown
	if len(breakdown) == 0 {
		breakdown = json.RawMessage(`{}`)
	}
	if _, err := q.Exec(ctx, upsertSessionResultQuery,
		result.SessionID, result.PrimaryScore, result.MaxScore, result.TestScore, result.Grade,
		[]byte(breakdown),
	); err != nil {
		return wrapWriteError("сохранить результат сессии", err)
	}

	return nil
}

// GetSessionResult возвращает результат сессии.
func (r *PgRepository) GetSessionResult(ctx context.Context, q db.Querier, sessionID uuid.UUID) (*SessionResult, error) {
	var (
		result    SessionResult
		breakdown []byte
	)
	err := q.QueryRow(ctx, getSessionResultQuery, sessionID).Scan(
		&result.SessionID, &result.PrimaryScore, &result.MaxScore, &result.TestScore,
		&result.Grade, &breakdown,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("прочитать результат сессии: %w", err)
	}
	result.Breakdown = breakdown

	return &result, nil
}

// HasCompletedDiagnostic сообщает, есть ли у ученика завершённая диагностика.
func (r *PgRepository) HasCompletedDiagnostic(ctx context.Context, q db.Querier, userID, examID uuid.UUID) (bool, error) {
	var exists bool
	if err := q.QueryRow(ctx, hasCompletedDiagnosticQuery, userID, examID).Scan(&exists); err != nil {
		return false, fmt.Errorf("проверить завершённую диагностику: %w", err)
	}

	return exists, nil
}

func scanPlan(row rowScanner) (*Plan, error) {
	var plan Plan
	err := row.Scan(
		&plan.ID, &plan.UserID, &plan.ExamID, &plan.ExamDate, &plan.TargetScore,
		&plan.WeeklyMinutes, &plan.Status, &plan.Version, &plan.LastReplannedAt,
		&plan.CreatedAt, &plan.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("прочитать учебный план: %w", err)
	}

	return &plan, nil
}

func scanSession(row rowScanner) (*Session, error) {
	var session Session
	err := row.Scan(
		&session.ID, &session.UserID, &session.ExamID, &session.Kind, &session.VariantID,
		&session.Status, &session.StartedAt, &session.DeadlineAt, &session.FinishedAt,
		&session.ScorePrimary, &session.PlanItemID, &session.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("прочитать сессию: %w", err)
	}

	return &session, nil
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
