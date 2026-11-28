package learning

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"oge-ege/internal/content"
	"oge-ege/internal/profile"
	"oge-ege/internal/shared/clock"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestSetupGoalValidation(t *testing.T) {
	draft := &fakeContentProvider{exam: content.Exam{ID: uuid.New(), Code: "draft", Status: content.StatusDraft}}
	published := &fakeContentProvider{exam: content.Exam{ID: uuid.New(), Code: "published", Status: content.StatusPublished}}
	future := time.Now().UTC().AddDate(0, 0, 30).Format(dateLayout)

	tests := []struct {
		name    string
		content *fakeContentProvider
		input   GoalInput
		want    error
	}{
		{
			name:    "пустой трек",
			content: published,
			input:   GoalInput{ExamDate: future, TargetScore: 50, WeeklyMinutes: 300},
			want:    ErrInvalidExamCode,
		},
		{
			name:    "некорректная дата",
			content: published,
			input:   GoalInput{ExamCode: "published", ExamDate: "not-a-date", TargetScore: 50, WeeklyMinutes: 300},
			want:    ErrInvalidExamDate,
		},
		{
			name:    "прошлая дата",
			content: published,
			input:   GoalInput{ExamCode: "published", ExamDate: "2000-01-01", TargetScore: 50, WeeklyMinutes: 300},
			want:    ErrInvalidExamDate,
		},
		{
			name:    "отрицательный балл",
			content: published,
			input:   GoalInput{ExamCode: "published", ExamDate: future, TargetScore: -1, WeeklyMinutes: 300},
			want:    ErrInvalidTargetScore,
		},
		{
			name:    "балл больше 100",
			content: published,
			input:   GoalInput{ExamCode: "published", ExamDate: future, TargetScore: 101, WeeklyMinutes: 300},
			want:    ErrInvalidTargetScore,
		},
		{
			name:    "нулевое время",
			content: published,
			input:   GoalInput{ExamCode: "published", ExamDate: future, TargetScore: 50, WeeklyMinutes: 0},
			want:    ErrInvalidWeeklyMinutes,
		},
		{
			name:    "время больше недели",
			content: published,
			input:   GoalInput{ExamCode: "published", ExamDate: future, TargetScore: 50, WeeklyMinutes: 10081},
			want:    ErrInvalidWeeklyMinutes,
		},
		{
			name:    "неопубликованный трек",
			content: draft,
			input:   GoalInput{ExamCode: "draft", ExamDate: future, TargetScore: 50, WeeklyMinutes: 300},
			want:    ErrExamUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(nil, nil, nil, clock.New(), &fakeGoalProvider{}, tt.content, testLogger())
			if _, err := service.SetupGoal(context.Background(), uuid.New(), tt.input); !errors.Is(err, tt.want) {
				t.Fatalf("ошибка = %v, ожидалась %v", err, tt.want)
			}
		})
	}
}

func TestPlanViewWithoutGoal(t *testing.T) {
	service := NewService(nil, nil, nil, clock.New(), &fakeGoalProvider{}, &fakeContentProvider{}, testLogger())
	view, err := service.PlanView(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if view.HasGoal || !view.Empty || !strings.Contains(view.Reason, "цель") {
		t.Fatalf("пустое состояние плана = %+v", view)
	}
}

func TestPlanViewPastExamDate(t *testing.T) {
	goals := &fakeGoalProvider{goal: &profile.StudyGoal{
		ExamDate:    time.Now().UTC().AddDate(0, 0, -1),
		TargetScore: 50,
	}}
	service := NewService(nil, nil, nil, clock.New(), goals, &fakeContentProvider{}, testLogger())
	view, err := service.PlanView(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if !view.HasGoal || !view.Empty || !strings.Contains(view.Reason, "прошла") {
		t.Fatalf("состояние плана = %+v", view)
	}
}
