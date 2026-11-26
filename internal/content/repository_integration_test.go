package content

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"oge-ege/internal/shared/db"
)

func TestRepositoryUniquenessIntegration(t *testing.T) {
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
	now := time.Now().UTC()
	exam := &Exam{
		ID: uuid.Must(uuid.NewV7()), Code: "repo_it_" + uuid.Must(uuid.NewV7()).String(),
		Title: "Тест", Subject: "math", Status: StatusDraft, CreatedAt: now, UpdatedAt: now,
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM content.topics WHERE exam_id = $1`, exam.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM content.kim_numbers WHERE exam_year_id IN (SELECT id FROM content.exam_years WHERE exam_id = $1)`, exam.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM content.exam_years WHERE exam_id = $1`, exam.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM content.exams WHERE id = $1`, exam.ID)
	})

	if err := repo.CreateExam(ctx, pool, exam); err != nil {
		t.Fatalf("создать трек: %v", err)
	}
	if err := repo.CreateExam(ctx, pool, exam); !errors.Is(err, ErrConflict) {
		t.Fatalf("дубликат кода трека: ожидалась ошибка конфликта, получено %v", err)
	}
	if found, err := repo.GetExamByCode(ctx, pool, exam.Code); err != nil || found.ID != exam.ID {
		t.Fatalf("найти трек по коду: %v, %+v", err, found)
	}

	year := &ExamYear{
		ID: uuid.Must(uuid.NewV7()), ExamID: exam.ID, Year: 2026,
		DurationMinutes: 235, Status: StatusDraft, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateExamYear(ctx, pool, year); err != nil {
		t.Fatalf("создать учебный год: %v", err)
	}
	if err := repo.CreateExamYear(ctx, pool, year); !errors.Is(err, ErrConflict) {
		t.Fatalf("дубликат учебного года: ожидалась ошибка конфликта, получено %v", err)
	}

	kim := &KimNumber{
		ID: uuid.Must(uuid.NewV7()), ExamYearID: year.ID, Number: 1, Part: 1,
		AnswerType: AnswerTypeShort, MaxPrimaryScore: 1, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateKimNumber(ctx, pool, kim); err != nil {
		t.Fatalf("создать номер КИМ: %v", err)
	}
	if err := repo.CreateKimNumber(ctx, pool, kim); !errors.Is(err, ErrConflict) {
		t.Fatalf("дубликат номера КИМ: ожидалась ошибка конфликта, получено %v", err)
	}

	topic := &Topic{
		ID: uuid.Must(uuid.NewV7()), ExamID: exam.ID, Code: "numbers", Title: "Числа",
		OrderIndex: 1, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateTopic(ctx, pool, topic); err != nil {
		t.Fatalf("создать тему: %v", err)
	}
	if err := repo.CreateTopic(ctx, pool, topic); !errors.Is(err, ErrConflict) {
		t.Fatalf("дубликат кода темы: ожидалась ошибка конфликта, получено %v", err)
	}
}
