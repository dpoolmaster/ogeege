package content

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"oge-ege/internal/shared/clock"
	"oge-ege/internal/shared/db"
	"oge-ege/internal/shared/storage"
)

type txOnly struct {
	tx pgx.Tx
}

func (m txOnly) WithTx(_ context.Context, fn func(q db.Querier) error) error {
	return fn(m.tx)
}

func TestServiceIntegration(t *testing.T) {
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

	service := NewService(NewPgRepository(), tx, txOnly{tx: tx}, clock.New(), rand.New(rand.NewSource(1)), discardLogger())

	exam := createExam(t, ctx, service, "it_"+uuid.Must(uuid.NewV7()).String())
	draftExam, err := service.CreateExam(ctx, ExamInput{Code: "it_draft_" + uuid.Must(uuid.NewV7()).String(), Title: "Черновик", Subject: "math"})
	if err != nil {
		t.Fatalf("создать черновик трека: %v", err)
	}
	year := createYear(t, ctx, service, exam.ID)
	kimShort := createKim(t, ctx, service, year.ID, 1, 1, AnswerTypeShort, 1)
	kimExtended := createKim(t, ctx, service, year.ID, 2, 2, AnswerTypeExtended, 2)

	parent := createTopic(t, ctx, service, exam.ID, nil, "parent", "Родитель")
	child := createTopic(t, ctx, service, exam.ID, &parent.ID, "child", "Ребёнок")
	foreignParent := createTopic(t, ctx, service, draftExam.ID, nil, "foreign", "Чужой родитель")
	if _, err := service.CreateTopic(ctx, TopicInput{
		ExamID: exam.ID, ParentID: &foreignParent.ID, Code: "bad_parent", Title: "Плохая тема", OrderIndex: 1,
	}); err == nil {
		t.Fatal("ожидалась ошибка для родителя из другого трека")
	}
	for _, kim := range []*KimNumber{kimShort, kimExtended} {
		if _, err := service.SetKimTopics(ctx, kim.ID, []KimTopicInput{{TopicID: child.ID, Weight: 1}}); err != nil {
			t.Fatalf("связать тему: %v", err)
		}
	}

	theory := createTheory(t, ctx, service, child.ID)
	activeTheory, err := service.GetActiveTheory(ctx, child.ID)
	if err != nil {
		t.Fatalf("действующая теория: %v", err)
	}
	if activeTheory.ID != theory.ID {
		t.Fatalf("действующая теория = %s", activeTheory.ID)
	}
	if _, err := service.CreateTheoryVersion(ctx, TheoryInput{TopicID: child.ID, ContentMD: "Плохая формула $x^{2$"}); err == nil {
		t.Fatal("ожидалась ошибка формулы")
	}
	theoryVersions, err := service.ListTheoryVersions(ctx, child.ID)
	if err != nil || len(theoryVersions) != 1 {
		t.Fatalf("история теории: %v, %d", err, len(theoryVersions))
	}
	draft, err := service.CreateTheoryVersion(ctx, TheoryInput{TopicID: child.ID, ContentMD: "Черновик"})
	if err != nil {
		t.Fatalf("создать черновик теории: %v", err)
	}
	updatedDraft, err := service.UpdateTheory(ctx, draft.ID, TheoryInput{ContentMD: "Черновик изменён $x$"})
	if err != nil {
		t.Fatalf("изменить черновик теории: %v", err)
	}
	if updatedDraft.ContentMD != "Черновик изменён $x$" || updatedDraft.Version != draft.Version {
		t.Fatalf("черновик теории = %+v", updatedDraft)
	}
	if _, err := service.UpdateTheory(ctx, theory.ID, TheoryInput{ContentMD: "Нельзя"}); !errors.Is(err, ErrConflict) {
		t.Fatalf("опубликованная версия не должна изменяться: %v", err)
	}

	first := createTask(t, ctx, service, year.ID, kimShort.ID, child.ID, "Задание первое $1+1$")
	published, err := service.PublishTask(ctx, first.ID)
	if err != nil {
		t.Fatalf("опубликовать задание: %v", err)
	}
	if published.Status != StatusPublished || !published.IsPublic {
		t.Fatalf("задание не опубликовано: %+v", published)
	}

	updatedInput := TaskInput{
		ExamYearID: year.ID, KimNumberID: kimShort.ID, TopicID: &child.ID,
		Difficulty: DifficultyEasy, StatementMD: "Задание первое изменено $1+1$",
		AnswerSpec: json.RawMessage(`{"kind":"number","accepted":["2"]}`), Source: "seed",
	}
	updated, err := service.UpdateTask(ctx, first.ID, updatedInput)
	if err != nil {
		t.Fatalf("обновить задание: %v", err)
	}
	if updated.Version != 2 {
		t.Fatalf("версия = %d, ожидалось 2", updated.Version)
	}
	versions, err := service.ListTaskVersions(ctx, first.ID)
	if err != nil || len(versions) != 1 {
		t.Fatalf("история версий: %v, %d", err, len(versions))
	}

	public, err := service.ListPublicTasks(ctx, TaskFilter{ExamYearID: &year.ID})
	if err != nil {
		t.Fatalf("публичные задания: %v", err)
	}
	if len(public) != 1 || public[0].ID != first.ID {
		t.Fatalf("публичные задания = %+v", public)
	}
	if err := service.DeleteTask(ctx, first.ID); !errors.Is(err, ErrConflict) {
		t.Fatalf("удаление опубликованного задания: ожидалась ошибка, получено %v", err)
	}
	if err := service.DeleteExam(ctx, exam.ID); !errors.Is(err, ErrConflict) {
		t.Fatalf("удаление опубликованного трека: ожидалась ошибка, получено %v", err)
	}

	importStatement := "Импортированное задание $2+2$"
	report, err := service.ImportTasks(ctx, []ImportItem{
		{
			ExamYearID: year.ID, KimNumberID: kimShort.ID, TopicID: &child.ID,
			Difficulty: DifficultyEasy, StatementMD: importStatement,
			AnswerSpec: json.RawMessage(`{"kind":"number","accepted":["4"]}`), Source: "fipi",
			SourceRef: "https://fipi.ru",
		},
		{
			ExamYearID: year.ID, KimNumberID: kimShort.ID, TopicID: &child.ID,
			Difficulty: DifficultyEasy, StatementMD: updatedInput.StatementMD,
			AnswerSpec: json.RawMessage(`{"kind":"number","accepted":["2"]}`), Source: "fipi",
		},
		{
			ExamYearID: year.ID, KimNumberID: kimShort.ID, TopicID: &child.ID,
			Difficulty: DifficultyEasy, StatementMD: "Плохая формула $x^{2$",
			AnswerSpec: json.RawMessage(`{"kind":"number","accepted":["1"]}`), Source: "fipi",
		},
	})
	if err != nil {
		t.Fatalf("импорт: %v", err)
	}
	if len(report.Created) != 1 || len(report.Skipped) != 1 || len(report.Rejected) != 1 {
		t.Fatalf("отчёт импорта = %+v", report)
	}
	imported, err := service.GetTask(ctx, report.Created[0].TaskID)
	if err != nil {
		t.Fatalf("найти импортированное задание: %v", err)
	}
	if imported.Status != StatusDraft || imported.Source != "fipi" || imported.SourceRef != "https://fipi.ru" {
		t.Fatalf("импортированное задание = %+v", imported)
	}
	publicAfterImport, err := service.ListPublicTasks(ctx, TaskFilter{ExamYearID: &year.ID})
	if err != nil {
		t.Fatalf("публичные задания после импорта: %v", err)
	}
	if len(publicAfterImport) != 1 || publicAfterImport[0].ID != first.ID {
		t.Fatalf("импортированный черновик попал в публичную выдачу: %+v", publicAfterImport)
	}

	key, err := storage.ImageKey(first.ID, "image/png")
	if err != nil {
		t.Fatalf("сформировать ключ изображения: %v", err)
	}
	attachment, err := service.AddTaskAttachment(ctx, first.ID, TaskAttachmentInput{Kind: "image", StorageKey: key, Alt: "рисунок"})
	if err != nil {
		t.Fatalf("привязать изображение: %v", err)
	}
	if attachment.URL != "/media/"+key {
		t.Fatalf("URL изображения = %q", attachment.URL)
	}
	if _, err := service.AddTaskAttachment(ctx, first.ID, TaskAttachmentInput{Kind: "image", StorageKey: "../evil.png"}); err == nil {
		t.Fatal("ожидалась ошибка для чужого ключа изображения")
	}

	rubric, err := service.CreateRubric(ctx, RubricInput{
		KimNumberID: kimExtended.ID, ExamYearID: year.ID,
		Criteria: RubricCriteria{{Code: "C1", Description: "Решение", MaxPoints: 2}},
	})
	if err != nil {
		t.Fatalf("создать рубрику: %v", err)
	}
	if _, err := service.PublishRubric(ctx, rubric.ID); err != nil {
		t.Fatalf("опубликовать рубрику: %v", err)
	}
	if _, err := service.GetActiveRubric(ctx, kimExtended.ID, year.ID); err != nil {
		t.Fatalf("действующая рубрика: %v", err)
	}

	scales := make([]ScoreScaleEntry, 0, 4)
	for primary := 0; primary <= 3; primary++ {
		scales = append(scales, ScoreScaleEntry{PrimaryScore: primary, TestScore: primary})
	}
	if _, err := service.ReplaceScoreScales(ctx, ScoreScaleInput{ExamYearID: year.ID, Entries: scales}); err != nil {
		t.Fatalf("сохранить шкалу: %v", err)
	}
	if _, err := service.ReplaceScoreScales(ctx, ScoreScaleInput{ExamYearID: year.ID, Entries: scales[1:]}); err == nil {
		t.Fatal("ожидалась ошибка разрыва шкалы")
	}

	second := createTask(t, ctx, service, year.ID, kimExtended.ID, child.ID, "Задание второе $3+3$")
	if _, err := service.PublishTask(ctx, second.ID); err != nil {
		t.Fatalf("опубликовать второе задание: %v", err)
	}

	variant, err := service.CreateVariant(ctx, VariantInput{
		ExamYearID: year.ID, Title: "Ручной", Kind: VariantMock, DurationMinutes: 60,
		Tasks: []VariantTaskInput{{TaskID: first.ID, Points: 1}, {TaskID: second.ID, Points: 2}},
	})
	if err != nil {
		t.Fatalf("создать вариант: %v", err)
	}
	if len(variant.Tasks) != 2 || variant.Tasks[1].Position != 2 {
		t.Fatalf("позиции варианта = %+v", variant.Tasks)
	}

	generated, err := service.GenerateVariant(ctx, GenerateVariantInput{
		ExamYearID: year.ID, Title: "Сгенерированный", Kind: VariantMock, DurationMinutes: 60,
		Rules: []GenerationRule{{KimNumberID: kimShort.ID, Count: 1}},
	})
	if err != nil {
		t.Fatalf("сгенерировать вариант: %v", err)
	}
	if len(generated.Tasks) != 1 {
		t.Fatalf("позиции сгенерированного варианта = %+v", generated.Tasks)
	}
	before, err := service.CountVariants(ctx, year.ID)
	if err != nil {
		t.Fatalf("посчитать варианты: %v", err)
	}
	if _, err := service.GenerateVariant(ctx, GenerateVariantInput{
		ExamYearID: year.ID, Title: "Недостаточно", Kind: VariantMock, DurationMinutes: 60,
		Rules: []GenerationRule{{KimNumberID: kimShort.ID, Count: 10}},
	}); !errors.Is(err, ErrInsufficient) {
		t.Fatalf("ожидалась ошибка нехватки заданий, получено %v", err)
	}
	after, err := service.CountVariants(ctx, year.ID)
	if err != nil {
		t.Fatalf("посчитать варианты: %v", err)
	}
	if before != after {
		t.Fatalf("при отказе сохранён частичный вариант: %d -> %d", before, after)
	}

	available, err := service.IsExamAvailable(ctx, exam.ID)
	if err != nil || !available {
		t.Fatalf("опубликованный трек должен быть доступен: %v, %v", available, err)
	}
	if available, _ := service.IsExamAvailable(ctx, draftExam.ID); available {
		t.Fatal("неопубликованный трек не должен быть доступен")
	}
	if available, _ := service.IsExamAvailable(ctx, uuid.Must(uuid.NewV7())); available {
		t.Fatal("несуществующий трек не должен быть доступен")
	}

	publishedExams, err := service.ListPublishedExams(ctx)
	if err != nil {
		t.Fatalf("опубликованные треки: %v", err)
	}
	hasDraft := false
	for _, item := range publishedExams {
		if item.ID == draftExam.ID {
			hasDraft = true
		}
	}
	if hasDraft {
		t.Fatal("черновик трека попал в опубликованную выдачу")
	}

	page1, err := service.ListExams(ctx, ExamFilter{Query: "it_", Limit: 1})
	if err != nil || len(page1) != 1 {
		t.Fatalf("первая страница треков: %v, %d", err, len(page1))
	}
	page2, err := service.ListExams(ctx, ExamFilter{Query: "it_", Cursor: page1[0].ID, Limit: 1})
	if err != nil || len(page2) != 1 {
		t.Fatalf("вторая страница треков: %v, %d", err, len(page2))
	}
	if page1[0].ID == page2[0].ID {
		t.Fatal("страницы пагинации пересекаются")
	}

	if _, err := service.ArchiveTask(ctx, first.ID); err != nil {
		t.Fatalf("архивировать задание: %v", err)
	}
	publicAfterArchive, err := service.ListPublicTasks(ctx, TaskFilter{ExamYearID: &year.ID})
	if err != nil {
		t.Fatalf("публичные задания после архива: %v", err)
	}
	for _, item := range publicAfterArchive {
		if item.ID == first.ID {
			t.Fatalf("архивное задание осталось в публичной выдаче: %+v", publicAfterArchive)
		}
	}
}

func createExam(t *testing.T, ctx context.Context, service *Service, code string) *Exam {
	t.Helper()
	exam, err := service.CreateExam(ctx, ExamInput{Code: code, Title: "Тест", Subject: "math"})
	if err != nil {
		t.Fatalf("создать трек: %v", err)
	}
	if _, err := service.PublishExam(ctx, exam.ID); err != nil {
		t.Fatalf("опубликовать трек: %v", err)
	}
	return exam
}

func createYear(t *testing.T, ctx context.Context, service *Service, examID uuid.UUID) *ExamYear {
	t.Helper()
	year, err := service.CreateExamYear(ctx, ExamYearInput{ExamID: examID, Year: 2026, DurationMinutes: 235})
	if err != nil {
		t.Fatalf("создать учебный год: %v", err)
	}
	if _, err := service.PublishExamYear(ctx, year.ID); err != nil {
		t.Fatalf("опубликовать учебный год: %v", err)
	}
	return year
}

func createKim(t *testing.T, ctx context.Context, service *Service, yearID uuid.UUID, number, part int, answerType AnswerType, score int) *KimNumber {
	t.Helper()
	kim, err := service.CreateKimNumber(ctx, KimNumberInput{
		ExamYearID: yearID, Number: number, Part: part, AnswerType: answerType, MaxPrimaryScore: score,
	})
	if err != nil {
		t.Fatalf("создать номер КИМ: %v", err)
	}
	return kim
}

func createTopic(t *testing.T, ctx context.Context, service *Service, examID uuid.UUID, parentID *uuid.UUID, code, title string) *Topic {
	t.Helper()
	topic, err := service.CreateTopic(ctx, TopicInput{ExamID: examID, ParentID: parentID, Code: code, Title: title, OrderIndex: 1})
	if err != nil {
		t.Fatalf("создать тему: %v", err)
	}
	return topic
}

func createTheory(t *testing.T, ctx context.Context, service *Service, topicID uuid.UUID) *Theory {
	t.Helper()
	theory, err := service.CreateTheoryVersion(ctx, TheoryInput{TopicID: topicID, ContentMD: "Текст $x^2$"})
	if err != nil {
		t.Fatalf("создать теорию: %v", err)
	}
	published, err := service.PublishTheory(ctx, theory.ID)
	if err != nil {
		t.Fatalf("опубликовать теорию: %v", err)
	}
	return published
}

func createTask(t *testing.T, ctx context.Context, service *Service, yearID, kimID, topicID uuid.UUID, statement string) *Task {
	t.Helper()
	task, err := service.CreateTask(ctx, TaskInput{
		ExamYearID: yearID, KimNumberID: kimID, TopicID: &topicID,
		Difficulty: DifficultyEasy, StatementMD: statement,
		AnswerSpec: json.RawMessage(`{"kind":"number","accepted":["1"]}`), Source: "seed",
	})
	if err != nil {
		t.Fatalf("создать задание: %v", err)
	}
	return task
}
