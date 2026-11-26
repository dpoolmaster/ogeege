// Package admin реализует административный API, аудит и управление пользователями.
package admin

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"

	"oge-ege/internal/auth"
)

// ErrForbidden возвращается при недостаточных правах.
var ErrForbidden = errors.New("недостаточно прав")

// Actor — автор административного действия.
type Actor struct {
	ID   uuid.UUID
	Role auth.Role
	IP   string
}

// AuditEntry — запись журнала аудита.
type AuditEntry struct {
	ID        uuid.UUID
	ActorID   uuid.UUID
	ActorRole string
	Action    string
	Entity    string
	EntityID  *uuid.UUID
	Before    json.RawMessage
	After     json.RawMessage
	IP        string
	CreatedAt time.Time
}

// AuditFilter — фильтр журнала аудита.
type AuditFilter struct {
	Cursor  uuid.UUID
	Limit   int
	ActorID *uuid.UUID
	Action  string
	Entity  string
	From    *time.Time
	To      *time.Time
}

// UserSummary — безопасное представление пользователя для админки.
type UserSummary struct {
	ID        uuid.UUID
	Email     string
	Role      auth.Role
	Status    auth.Status
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewUserSummary создаёт безопасное представление пользователя.
func NewUserSummary(user *auth.User) UserSummary {
	return UserSummary{
		ID:        user.ID,
		Email:     user.Email,
		Role:      user.Role,
		Status:    user.Status,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}
