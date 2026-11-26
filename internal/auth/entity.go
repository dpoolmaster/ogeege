// Package auth реализует учётные записи, сессии и проверку прав.
package auth

import (
	"time"

	"github.com/google/uuid"
)

// Role — роль пользователя.
type Role string

const (
	RoleStudent   Role = "student"
	RoleParent    Role = "parent"
	RoleMethodist Role = "methodist"
	RoleAdmin     Role = "admin"
)

// Valid сообщает, известна ли роль.
func (r Role) Valid() bool {
	switch r {
	case RoleStudent, RoleParent, RoleMethodist, RoleAdmin:
		return true
	default:
		return false
	}
}

// IsPublic сообщает, разрешена ли роль при публичной регистрации.
func (r Role) IsPublic() bool {
	return r == RoleStudent || r == RoleParent
}

// Status — состояние учётной записи.
type Status string

const (
	StatusUnverified Status = "unverified"
	StatusActive     Status = "active"
	StatusBlocked    Status = "blocked"
	StatusAnonymized Status = "anonymized"
)

// User — учётная запись.
type User struct {
	ID              uuid.UUID
	Email           string
	PasswordHash    string
	Role            Role
	Status          Status
	EmailVerifiedAt *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Verification — токен подтверждения email или сброса пароля.
type Verification struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	TokenHash  []byte
	ExpiresAt  time.Time
	ConsumedAt *time.Time
	CreatedAt  time.Time
}

// Session — серверная сессия.
type Session struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	TokenHash  []byte
	UserAgent  string
	IP         string
	ExpiresAt  time.Time
	RevokedAt  *time.Time
	LastSeenAt time.Time
	CreatedAt  time.Time
}

// LoginAttempt — запись попытки входа.
type LoginAttempt struct {
	ID        uuid.UUID
	Email     string
	IP        string
	Success   bool
	CreatedAt time.Time
}

// ConsentGrant — согласие, предоставленное при регистрации.
type ConsentGrant struct {
	Kind            string
	Granted         bool
	DocumentVersion string
}

// Principal — минимальные данные пользователя для контекста запроса.
type Principal struct {
	ID     uuid.UUID
	Email  string
	Role   Role
	Status Status
}
