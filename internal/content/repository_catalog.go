package content

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"oge-ege/internal/shared/db"
)

const kimNumberColumns = `
    id,
    exam_year_id,
    number,
    part,
    answer_type,
    max_primary_score::int,
    COALESCE(title, ''),
    created_at,
    updated_at
`

const createKimNumberQuery = `
INSERT INTO content.kim_numbers (
    id, exam_year_id, number, part, answer_type, max_primary_score, title, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''), $8, $8)
`

const updateKimNumberQuery = `
UPDATE content.kim_numbers
SET number = $2,
    part = $3,
    answer_type = $4,
    max_primary_score = $5,
    title = NULLIF($6, ''),
    updated_at = $7
WHERE id = $1
`

const getKimNumberQuery = `
SELECT ` + kimNumberColumns + `
FROM content.kim_numbers
WHERE id = $1
`

const getKimNumberByNumberQuery = `
SELECT ` + kimNumberColumns + `
FROM content.kim_numbers
WHERE exam_year_id = $1 AND number = $2
`

const listKimNumbersQuery = `
SELECT ` + kimNumberColumns + `
FROM content.kim_numbers
WHERE ($1::uuid IS NULL OR id > $1)
  AND ($2::uuid IS NULL OR exam_year_id = $2)
ORDER BY id
LIMIT $3
`

const deleteKimNumberQuery = `DELETE FROM content.kim_numbers WHERE id = $1`

const sumMaxPrimaryScoreQuery = `
SELECT COALESCE(SUM(max_primary_score), 0)::int
FROM content.kim_numbers
WHERE exam_year_id = $1
`

// CreateKimNumber сохраняет номер КИМ.
func (r *PgRepository) CreateKimNumber(ctx context.Context, q db.Querier, number *KimNumber) error {
	if _, err := q.Exec(ctx, createKimNumberQuery,
		number.ID, number.ExamYearID, number.Number, number.Part, string(number.AnswerType),
		number.MaxPrimaryScore, number.Title, number.CreatedAt,
	); err != nil {
		return wrapWriteError("создать номер КИМ", err)
	}

	return nil
}

// UpdateKimNumber обновляет номер КИМ.
func (r *PgRepository) UpdateKimNumber(ctx context.Context, q db.Querier, number *KimNumber) error {
	if _, err := q.Exec(ctx, updateKimNumberQuery,
		number.ID, number.Number, number.Part, string(number.AnswerType),
		number.MaxPrimaryScore, number.Title, number.UpdatedAt,
	); err != nil {
		return wrapWriteError("обновить номер КИМ", err)
	}

	return nil
}

// GetKimNumber возвращает номер КИМ по идентификатору.
func (r *PgRepository) GetKimNumber(ctx context.Context, q db.Querier, id uuid.UUID) (*KimNumber, error) {
	return scanKimNumber(q.QueryRow(ctx, getKimNumberQuery, id))
}

// GetKimNumberByNumber возвращает номер КИМ учебного года.
func (r *PgRepository) GetKimNumberByNumber(ctx context.Context, q db.Querier, examYearID uuid.UUID, number int) (*KimNumber, error) {
	return scanKimNumber(q.QueryRow(ctx, getKimNumberByNumberQuery, examYearID, number))
}

// ListKimNumbers возвращает страницу номеров КИМ.
func (r *PgRepository) ListKimNumbers(ctx context.Context, q db.Querier, filter KimNumberFilter) ([]KimNumber, error) {
	rows, err := q.Query(ctx, listKimNumbersQuery,
		nullableUUID(filter.Cursor), nullableUUIDPtr(filter.ExamYearID), normalizeLimit(filter.Limit),
	)
	if err != nil {
		return nil, fmt.Errorf("выбрать номера КИМ: %w", err)
	}
	defer rows.Close()

	var numbers []KimNumber
	for rows.Next() {
		number, err := scanKimNumber(rows)
		if err != nil {
			return nil, err
		}
		numbers = append(numbers, *number)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("прочитать номера КИМ: %w", err)
	}

	return numbers, nil
}

// DeleteKimNumber удаляет номер КИМ.
func (r *PgRepository) DeleteKimNumber(ctx context.Context, q db.Querier, id uuid.UUID) error {
	if _, err := q.Exec(ctx, deleteKimNumberQuery, id); err != nil {
		return fmt.Errorf("удалить номер КИМ: %w", err)
	}

	return nil
}

// SumMaxPrimaryScore возвращает сумму максимальных баллов учебного года.
func (r *PgRepository) SumMaxPrimaryScore(ctx context.Context, q db.Querier, examYearID uuid.UUID) (int, error) {
	var sum int
	if err := q.QueryRow(ctx, sumMaxPrimaryScoreQuery, examYearID).Scan(&sum); err != nil {
		return 0, fmt.Errorf("посчитать максимальный первичный балл: %w", err)
	}

	return sum, nil
}

const topicColumns = `
    id,
    exam_id,
    parent_id,
    code,
    title,
    order_index,
    created_at,
    updated_at
`

const createTopicQuery = `
INSERT INTO content.topics (id, exam_id, parent_id, code, title, order_index, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
`

const updateTopicQuery = `
UPDATE content.topics
SET parent_id = $2, code = $3, title = $4, order_index = $5, updated_at = $6
WHERE id = $1
`

const getTopicQuery = `
SELECT ` + topicColumns + `
FROM content.topics
WHERE id = $1
`

const getTopicByCodeQuery = `
SELECT ` + topicColumns + `
FROM content.topics
WHERE exam_id = $1 AND code = $2
`

const listTopicsQuery = `
SELECT ` + topicColumns + `
FROM content.topics
WHERE ($1::uuid IS NULL OR id > $1)
  AND ($2::uuid IS NULL OR exam_id = $2)
  AND ($3::uuid IS NULL OR parent_id = $3)
ORDER BY id
LIMIT $4
`

const deleteTopicQuery = `DELETE FROM content.topics WHERE id = $1`

const deleteKimTopicsQuery = `DELETE FROM content.kim_topics WHERE kim_number_id = $1`

const insertKimTopicQuery = `
INSERT INTO content.kim_topics (kim_number_id, topic_id, weight)
VALUES ($1, $2, $3)
`

const listKimTopicsQuery = `
SELECT kim_number_id, topic_id, weight::float8
FROM content.kim_topics
WHERE kim_number_id = $1
ORDER BY topic_id
`

const getKimTopicQuery = `
SELECT kim_number_id, topic_id, weight::float8
FROM content.kim_topics
WHERE kim_number_id = $1 AND topic_id = $2
`

// CreateTopic сохраняет тему.
func (r *PgRepository) CreateTopic(ctx context.Context, q db.Querier, topic *Topic) error {
	if _, err := q.Exec(ctx, createTopicQuery,
		topic.ID, topic.ExamID, topic.ParentID, topic.Code, topic.Title, topic.OrderIndex, topic.CreatedAt,
	); err != nil {
		return wrapWriteError("создать тему", err)
	}

	return nil
}

// UpdateTopic обновляет тему.
func (r *PgRepository) UpdateTopic(ctx context.Context, q db.Querier, topic *Topic) error {
	if _, err := q.Exec(ctx, updateTopicQuery,
		topic.ID, topic.ParentID, topic.Code, topic.Title, topic.OrderIndex, topic.UpdatedAt,
	); err != nil {
		return wrapWriteError("обновить тему", err)
	}

	return nil
}

// GetTopic возвращает тему по идентификатору.
func (r *PgRepository) GetTopic(ctx context.Context, q db.Querier, id uuid.UUID) (*Topic, error) {
	return scanTopic(q.QueryRow(ctx, getTopicQuery, id))
}

// GetTopicByCode возвращает тему трека по коду.
func (r *PgRepository) GetTopicByCode(ctx context.Context, q db.Querier, examID uuid.UUID, code string) (*Topic, error) {
	return scanTopic(q.QueryRow(ctx, getTopicByCodeQuery, examID, code))
}

// ListTopics возвращает страницу тем.
func (r *PgRepository) ListTopics(ctx context.Context, q db.Querier, filter TopicFilter) ([]Topic, error) {
	rows, err := q.Query(ctx, listTopicsQuery,
		nullableUUID(filter.Cursor), nullableUUIDPtr(filter.ExamID), nullableUUIDPtr(filter.ParentID), normalizeLimit(filter.Limit),
	)
	if err != nil {
		return nil, fmt.Errorf("выбрать темы: %w", err)
	}
	defer rows.Close()

	var topics []Topic
	for rows.Next() {
		topic, err := scanTopic(rows)
		if err != nil {
			return nil, err
		}
		topics = append(topics, *topic)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("прочитать темы: %w", err)
	}

	return topics, nil
}

// DeleteTopic удаляет тему.
func (r *PgRepository) DeleteTopic(ctx context.Context, q db.Querier, id uuid.UUID) error {
	if _, err := q.Exec(ctx, deleteTopicQuery, id); err != nil {
		return fmt.Errorf("удалить тему: %w", err)
	}

	return nil
}

// ReplaceKimTopics заменяет связи номера КИМ с темами.
func (r *PgRepository) ReplaceKimTopics(ctx context.Context, q db.Querier, kimNumberID uuid.UUID, topics []KimTopic) error {
	if _, err := q.Exec(ctx, deleteKimTopicsQuery, kimNumberID); err != nil {
		return fmt.Errorf("очистить связи тем: %w", err)
	}

	for _, topic := range topics {
		if _, err := q.Exec(ctx, insertKimTopicQuery, kimNumberID, topic.TopicID, topic.Weight); err != nil {
			return wrapWriteError("сохранить связь темы", err)
		}
	}

	return nil
}

// ListKimTopics возвращает связи номера КИМ с темами.
func (r *PgRepository) ListKimTopics(ctx context.Context, q db.Querier, kimNumberID uuid.UUID) ([]KimTopic, error) {
	rows, err := q.Query(ctx, listKimTopicsQuery, kimNumberID)
	if err != nil {
		return nil, fmt.Errorf("выбрать связи тем: %w", err)
	}
	defer rows.Close()

	var topics []KimTopic
	for rows.Next() {
		var topic KimTopic
		if err := rows.Scan(&topic.KimNumberID, &topic.TopicID, &topic.Weight); err != nil {
			return nil, fmt.Errorf("прочитать связь темы: %w", err)
		}
		topics = append(topics, topic)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("прочитать связи тем: %w", err)
	}

	return topics, nil
}

// GetKimTopic возвращает связь номера КИМ с темой.
func (r *PgRepository) GetKimTopic(ctx context.Context, q db.Querier, kimNumberID, topicID uuid.UUID) (*KimTopic, error) {
	var topic KimTopic
	err := q.QueryRow(ctx, getKimTopicQuery, kimNumberID, topicID).Scan(&topic.KimNumberID, &topic.TopicID, &topic.Weight)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("прочитать связь темы: %w", err)
	}

	return &topic, nil
}

func scanKimNumber(row rowScanner) (*KimNumber, error) {
	var number KimNumber
	err := row.Scan(
		&number.ID,
		&number.ExamYearID,
		&number.Number,
		&number.Part,
		&number.AnswerType,
		&number.MaxPrimaryScore,
		&number.Title,
		&number.CreatedAt,
		&number.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("прочитать номер КИМ: %w", err)
	}

	return &number, nil
}

func scanTopic(row rowScanner) (*Topic, error) {
	var topic Topic
	err := row.Scan(
		&topic.ID,
		&topic.ExamID,
		&topic.ParentID,
		&topic.Code,
		&topic.Title,
		&topic.OrderIndex,
		&topic.CreatedAt,
		&topic.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("прочитать тему: %w", err)
	}

	return &topic, nil
}
