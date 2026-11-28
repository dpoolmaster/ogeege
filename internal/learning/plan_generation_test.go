package learning

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestBuildWeekBlueprintsSingleWeek(t *testing.T) {
	current := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	exam := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)

	weeks := BuildWeekBlueprints(current, exam, 600, nil, false)
	if len(weeks) != 1 {
		t.Fatalf("недель = %d, ожидалось 1", len(weeks))
	}
	if !weeks[0].WeekStart.Equal(time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("начало недели = %s", weeks[0].WeekStart)
	}
	if weeks[0].Phase != PhaseDiagnostics {
		t.Fatalf("фаза = %q", weeks[0].Phase)
	}
	if len(weeks[0].Items) != 7 {
		t.Fatalf("пунктов = %d, ожидалось 7", len(weeks[0].Items))
	}
	if weeks[0].Items[0].Kind != KindDiagnostic {
		t.Fatalf("первый пункт = %q", weeks[0].Items[0].Kind)
	}
}

func TestBuildWeekBlueprintsTwoWeeks(t *testing.T) {
	current := time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)
	exam := time.Date(2026, 11, 27, 0, 0, 0, 0, time.UTC)

	weeks := BuildWeekBlueprints(current, exam, 300, nil, false)
	if len(weeks) != 2 {
		t.Fatalf("недель = %d, ожидалось 2", len(weeks))
	}
	if weeks[0].Phase != PhaseDiagnostics || weeks[1].Phase != PhaseReview {
		t.Fatalf("фазы = %q, %q", weeks[0].Phase, weeks[1].Phase)
	}
}

func TestBuildWeekBlueprintsPhases(t *testing.T) {
	current := time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)
	exam := current.AddDate(0, 0, 7*4+4)

	weeks := BuildWeekBlueprints(current, exam, 300, nil, false)
	want := []Phase{PhaseDiagnostics, PhaseBase, PhaseBase, PhasePractice, PhaseReview}
	if len(weeks) != len(want) {
		t.Fatalf("недель = %d, ожидалось %d", len(weeks), len(want))
	}
	for index, phase := range want {
		if weeks[index].Phase != phase {
			t.Fatalf("неделя %d: фаза %q, ожидалась %q", index, weeks[index].Phase, phase)
		}
	}
}

func TestBuildWeekBlueprintsBudget(t *testing.T) {
	current := time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)
	exam := current.AddDate(0, 0, 6)

	weeks := BuildWeekBlueprints(current, exam, 600, nil, false)
	total := 0
	for _, item := range weeks[0].Items {
		total += item.TaskCount * 5
	}
	if total > 600 {
		t.Fatalf("нагрузка недели = %d минут, бюджет 600", total)
	}
}

func TestBuildWeekBlueprintsSmallBudget(t *testing.T) {
	current := time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)
	exam := current.AddDate(0, 0, 6)

	weeks := BuildWeekBlueprints(current, exam, 3, nil, true)
	if len(weeks[0].Items) != 3 {
		t.Fatalf("пунктов = %d, ожидалось 3", len(weeks[0].Items))
	}
	for _, item := range weeks[0].Items {
		if item.TaskCount < 1 {
			t.Fatal("task_count должен быть не меньше 1")
		}
		if item.Kind == KindDiagnostic {
			t.Fatal("при пройденной диагностике диагностический пункт не создаётся")
		}
	}
}

func TestBuildWeekBlueprintsPastExam(t *testing.T) {
	current := time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)
	exam := current.AddDate(0, 0, -1)

	if weeks := BuildWeekBlueprints(current, exam, 300, nil, false); weeks != nil {
		t.Fatalf("для прошедшего экзамена план не строится, получено %d недель", len(weeks))
	}
}

func TestBuildWeekBlueprintsUsesTopics(t *testing.T) {
	current := time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)
	exam := current.AddDate(0, 0, 13)
	topicID := uuid.New()

	weeks := BuildWeekBlueprints(current, exam, 300, []TopicPriority{{TopicID: topicID, Title: "Тема"}}, false)
	found := false
	for _, week := range weeks {
		for _, item := range week.Items {
			if item.TopicID != nil && *item.TopicID == topicID {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("пункты плана должны ссылаться на темы")
	}
}
