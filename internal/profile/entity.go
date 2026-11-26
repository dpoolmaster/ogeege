// Package profile реализует профиль ученика, согласия, цели и настройки.
package profile

import (
	"time"

	"github.com/google/uuid"
)

// Profile — персональные данные ученика.
type Profile struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	FirstName string
	LastName  string
	BirthDate *time.Time
	Grade     *int
	Timezone  string
	Contacts  map[string]any
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Consent — зафиксированное согласие.
type Consent struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	Kind            string
	Granted         bool
	DocumentVersion string
	GrantedAt       time.Time
	IP              string
}

// StudyGoal — учебная цель.
type StudyGoal struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	ExamID        uuid.UUID
	ExamDate      time.Time
	TargetScore   int
	WeeklyMinutes int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Settings — настройки уведомлений.
type Settings struct {
	ID                 uuid.UUID
	UserID             uuid.UUID
	EmailNotifications bool
	WeeklyReport       bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
