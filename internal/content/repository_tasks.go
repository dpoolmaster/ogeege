package content

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"oge-ege/internal/shared/db"
)

const taskColumns = `
    id,
    exam_year_id,
    kim_number_id,
    topic_id,
    difficulty,
    statement_md,
    answer_spec,
    COALESCE(answer_display, ''),
    source,
    COALESCE(source_ref, ''),
    status,
    is_public,
    content_hash,
    version,
    created_by,
    created_at,
    updated_at
`

const createTaskQuery = `
INSERT INTO content.tasks (
    id, exam_year_id, kim_number_id, topic_id, difficulty, statement_md, answer_spec,
    answer_display, source, source_ref, status, is_public, content_hash, version, created_by,
    created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7::jsonb,
    NULLIF($8, ''), $9, NULLIF($10, ''), $11, $12, $13, $14, $15,
    $16, $16
)
`

const updateTaskQuery = `
UPDATE content.tasks
SET kim_number_id = $2,
    topic_id = $3,
    difficulty = $4,
    statement_md = $5,
    answer_spec = $6::jsonb,
    answer_display = NULLIF($7, ''),
    source = $8,
    source_ref = NULLIF($9, ''),
    status = $10,
    is_public = $11,
    content_hash = $12,
    version = $13,
    updated_at = $14
WHERE id = $1
`

const getTaskQuery = `
SELECT ` + taskColumns + `
FROM content.tasks
WHERE id = $1
`

const getTaskByHashQuery = `
SELECT ` + taskColumns + `
FROM content.tasks
WHERE content_hash = $1
`

const listTasksQuery = `
SELECT ` + taskColumns + `
FROM content.tasks
WHERE ($1::uuid IS NULL OR id > $1)
  AND ($2::uuid IS NULL OR exam_year_id = $2)
  AND ($3::uuid IS NULL OR kim_number_id = $3)
  AND ($4::uuid IS NULL OR topic_id = $4)
  AND ($5::text IS NULL OR status = $5)
  AND ($6::text = '' OR statement_md ILIKE '%' || $6 || '%')
ORDER BY id
LIMIT $7
`

const listPublicTasksQuery = `
SELECT ` + taskColumns + `
FROM content.tasks
WHERE status = 'published' AND is_public = true
  AND ($1::uuid IS NULL OR id > $1)
  AND ($2::uuid IS NULL OR exam_year_id = $2)
  AND ($3::uuid IS NULL OR kim_number_id = $3)
  AND ($4::uuid IS NULL OR topic_id = $4)
  AND ($5::text = '' OR statement_md ILIKE '%' || $5 || '%')
  AND ($6::uuid IS NULL OR exam_year_id IN (
      SELECT id FROM content.exam_years WHERE exam_id = $6
  ))
  AND ($7::text[] IS NULL OR answer_spec->>'kind' = ANY($7::text[]))
  AND (NOT $8::boolean OR NOT EXISTS (
      SELECT 1
      FROM content.variant_tasks vt
      JOIN content.variants v ON v.id = vt.variant_id
      WHERE vt.task_id = content.tasks.id
        AND v.kind = 'mock'
        AND v.status = 'published'
  ))
ORDER BY id
LIMIT $9
`

const setTaskStatusQuery = `
UPDATE content.tasks
SET status = $2, is_public = $3, updated_at = $4
WHERE id = $1
`

const deleteTaskQuery = `DELETE FROM content.tasks WHERE id = $1`

const createTaskVersionQuery = `
INSERT INTO content.task_versions (
    id, task_id, version, statement_md, answer_spec, answer_display, difficulty,
    kim_number_id, topic_id, source, source_ref, created_by, created_at
) VALUES ($1, $2, $3, $4, $5::jsonb, NULLIF($6, ''), $7, $8, $9, $10, NULLIF($11, ''), $12, $13)
`

const taskVersionColumns = `
    id,
    task_id,
    version,
    statement_md,
    answer_spec,
    COALESCE(answer_display, ''),
    difficulty,
    kim_number_id,
    topic_id,
    source,
    COALESCE(source_ref, ''),
    created_by,
    created_at
`

const listTaskVersionsQuery = `
SELECT ` + taskVersionColumns + `
FROM content.task_versions
WHERE task_id = $1
ORDER BY version DESC
`

const listCandidateTasksQuery = `
SELECT ` + taskColumns + `
FROM content.tasks
WHERE status = 'published' AND is_public = true
  AND exam_year_id = $1
  AND kim_number_id = $2
  AND ($3::text IS NULL OR difficulty = $3)
  AND ($4::uuid IS NULL OR topic_id = $4)
ORDER BY id
`

const createTaskAttachmentQuery = `
INSERT INTO content.task_attachments (id, task_id, kind, storage_key, url, alt, created_at)
VALUES ($1, $2, $3, $4, NULLIF($5, ''), NULLIF($6, ''), $7)
`

const listTaskAttachmentsQuery = `
SELECT id, task_id, kind, storage_key, COALESCE(url, ''), COALESCE(alt, ''), created_at
FROM content.task_attachments
WHERE task_id = $1
ORDER BY created_at
`

const listTaskOptionsQuery = `
SELECT id, task_id, position, content_md, is_correct
FROM content.task_options
WHERE task_id = $1
ORDER BY position
`

const deleteTaskOptionsQuery = `DELETE FROM content.task_options WHERE task_id = $1`

const createTaskOptionQuery = `
INSERT INTO content.task_options (id, task_id, position, content_md, is_correct)
VALUES ($1, $2, $3, $4, $5)
`

// CreateTask сохраняет активную версию задания.
func (r *PgRepository) CreateTask(ctx context.Context, q db.Querier, task *Task) error {
	if _, err := q.Exec(ctx, createTaskQuery,
		task.ID, task.ExamYearID, task.KimNumberID, task.TopicID, string(task.Difficulty),
		task.StatementMD, []byte(task.AnswerSpec), task.AnswerDisplay, task.Source, task.SourceRef,
		string(task.Status), task.IsPublic, task.ContentHash, task.Version, task.CreatedBy, task.CreatedAt,
	); err != nil {
		return wrapWriteError("создать задание", err)
	}

	return nil
}

// UpdateTask обновляет активную версию задания.
func (r *PgRepository) UpdateTask(ctx context.Context, q db.Querier, task *Task) error {
	if _, err := q.Exec(ctx, updateTaskQuery,
		task.ID, task.KimNumberID, task.TopicID, string(task.Difficulty), task.StatementMD,
		[]byte(task.AnswerSpec), task.AnswerDisplay, task.Source, task.SourceRef, string(task.Status),
		task.IsPublic, task.ContentHash, task.Version, task.UpdatedAt,
	); err != nil {
		return wrapWriteError("обновить задание", err)
	}

	return nil
}

// GetTask возвращает задание по идентификатору.
func (r *PgRepository) GetTask(ctx context.Context, q db.Querier, id uuid.UUID) (*Task, error) {
	return scanTask(q.QueryRow(ctx, getTaskQuery, id))
}

// GetTaskByHash возвращает задание по хешу условия.
func (r *PgRepository) GetTaskByHash(ctx context.Context, q db.Querier, hash []byte) (*Task, error) {
	return scanTask(q.QueryRow(ctx, getTaskByHashQuery, hash))
}

// ListTasks возвращает страницу заданий.
func (r *PgRepository) ListTasks(ctx context.Context, q db.Querier, filter TaskFilter) ([]Task, error) {
	rows, err := q.Query(ctx, listTasksQuery,
		nullableUUID(filter.Cursor), nullableUUIDPtr(filter.ExamYearID), nullableUUIDPtr(filter.KimNumberID),
		nullableUUIDPtr(filter.TopicID), nullableStatus(filter.Status), filter.Query, normalizeLimit(filter.Limit),
	)
	if err != nil {
		return nil, fmt.Errorf("выбрать задания: %w", err)
	}
	defer rows.Close()

	return collectTasks(rows)
}

// ListPublicTasks возвращает опубликованные публичные задания.
func (r *PgRepository) ListPublicTasks(ctx context.Context, q db.Querier, filter TaskFilter) ([]Task, error) {
	rows, err := q.Query(ctx, listPublicTasksQuery,
		nullableUUID(filter.Cursor), nullableUUIDPtr(filter.ExamYearID), nullableUUIDPtr(filter.KimNumberID),
		nullableUUIDPtr(filter.TopicID), filter.Query, nullableUUIDPtr(filter.ExamID),
		nullableStrings(filter.Kinds), filter.ExcludePublishedMock, normalizeLimit(filter.Limit),
	)
	if err != nil {
		return nil, fmt.Errorf("выбрать публичные задания: %w", err)
	}
	defer rows.Close()

	return collectTasks(rows)
}

// SetTaskStatus меняет статус задания.
func (r *PgRepository) SetTaskStatus(ctx context.Context, q db.Querier, id uuid.UUID, status Status, isPublic bool, now time.Time) error {
	if _, err := q.Exec(ctx, setTaskStatusQuery, id, string(status), isPublic, now); err != nil {
		return fmt.Errorf("изменить статус задания: %w", err)
	}

	return nil
}

// DeleteTask удаляет задание.
func (r *PgRepository) DeleteTask(ctx context.Context, q db.Querier, id uuid.UUID) error {
	if _, err := q.Exec(ctx, deleteTaskQuery, id); err != nil {
		return fmt.Errorf("удалить задание: %w", err)
	}

	return nil
}

// CreateTaskVersion сохраняет историческую версию задания.
func (r *PgRepository) CreateTaskVersion(ctx context.Context, q db.Querier, version *TaskVersion) error {
	if _, err := q.Exec(ctx, createTaskVersionQuery,
		version.ID, version.TaskID, version.Version, version.StatementMD, []byte(version.AnswerSpec),
		version.AnswerDisplay, string(version.Difficulty), version.KimNumberID, version.TopicID,
		version.Source, version.SourceRef, version.CreatedBy, version.CreatedAt,
	); err != nil {
		return wrapWriteError("сохранить версию задания", err)
	}

	return nil
}

// ListTaskVersions возвращает историю версий задания.
func (r *PgRepository) ListTaskVersions(ctx context.Context, q db.Querier, taskID uuid.UUID) ([]TaskVersion, error) {
	rows, err := q.Query(ctx, listTaskVersionsQuery, taskID)
	if err != nil {
		return nil, fmt.Errorf("выбрать версии задания: %w", err)
	}
	defer rows.Close()

	var versions []TaskVersion
	for rows.Next() {
		version, err := scanTaskVersion(rows)
		if err != nil {
			return nil, err
		}
		versions = append(versions, *version)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("прочитать версии задания: %w", err)
	}

	return versions, nil
}

// ListCandidateTasks возвращает опубликованные задания для генерации варианта.
func (r *PgRepository) ListCandidateTasks(ctx context.Context, q db.Querier, examYearID, kimNumberID uuid.UUID, difficulty *Difficulty, topicID *uuid.UUID) ([]Task, error) {
	rows, err := q.Query(ctx, listCandidateTasksQuery,
		examYearID, kimNumberID, nullableDifficulty(difficulty), nullableUUIDPtr(topicID),
	)
	if err != nil {
		return nil, fmt.Errorf("выбрать задания для генерации: %w", err)
	}
	defer rows.Close()

	return collectTasks(rows)
}

// CreateTaskAttachment сохраняет изображение задания.
func (r *PgRepository) CreateTaskAttachment(ctx context.Context, q db.Querier, attachment *TaskAttachment) error {
	if _, err := q.Exec(ctx, createTaskAttachmentQuery,
		attachment.ID, attachment.TaskID, attachment.Kind, attachment.StorageKey,
		attachment.URL, attachment.Alt, attachment.CreatedAt,
	); err != nil {
		return wrapWriteError("сохранить изображение задания", err)
	}

	return nil
}

// ListTaskAttachments возвращает изображения задания.
func (r *PgRepository) ListTaskAttachments(ctx context.Context, q db.Querier, taskID uuid.UUID) ([]TaskAttachment, error) {
	rows, err := q.Query(ctx, listTaskAttachmentsQuery, taskID)
	if err != nil {
		return nil, fmt.Errorf("выбрать изображения задания: %w", err)
	}
	defer rows.Close()

	var attachments []TaskAttachment
	for rows.Next() {
		var attachment TaskAttachment
		if err := rows.Scan(
			&attachment.ID, &attachment.TaskID, &attachment.Kind, &attachment.StorageKey,
			&attachment.URL, &attachment.Alt, &attachment.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("прочитать изображение задания: %w", err)
		}
		attachments = append(attachments, attachment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("прочитать изображения задания: %w", err)
	}

	return attachments, nil
}

// ListTaskOptions возвращает варианты ответа задания.
func (r *PgRepository) ListTaskOptions(ctx context.Context, q db.Querier, taskID uuid.UUID) ([]TaskOption, error) {
	rows, err := q.Query(ctx, listTaskOptionsQuery, taskID)
	if err != nil {
		return nil, fmt.Errorf("выбрать варианты задания: %w", err)
	}
	defer rows.Close()

	var options []TaskOption
	for rows.Next() {
		var option TaskOption
		if err := rows.Scan(&option.ID, &option.TaskID, &option.Position, &option.ContentMD, &option.IsCorrect); err != nil {
			return nil, fmt.Errorf("прочитать вариант задания: %w", err)
		}
		options = append(options, option)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("прочитать варианты задания: %w", err)
	}

	return options, nil
}

// ReplaceTaskOptions заменяет набор вариантов ответа задания.
func (r *PgRepository) ReplaceTaskOptions(ctx context.Context, q db.Querier, taskID uuid.UUID, options []TaskOption) error {
	if _, err := q.Exec(ctx, deleteTaskOptionsQuery, taskID); err != nil {
		return fmt.Errorf("удалить варианты задания: %w", err)
	}
	for _, option := range options {
		if _, err := q.Exec(ctx, createTaskOptionQuery,
			option.ID, taskID, option.Position, option.ContentMD, option.IsCorrect,
		); err != nil {
			return wrapWriteError("сохранить вариант задания", err)
		}
	}

	return nil
}

func collectTasks(rows pgx.Rows) ([]Task, error) {
	var tasks []Task
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, *task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("прочитать задания: %w", err)
	}

	return tasks, nil
}

func scanTask(row rowScanner) (*Task, error) {
	var (
		task       Task
		answerSpec []byte
	)
	err := row.Scan(
		&task.ID,
		&task.ExamYearID,
		&task.KimNumberID,
		&task.TopicID,
		&task.Difficulty,
		&task.StatementMD,
		&answerSpec,
		&task.AnswerDisplay,
		&task.Source,
		&task.SourceRef,
		&task.Status,
		&task.IsPublic,
		&task.ContentHash,
		&task.Version,
		&task.CreatedBy,
		&task.CreatedAt,
		&task.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("прочитать задание: %w", err)
	}
	task.AnswerSpec = answerSpec

	return &task, nil
}

func scanTaskVersion(row rowScanner) (*TaskVersion, error) {
	var (
		version    TaskVersion
		answerSpec []byte
	)
	err := row.Scan(
		&version.ID,
		&version.TaskID,
		&version.Version,
		&version.StatementMD,
		&answerSpec,
		&version.AnswerDisplay,
		&version.Difficulty,
		&version.KimNumberID,
		&version.TopicID,
		&version.Source,
		&version.SourceRef,
		&version.CreatedBy,
		&version.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("прочитать версию задания: %w", err)
	}
	version.AnswerSpec = answerSpec

	return &version, nil
}

func nullableDifficulty(difficulty *Difficulty) any {
	if difficulty == nil {
		return nil
	}

	return string(*difficulty)
}
