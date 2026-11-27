package content

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"oge-ege/internal/shared/db"
)

type demoRepo struct {
	Repository
	tasks  []Task
	filter TaskFilter
}

func (r *demoRepo) ListPublicTasks(_ context.Context, _ db.Querier, filter TaskFilter) ([]Task, error) {
	r.filter = filter
	return r.tasks, nil
}

type reverseShuffler struct{}

func (reverseShuffler) Shuffle(n int, swap func(i, j int)) {
	for i := 0; i < n/2; i++ {
		swap(i, n-1-i)
	}
}

func TestListDemoTasks(t *testing.T) {
	examYearID := uuid.Must(uuid.NewV7())
	tasks := make([]Task, 0, 9)
	for i := 0; i < 6; i++ {
		tasks = append(tasks, Task{ID: uuid.Must(uuid.NewV7()), ExamYearID: examYearID, KimNumberID: uuid.Must(uuid.NewV7())})
	}
	tasks = append(tasks,
		Task{ID: uuid.Must(uuid.NewV7()), ExamYearID: examYearID, KimNumberID: tasks[0].KimNumberID},
		Task{ID: uuid.Must(uuid.NewV7()), ExamYearID: examYearID, KimNumberID: tasks[1].KimNumberID},
	)

	repo := &demoRepo{tasks: tasks}
	service := NewService(repo, nil, fakeTx{}, fixedClock{}, reverseShuffler{}, discardLogger())

	result, err := service.ListDemoTasks(context.Background(), examYearID, 5)
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if len(result) != 5 {
		t.Fatalf("получено %d заданий, ожидалось 5", len(result))
	}
	if !repo.filter.ExcludePublishedMock {
		t.Fatal("задания опубликованных пробников должны исключаться")
	}
	if len(repo.filter.Kinds) == 0 {
		t.Fatal("фильтр должен содержать проверяемые kind")
	}

	seen := make(map[uuid.UUID]struct{}, len(result))
	for _, task := range result {
		if _, ok := seen[task.KimNumberID]; ok {
			t.Fatalf("номер КИМ %s повторяется", task.KimNumberID)
		}
		seen[task.KimNumberID] = struct{}{}
	}
}

func TestListDemoTasksClampsLimit(t *testing.T) {
	examYearID := uuid.Must(uuid.NewV7())
	tasks := make([]Task, 0, 10)
	for i := 0; i < 10; i++ {
		tasks = append(tasks, Task{ID: uuid.Must(uuid.NewV7()), ExamYearID: examYearID, KimNumberID: uuid.Must(uuid.NewV7())})
	}

	service := NewService(&demoRepo{tasks: tasks}, nil, fakeTx{}, fixedClock{}, reverseShuffler{}, discardLogger())
	result, err := service.ListDemoTasks(context.Background(), examYearID, 10)
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if len(result) != 7 {
		t.Fatalf("получено %d заданий, ожидалось 7", len(result))
	}
}

func TestListDemoTasksInsufficient(t *testing.T) {
	examYearID := uuid.Must(uuid.NewV7())
	tasks := []Task{
		{ID: uuid.Must(uuid.NewV7()), ExamYearID: examYearID, KimNumberID: uuid.Must(uuid.NewV7())},
		{ID: uuid.Must(uuid.NewV7()), ExamYearID: examYearID, KimNumberID: uuid.Must(uuid.NewV7())},
	}

	service := NewService(&demoRepo{tasks: tasks}, nil, fakeTx{}, fixedClock{}, reverseShuffler{}, discardLogger())
	if _, err := service.ListDemoTasks(context.Background(), examYearID, 5); !errors.Is(err, ErrInsufficient) {
		t.Fatalf("ожидалась ErrInsufficient, получено %v", err)
	}
}
