package learning

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"oge-ege/internal/content"
	"oge-ege/internal/profile"
	"oge-ege/internal/shared/clock"
	"oge-ege/internal/shared/db"
)

type fakeGoalProvider struct {
	goal    *profile.StudyGoal
	saveErr error
}

func (f *fakeGoalProvider) GetCurrentGoal(context.Context, uuid.UUID) (*profile.StudyGoal, error) {
	if f.goal == nil {
		return nil, profile.ErrNotFound
	}

	return f.goal, nil
}

func (f *fakeGoalProvider) SaveGoal(_ context.Context, userID uuid.UUID, in profile.GoalInput) (*profile.StudyGoal, error) {
	if f.saveErr != nil {
		return nil, f.saveErr
	}
	examID, err := uuid.Parse(in.ExamID)
	if err != nil {
		return nil, profile.ErrInvalidExamID
	}
	examDate, err := time.Parse(dateLayout, in.ExamDate)
	if err != nil {
		return nil, profile.ErrInvalidExamDate
	}
	now := time.Now().UTC()
	f.goal = &profile.StudyGoal{
		ID:            uuid.New(),
		UserID:        userID,
		ExamID:        examID,
		ExamDate:      examDate,
		TargetScore:   in.TargetScore,
		WeeklyMinutes: in.WeeklyMinutes,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	return f.goal, nil
}

type fakeContentProvider struct {
	exam   content.Exam
	year   content.ExamYear
	topics []content.Topic
	kims   []content.KimNumber
	links  []content.KimTopic
	tasks  []content.Task
}

func (f *fakeContentProvider) GetExam(context.Context, uuid.UUID) (*content.Exam, error) {
	return &f.exam, nil
}

func (f *fakeContentProvider) GetExamByCode(context.Context, string) (*content.Exam, error) {
	return &f.exam, nil
}

func (f *fakeContentProvider) ListPublishedExams(context.Context) ([]content.Exam, error) {
	return []content.Exam{f.exam}, nil
}

func (f *fakeContentProvider) GetLatestPublishedExamYear(context.Context, uuid.UUID) (*content.ExamYear, error) {
	return &f.year, nil
}

func (f *fakeContentProvider) ListKimNumbers(context.Context, content.KimNumberFilter) ([]content.KimNumber, error) {
	return f.kims, nil
}

func (f *fakeContentProvider) ListTopics(context.Context, content.TopicFilter) ([]content.Topic, error) {
	return f.topics, nil
}

func (f *fakeContentProvider) ListKimTopicsByExamYear(context.Context, uuid.UUID) ([]content.KimTopic, error) {
	return f.links, nil
}

func (f *fakeContentProvider) ListPublicTasks(context.Context, content.TaskFilter) ([]content.Task, error) {
	return f.tasks, nil
}

func (f *fakeContentProvider) GetTask(context.Context, uuid.UUID) (*content.Task, error) {
	return nil, content.ErrNotFound
}

func (f *fakeContentProvider) ListTaskAttachments(context.Context, uuid.UUID) ([]content.TaskAttachment, error) {
	return nil, nil
}

func TestServiceIntegration(t *testing.T) {
	databaseURL, _ := os.LookupEnv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL не задан")
	}

	ctx := context.Background()
	pool, err := db.Open(ctx, databaseURL, 0, 2)
	if err != nil {
		t.Fatalf("открыть пул: %v", err)
	}
	defer pool.Close()

	userID := uuid.New()
	cleanup := func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM learning.sessions WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM learning.study_plans WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM learning.topic_mastery WHERE user_id = $1`, userID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM learning.kim_mastery WHERE user_id = $1`, userID)
	}
	t.Cleanup(cleanup)

	examID := uuid.New()
	yearID := uuid.New()
	topic1 := content.Topic{ID: uuid.New(), ExamID: examID, Code: "t1", Title: "Тема 1", OrderIndex: 1}
	topic2 := content.Topic{ID: uuid.New(), ExamID: examID, Code: "t2", Title: "Тема 2", OrderIndex: 2}
	kim1 := content.KimNumber{ID: uuid.New(), ExamYearID: yearID, Number: 1, Part: 1, AnswerType: content.AnswerTypeShort, MaxPrimaryScore: 1}
	kim2 := content.KimNumber{ID: uuid.New(), ExamYearID: yearID, Number: 2, Part: 1, AnswerType: content.AnswerTypeShort, MaxPrimaryScore: 1}
	now := time.Now().UTC()
	task1 := content.Task{
		ID: uuid.New(), ExamYearID: yearID, KimNumberID: kim1.ID, TopicID: &topic1.ID,
		Difficulty: content.DifficultyEasy, StatementMD: "Первое", AnswerSpec: json.RawMessage(`{"kind":"number","accepted":["5"]}`),
		Status: content.StatusPublished, IsPublic: true, Version: 1, CreatedAt: now,
	}
	task2 := content.Task{
		ID: uuid.New(), ExamYearID: yearID, KimNumberID: kim2.ID, TopicID: &topic2.ID,
		Difficulty: content.DifficultyEasy, StatementMD: "Второе", AnswerSpec: json.RawMessage(`{"kind":"number","accepted":["7"]}`),
		Status: content.StatusPublished, IsPublic: true, Version: 1, CreatedAt: now,
	}

	contentProvider := &fakeContentProvider{
		exam:   content.Exam{ID: examID, Code: "it_math", Title: "Тестовый трек", Subject: "math", Status: content.StatusPublished},
		year:   content.ExamYear{ID: yearID, ExamID: examID, Year: 2027, DurationMinutes: 180, Status: content.StatusPublished},
		topics: []content.Topic{topic1, topic2},
		kims:   []content.KimNumber{kim1, kim2},
		links: []content.KimTopic{
			{KimNumberID: kim1.ID, TopicID: topic1.ID, Weight: 1},
			{KimNumberID: kim2.ID, TopicID: topic2.ID, Weight: 1},
		},
		tasks: []content.Task{task1, task2},
	}
	goals := &fakeGoalProvider{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := NewService(NewPgRepository(), pool, db.NewTxManager(pool), clock.New(), goals, contentProvider, logger)

	plan, err := service.SetupGoal(ctx, userID, GoalInput{
		ExamCode:      "it_math",
		ExamDate:      time.Now().UTC().AddDate(0, 0, 30).Format(dateLayout),
		TargetScore:   60,
		WeeklyMinutes: 300,
	})
	if err != nil {
		t.Fatalf("цель: %v", err)
	}
	if plan.Version != 1 || plan.Status != PlanActive {
		t.Fatalf("план = %+v", plan)
	}

	weeks, items, err := service.loadPlanTree(ctx, plan.ID)
	if err != nil {
		t.Fatalf("прочитать план: %v", err)
	}
	if len(weeks) == 0 || len(items) == 0 {
		t.Fatal("план должен содержать недели и пункты")
	}

	repeated, err := service.SetupGoal(ctx, userID, GoalInput{
		ExamCode:      "it_math",
		ExamDate:      time.Now().UTC().AddDate(0, 0, 30).Format(dateLayout),
		TargetScore:   60,
		WeeklyMinutes: 300,
	})
	if err != nil {
		t.Fatalf("повторная цель: %v", err)
	}
	if repeated.Version != 2 {
		t.Fatalf("версия повторного плана = %d", repeated.Version)
	}
	var totalPlans, activePlans, supersededPlans int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM learning.study_plans WHERE user_id = $1`, userID).Scan(&totalPlans); err != nil {
		t.Fatalf("посчитать планы: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM learning.study_plans WHERE user_id = $1 AND status = 'active'`, userID).Scan(&activePlans); err != nil {
		t.Fatalf("посчитать активные планы: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM learning.study_plans WHERE user_id = $1 AND status = 'superseded'`, userID).Scan(&supersededPlans); err != nil {
		t.Fatalf("посчитать историю планов: %v", err)
	}
	if totalPlans != 2 || activePlans != 1 || supersededPlans != 1 {
		t.Fatalf("планы: всего %d, активных %d, история %d", totalPlans, activePlans, supersededPlans)
	}
	plan = repeated

	prepared, err := service.PrepareDiagnostic(ctx, userID, "")
	if err != nil {
		t.Fatalf("подготовить диагностику: %v", err)
	}
	if !prepared.Available || prepared.TaskCount != 2 {
		t.Fatalf("диагностика = %+v", prepared)
	}

	session, err := service.StartDiagnostic(ctx, userID, "")
	if err != nil {
		t.Fatalf("начать диагностику: %v", err)
	}
	sessionItems, err := service.repo.ListSessionItems(ctx, service.db, session.ID)
	if err != nil || len(sessionItems) != 2 {
		t.Fatalf("задания сессии: %v, %d", err, len(sessionItems))
	}

	contentProvider.tasks[0].AnswerSpec = json.RawMessage(`{"kind":"number","accepted":["999"]}`)
	if _, err := service.SaveAnswer(ctx, userID, session.ID, sessionItems[0].ID, "5"); err != nil {
		t.Fatalf("сохранить верный ответ: %v", err)
	}
	if _, err := service.SaveAnswer(ctx, userID, session.ID, sessionItems[1].ID, "9"); err != nil {
		t.Fatalf("сохранить неверный ответ: %v", err)
	}

	result, err := service.CompleteSession(ctx, userID, session.ID)
	if err != nil {
		t.Fatalf("завершить сессию: %v", err)
	}
	if result.PrimaryScore != 1 || result.MaxScore != 2 {
		t.Fatalf("результат = %+v", result)
	}

	gaps, err := service.Gaps(ctx, userID)
	if err != nil {
		t.Fatalf("карта пробелов: %v", err)
	}
	if !gaps.HasData {
		t.Fatal("после диагностики карта должна содержать данные")
	}

	replanned, err := service.Replan(ctx, userID)
	if err != nil {
		t.Fatalf("пересчитать план: %v", err)
	}
	if replanned.Version < 2 {
		t.Fatalf("версия после диагностики = %d", replanned.Version)
	}
	repeatedReplan, err := service.Replan(ctx, userID)
	if err != nil {
		t.Fatalf("повторный пересчёт: %v", err)
	}
	if repeatedReplan.Version != replanned.Version {
		t.Fatalf("повторный пересчёт создал версию %d вместо %d", repeatedReplan.Version, replanned.Version)
	}

	var activeCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM learning.study_plans WHERE user_id = $1 AND status = 'active'`, userID).Scan(&activeCount); err != nil {
		t.Fatalf("посчитать активные планы: %v", err)
	}
	if activeCount != 1 {
		t.Fatalf("активных планов = %d, ожидался 1", activeCount)
	}

	view, err := service.Result(ctx, userID, session.ID)
	if err != nil {
		t.Fatalf("результат сессии: %v", err)
	}
	if len(view.Items) != 2 {
		t.Fatalf("строк разбивки = %d", len(view.Items))
	}
}

func TestSetupGoalFailureIntegration(t *testing.T) {
	databaseURL, _ := os.LookupEnv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL не задан")
	}

	ctx := context.Background()
	pool, err := db.Open(ctx, databaseURL, 0, 2)
	if err != nil {
		t.Fatalf("открыть пул: %v", err)
	}
	defer pool.Close()

	userID := uuid.New()
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM learning.study_plans WHERE user_id = $1`, userID)
	})

	examID := uuid.New()
	contentProvider := &fakeContentProvider{exam: content.Exam{ID: examID, Code: "it_math", Status: content.StatusPublished}}
	goals := &fakeGoalProvider{saveErr: errors.New("цель не сохранена")}
	service := NewService(NewPgRepository(), pool, db.NewTxManager(pool), clock.New(), goals, contentProvider, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if _, err := service.SetupGoal(ctx, userID, GoalInput{
		ExamCode:      "it_math",
		ExamDate:      time.Now().UTC().AddDate(0, 0, 30).Format(dateLayout),
		TargetScore:   60,
		WeeklyMinutes: 300,
	}); err == nil {
		t.Fatal("ожидалась ошибка сохранения цели")
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM learning.study_plans WHERE user_id = $1`, userID).Scan(&count); err != nil {
		t.Fatalf("посчитать планы: %v", err)
	}
	if count != 0 {
		t.Fatalf("при ошибке цели создано планов: %d", count)
	}
}

func TestResultNotReadyIntegration(t *testing.T) {
	databaseURL, _ := os.LookupEnv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL не задан")
	}

	ctx := context.Background()
	pool, err := db.Open(ctx, databaseURL, 0, 2)
	if err != nil {
		t.Fatalf("открыть пул: %v", err)
	}
	defer pool.Close()

	userID := uuid.New()
	examID := uuid.New()
	now := time.Now().UTC()
	session := &Session{
		ID: uuid.New(), UserID: userID, ExamID: examID, Kind: SessionKindDiagnostic,
		Status: SessionInProgress, StartedAt: now, CreatedAt: now,
	}
	repo := NewPgRepository()
	if err := repo.CreateSession(ctx, pool, session); err != nil {
		t.Fatalf("создать сессию: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM learning.sessions WHERE user_id = $1`, userID)
	})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := NewService(repo, pool, db.NewTxManager(pool), clock.New(), &fakeGoalProvider{}, &fakeContentProvider{}, logger)
	if _, err := service.Result(ctx, userID, session.ID); !errors.Is(err, ErrResultNotReady) {
		t.Fatalf("ожидалась ошибка незавершённой сессии, получено %v", err)
	}
}
