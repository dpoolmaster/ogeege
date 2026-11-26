package admin

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"oge-ege/internal/shared/db"
)

// Repository описывает хранилище журнала аудита.
type Repository interface {
	InsertAudit(ctx context.Context, q db.Querier, entry *AuditEntry) error
	ListAudit(ctx context.Context, q db.Querier, filter AuditFilter) ([]AuditEntry, error)
}

const insertAuditQuery = `
INSERT INTO admin.audit_log (
    id, actor_id, actor_role, action, entity, entity_id, before_json, after_json, ip, created_at
) VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8::jsonb, NULLIF($9, '')::inet, $10)
`

const listAuditQuery = `
SELECT
    id,
    actor_id,
    actor_role,
    action,
    entity,
    entity_id,
    before_json,
    after_json,
    COALESCE(host(ip), ''),
    created_at
FROM admin.audit_log
WHERE ($1::uuid IS NULL OR id > $1)
  AND ($2::uuid IS NULL OR actor_id = $2)
  AND ($3::text = '' OR action = $3)
  AND ($4::text = '' OR entity = $4)
  AND ($5::timestamptz IS NULL OR created_at >= $5)
  AND ($6::timestamptz IS NULL OR created_at <= $6)
ORDER BY id
LIMIT $7
`

// PgRepository работает с PostgreSQL.
type PgRepository struct{}

// NewPgRepository создаёт репозиторий admin.
func NewPgRepository() *PgRepository {
	return &PgRepository{}
}

// InsertAudit сохраняет запись аудита.
func (r *PgRepository) InsertAudit(ctx context.Context, q db.Querier, entry *AuditEntry) error {
	if _, err := q.Exec(ctx, insertAuditQuery,
		entry.ID, entry.ActorID, entry.ActorRole, entry.Action, entry.Entity, entry.EntityID,
		nullableJSON(entry.Before), nullableJSON(entry.After), entry.IP, entry.CreatedAt,
	); err != nil {
		return fmt.Errorf("сохранить запись аудита: %w", err)
	}

	return nil
}

// ListAudit возвращает страницу журнала аудита.
func (r *PgRepository) ListAudit(ctx context.Context, q db.Querier, filter AuditFilter) ([]AuditEntry, error) {
	rows, err := q.Query(ctx, listAuditQuery,
		nullableUUID(filter.Cursor), nullableUUIDPtr(filter.ActorID), filter.Action, filter.Entity,
		filter.From, filter.To, normalizeLimit(filter.Limit),
	)
	if err != nil {
		return nil, fmt.Errorf("выбрать журнал аудита: %w", err)
	}
	defer rows.Close()

	var entries []AuditEntry
	for rows.Next() {
		var entry AuditEntry
		if err := rows.Scan(
			&entry.ID, &entry.ActorID, &entry.ActorRole, &entry.Action, &entry.Entity, &entry.EntityID,
			&entry.Before, &entry.After, &entry.IP, &entry.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("прочитать запись аудита: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("прочитать журнал аудита: %w", err)
	}

	return entries, nil
}

func nullableJSON(value []byte) any {
	if len(value) == 0 {
		return nil
	}

	return string(value)
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
