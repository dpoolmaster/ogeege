package profile

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"oge-ege/internal/shared/db"
)

func TestPgRepositoryIntegration(t *testing.T) {
	databaseURL, ok := os.LookupEnv("TEST_DATABASE_URL")
	if !ok || databaseURL == "" {
		t.Skip("TEST_DATABASE_URL не задан")
	}

	ctx := context.Background()
	pool, err := db.Open(ctx, databaseURL, 0, 2)
	if err != nil {
		t.Fatalf("открыть пул: %v", err)
	}
	defer pool.Close()

	repo := NewPgRepository()
	userID := uuid.Must(uuid.NewV7())
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM profile.settings WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM profile.study_goals WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM profile.consents WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM profile.profiles WHERE user_id = $1`, userID)
	})

	now := time.Now()
	birthDate := time.Date(2010, 5, 1, 0, 0, 0, 0, time.UTC)
	grade := 9
	profile := &Profile{
		ID:        uuid.Must(uuid.NewV7()),
		UserID:    userID,
		FirstName: "Иван",
		LastName:  "Петров",
		BirthDate: &birthDate,
		Grade:     &grade,
		Timezone:  "Europe/Moscow",
		Contacts:  map[string]any{"phone": "+79990000000"},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := repo.UpsertProfile(ctx, pool, profile); err != nil {
		t.Fatalf("создать профиль: %v", err)
	}

	profile.FirstName = "Пётр"
	if err := repo.UpsertProfile(ctx, pool, profile); err != nil {
		t.Fatalf("обновить профиль: %v", err)
	}

	stored, err := repo.GetProfile(ctx, pool, userID)
	if err != nil {
		t.Fatalf("прочитать профиль: %v", err)
	}
	if stored.FirstName != "Пётр" || stored.BirthDate == nil {
		t.Fatalf("профиль = %+v", stored)
	}

	consent := &Consent{
		ID:              uuid.Must(uuid.NewV7()),
		UserID:          userID,
		Kind:            "personal_data",
		Granted:         true,
		DocumentVersion: "1",
		GrantedAt:       now,
		IP:              "127.0.0.1",
	}
	if err := repo.CreateConsent(ctx, pool, consent); err != nil {
		t.Fatalf("создать согласие: %v", err)
	}
	latest, err := repo.GetLatestConsent(ctx, pool, userID, "personal_data")
	if err != nil {
		t.Fatalf("прочитать согласие: %v", err)
	}
	if !latest.Granted || latest.IP != "127.0.0.1" {
		t.Fatalf("согласие = %+v", latest)
	}

	goal := &StudyGoal{
		ID:            uuid.Must(uuid.NewV7()),
		UserID:        userID,
		ExamID:        uuid.Must(uuid.NewV7()),
		ExamDate:      time.Date(2027, 6, 1, 0, 0, 0, 0, time.UTC),
		TargetScore:   80,
		WeeklyMinutes: 300,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := repo.CreateStudyGoal(ctx, pool, goal); err != nil {
		t.Fatalf("создать цель: %v", err)
	}
	current, err := repo.GetCurrentGoal(ctx, pool, userID)
	if err != nil {
		t.Fatalf("прочитать цель: %v", err)
	}
	if current.TargetScore != 80 {
		t.Fatalf("цель = %+v", current)
	}

	settings := &Settings{
		ID:                 uuid.Must(uuid.NewV7()),
		UserID:             userID,
		EmailNotifications: false,
		WeeklyReport:       true,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := repo.UpsertSettings(ctx, pool, settings); err != nil {
		t.Fatalf("создать настройки: %v", err)
	}
	storedSettings, err := repo.GetSettings(ctx, pool, userID)
	if err != nil {
		t.Fatalf("прочитать настройки: %v", err)
	}
	if storedSettings.EmailNotifications {
		t.Fatalf("настройки = %+v", storedSettings)
	}

	if err := repo.DeleteProfile(ctx, pool, userID); err != nil {
		t.Fatalf("удалить профиль: %v", err)
	}
	if _, err := repo.GetProfile(ctx, pool, userID); err == nil {
		t.Fatal("профиль не удалён")
	}
}
