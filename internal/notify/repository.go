package notify

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

// Repository описывает хранилище outbox и шаблонов.
type Repository interface {
	InsertOutbox(ctx context.Context, q db.Querier, item *OutboxItem) error
	GetActiveTemplate(ctx context.Context, q db.Querier, code string) (*Template, error)
	FetchDue(ctx context.Context, q db.Querier, now time.Time, limit int) ([]OutboxItem, error)
	MarkSent(ctx context.Context, q db.Querier, id uuid.UUID, now time.Time, attempts int) error
	Reschedule(ctx context.Context, q db.Querier, id uuid.UUID, attempts int, scheduledAt time.Time, lastError string) error
	MarkFailed(ctx context.Context, q db.Querier, id uuid.UUID, attempts int, lastError string) error
	GetOutbox(ctx context.Context, q db.Querier, id uuid.UUID) (*OutboxItem, error)
}

const insertOutboxQuery = `
INSERT INTO notify.outbox (
    id,
    user_id,
    channel,
    recipient,
    template,
    payload_json,
    status,
    attempts,
    scheduled_at,
    dedup_key
) VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8, $9, $10)
ON CONFLICT DO NOTHING
`

const getActiveTemplateQuery = `
SELECT
    id,
    code,
    subject,
    body_md,
    channel,
    version,
    status
FROM notify.templates
WHERE code = $1 AND status = 'active' AND channel = 'email'
ORDER BY version DESC
LIMIT 1
`

const fetchDueQuery = `
SELECT
    id,
    user_id,
    channel,
    recipient,
    template,
    payload_json,
    status,
    attempts,
    scheduled_at,
    sent_at,
    last_error,
    dedup_key,
    created_at,
    updated_at
FROM notify.outbox
WHERE status = 'pending' AND scheduled_at <= $1 AND attempts < $2
ORDER BY scheduled_at, created_at
LIMIT $3
FOR UPDATE SKIP LOCKED
`

const markSentQuery = `
UPDATE notify.outbox
SET status = 'sent', attempts = $2, sent_at = $3, last_error = NULL, updated_at = now()
WHERE id = $1
`

const rescheduleQuery = `
UPDATE notify.outbox
SET status = 'pending', attempts = $2, scheduled_at = $3, last_error = $4, updated_at = now()
WHERE id = $1
`

const markFailedQuery = `
UPDATE notify.outbox
SET status = 'failed', attempts = $2, last_error = $3, updated_at = now()
WHERE id = $1
`

const getOutboxQuery = `
SELECT
    id,
    user_id,
    channel,
    recipient,
    template,
    payload_json,
    status,
    attempts,
    scheduled_at,
    sent_at,
    last_error,
    dedup_key,
    created_at,
    updated_at
FROM notify.outbox
WHERE id = $1
`

// PgRepository работает с PostgreSQL.
type PgRepository struct{}

// NewPgRepository создаёт репозиторий notify.
func NewPgRepository() *PgRepository {
	return &PgRepository{}
}

// InsertOutbox сохраняет письмо; дубликат по dedup_key игнорируется.
func (r *PgRepository) InsertOutbox(ctx context.Context, q db.Querier, item *OutboxItem) error {
	payload, err := json.Marshal(item.Payload)
	if err != nil {
		return fmt.Errorf("сериализовать payload письма: %w", err)
	}

	var userID *uuid.UUID
	if item.UserID != uuid.Nil {
		userID = &item.UserID
	}

	var dedupKey *string
	if item.DedupKey != "" {
		dedupKey = &item.DedupKey
	}

	if _, err := q.Exec(ctx, insertOutboxQuery,
		item.ID,
		userID,
		string(item.Channel),
		item.Recipient,
		item.Template,
		string(payload),
		string(item.Status),
		item.Attempts,
		item.ScheduledAt,
		dedupKey,
	); err != nil {
		return fmt.Errorf("сохранить письмо в outbox: %w", err)
	}

	return nil
}

// GetActiveTemplate возвращает активную версию шаблона по коду.
func (r *PgRepository) GetActiveTemplate(ctx context.Context, q db.Querier, code string) (*Template, error) {
	var template Template
	err := q.QueryRow(ctx, getActiveTemplateQuery, code).Scan(
		&template.ID,
		&template.Code,
		&template.Subject,
		&template.BodyMD,
		&template.Channel,
		&template.Version,
		&template.Status,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("активный шаблон %q не найден", code)
	}
	if err != nil {
		return nil, fmt.Errorf("прочитать шаблон %q: %w", code, err)
	}

	return &template, nil
}

// FetchDue блокирует и возвращает пачку писем к отправке.
func (r *PgRepository) FetchDue(ctx context.Context, q db.Querier, now time.Time, limit int) ([]OutboxItem, error) {
	rows, err := q.Query(ctx, fetchDueQuery, now, MaxAttempts, limit)
	if err != nil {
		return nil, fmt.Errorf("выбрать письма из outbox: %w", err)
	}
	defer rows.Close()

	var items []OutboxItem
	for rows.Next() {
		item, err := scanOutbox(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("прочитать письма из outbox: %w", err)
	}

	return items, nil
}

// MarkSent помечает письмо отправленным.
func (r *PgRepository) MarkSent(ctx context.Context, q db.Querier, id uuid.UUID, now time.Time, attempts int) error {
	if _, err := q.Exec(ctx, markSentQuery, id, attempts, now); err != nil {
		return fmt.Errorf("пометить письмо отправленным: %w", err)
	}

	return nil
}

// Reschedule переносит повторную попытку.
func (r *PgRepository) Reschedule(ctx context.Context, q db.Querier, id uuid.UUID, attempts int, scheduledAt time.Time, lastError string) error {
	if _, err := q.Exec(ctx, rescheduleQuery, id, attempts, scheduledAt, lastError); err != nil {
		return fmt.Errorf("перенести повторную отправку: %w", err)
	}

	return nil
}

// MarkFailed помечает письмо окончательно неуспешным.
func (r *PgRepository) MarkFailed(ctx context.Context, q db.Querier, id uuid.UUID, attempts int, lastError string) error {
	if _, err := q.Exec(ctx, markFailedQuery, id, attempts, lastError); err != nil {
		return fmt.Errorf("пометить письмо неуспешным: %w", err)
	}

	return nil
}

// GetOutbox возвращает письмо по идентификатору.
func (r *PgRepository) GetOutbox(ctx context.Context, q db.Querier, id uuid.UUID) (*OutboxItem, error) {
	row := q.QueryRow(ctx, getOutboxQuery, id)
	item, err := scanOutbox(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("письмо %s не найдено", id)
	}
	if err != nil {
		return nil, err
	}

	return &item, nil
}

type outboxScanner interface {
	Scan(dest ...any) error
}

func scanOutbox(row outboxScanner) (OutboxItem, error) {
	var (
		item     OutboxItem
		payload  []byte
		userID   *uuid.UUID
		dedupKey *string
	)

	err := row.Scan(
		&item.ID,
		&userID,
		&item.Channel,
		&item.Recipient,
		&item.Template,
		&payload,
		&item.Status,
		&item.Attempts,
		&item.ScheduledAt,
		&item.SentAt,
		&item.LastError,
		&dedupKey,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return OutboxItem{}, err
	}

	if userID != nil {
		item.UserID = *userID
	}
	if dedupKey != nil {
		item.DedupKey = *dedupKey
	}
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &item.Payload); err != nil {
			return OutboxItem{}, fmt.Errorf("разобрать payload письма: %w", err)
		}
	}

	return item, nil
}
