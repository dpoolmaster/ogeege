package profile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"oge-ege/internal/shared/db"
)

// Repository описывает хранилище модуля profile.
type Repository interface {
	GetProfile(ctx context.Context, q db.Querier, userID uuid.UUID) (*Profile, error)
	UpsertProfile(ctx context.Context, q db.Querier, profile *Profile) error
	DeleteProfile(ctx context.Context, q db.Querier, userID uuid.UUID) error

	CreateConsent(ctx context.Context, q db.Querier, consent *Consent) error
	GetLatestConsent(ctx context.Context, q db.Querier, userID uuid.UUID, kind string) (*Consent, error)
	AnonymizeConsents(ctx context.Context, q db.Querier, userID, anonymizedUserID uuid.UUID) error

	CreateStudyGoal(ctx context.Context, q db.Querier, goal *StudyGoal) error
	GetCurrentGoal(ctx context.Context, q db.Querier, userID uuid.UUID) (*StudyGoal, error)
	DeleteStudyGoals(ctx context.Context, q db.Querier, userID uuid.UUID) error

	GetSettings(ctx context.Context, q db.Querier, userID uuid.UUID) (*Settings, error)
	UpsertSettings(ctx context.Context, q db.Querier, settings *Settings) error
	DeleteSettings(ctx context.Context, q db.Querier, userID uuid.UUID) error
}

const getProfileQuery = `
SELECT
    id,
    user_id,
    COALESCE(first_name, ''),
    COALESCE(last_name, ''),
    birth_date,
    grade,
    COALESCE(timezone, ''),
    contacts,
    created_at,
    updated_at
FROM profile.profiles
WHERE user_id = $1
`

const upsertProfileQuery = `
INSERT INTO profile.profiles (
    id, user_id, first_name, last_name, birth_date, grade, timezone, contacts, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9, $9)
ON CONFLICT (user_id) DO UPDATE SET
    first_name = EXCLUDED.first_name,
    last_name = EXCLUDED.last_name,
    birth_date = EXCLUDED.birth_date,
    grade = EXCLUDED.grade,
    timezone = EXCLUDED.timezone,
    contacts = EXCLUDED.contacts,
    updated_at = EXCLUDED.updated_at
`

const deleteProfileQuery = `DELETE FROM profile.profiles WHERE user_id = $1`

const createConsentQuery = `
INSERT INTO profile.consents (id, user_id, kind, granted, document_version, granted_at, ip)
VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, '')::inet)
`

const getLatestConsentQuery = `
SELECT id, user_id, kind, granted, document_version, granted_at, COALESCE(host(ip), '')
FROM profile.consents
WHERE user_id = $1 AND kind = $2
ORDER BY granted_at DESC
LIMIT 1
`

const anonymizeConsentsQuery = `
UPDATE profile.consents SET user_id = $2 WHERE user_id = $1
`

const createStudyGoalQuery = `
INSERT INTO profile.study_goals (
    id, user_id, exam_id, exam_date, target_score, weekly_minutes, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
`

const getCurrentGoalQuery = `
SELECT id, user_id, exam_id, exam_date, target_score, weekly_minutes, created_at, updated_at
FROM profile.study_goals
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT 1
`

const deleteStudyGoalsQuery = `DELETE FROM profile.study_goals WHERE user_id = $1`

const getSettingsQuery = `
SELECT id, user_id, email_notifications, weekly_report, created_at, updated_at
FROM profile.settings
WHERE user_id = $1
`

const upsertSettingsQuery = `
INSERT INTO profile.settings (id, user_id, email_notifications, weekly_report, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $5)
ON CONFLICT (user_id) DO UPDATE SET
    email_notifications = EXCLUDED.email_notifications,
    weekly_report = EXCLUDED.weekly_report,
    updated_at = EXCLUDED.updated_at
`

const deleteSettingsQuery = `DELETE FROM profile.settings WHERE user_id = $1`

// PgRepository работает с PostgreSQL.
type PgRepository struct{}

// NewPgRepository создаёт репозиторий profile.
func NewPgRepository() *PgRepository {
	return &PgRepository{}
}

// GetProfile возвращает профиль пользователя.
func (r *PgRepository) GetProfile(ctx context.Context, q db.Querier, userID uuid.UUID) (*Profile, error) {
	var (
		profile  Profile
		contacts []byte
	)

	err := q.QueryRow(ctx, getProfileQuery, userID).Scan(
		&profile.ID,
		&profile.UserID,
		&profile.FirstName,
		&profile.LastName,
		&profile.BirthDate,
		&profile.Grade,
		&profile.Timezone,
		&contacts,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("профиль не найден")
	}
	if err != nil {
		return nil, fmt.Errorf("прочитать профиль: %w", err)
	}
	if len(contacts) > 0 {
		if err := json.Unmarshal(contacts, &profile.Contacts); err != nil {
			return nil, fmt.Errorf("разобрать контакты профиля: %w", err)
		}
	}

	return &profile, nil
}

// UpsertProfile создаёт или обновляет профиль.
func (r *PgRepository) UpsertProfile(ctx context.Context, q db.Querier, profile *Profile) error {
	contacts, err := json.Marshal(profile.Contacts)
	if err != nil {
		return fmt.Errorf("сериализовать контакты профиля: %w", err)
	}

	if _, err := q.Exec(ctx, upsertProfileQuery,
		profile.ID,
		profile.UserID,
		profile.FirstName,
		profile.LastName,
		profile.BirthDate,
		profile.Grade,
		profile.Timezone,
		string(contacts),
		profile.UpdatedAt,
	); err != nil {
		return fmt.Errorf("сохранить профиль: %w", err)
	}

	return nil
}

// DeleteProfile удаляет профиль.
func (r *PgRepository) DeleteProfile(ctx context.Context, q db.Querier, userID uuid.UUID) error {
	if _, err := q.Exec(ctx, deleteProfileQuery, userID); err != nil {
		return fmt.Errorf("удалить профиль: %w", err)
	}

	return nil
}

// CreateConsent сохраняет согласие.
func (r *PgRepository) CreateConsent(ctx context.Context, q db.Querier, consent *Consent) error {
	if _, err := q.Exec(ctx, createConsentQuery,
		consent.ID, consent.UserID, consent.Kind, consent.Granted, consent.DocumentVersion, consent.GrantedAt, consent.IP,
	); err != nil {
		return fmt.Errorf("сохранить согласие: %w", err)
	}

	return nil
}

// GetLatestConsent возвращает последнее согласие указанного вида.
func (r *PgRepository) GetLatestConsent(ctx context.Context, q db.Querier, userID uuid.UUID, kind string) (*Consent, error) {
	var consent Consent
	err := q.QueryRow(ctx, getLatestConsentQuery, userID, kind).Scan(
		&consent.ID,
		&consent.UserID,
		&consent.Kind,
		&consent.Granted,
		&consent.DocumentVersion,
		&consent.GrantedAt,
		&consent.IP,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("согласие не найдено")
	}
	if err != nil {
		return nil, fmt.Errorf("прочитать согласие: %w", err)
	}

	return &consent, nil
}

// AnonymizeConsents отвязывает согласия от пользователя.
func (r *PgRepository) AnonymizeConsents(ctx context.Context, q db.Querier, userID, anonymizedUserID uuid.UUID) error {
	if _, err := q.Exec(ctx, anonymizeConsentsQuery, userID, anonymizedUserID); err != nil {
		return fmt.Errorf("обезличить согласия: %w", err)
	}

	return nil
}

// CreateStudyGoal сохраняет новую учебную цель.
func (r *PgRepository) CreateStudyGoal(ctx context.Context, q db.Querier, goal *StudyGoal) error {
	if _, err := q.Exec(ctx, createStudyGoalQuery,
		goal.ID, goal.UserID, goal.ExamID, goal.ExamDate, goal.TargetScore, goal.WeeklyMinutes, goal.CreatedAt,
	); err != nil {
		return fmt.Errorf("сохранить учебную цель: %w", err)
	}

	return nil
}

// GetCurrentGoal возвращает последнюю учебную цель.
func (r *PgRepository) GetCurrentGoal(ctx context.Context, q db.Querier, userID uuid.UUID) (*StudyGoal, error) {
	var goal StudyGoal
	err := q.QueryRow(ctx, getCurrentGoalQuery, userID).Scan(
		&goal.ID,
		&goal.UserID,
		&goal.ExamID,
		&goal.ExamDate,
		&goal.TargetScore,
		&goal.WeeklyMinutes,
		&goal.CreatedAt,
		&goal.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("учебная цель не найдена")
	}
	if err != nil {
		return nil, fmt.Errorf("прочитать учебную цель: %w", err)
	}

	return &goal, nil
}

// DeleteStudyGoals удаляет цели пользователя.
func (r *PgRepository) DeleteStudyGoals(ctx context.Context, q db.Querier, userID uuid.UUID) error {
	if _, err := q.Exec(ctx, deleteStudyGoalsQuery, userID); err != nil {
		return fmt.Errorf("удалить учебные цели: %w", err)
	}

	return nil
}

// GetSettings возвращает настройки пользователя.
func (r *PgRepository) GetSettings(ctx context.Context, q db.Querier, userID uuid.UUID) (*Settings, error) {
	var settings Settings
	err := q.QueryRow(ctx, getSettingsQuery, userID).Scan(
		&settings.ID,
		&settings.UserID,
		&settings.EmailNotifications,
		&settings.WeeklyReport,
		&settings.CreatedAt,
		&settings.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("настройки не найдены")
	}
	if err != nil {
		return nil, fmt.Errorf("прочитать настройки: %w", err)
	}

	return &settings, nil
}

// UpsertSettings создаёт или обновляет настройки.
func (r *PgRepository) UpsertSettings(ctx context.Context, q db.Querier, settings *Settings) error {
	if _, err := q.Exec(ctx, upsertSettingsQuery,
		settings.ID, settings.UserID, settings.EmailNotifications, settings.WeeklyReport, settings.UpdatedAt,
	); err != nil {
		return fmt.Errorf("сохранить настройки: %w", err)
	}

	return nil
}

// DeleteSettings удаляет настройки пользователя.
func (r *PgRepository) DeleteSettings(ctx context.Context, q db.Querier, userID uuid.UUID) error {
	if _, err := q.Exec(ctx, deleteSettingsQuery, userID); err != nil {
		return fmt.Errorf("удалить настройки: %w", err)
	}

	return nil
}
