// Package notify реализует email-уведомления через outbox.
package notify

import (
	"time"

	"github.com/google/uuid"
)

// Channel — канал доставки уведомления.
type Channel string

// ChannelEmail — доставка по электронной почте.
const ChannelEmail Channel = "email"

// Status — состояние письма в outbox.
type Status string

const (
	StatusPending Status = "pending"
	StatusSent    Status = "sent"
	StatusFailed  Status = "failed"
)

// MaxAttempts — предельное число попыток отправки.
const MaxAttempts = 5

// TemplateCodeVerification — код шаблона подтверждения email.
const TemplateCodeVerification = "email_verification"

// TemplateCodePasswordReset — код шаблона сброса пароля.
const TemplateCodePasswordReset = "password_reset"

// TemplateCodeLeadNotification — код шаблона уведомления о заявке.
const TemplateCodeLeadNotification = "lead_notification"

// Message описывает письмо, которое нужно поставить в outbox.
type Message struct {
	UserID    uuid.UUID
	Channel   Channel
	Recipient string
	Template  string
	Payload   map[string]any
	DedupKey  string
}

// Template — версия шаблона письма.
type Template struct {
	ID      uuid.UUID
	Code    string
	Subject string
	BodyMD  string
	Channel Channel
	Version int
	Status  string
}

// OutboxItem — сохранённое письмо.
type OutboxItem struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Channel     Channel
	Recipient   string
	Template    string
	Payload     map[string]any
	Status      Status
	Attempts    int
	ScheduledAt time.Time
	SentAt      *time.Time
	LastError   *string
	DedupKey    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
