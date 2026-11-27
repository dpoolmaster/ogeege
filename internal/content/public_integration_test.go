package content

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/google/uuid"

	"oge-ege/internal/shared/clock"
	"oge-ege/internal/shared/db"
)

func TestPublicReadingIntegration(t *testing.T) {
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

	service := NewService(NewPgRepository(), tx, txOnly{tx: tx}, clock.New(), nil, discardLogger())
	exam := createExam(t, ctx, service, "it_public_"+uuid.Must(uuid.NewV7()).String())
	year := createYear(t, ctx, service, exam.ID)

	draftYear, err := service.CreateExamYear(ctx, ExamYearInput{ExamID: exam.ID, Year: 2027, DurationMinutes: 235})
	if err != nil {
		t.Fatalf("создать черновик года: %v", err)
	}
	latest, err := service.GetLatestPublishedExamYear(ctx, exam.ID)
	if err != nil {
		t.Fatalf("последний опубликованный год: %v", err)
	}
	if latest.ID != year.ID {
		t.Fatalf("выбран год %s вместо %s (черновик %s)", latest.ID, year.ID, draftYear.ID)
	}

	kimOne := createKim(t, ctx, service, year.ID, 1, 1, AnswerTypeShort, 1)
	kimTwo := createKim(t, ctx, service, year.ID, 2, 1, AnswerTypeShort, 1)
	topic := createTopic(t, ctx, service, exam.ID, nil, "public", "Публичная тема")
	for _, kim := range []*KimNumber{kimOne, kimTwo} {
		if _, err := service.SetKimTopics(ctx, kim.ID, []KimTopicInput{{TopicID: topic.ID, Weight: 1}}); err != nil {
			t.Fatalf("связать тему с номером КИМ: %v", err)
		}
	}

	numberTask := createTask(t, ctx, service, year.ID, kimOne.ID, topic.ID, "Публичное числовое задание")
	if _, err := service.PublishTask(ctx, numberTask.ID); err != nil {
		t.Fatalf("опубликовать числовое задание: %v", err)
	}

	textTask, err := service.CreateTask(ctx, TaskInput{
		ExamYearID: year.ID, KimNumberID: kimTwo.ID, TopicID: &topic.ID,
		Difficulty: DifficultyEasy, StatementMD: "Публичное текстовое задание",
		AnswerSpec: json.RawMessage(`{"kind":"text","accepted":["да"]}`), Source: "seed",
	})
	if err != nil {
		t.Fatalf("создать текстовое задание: %v", err)
	}
	if _, err := service.PublishTask(ctx, textTask.ID); err != nil {
		t.Fatalf("опубликовать текстовое задание: %v", err)
	}

	draftTask := createTask(t, ctx, service, year.ID, kimOne.ID, topic.ID, "Черновик публичного задания")
	privateTask := createTask(t, ctx, service, year.ID, kimOne.ID, topic.ID, "Приватное задание")
	if _, err := service.PublishTask(ctx, privateTask.ID); err != nil {
		t.Fatalf("опубликовать приватное задание: %v", err)
	}
	if err := service.repo.SetTaskStatus(ctx, tx, privateTask.ID, StatusPublished, false, clock.New().Now()); err != nil {
		t.Fatalf("снять публичность: %v", err)
	}

	mockTask := createTask(t, ctx, service, year.ID, kimTwo.ID, topic.ID, "Задание опубликованного пробника")
	if _, err := service.PublishTask(ctx, mockTask.ID); err != nil {
		t.Fatalf("опубликовать задание пробника: %v", err)
	}
	variant, err := service.CreateVariant(ctx, VariantInput{
		ExamYearID: year.ID, Title: "Пробник", Kind: VariantMock, DurationMinutes: 60,
		Tasks: []VariantTaskInput{{TaskID: mockTask.ID, Points: 1}},
	})
	if err != nil {
		t.Fatalf("создать пробник: %v", err)
	}
	if _, err := service.PublishVariant(ctx, variant.ID); err != nil {
		t.Fatalf("опубликовать пробник: %v", err)
	}

	tasks, err := service.ListPublicTasks(ctx, TaskFilter{
		ExamID:               &exam.ID,
		Kinds:                []string{"number"},
		ExcludePublishedMock: true,
		Limit:                50,
	})
	if err != nil {
		t.Fatalf("публичные задания: %v", err)
	}
	ids := make(map[uuid.UUID]bool)
	for _, task := range tasks {
		ids[task.ID] = true
	}
	if !ids[numberTask.ID] {
		t.Fatal("опубликованное числовое задание не попало в выдачу")
	}
	if ids[textTask.ID] {
		t.Fatal("задание другого kind попало в выдачу")
	}
	if ids[draftTask.ID] || ids[privateTask.ID] || ids[mockTask.ID] {
		t.Fatalf("недоступное задание попало в выдачу: %+v", ids)
	}

	extra := createTask(t, ctx, service, year.ID, kimOne.ID, topic.ID, "Второе числовое задание для пагинации")
	if _, err := service.PublishTask(ctx, extra.ID); err != nil {
		t.Fatalf("опубликовать второе задание: %v", err)
	}
	pageOne, err := service.ListPublicTasks(ctx, TaskFilter{ExamID: &exam.ID, Kinds: []string{"number"}, ExcludePublishedMock: true, Limit: 1})
	if err != nil || len(pageOne) != 1 {
		t.Fatalf("первая страница: %v, %d", err, len(pageOne))
	}
	pageTwo, err := service.ListPublicTasks(ctx, TaskFilter{
		ExamID: &exam.ID, Kinds: []string{"number"}, ExcludePublishedMock: true, Cursor: pageOne[0].ID, Limit: 1,
	})
	if err != nil || len(pageTwo) != 1 {
		t.Fatalf("вторая страница: %v, %d", err, len(pageTwo))
	}
	if pageOne[0].ID == pageTwo[0].ID {
		t.Fatal("страницы пагинации пересекаются")
	}
}
