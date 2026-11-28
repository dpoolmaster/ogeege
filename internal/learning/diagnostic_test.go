package learning

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"oge-ege/internal/content"
)

func TestBuildDiagnosticSampleCoversKimNumbers(t *testing.T) {
	now := time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)
	kim1 := content.KimNumber{ID: uuid.New(), Number: 1}
	kim2 := content.KimNumber{ID: uuid.New(), Number: 2}
	tasks := []content.Task{
		{ID: uuid.New(), KimNumberID: kim2.ID, Difficulty: content.DifficultyHard, CreatedAt: now},
		{ID: uuid.New(), KimNumberID: kim1.ID, Difficulty: content.DifficultyMedium, CreatedAt: now},
		{ID: uuid.New(), KimNumberID: kim1.ID, Difficulty: content.DifficultyEasy, CreatedAt: now.Add(time.Hour)},
	}

	sample, duration, err := BuildDiagnosticSample([]content.KimNumber{kim2, kim1}, tasks)
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if len(sample) != 2 {
		t.Fatalf("заданий в выборке = %d, ожидалось 2", len(sample))
	}
	if sample[0].KimNumberID != kim1.ID || sample[1].KimNumberID != kim2.ID {
		t.Fatal("выборка должна быть упорядочена по номеру КИМ")
	}
	if sample[0].Difficulty != content.DifficultyEasy {
		t.Fatalf("для номера 1 выбрано задание сложности %q", sample[0].Difficulty)
	}
	if duration != 4 {
		t.Fatalf("длительность = %d, ожидалось 4", duration)
	}
}

func TestBuildDiagnosticSampleInsufficient(t *testing.T) {
	kim1 := content.KimNumber{ID: uuid.New(), Number: 1}
	kim2 := content.KimNumber{ID: uuid.New(), Number: 2}
	tasks := []content.Task{{ID: uuid.New(), KimNumberID: kim1.ID, Difficulty: content.DifficultyEasy}}

	if _, _, err := BuildDiagnosticSample([]content.KimNumber{kim1, kim2}, tasks); !errors.Is(err, ErrInsufficientContent) {
		t.Fatalf("ожидалась ошибка нехватки контента, получено %v", err)
	}
}

func TestBuildDiagnosticSampleKeepsCoverageOverDuration(t *testing.T) {
	numbers := make([]content.KimNumber, 0, 20)
	tasks := make([]content.Task, 0, 20)
	for index := 1; index <= 20; index++ {
		kim := content.KimNumber{ID: uuid.New(), Number: index}
		numbers = append(numbers, kim)
		tasks = append(tasks, content.Task{ID: uuid.New(), KimNumberID: kim.ID, Difficulty: content.DifficultyHard})
	}

	sample, duration, err := BuildDiagnosticSample(numbers, tasks)
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if len(sample) != len(numbers) {
		t.Fatalf("покрытие потеряно: %d из %d", len(sample), len(numbers))
	}
	if duration != 60 {
		t.Fatalf("длительность = %d, ожидалось 60", duration)
	}
}
