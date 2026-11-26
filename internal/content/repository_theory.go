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

const theoryColumns = `
    id,
    topic_id,
    content_md,
    version,
    status,
    updated_by,
    created_at,
    updated_at
`

const createTheoryQuery = `
INSERT INTO content.theory (id, topic_id, content_md, version, status, updated_by, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
`

const getTheoryQuery = `
SELECT ` + theoryColumns + `
FROM content.theory
WHERE id = $1
`

const updateTheoryQuery = `
UPDATE content.theory
SET content_md = $2, updated_by = $3, updated_at = $4
WHERE id = $1
`

const getActiveTheoryQuery = `
SELECT ` + theoryColumns + `
FROM content.theory
WHERE topic_id = $1 AND status = 'published'
ORDER BY version DESC
LIMIT 1
`

const getLatestTheoryVersionQuery = `
SELECT COALESCE(MAX(version), 0)
FROM content.theory
WHERE topic_id = $1
`

const listTheoryVersionsQuery = `
SELECT ` + theoryColumns + `
FROM content.theory
WHERE topic_id = $1
ORDER BY version DESC
`

const setTheoryStatusQuery = `
UPDATE content.theory
SET status = $2, updated_at = $3
WHERE id = $1
`

// CreateTheory сохраняет новую версию теории.
func (r *PgRepository) CreateTheory(ctx context.Context, q db.Querier, theory *Theory) error {
	if _, err := q.Exec(ctx, createTheoryQuery,
		theory.ID, theory.TopicID, theory.ContentMD, theory.Version, string(theory.Status), theory.UpdatedBy, theory.CreatedAt,
	); err != nil {
		return wrapWriteError("создать версию теории", err)
	}

	return nil
}

// GetTheory возвращает версию теории по идентификатору.
func (r *PgRepository) GetTheory(ctx context.Context, q db.Querier, id uuid.UUID) (*Theory, error) {
	return scanTheory(q.QueryRow(ctx, getTheoryQuery, id))
}

// UpdateTheory обновляет текст черновой версии теории.
func (r *PgRepository) UpdateTheory(ctx context.Context, q db.Querier, theory *Theory) error {
	if _, err := q.Exec(ctx, updateTheoryQuery,
		theory.ID, theory.ContentMD, theory.UpdatedBy, theory.UpdatedAt,
	); err != nil {
		return wrapWriteError("обновить версию теории", err)
	}

	return nil
}

// GetActiveTheory возвращает действующую опубликованную версию теории.
func (r *PgRepository) GetActiveTheory(ctx context.Context, q db.Querier, topicID uuid.UUID) (*Theory, error) {
	return scanTheory(q.QueryRow(ctx, getActiveTheoryQuery, topicID))
}

// GetLatestTheoryVersion возвращает последний номер версии теории.
func (r *PgRepository) GetLatestTheoryVersion(ctx context.Context, q db.Querier, topicID uuid.UUID) (int, error) {
	var version int
	if err := q.QueryRow(ctx, getLatestTheoryVersionQuery, topicID).Scan(&version); err != nil {
		return 0, fmt.Errorf("получить версию теории: %w", err)
	}

	return version, nil
}

// ListTheoryVersions возвращает историю версий теории.
func (r *PgRepository) ListTheoryVersions(ctx context.Context, q db.Querier, topicID uuid.UUID) ([]Theory, error) {
	rows, err := q.Query(ctx, listTheoryVersionsQuery, topicID)
	if err != nil {
		return nil, fmt.Errorf("выбрать версии теории: %w", err)
	}
	defer rows.Close()

	var versions []Theory
	for rows.Next() {
		version, err := scanTheory(rows)
		if err != nil {
			return nil, err
		}
		versions = append(versions, *version)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("прочитать версии теории: %w", err)
	}

	return versions, nil
}

// SetTheoryStatus меняет статус версии теории.
func (r *PgRepository) SetTheoryStatus(ctx context.Context, q db.Querier, id uuid.UUID, status Status, now time.Time) error {
	if _, err := q.Exec(ctx, setTheoryStatusQuery, id, string(status), now); err != nil {
		return fmt.Errorf("изменить статус теории: %w", err)
	}

	return nil
}

func scanTheory(row rowScanner) (*Theory, error) {
	var theory Theory
	err := row.Scan(
		&theory.ID,
		&theory.TopicID,
		&theory.ContentMD,
		&theory.Version,
		&theory.Status,
		&theory.UpdatedBy,
		&theory.CreatedAt,
		&theory.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("прочитать теорию: %w", err)
	}

	return &theory, nil
}
