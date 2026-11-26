package profile

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"oge-ege/internal/auth"
	"oge-ege/internal/shared/db"
)

type testClock struct {
	now time.Time
}

func (c *testClock) Now() time.Time { return c.now }

type fakeTx struct{}

func (fakeTx) WithTx(_ context.Context, fn func(q db.Querier) error) error {
	return fn(nil)
}

type fakeRepo struct {
	profiles map[uuid.UUID]*Profile
	consents map[uuid.UUID][]*Consent
	goals    map[uuid.UUID][]*StudyGoal
	settings map[uuid.UUID]*Settings
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		profiles: make(map[uuid.UUID]*Profile),
		consents: make(map[uuid.UUID][]*Consent),
		goals:    make(map[uuid.UUID][]*StudyGoal),
		settings: make(map[uuid.UUID]*Settings),
	}
}

func (f *fakeRepo) GetProfile(_ context.Context, _ db.Querier, userID uuid.UUID) (*Profile, error) {
	profile, ok := f.profiles[userID]
	if !ok {
		return nil, errors.New("профиль не найден")
	}
	return profile, nil
}

func (f *fakeRepo) UpsertProfile(_ context.Context, _ db.Querier, profile *Profile) error {
	copied := *profile
	f.profiles[profile.UserID] = &copied
	return nil
}

func (f *fakeRepo) DeleteProfile(_ context.Context, _ db.Querier, userID uuid.UUID) error {
	delete(f.profiles, userID)
	return nil
}

func (f *fakeRepo) CreateConsent(_ context.Context, _ db.Querier, consent *Consent) error {
	copied := *consent
	f.consents[consent.UserID] = append(f.consents[consent.UserID], &copied)
	return nil
}

func (f *fakeRepo) GetLatestConsent(_ context.Context, _ db.Querier, userID uuid.UUID, kind string) (*Consent, error) {
	var latest *Consent
	for _, consent := range f.consents[userID] {
		if consent.Kind == kind {
			if latest == nil || !consent.GrantedAt.Before(latest.GrantedAt) {
				latest = consent
			}
		}
	}
	if latest == nil {
		return nil, errors.New("согласие не найдено")
	}
	return latest, nil
}

func (f *fakeRepo) AnonymizeConsents(_ context.Context, _ db.Querier, userID, anonymizedUserID uuid.UUID) error {
	consents := f.consents[userID]
	delete(f.consents, userID)
	f.consents[anonymizedUserID] = consents
	return nil
}

func (f *fakeRepo) CreateStudyGoal(_ context.Context, _ db.Querier, goal *StudyGoal) error {
	copied := *goal
	f.goals[goal.UserID] = append(f.goals[goal.UserID], &copied)
	return nil
}

func (f *fakeRepo) GetCurrentGoal(_ context.Context, _ db.Querier, userID uuid.UUID) (*StudyGoal, error) {
	goals := f.goals[userID]
	if len(goals) == 0 {
		return nil, errors.New("цель не найдена")
	}
	return goals[len(goals)-1], nil
}

func (f *fakeRepo) DeleteStudyGoals(_ context.Context, _ db.Querier, userID uuid.UUID) error {
	delete(f.goals, userID)
	return nil
}

func (f *fakeRepo) GetSettings(_ context.Context, _ db.Querier, userID uuid.UUID) (*Settings, error) {
	settings, ok := f.settings[userID]
	if !ok {
		return nil, errors.New("настройки не найдены")
	}
	return settings, nil
}

func (f *fakeRepo) UpsertSettings(_ context.Context, _ db.Querier, settings *Settings) error {
	copied := *settings
	f.settings[settings.UserID] = &copied
	return nil
}

func (f *fakeRepo) DeleteSettings(_ context.Context, _ db.Querier, userID uuid.UUID) error {
	delete(f.settings, userID)
	return nil
}

type fakeAnonymizer struct {
	calls []uuid.UUID
	err   error
}

func (f *fakeAnonymizer) AnonymizeAccount(_ context.Context, userID uuid.UUID) error {
	f.calls = append(f.calls, userID)
	return f.err
}

type testEnv struct {
	service    *Service
	repo       *fakeRepo
	anonymizer *fakeAnonymizer
	clock      *testClock
}

func newTestEnv() *testEnv {
	repo := newFakeRepo()
	anonymizer := &fakeAnonymizer{}
	clk := &testClock{now: time.Date(2026, time.November, 26, 12, 0, 0, 0, time.UTC)}
	service := NewService(repo, nil, fakeTx{}, clk, anonymizer, slog.New(slog.NewTextHandler(io.Discard, nil)))

	return &testEnv{service: service, repo: repo, anonymizer: anonymizer, clock: clk}
}

func TestAge(t *testing.T) {
	tests := []struct {
		name      string
		birthDate string
		now       string
		wantAge   int
		wantMinor bool
	}{
		{name: "ровно 18", birthDate: "2008-11-26", now: "2026-11-26", wantAge: 18},
		{name: "17 до дня рождения", birthDate: "2008-11-27", now: "2026-11-26", wantAge: 17, wantMinor: true},
		{name: "17 лет", birthDate: "2009-01-01", now: "2026-11-26", wantAge: 17, wantMinor: true},
		{name: "18 после дня рождения", birthDate: "2008-01-01", now: "2026-11-26", wantAge: 18},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			birth, _ := time.Parse(dateLayout, tt.birthDate)
			now, _ := time.Parse(dateLayout, tt.now)
			if got := Age(birth, now); got != tt.wantAge {
				t.Fatalf("возраст = %d, ожидался %d", got, tt.wantAge)
			}
			if got := IsMinor(birth, now); got != tt.wantMinor {
				t.Fatalf("несовершеннолетний = %t", got)
			}
		})
	}
}

func TestSaveProfile(t *testing.T) {
	userID := uuid.Must(uuid.NewV7())
	grade := 9

	t.Run("без даты рождения для ученика", func(t *testing.T) {
		env := newTestEnv()
		_, err := env.service.SaveProfile(context.Background(), userID, auth.RoleStudent, ProfileInput{
			FirstName: "Иван",
			LastName:  "Петров",
		})
		if !errors.Is(err, ErrBirthDateRequired) {
			t.Fatalf("ошибка = %v", err)
		}
	})

	t.Run("некорректный часовой пояс", func(t *testing.T) {
		env := newTestEnv()
		_, err := env.service.SaveProfile(context.Background(), userID, auth.RoleStudent, ProfileInput{
			FirstName: "Иван",
			LastName:  "Петров",
			BirthDate: "2010-05-01",
			Timezone:  "Mars/Base",
		})
		if !errors.Is(err, ErrInvalidTimezone) {
			t.Fatalf("ошибка = %v", err)
		}
	})

	t.Run("некорректный класс", func(t *testing.T) {
		env := newTestEnv()
		invalidGrade := 12
		_, err := env.service.SaveProfile(context.Background(), userID, auth.RoleStudent, ProfileInput{
			FirstName: "Иван",
			LastName:  "Петров",
			BirthDate: "2010-05-01",
			Grade:     &invalidGrade,
		})
		if !errors.Is(err, ErrInvalidGrade) {
			t.Fatalf("ошибка = %v", err)
		}
	})

	t.Run("успех", func(t *testing.T) {
		env := newTestEnv()
		profile, err := env.service.SaveProfile(context.Background(), userID, auth.RoleStudent, ProfileInput{
			FirstName: " Иван ",
			LastName:  "Петров",
			BirthDate: "2010-05-01",
			Grade:     &grade,
			Timezone:  "Europe/Moscow",
			Contacts:  map[string]any{"phone": "+79990000000"},
		})
		if err != nil {
			t.Fatalf("неожиданная ошибка: %v", err)
		}
		if profile.FirstName != "Иван" || profile.Timezone != "Europe/Moscow" {
			t.Fatalf("профиль = %+v", profile)
		}

		stored, err := env.service.GetProfile(context.Background(), userID)
		if err != nil {
			t.Fatalf("чтение профиля: %v", err)
		}
		if stored.BirthDate == nil {
			t.Fatal("дата рождения не сохранена")
		}
	})
}

func TestConsents(t *testing.T) {
	env := newTestEnv()
	userID := uuid.Must(uuid.NewV7())

	err := env.service.RecordRegistrationConsents(context.Background(), userID, []auth.ConsentGrant{
		{Kind: auth.ConsentPersonalData, Granted: true},
		{Kind: auth.ConsentLegalRepresentative, Granted: false},
	}, "127.0.0.1")
	if err != nil {
		t.Fatalf("сохранение согласий: %v", err)
	}

	stored := env.repo.consents[userID]
	if len(stored) != 2 {
		t.Fatalf("согласий = %d", len(stored))
	}
	if stored[0].DocumentVersion != auth.DefaultConsentVersion || stored[0].IP != "127.0.0.1" {
		t.Fatalf("согласие = %+v", stored[0])
	}

	ok, err := env.service.HasRequiredConsent(context.Background(), userID, auth.ConsentPersonalData, auth.DefaultConsentVersion)
	if err != nil || !ok {
		t.Fatalf("обязательное согласие не найдено: %v", err)
	}

	ok, err = env.service.HasRequiredConsent(context.Background(), userID, auth.ConsentPersonalData, "2")
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if ok {
		t.Fatal("согласие старой версии принято для новой версии")
	}
}

func TestCanUsePaidLearning(t *testing.T) {
	env := newTestEnv()
	userID := uuid.Must(uuid.NewV7())

	if _, err := env.service.SaveProfile(context.Background(), userID, auth.RoleStudent, ProfileInput{
		FirstName: "Иван",
		LastName:  "Петров",
		BirthDate: "2010-05-01",
	}); err != nil {
		t.Fatalf("профиль: %v", err)
	}

	if err := env.service.CanUsePaidLearning(context.Background(), userID); err == nil {
		t.Fatal("несовершеннолетний допущен без согласия представителя")
	}

	if err := env.service.RecordRegistrationConsents(context.Background(), userID, []auth.ConsentGrant{
		{Kind: auth.ConsentLegalRepresentative, Granted: true},
	}, "127.0.0.1"); err != nil {
		t.Fatalf("согласие: %v", err)
	}

	if err := env.service.CanUsePaidLearning(context.Background(), userID); err != nil {
		t.Fatalf("согласие представителя не учтено: %v", err)
	}
}

func TestSaveGoal(t *testing.T) {
	env := newTestEnv()
	userID := uuid.Must(uuid.NewV7())
	examID := uuid.Must(uuid.NewV7())

	t.Run("некорректная дата", func(t *testing.T) {
		_, err := env.service.SaveGoal(context.Background(), userID, GoalInput{
			ExamID:        examID.String(),
			ExamDate:      "2020-01-01",
			TargetScore:   80,
			WeeklyMinutes: 300,
		})
		if !errors.Is(err, ErrInvalidExamDate) {
			t.Fatalf("ошибка = %v", err)
		}
	})

	t.Run("некорректный балл", func(t *testing.T) {
		_, err := env.service.SaveGoal(context.Background(), userID, GoalInput{
			ExamID:        examID.String(),
			ExamDate:      "2027-06-01",
			TargetScore:   101,
			WeeklyMinutes: 300,
		})
		if !errors.Is(err, ErrInvalidTargetScore) {
			t.Fatalf("ошибка = %v", err)
		}
	})

	t.Run("сохранение и история", func(t *testing.T) {
		first, err := env.service.SaveGoal(context.Background(), userID, GoalInput{
			ExamID:        examID.String(),
			ExamDate:      "2027-06-01",
			TargetScore:   80,
			WeeklyMinutes: 300,
		})
		if err != nil {
			t.Fatalf("первая цель: %v", err)
		}
		if first.TargetScore != 80 {
			t.Fatalf("балл = %d", first.TargetScore)
		}

		second, err := env.service.SaveGoal(context.Background(), userID, GoalInput{
			ExamID:        examID.String(),
			ExamDate:      "2027-06-01",
			TargetScore:   90,
			WeeklyMinutes: 400,
		})
		if err != nil {
			t.Fatalf("вторая цель: %v", err)
		}
		if second.TargetScore != 90 {
			t.Fatalf("балл = %d", second.TargetScore)
		}

		if len(env.repo.goals[userID]) != 2 {
			t.Fatalf("история целей = %d", len(env.repo.goals[userID]))
		}

		current, err := env.service.GetCurrentGoal(context.Background(), userID)
		if err != nil {
			t.Fatalf("текущая цель: %v", err)
		}
		if current.TargetScore != 90 {
			t.Fatalf("текущий балл = %d", current.TargetScore)
		}
	})
}

func TestSettings(t *testing.T) {
	env := newTestEnv()
	userID := uuid.Must(uuid.NewV7())

	settings, err := env.service.GetSettings(context.Background(), userID)
	if err != nil {
		t.Fatalf("настройки по умолчанию: %v", err)
	}
	if !settings.EmailNotifications || !settings.WeeklyReport {
		t.Fatal("некорректные значения по умолчанию")
	}

	saved, err := env.service.SaveSettings(context.Background(), userID, false, true)
	if err != nil {
		t.Fatalf("сохранение настроек: %v", err)
	}
	if saved.EmailNotifications {
		t.Fatal("email-уведомления не отключены")
	}

	loaded, err := env.service.GetSettings(context.Background(), userID)
	if err != nil {
		t.Fatalf("чтение настроек: %v", err)
	}
	if loaded.EmailNotifications || !loaded.WeeklyReport {
		t.Fatalf("настройки = %+v", loaded)
	}
}

func TestDeleteAccount(t *testing.T) {
	env := newTestEnv()
	userID := uuid.Must(uuid.NewV7())

	if _, err := env.service.SaveProfile(context.Background(), userID, auth.RoleStudent, ProfileInput{
		FirstName: "Иван",
		LastName:  "Петров",
		BirthDate: "2010-05-01",
	}); err != nil {
		t.Fatalf("профиль: %v", err)
	}
	if err := env.service.RecordRegistrationConsents(context.Background(), userID, []auth.ConsentGrant{
		{Kind: auth.ConsentPersonalData, Granted: true},
	}, "127.0.0.1"); err != nil {
		t.Fatalf("согласия: %v", err)
	}
	if _, err := env.service.SaveSettings(context.Background(), userID, true, true); err != nil {
		t.Fatalf("настройки: %v", err)
	}

	if err := env.service.DeleteAccount(context.Background(), userID); err != nil {
		t.Fatalf("удаление аккаунта: %v", err)
	}

	if len(env.anonymizer.calls) != 1 || env.anonymizer.calls[0] != userID {
		t.Fatalf("обезличивание auth не вызвано: %v", env.anonymizer.calls)
	}
	if _, ok := env.repo.profiles[userID]; ok {
		t.Fatal("профиль не удалён")
	}
	if _, ok := env.repo.settings[userID]; ok {
		t.Fatal("настройки не удалены")
	}
	if len(env.repo.consents[userID]) != 0 {
		t.Fatal("согласия не обезличены")
	}

	found := false
	for _, consents := range env.repo.consents {
		if len(consents) > 0 {
			found = true
		}
	}
	if !found {
		t.Fatal("обезличенные согласия потеряны")
	}
}

func TestSaveProfileRejectsInvalidDate(t *testing.T) {
	env := newTestEnv()
	_, err := env.service.SaveProfile(context.Background(), uuid.Must(uuid.NewV7()), auth.RoleStudent, ProfileInput{
		FirstName: "Иван",
		LastName:  "Петров",
		BirthDate: "01.05.2010",
	})
	if !errors.Is(err, ErrInvalidBirthDate) {
		t.Fatalf("ошибка = %v", err)
	}
}

func TestProfileForAnotherUserIsIsolated(t *testing.T) {
	env := newTestEnv()
	first := uuid.Must(uuid.NewV7())
	second := uuid.Must(uuid.NewV7())

	if _, err := env.service.SaveProfile(context.Background(), first, auth.RoleStudent, ProfileInput{
		FirstName: "Первый",
		LastName:  "Ученик",
		BirthDate: "2010-05-01",
	}); err != nil {
		t.Fatalf("профиль: %v", err)
	}

	if _, err := env.service.GetProfile(context.Background(), second); !errors.Is(err, ErrNotFound) {
		t.Fatalf("чужой профиль доступен: %v", err)
	}

	profile, err := env.service.GetProfile(context.Background(), first)
	if err != nil {
		t.Fatalf("свой профиль: %v", err)
	}
	if !strings.HasPrefix(profile.FirstName, "Первый") {
		t.Fatalf("профиль = %+v", profile)
	}
}
