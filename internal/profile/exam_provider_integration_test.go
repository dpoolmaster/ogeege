package profile

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"oge-ege/internal/content"
	"oge-ege/internal/shared/clock"
	"oge-ege/internal/shared/db"
)

type txOnlyManager struct {
	tx pgx.Tx
}

func (m txOnlyManager) WithTx(_ context.Context, fn func(q db.Querier) error) error {
	return fn(m.tx)
}

func TestSaveGoalIntegrationWithContent(t *testing.T) {
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

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("начать транзакцию: %v", err)
	}
	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	contentService := content.NewService(content.NewPgRepository(), tx, txOnlyManager{tx: tx}, clock.New(), nil, logger)
	profileService := NewService(NewPgRepository(), tx, txOnlyManager{tx: tx}, clock.New(), nil, logger, contentService)

	exam, err := contentService.CreateExam(ctx, content.ExamInput{
		Code: "it_goal_" + uuid.Must(uuid.NewV7()).String(), Title: "Трек", Subject: "math",
	})
	if err != nil {
		t.Fatalf("создать трек: %v", err)
	}

	userID := uuid.Must(uuid.NewV7())
	input := GoalInput{ExamID: exam.ID.String(), ExamDate: "2030-06-01", TargetScore: 80, WeeklyMinutes: 300}

	if _, err := profileService.SaveGoal(ctx, userID, input); !errors.Is(err, ErrExamUnavailable) {
		t.Fatalf("черновик трека: ожидалась ошибка, получено %v", err)
	}

	if _, err := contentService.PublishExam(ctx, exam.ID); err != nil {
		t.Fatalf("опубликовать трек: %v", err)
	}
	goal, err := profileService.SaveGoal(ctx, userID, input)
	if err != nil {
		t.Fatalf("сохранить цель: %v", err)
	}
	if goal.ExamID != exam.ID {
		t.Fatalf("exam_id = %s", goal.ExamID)
	}
}
