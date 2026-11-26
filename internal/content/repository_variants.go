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

const variantColumns = `
    id,
    exam_year_id,
    title,
    kind,
    duration_minutes,
    status,
    created_at,
    updated_at
`

const createVariantQuery = `
INSERT INTO content.variants (id, exam_year_id, title, kind, duration_minutes, status, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
`

const createVariantTaskQuery = `
INSERT INTO content.variant_tasks (id, variant_id, position, task_id, points)
VALUES ($1, $2, $3, $4, $5)
`

const getVariantQuery = `
SELECT ` + variantColumns + `
FROM content.variants
WHERE id = $1
`

const listVariantTasksQuery = `
SELECT id, variant_id, position, task_id, points::int
FROM content.variant_tasks
WHERE variant_id = $1
ORDER BY position
`

const listVariantsQuery = `
SELECT ` + variantColumns + `
FROM content.variants
WHERE ($1::uuid IS NULL OR id > $1)
  AND ($2::uuid IS NULL OR exam_year_id = $2)
  AND ($3::text IS NULL OR status = $3)
ORDER BY id
LIMIT $4
`

const setVariantStatusQuery = `
UPDATE content.variants
SET status = $2, updated_at = $3
WHERE id = $1
`

const countVariantsQuery = `
SELECT count(*)
FROM content.variants
WHERE exam_year_id = $1
`

// CreateVariant сохраняет вариант и его позиции.
func (r *PgRepository) CreateVariant(ctx context.Context, q db.Querier, variant *Variant) error {
	if _, err := q.Exec(ctx, createVariantQuery,
		variant.ID, variant.ExamYearID, variant.Title, string(variant.Kind),
		variant.DurationMinutes, string(variant.Status), variant.CreatedAt,
	); err != nil {
		return wrapWriteError("создать вариант", err)
	}

	for _, task := range variant.Tasks {
		if _, err := q.Exec(ctx, createVariantTaskQuery,
			task.ID, variant.ID, task.Position, task.TaskID, task.Points,
		); err != nil {
			return wrapWriteError("сохранить позицию варианта", err)
		}
	}

	return nil
}

// GetVariant возвращает вариант с позициями.
func (r *PgRepository) GetVariant(ctx context.Context, q db.Querier, id uuid.UUID) (*Variant, error) {
	variant, err := scanVariant(q.QueryRow(ctx, getVariantQuery, id))
	if err != nil {
		return nil, err
	}

	tasks, err := r.ListVariantTasks(ctx, q, id)
	if err != nil {
		return nil, err
	}
	variant.Tasks = tasks

	return variant, nil
}

// ListVariantTasks возвращает позиции варианта.
func (r *PgRepository) ListVariantTasks(ctx context.Context, q db.Querier, variantID uuid.UUID) ([]VariantTask, error) {
	rows, err := q.Query(ctx, listVariantTasksQuery, variantID)
	if err != nil {
		return nil, fmt.Errorf("выбрать позиции варианта: %w", err)
	}
	defer rows.Close()

	var tasks []VariantTask
	for rows.Next() {
		var task VariantTask
		if err := rows.Scan(&task.ID, &task.VariantID, &task.Position, &task.TaskID, &task.Points); err != nil {
			return nil, fmt.Errorf("прочитать позицию варианта: %w", err)
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("прочитать позиции варианта: %w", err)
	}

	return tasks, nil
}

// ListVariants возвращает страницу вариантов.
func (r *PgRepository) ListVariants(ctx context.Context, q db.Querier, filter VariantFilter) ([]Variant, error) {
	rows, err := q.Query(ctx, listVariantsQuery,
		nullableUUID(filter.Cursor), nullableUUIDPtr(filter.ExamYearID), nullableStatus(filter.Status), normalizeLimit(filter.Limit),
	)
	if err != nil {
		return nil, fmt.Errorf("выбрать варианты: %w", err)
	}
	defer rows.Close()

	var variants []Variant
	for rows.Next() {
		variant, err := scanVariant(rows)
		if err != nil {
			return nil, err
		}
		variants = append(variants, *variant)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("прочитать варианты: %w", err)
	}

	return variants, nil
}

// SetVariantStatus меняет статус варианта.
func (r *PgRepository) SetVariantStatus(ctx context.Context, q db.Querier, id uuid.UUID, status Status, now time.Time) error {
	if _, err := q.Exec(ctx, setVariantStatusQuery, id, string(status), now); err != nil {
		return fmt.Errorf("изменить статус варианта: %w", err)
	}

	return nil
}

// CountVariants считает варианты учебного года.
func (r *PgRepository) CountVariants(ctx context.Context, q db.Querier, examYearID uuid.UUID) (int, error) {
	var count int
	if err := q.QueryRow(ctx, countVariantsQuery, examYearID).Scan(&count); err != nil {
		return 0, fmt.Errorf("посчитать варианты: %w", err)
	}

	return count, nil
}

func scanVariant(row rowScanner) (*Variant, error) {
	var variant Variant
	err := row.Scan(
		&variant.ID,
		&variant.ExamYearID,
		&variant.Title,
		&variant.Kind,
		&variant.DurationMinutes,
		&variant.Status,
		&variant.CreatedAt,
		&variant.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("прочитать вариант: %w", err)
	}

	return &variant, nil
}
