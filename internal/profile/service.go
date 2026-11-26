package profile

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"oge-ege/internal/auth"
	"oge-ege/internal/shared/clock"
	"oge-ege/internal/shared/db"
	"oge-ege/internal/shared/ids"
)

const dateLayout = "2006-01-02"

var (
	// ErrBirthDateRequired возвращается без даты рождения ученика.
	ErrBirthDateRequired = errors.New("дата рождения обязательна для ученика")
	// ErrInvalidBirthDate возвращается при некорректной дате рождения.
	ErrInvalidBirthDate = errors.New("некорректная дата рождения")
	// ErrInvalidTimezone возвращается при неизвестном часовом поясе.
	ErrInvalidTimezone = errors.New("некорректный часовой пояс")
	// ErrInvalidGrade возвращается при недопустимом классе.
	ErrInvalidGrade = errors.New("класс должен быть от 1 до 11")
	// ErrInvalidExamID возвращается при некорректном идентификаторе трека.
	ErrInvalidExamID = errors.New("некорректный идентификатор трека")
	// ErrExamUnavailable возвращается, когда трек не существует или не опубликован.
	ErrExamUnavailable = errors.New("трек не найден или не опубликован")
	// ErrInvalidExamDate возвращается при дате экзамена в прошлом.
	ErrInvalidExamDate = errors.New("дата экзамена должна быть в будущем")
	// ErrInvalidTargetScore возвращается при недопустимом целевом балле.
	ErrInvalidTargetScore = errors.New("целевой балл должен быть от 0 до 100")
	// ErrInvalidWeeklyMinutes возвращается при недопустимом недельном времени.
	ErrInvalidWeeklyMinutes = errors.New("недельное время должно быть от 1 до 10080 минут")
	// ErrNotFound возвращается, когда данные профиля отсутствуют.
	ErrNotFound = errors.New("данные не найдены")
)

// TxManager выполняет функцию в транзакции.
type TxManager interface {
	WithTx(ctx context.Context, fn func(q db.Querier) error) error
}

// AccountAnonymizer обезличивает учётную запись в модуле auth.
type AccountAnonymizer interface {
	AnonymizeAccount(ctx context.Context, userID uuid.UUID) error
}

// ExamProvider проверяет доступность учебного трека.
type ExamProvider interface {
	IsExamAvailable(ctx context.Context, examID uuid.UUID) (bool, error)
}

// ProfileInput содержит данные для сохранения профиля.
type ProfileInput struct {
	FirstName string
	LastName  string
	BirthDate string
	Grade     *int
	Timezone  string
	Contacts  map[string]any
}

// GoalInput содержит данные учебной цели.
type GoalInput struct {
	ExamID        string
	ExamDate      string
	TargetScore   int
	WeeklyMinutes int
}

// Service содержит сценарии модуля profile.
type Service struct {
	repo         Repository
	db           db.Querier
	tx           TxManager
	clock        clock.Clock
	anonymizer   AccountAnonymizer
	examProvider ExamProvider
	logger       *slog.Logger
}

// NewService создаёт сервис profile.
func NewService(repo Repository, database db.Querier, tx TxManager, clk clock.Clock, anonymizer AccountAnonymizer, logger *slog.Logger, examProviders ...ExamProvider) *Service {
	service := &Service{
		repo:       repo,
		db:         database,
		tx:         tx,
		clock:      clk,
		anonymizer: anonymizer,
		logger:     logger,
	}
	if len(examProviders) > 0 {
		service.examProvider = examProviders[0]
	}

	return service
}

// GetProfile возвращает профиль пользователя.
func (s *Service) GetProfile(ctx context.Context, userID uuid.UUID) (*Profile, error) {
	profile, err := s.repo.GetProfile(ctx, s.db, userID)
	if err != nil {
		return nil, ErrNotFound
	}

	return profile, nil
}

// SaveProfile проверяет и сохраняет профиль.
func (s *Service) SaveProfile(ctx context.Context, userID uuid.UUID, role auth.Role, in ProfileInput) (*Profile, error) {
	firstName := strings.TrimSpace(in.FirstName)
	lastName := strings.TrimSpace(in.LastName)
	if len([]rune(firstName)) > 100 || len([]rune(lastName)) > 100 {
		return nil, fmt.Errorf("имя и фамилия не должны превышать 100 символов")
	}

	var birthDate *time.Time
	if strings.TrimSpace(in.BirthDate) != "" {
		parsed, err := time.Parse(dateLayout, strings.TrimSpace(in.BirthDate))
		if err != nil {
			return nil, ErrInvalidBirthDate
		}
		now := s.clock.Now()
		if parsed.After(time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)) {
			return nil, ErrInvalidBirthDate
		}
		birthDate = &parsed
	} else if role == auth.RoleStudent {
		return nil, ErrBirthDateRequired
	}

	if in.Grade != nil && (*in.Grade < 1 || *in.Grade > 11) {
		return nil, ErrInvalidGrade
	}

	timezone := strings.TrimSpace(in.Timezone)
	if timezone == "" {
		timezone = "UTC"
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		return nil, ErrInvalidTimezone
	}

	now := s.clock.Now()
	id, err := ids.New()
	if err != nil {
		return nil, err
	}

	contacts := in.Contacts
	if contacts == nil {
		contacts = map[string]any{}
	}

	profile := &Profile{
		ID:        id,
		UserID:    userID,
		FirstName: firstName,
		LastName:  lastName,
		BirthDate: birthDate,
		Grade:     in.Grade,
		Timezone:  timezone,
		Contacts:  contacts,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.UpsertProfile(ctx, s.db, profile); err != nil {
		return nil, err
	}

	return s.repo.GetProfile(ctx, s.db, userID)
}

// RecordRegistrationConsents сохраняет согласия, предоставленные при регистрации.
func (s *Service) RecordRegistrationConsents(ctx context.Context, userID uuid.UUID, grants []auth.ConsentGrant, ip string) error {
	now := s.clock.Now()

	return s.tx.WithTx(ctx, func(q db.Querier) error {
		for _, grant := range grants {
			id, err := ids.New()
			if err != nil {
				return err
			}
			version := grant.DocumentVersion
			if version == "" {
				version = auth.DefaultConsentVersion
			}

			if err := s.repo.CreateConsent(ctx, q, &Consent{
				ID:              id,
				UserID:          userID,
				Kind:            grant.Kind,
				Granted:         grant.Granted,
				DocumentVersion: version,
				GrantedAt:       now,
				IP:              ip,
			}); err != nil {
				return err
			}
		}

		return nil
	})
}

// HasRequiredConsent проверяет наличие действующего согласия указанного вида.
func (s *Service) HasRequiredConsent(ctx context.Context, userID uuid.UUID, kind, version string) (bool, error) {
	consent, err := s.repo.GetLatestConsent(ctx, s.db, userID, kind)
	if err != nil {
		return false, nil
	}

	return consent.Granted && consent.DocumentVersion == version, nil
}

// Age рассчитывает возраст на указанную дату.
func Age(birthDate, now time.Time) int {
	birth := time.Date(birthDate.Year(), birthDate.Month(), birthDate.Day(), 0, 0, 0, 0, time.UTC)
	current := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	years := current.Year() - birth.Year()
	if current.Month() < birth.Month() || (current.Month() == birth.Month() && current.Day() < birth.Day()) {
		years--
	}

	return years
}

// IsMinor сообщает, является ли пользователь несовершеннолетним.
func IsMinor(birthDate, now time.Time) bool {
	return Age(birthDate, now) < 18
}

// CanUsePaidLearning проверяет возраст и согласие законного представителя.
func (s *Service) CanUsePaidLearning(ctx context.Context, userID uuid.UUID) error {
	profile, err := s.repo.GetProfile(ctx, s.db, userID)
	if err != nil || profile.BirthDate == nil {
		return ErrBirthDateRequired
	}
	if !IsMinor(*profile.BirthDate, s.clock.Now()) {
		return nil
	}

	ok, err := s.HasRequiredConsent(ctx, userID, auth.ConsentLegalRepresentative, auth.DefaultConsentVersion)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("требуется согласие законного представителя")
	}

	return nil
}

// GetCurrentGoal возвращает текущую учебную цель.
func (s *Service) GetCurrentGoal(ctx context.Context, userID uuid.UUID) (*StudyGoal, error) {
	goal, err := s.repo.GetCurrentGoal(ctx, s.db, userID)
	if err != nil {
		return nil, ErrNotFound
	}

	return goal, nil
}

// SaveGoal проверяет и сохраняет новую учебную цель, сохраняя историю.
func (s *Service) SaveGoal(ctx context.Context, userID uuid.UUID, in GoalInput) (*StudyGoal, error) {
	examID, err := uuid.Parse(strings.TrimSpace(in.ExamID))
	if err != nil {
		return nil, ErrInvalidExamID
	}
	if s.examProvider != nil {
		available, err := s.examProvider.IsExamAvailable(ctx, examID)
		if err != nil {
			return nil, fmt.Errorf("проверить трек: %w", err)
		}
		if !available {
			return nil, ErrExamUnavailable
		}
	}

	examDate, err := time.Parse(dateLayout, strings.TrimSpace(in.ExamDate))
	if err != nil {
		return nil, ErrInvalidExamDate
	}
	now := s.clock.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if examDate.Before(today) {
		return nil, ErrInvalidExamDate
	}
	if in.TargetScore < 0 || in.TargetScore > 100 {
		return nil, ErrInvalidTargetScore
	}
	if in.WeeklyMinutes < 1 || in.WeeklyMinutes > 10080 {
		return nil, ErrInvalidWeeklyMinutes
	}

	id, err := ids.New()
	if err != nil {
		return nil, err
	}

	goal := &StudyGoal{
		ID:            id,
		UserID:        userID,
		ExamID:        examID,
		ExamDate:      examDate,
		TargetScore:   in.TargetScore,
		WeeklyMinutes: in.WeeklyMinutes,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.repo.CreateStudyGoal(ctx, s.db, goal); err != nil {
		return nil, err
	}

	return goal, nil
}

// GetSettings возвращает настройки; при отсутствии записи — значения по умолчанию.
func (s *Service) GetSettings(ctx context.Context, userID uuid.UUID) (*Settings, error) {
	settings, err := s.repo.GetSettings(ctx, s.db, userID)
	if err != nil {
		return &Settings{
			UserID:             userID,
			EmailNotifications: true,
			WeeklyReport:       true,
		}, nil
	}

	return settings, nil
}

// SaveSettings сохраняет настройки уведомлений.
func (s *Service) SaveSettings(ctx context.Context, userID uuid.UUID, emailNotifications, weeklyReport bool) (*Settings, error) {
	now := s.clock.Now()
	id, err := ids.New()
	if err != nil {
		return nil, err
	}

	settings := &Settings{
		ID:                 id,
		UserID:             userID,
		EmailNotifications: emailNotifications,
		WeeklyReport:       weeklyReport,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if err := s.repo.UpsertSettings(ctx, s.db, settings); err != nil {
		return nil, err
	}

	return settings, nil
}

// DeleteAccount обезличивает учётную запись и очищает профиль.
func (s *Service) DeleteAccount(ctx context.Context, userID uuid.UUID) error {
	if s.anonymizer == nil {
		return errors.New("обезличивание учётной записи не настроено")
	}
	if err := s.anonymizer.AnonymizeAccount(ctx, userID); err != nil {
		return fmt.Errorf("обезличить учётную запись: %w", err)
	}

	anonymizedID, err := ids.New()
	if err != nil {
		return err
	}

	return s.tx.WithTx(ctx, func(q db.Querier) error {
		if err := s.repo.AnonymizeConsents(ctx, q, userID, anonymizedID); err != nil {
			return err
		}
		if err := s.repo.DeleteProfile(ctx, q, userID); err != nil {
			return err
		}
		if err := s.repo.DeleteStudyGoals(ctx, q, userID); err != nil {
			return err
		}

		return s.repo.DeleteSettings(ctx, q, userID)
	})
}
