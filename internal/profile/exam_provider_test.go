package profile

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fakeExamProvider struct {
	available bool
	err       error
	calls     []uuid.UUID
}

func (f *fakeExamProvider) IsExamAvailable(_ context.Context, examID uuid.UUID) (bool, error) {
	f.calls = append(f.calls, examID)
	return f.available, f.err
}

func newProviderEnv(provider ExamProvider) (*Service, *fakeRepo) {
	repo := newFakeRepo()
	clk := &testClock{now: time.Date(2026, time.November, 26, 12, 0, 0, 0, time.UTC)}
	service := NewService(repo, nil, fakeTx{}, clk, &fakeAnonymizer{}, slog.New(slog.NewTextHandler(io.Discard, nil)), provider)

	return service, repo
}

func TestSaveGoalChecksExamAvailability(t *testing.T) {
	userID := uuid.Must(uuid.NewV7())
	examID := uuid.Must(uuid.NewV7())
	input := GoalInput{ExamID: examID.String(), ExamDate: "2027-06-01", TargetScore: 80, WeeklyMinutes: 300}

	t.Run("неопубликованный трек", func(t *testing.T) {
		provider := &fakeExamProvider{available: false}
		service, repo := newProviderEnv(provider)

		_, err := service.SaveGoal(context.Background(), userID, input)
		if !errors.Is(err, ErrExamUnavailable) {
			t.Fatalf("ошибка = %v", err)
		}
		if len(repo.goals[userID]) != 0 {
			t.Fatal("цель не должна сохраняться")
		}
		if len(provider.calls) != 1 || provider.calls[0] != examID {
			t.Fatalf("вызовы провайдера = %v", provider.calls)
		}
	})

	t.Run("ошибка провайдера не маскируется", func(t *testing.T) {
		provider := &fakeExamProvider{err: errors.New("БД недоступна")}
		service, repo := newProviderEnv(provider)

		_, err := service.SaveGoal(context.Background(), userID, input)
		if err == nil || errors.Is(err, ErrExamUnavailable) {
			t.Fatalf("ожидалась ошибка провайдера, получено %v", err)
		}
		if len(repo.goals[userID]) != 0 {
			t.Fatal("цель не должна сохраняться")
		}
	})

	t.Run("прежняя цель сохраняется", func(t *testing.T) {
		provider := &fakeExamProvider{available: true}
		service, repo := newProviderEnv(provider)

		first, err := service.SaveGoal(context.Background(), userID, input)
		if err != nil {
			t.Fatalf("первая цель: %v", err)
		}
		provider.available = false
		input.TargetScore = 90
		if _, err := service.SaveGoal(context.Background(), userID, input); !errors.Is(err, ErrExamUnavailable) {
			t.Fatalf("ошибка = %v", err)
		}

		current, err := service.GetCurrentGoal(context.Background(), userID)
		if err != nil {
			t.Fatalf("текущая цель: %v", err)
		}
		if current.ID != first.ID || current.TargetScore != first.TargetScore {
			t.Fatalf("прежняя цель изменилась: %+v", current)
		}
		if len(repo.goals[userID]) != 1 {
			t.Fatalf("история целей = %d", len(repo.goals[userID]))
		}
	})
}
