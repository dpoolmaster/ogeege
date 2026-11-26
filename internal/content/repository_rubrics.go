package content

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"oge-ege/internal/shared/db"
)

const rubricColumns = `
    id,
    kim_number_id,
    exam_year_id,
    version,
    criteria_json,
    status,
    created_at,
    updated_at
`

const createRubricQuery = `
INSERT INTO content.rubrics (
    id, kim_number_id, exam_year_id, version, criteria_json, status, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7, $7)
`

const getRubricQuery = `
SELECT ` + rubricColumns + `
FROM content.rubrics
WHERE id = $1
`

const getActiveRubricQuery = `
SELECT ` + rubricColumns + `
FROM content.rubrics
WHERE kim_number_id = $1 AND exam_year_id = $2 AND status = 'published'
ORDER BY version DESC
LIMIT 1
`

const getLatestRubricVersionQuery = `
SELECT COALESCE(MAX(version), 0)
FROM content.rubrics
WHERE kim_number_id = $1 AND exam_year_id = $2
`

const listRubricsQuery = `
SELECT ` + rubricColumns + `
FROM content.rubrics
WHERE ($1::uuid IS NULL OR id > $1)
  AND ($2::uuid IS NULL OR kim_number_id = $2)
  AND ($3::uuid IS NULL OR exam_year_id = $3)
  AND ($4::text IS NULL OR status = $4)
ORDER BY id
LIMIT $5
`

const setRubricStatusQuery = `
UPDATE content.rubrics
SET status = $2, updated_at = $3
WHERE id = $1
`

// CreateRubric сохраняет версию рубрики.
func (r *PgRepository) CreateRubric(ctx context.Context, q db.Querier, rubric *Rubric) error {
	criteria, err := marshalJSON(rubric.Criteria)
	if err != nil {
		return err
	}

	if _, err := q.Exec(ctx, createRubricQuery,
		rubric.ID, rubric.KimNumberID, rubric.ExamYearID, rubric.Version, string(criteria),
		string(rubric.Status), rubric.CreatedAt,
	); err != nil {
		return wrapWriteError("создать рубрику", err)
	}

	return nil
}

// GetRubric возвращает рубрику по идентификатору.
func (r *PgRepository) GetRubric(ctx context.Context, q db.Querier, id uuid.UUID) (*Rubric, error) {
	return scanRubric(q.QueryRow(ctx, getRubricQuery, id))
}

// GetActiveRubric возвращает действующую рубрику.
func (r *PgRepository) GetActiveRubric(ctx context.Context, q db.Querier, kimNumberID, examYearID uuid.UUID) (*Rubric, error) {
	return scanRubric(q.QueryRow(ctx, getActiveRubricQuery, kimNumberID, examYearID))
}

// GetLatestRubricVersion возвращает последнюю версию рубрики.
func (r *PgRepository) GetLatestRubricVersion(ctx context.Context, q db.Querier, kimNumberID, examYearID uuid.UUID) (int, error) {
	var version int
	if err := q.QueryRow(ctx, getLatestRubricVersionQuery, kimNumberID, examYearID).Scan(&version); err != nil {
		return 0, fmt.Errorf("получить версию рубрики: %w", err)
	}

	return version, nil
}

// ListRubrics возвращает страницу рубрик.
func (r *PgRepository) ListRubrics(ctx context.Context, q db.Querier, filter RubricFilter) ([]Rubric, error) {
	rows, err := q.Query(ctx, listRubricsQuery,
		nullableUUID(filter.Cursor), nullableUUIDPtr(filter.KimNumberID), nullableUUIDPtr(filter.ExamYearID),
		nullableStatus(filter.Status), normalizeLimit(filter.Limit),
	)
	if err != nil {
		return nil, fmt.Errorf("выбрать рубрики: %w", err)
	}
	defer rows.Close()

	var rubrics []Rubric
	for rows.Next() {
		rubric, err := scanRubric(rows)
		if err != nil {
			return nil, err
		}
		rubrics = append(rubrics, *rubric)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("прочитать рубрики: %w", err)
	}

	return rubrics, nil
}

// SetRubricStatus меняет статус рубрики.
func (r *PgRepository) SetRubricStatus(ctx context.Context, q db.Querier, id uuid.UUID, status Status, now time.Time) error {
	if _, err := q.Exec(ctx, setRubricStatusQuery, id, string(status), now); err != nil {
		return fmt.Errorf("изменить статус рубрики: %w", err)
	}

	return nil
}

func scanRubric(row rowScanner) (*Rubric, error) {
	var (
		rubric   Rubric
		criteria []byte
	)
	err := row.Scan(
		&rubric.ID,
		&rubric.KimNumberID,
		&rubric.ExamYearID,
		&rubric.Version,
		&criteria,
		&rubric.Status,
		&rubric.CreatedAt,
		&rubric.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("прочитать рубрику: %w", err)
	}
	if len(criteria) > 0 {
		if err := json.Unmarshal(criteria, &rubric.Criteria); err != nil {
			return nil, fmt.Errorf("разобрать критерии рубрики: %w", err)
		}
	}

	return &rubric, nil
}

const scoreScaleColumns = `
    id,
    exam_year_id,
    primary_score,
    test_score,
    grade,
    min_threshold,
    created_at,
    updated_at
`

const deleteScoreScalesQuery = `DELETE FROM content.score_scales WHERE exam_year_id = $1`

const insertScoreScaleQuery = `
INSERT INTO content.score_scales (
    id, exam_year_id, primary_score, test_score, grade, min_threshold, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, now(), now())
`

const listScoreScalesQuery = `
SELECT ` + scoreScaleColumns + `
FROM content.score_scales
WHERE exam_year_id = $1
ORDER BY primary_score
`

// ReplaceScoreScales заменяет шкалу перевода учебного года.
func (r *PgRepository) ReplaceScoreScales(ctx context.Context, q db.Querier, examYearID uuid.UUID, entries []ScoreScale) error {
	if _, err := q.Exec(ctx, deleteScoreScalesQuery, examYearID); err != nil {
		return fmt.Errorf("очистить шкалу перевода: %w", err)
	}

	for _, entry := range entries {
		if _, err := q.Exec(ctx, insertScoreScaleQuery,
			entry.ID, examYearID, entry.PrimaryScore, entry.TestScore, entry.Grade, entry.MinThreshold,
		); err != nil {
			return wrapWriteError("сохранить строку шкалы", err)
		}
	}

	return nil
}

// ListScoreScales возвращает шкалу перевода учебного года.
func (r *PgRepository) ListScoreScales(ctx context.Context, q db.Querier, examYearID uuid.UUID) ([]ScoreScale, error) {
	rows, err := q.Query(ctx, listScoreScalesQuery, examYearID)
	if err != nil {
		return nil, fmt.Errorf("выбрать шкалу перевода: %w", err)
	}
	defer rows.Close()

	var scales []ScoreScale
	for rows.Next() {
		var scale ScoreScale
		if err := rows.Scan(
			&scale.ID, &scale.ExamYearID, &scale.PrimaryScore, &scale.TestScore,
			&scale.Grade, &scale.MinThreshold, &scale.CreatedAt, &scale.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("прочитать строку шкалы: %w", err)
		}
		scales = append(scales, scale)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("прочитать шкалу перевода: %w", err)
	}

	return scales, nil
}
