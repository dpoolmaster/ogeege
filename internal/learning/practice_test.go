package learning

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"oge-ege/internal/content"
	"oge-ege/internal/profile"
	"oge-ege/internal/shared/clock"
	"oge-ege/internal/shared/db"
)

type practiceRepo struct {
	Repository
	active     *Session
	planItem   *PlanItem
	planExamID uuid.UUID
	created    *Session
	items      []SessionItem
	dueTopics  []uuid.UUID
	incorrect  []uuid.UUID
	markedDone []uuid.UUID
}

func (r *practiceRepo) GetInProgressSession(context.Context, db.Querier, uuid.UUID, uuid.UUID, string) (*Session, error) {
	if r.active == nil {
		return nil, ErrNotFound
	}

	return r.active, nil
}

func (r *practiceRepo) GetPlanItemForUser(context.Context, db.Querier, uuid.UUID, uuid.UUID) (*PlanItem, uuid.UUID, error) {
	if r.planItem == nil {
		return nil, uuid.Nil, ErrNotFound
	}

	return r.planItem, r.planExamID, nil
}

func (r *practiceRepo) CreateSession(_ context.Context, _ db.Querier, session *Session) error {
	r.created = session

	return nil
}

func (r *practiceRepo) CreateSessionItems(_ context.Context, _ db.Querier, items []SessionItem) error {
	r.items = items

	return nil
}

func (r *practiceRepo) ListDueTopicIDs(context.Context, db.Querier, uuid.UUID, time.Time) ([]uuid.UUID, error) {
	return r.dueTopics, nil
}

func (r *practiceRepo) ListIncorrectTaskIDs(context.Context, db.Querier, uuid.UUID) ([]uuid.UUID, error) {
	return r.incorrect, nil
}

func (r *practiceRepo) MarkItemDone(_ context.Context, _ db.Querier, itemID, _ uuid.UUID) error {
	r.markedDone = append(r.markedDone, itemID)

	return nil
}

type practiceContent struct {
	fakeContentProvider
	tasks      []content.Task
	byID       map[uuid.UUID]content.Task
	options    map[uuid.UUID][]content.TaskOption
	filtered   []content.Task
	lastFilter content.TaskFilter
}

func (f *practiceContent) ListPublicTasks(_ context.Context, filter content.TaskFilter) ([]content.Task, error) {
	f.lastFilter = filter
	result := make([]content.Task, 0, len(f.tasks))
	for _, task := range f.tasks {
		if filter.TopicID != nil && (task.TopicID == nil || *task.TopicID != *filter.TopicID) {
			continue
		}
		if filter.KimNumberID != nil && task.KimNumberID != *filter.KimNumberID {
			continue
		}
		result = append(result, task)
	}
	f.filtered = result

	return result, nil
}

func (f *practiceContent) GetTask(_ context.Context, id uuid.UUID) (*content.Task, error) {
	task, ok := f.byID[id]
	if !ok {
		return nil, content.ErrNotFound
	}

	return &task, nil
}

func (f *practiceContent) ListTaskOptions(_ context.Context, taskID uuid.UUID) ([]content.TaskOption, error) {
	return f.options[taskID], nil
}

type reverseShuffle struct{}

func (reverseShuffle) Shuffle(n int, swap func(i, j int)) {
	for i := 0; i < n/2; i++ {
		swap(i, n-1-i)
	}
}

func practiceFixture(t *testing.T, count int) (*Service, *practiceRepo, *practiceContent, content.Exam, content.ExamYear, content.Topic, content.KimNumber) {
	t.Helper()
	exam := content.Exam{ID: uuid.New(), Code: "math", Title: "Математика", Status: content.StatusPublished}
	year := content.ExamYear{ID: uuid.New(), ExamID: exam.ID, Year: 2027, Status: content.StatusPublished}
	topic := content.Topic{ID: uuid.New(), ExamID: exam.ID, Code: "t", Title: "Тема", OrderIndex: 1}
	kim := content.KimNumber{ID: uuid.New(), ExamYearID: year.ID, Number: 1, Part: 1, MaxPrimaryScore: 1}

	tasks := make([]content.Task, 0, count)
	byID := make(map[uuid.UUID]content.Task, count)
	for i := 0; i < count; i++ {
		task := content.Task{
			ID: uuid.New(), ExamYearID: year.ID, KimNumberID: kim.ID, TopicID: &topic.ID,
			Difficulty: content.DifficultyEasy, StatementMD: "Задание",
			AnswerSpec: json.RawMessage(`{"kind":"number","accepted":["1"]}`),
			Status:     content.StatusPublished, IsPublic: true, Version: 1,
		}
		tasks = append(tasks, task)
		byID[task.ID] = task
	}

	provider := &practiceContent{
		fakeContentProvider: fakeContentProvider{exam: exam, year: year, topics: []content.Topic{topic}, kims: []content.KimNumber{kim}},
		tasks:               tasks,
		byID:                byID,
	}
	repo := &practiceRepo{}
	goals := &fakeGoalProvider{goal: &profile.StudyGoal{ExamID: exam.ID}}
	service := NewService(repo, nil, fakeTx{}, clock.New(), goals, provider, reverseShuffle{}, testLogger())

	return service, repo, provider, exam, year, topic, kim
}

func TestStartPracticeByTopic(t *testing.T) {
	service, repo, _, _, _, topic, _ := practiceFixture(t, 15)

	session, err := service.StartPractice(context.Background(), uuid.New(), PracticeInput{TopicID: &topic.ID})
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if session.Kind != SessionKindPractice || session.PlanItemID != nil {
		t.Fatalf("сессия = %+v", session)
	}
	if len(repo.items) != defaultPracticeSize {
		t.Fatalf("заданий = %d, ожидалось %d", len(repo.items), defaultPracticeSize)
	}
}

func TestStartPracticeFromPlanCapsAt20(t *testing.T) {
	service, repo, _, exam, _, topic, kim := practiceFixture(t, 25)
	repo.planItem = &PlanItem{ID: uuid.New(), Kind: KindPractice, TopicID: &topic.ID, KimNumberID: &kim.ID, TaskCount: 30}
	repo.planExamID = exam.ID

	session, err := service.StartPractice(context.Background(), uuid.New(), PracticeInput{PlanItemID: &repo.planItem.ID})
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if session.PlanItemID == nil || *session.PlanItemID != repo.planItem.ID {
		t.Fatalf("пункт плана не сохранён: %+v", session)
	}
	if len(repo.items) != maxPracticeSize {
		t.Fatalf("заданий = %d, ожидалось %d", len(repo.items), maxPracticeSize)
	}
}

func TestStartPracticeNoTasks(t *testing.T) {
	service, _, _, _, _, topic, _ := practiceFixture(t, 0)

	_, err := service.StartPractice(context.Background(), uuid.New(), PracticeInput{TopicID: &topic.ID})
	if !errors.Is(err, ErrContentUnavailable) {
		t.Fatalf("ошибка = %v, ожидалась ErrContentUnavailable", err)
	}
}

func TestStartPracticeRequiresOneFilter(t *testing.T) {
	service, _, _, _, _, _, _ := practiceFixture(t, 3)

	_, err := service.StartPractice(context.Background(), uuid.New(), PracticeInput{})
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("ошибка = %v, ожидалась ошибка валидации", err)
	}
}

func TestStartPracticeResumesActive(t *testing.T) {
	service, repo, _, _, _, topic, _ := practiceFixture(t, 3)
	active := &Session{ID: uuid.New(), Kind: SessionKindPractice}
	repo.active = active

	session, err := service.StartPractice(context.Background(), uuid.New(), PracticeInput{TopicID: &topic.ID})
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if session.ID != active.ID {
		t.Fatal("должна возвращаться незавершённая практика")
	}
}

func TestCollectReviewTasks(t *testing.T) {
	service, repo, provider, exam, year, topic, _ := practiceFixture(t, 2)
	dueTask := provider.tasks[0]
	incorrectTask := provider.tasks[1]
	repo.dueTopics = []uuid.UUID{topic.ID}
	repo.incorrect = []uuid.UUID{incorrectTask.ID, dueTask.ID}
	provider.byID[incorrectTask.ID] = incorrectTask

	tasks, err := service.collectReviewTasks(context.Background(), uuid.New(), exam.ID)
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("кандидатов = %d, ожидалось 2 (без дубликата)", len(tasks))
	}
	if tasks[0].ExamYearID != year.ID {
		t.Fatalf("задание другого года: %+v", tasks[0])
	}
}

func TestCollectReviewTasksExcludesUnsupported(t *testing.T) {
	service, repo, provider, exam, _, _, _ := practiceFixture(t, 1)
	extended := provider.tasks[0]
	extended.AnswerSpec = json.RawMessage(`{"kind":"extended"}`)
	provider.byID[extended.ID] = extended
	repo.incorrect = []uuid.UUID{extended.ID}

	tasks, err := service.collectReviewTasks(context.Background(), uuid.New(), exam.ID)
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if len(tasks) != 0 {
		t.Fatal("задание без автопроверки не должно попадать в повторение")
	}
}

func TestNextReview(t *testing.T) {
	now := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name         string
		interval     int
		ease         float64
		success      bool
		wantInterval int
		wantEase     float64
	}{
		{name: "успех", interval: 1, ease: 2.0, success: true, wantInterval: 2, wantEase: 2.1},
		{name: "успех ограничен", interval: 50, ease: 2.5, success: true, wantInterval: 60, wantEase: 2.5},
		{name: "неудача", interval: 10, ease: 2.0, success: false, wantInterval: 1, wantEase: 1.8},
		{name: "минимум легкости", interval: 10, ease: 1.4, success: false, wantInterval: 1, wantEase: 1.3},
		{name: "начальные значения", interval: 0, ease: 0, success: false, wantInterval: 1, wantEase: 1.8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interval, ease, next := NextReview(tt.interval, tt.ease, tt.success, now)
			if interval != tt.wantInterval || ease != tt.wantEase {
				t.Fatalf("получено %d, %.2f; ожидалось %d, %.2f", interval, ease, tt.wantInterval, tt.wantEase)
			}
			if !next.Equal(now.AddDate(0, 0, interval)) {
				t.Fatalf("next = %v", next)
			}
		})
	}
}

type fakeTx struct{}

func (fakeTx) WithTx(ctx context.Context, fn func(q db.Querier) error) error {
	return fn(nil)
}

type completionRepo struct {
	Repository
	session      *Session
	items        []SessionItem
	answers      []Answer
	topics       []TopicMastery
	kims         []KimMastery
	result       *SessionResult
	markedDone   []uuid.UUID
	savedTopics  []TopicMastery
	savedKims    []KimMastery
	savedResults int
	completed    int
}

func (r *completionRepo) GetSessionByUser(context.Context, db.Querier, uuid.UUID, uuid.UUID) (*Session, error) {
	return r.session, nil
}

func (r *completionRepo) ListSessionItems(context.Context, db.Querier, uuid.UUID) ([]SessionItem, error) {
	return r.items, nil
}

func (r *completionRepo) ListAnswers(context.Context, db.Querier, uuid.UUID) ([]Answer, error) {
	return r.answers, nil
}

func (r *completionRepo) ListTopicMastery(context.Context, db.Querier, uuid.UUID) ([]TopicMastery, error) {
	return r.topics, nil
}

func (r *completionRepo) ListKimMastery(context.Context, db.Querier, uuid.UUID) ([]KimMastery, error) {
	return r.kims, nil
}

func (r *completionRepo) CompleteSession(context.Context, db.Querier, uuid.UUID, time.Time, float64) error {
	r.completed++
	r.session.Status = SessionCompleted

	return nil
}

func (r *completionRepo) UpsertSessionResult(_ context.Context, _ db.Querier, result *SessionResult) error {
	r.result = result
	r.savedResults++

	return nil
}

func (r *completionRepo) GetSessionResult(context.Context, db.Querier, uuid.UUID) (*SessionResult, error) {
	return r.result, nil
}

func (r *completionRepo) UpsertTopicMastery(_ context.Context, _ db.Querier, mastery *TopicMastery) error {
	r.savedTopics = append(r.savedTopics, *mastery)

	return nil
}

func (r *completionRepo) UpsertKimMastery(_ context.Context, _ db.Querier, mastery *KimMastery) error {
	r.savedKims = append(r.savedKims, *mastery)

	return nil
}

func (r *completionRepo) MarkItemDone(_ context.Context, _ db.Querier, itemID, _ uuid.UUID) error {
	r.markedDone = append(r.markedDone, itemID)

	return nil
}

func completionFixture(success bool) (*Service, *completionRepo, uuid.UUID, uuid.UUID) {
	examID := uuid.New()
	topicID := uuid.New()
	kimID := uuid.New()
	planItemID := uuid.New()
	itemID := uuid.New()
	snapshot, _ := json.Marshal(TaskSnapshot{
		TaskID: uuid.New(), StatementMD: "Задание", KimNumberID: kimID, TopicID: &topicID, MaxPoints: 1,
		AnswerSpec: json.RawMessage(`{"kind":"number","accepted":["1"]}`),
	})
	repo := &completionRepo{
		session: &Session{ID: uuid.New(), ExamID: examID, Kind: SessionKindPractice, Status: SessionInProgress, PlanItemID: &planItemID},
		items: []SessionItem{{
			ID: itemID, SessionID: uuid.New(), Position: 1, MaxPoints: 1, TaskSnapshot: snapshot, Status: SessionItemAnswered,
		}},
		answers: []Answer{{ID: uuid.New(), SessionItemID: itemID, RawAnswer: "1", IsCorrect: success, AwardedPoints: 1}},
	}
	service := NewService(repo, nil, fakeTx{}, clock.New(), &fakeGoalProvider{}, &fakeContentProvider{}, nil, testLogger())

	return service, repo, topicID, planItemID
}

func TestCompletePracticeMarksPlanItemAndInterval(t *testing.T) {
	service, repo, topicID, planItemID := completionFixture(true)

	if _, err := service.CompleteSession(context.Background(), uuid.New(), repo.session.ID); err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if len(repo.markedDone) != 1 || repo.markedDone[0] != planItemID {
		t.Fatalf("пункт плана не отмечен: %v", repo.markedDone)
	}
	if len(repo.savedTopics) != 1 || repo.savedTopics[0].TopicID != topicID {
		t.Fatalf("освоение темы не сохранено: %+v", repo.savedTopics)
	}
	if repo.savedTopics[0].IntervalDays != 2 || repo.savedTopics[0].Ease != 2.1 {
		t.Fatalf("интервал = %d, легкость = %.2f", repo.savedTopics[0].IntervalDays, repo.savedTopics[0].Ease)
	}
	if repo.savedTopics[0].NextReviewAt == nil {
		t.Fatal("дата повторения не рассчитана")
	}
}

func TestCompletePracticeFailureResetsInterval(t *testing.T) {
	service, repo, _, _ := completionFixture(false)

	if _, err := service.CompleteSession(context.Background(), uuid.New(), repo.session.ID); err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if repo.savedTopics[0].IntervalDays != 1 {
		t.Fatalf("интервал = %d, ожидался 1", repo.savedTopics[0].IntervalDays)
	}
	if repo.savedTopics[0].Ease != 1.8 {
		t.Fatalf("легкость = %.2f, ожидалась 1.8", repo.savedTopics[0].Ease)
	}
}

func TestCompletePracticeIdempotent(t *testing.T) {
	service, repo, _, _ := completionFixture(true)

	if _, err := service.CompleteSession(context.Background(), uuid.New(), repo.session.ID); err != nil {
		t.Fatalf("первое завершение: %v", err)
	}
	if _, err := service.CompleteSession(context.Background(), uuid.New(), repo.session.ID); err != nil {
		t.Fatalf("повторное завершение: %v", err)
	}
	if len(repo.markedDone) != 1 {
		t.Fatalf("пункт отмечен %d раз", len(repo.markedDone))
	}
	if repo.savedResults != 1 {
		t.Fatalf("результат сохранён %d раз", repo.savedResults)
	}
}

func TestStartPracticeFiltersComposition(t *testing.T) {
	service, _, provider, _, _, topic, _ := practiceFixture(t, 15)

	if _, err := service.StartPractice(context.Background(), uuid.New(), PracticeInput{TopicID: &topic.ID}); err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if !provider.lastFilter.ExcludePublishedMock {
		t.Fatal("опубликованные пробники должны исключаться")
	}
	if len(provider.lastFilter.Kinds) == 0 {
		t.Fatal("фильтр должен ограничивать типы проверяемых ответов")
	}
	for _, kind := range []string{"number", "text", "digits", "interval", "choice"} {
		found := false
		for _, candidate := range provider.lastFilter.Kinds {
			if candidate == kind {
				found = true
			}
		}
		if !found {
			t.Fatalf("kind %q отсутствует в фильтре", kind)
		}
	}
}

func TestStartPracticeShufflesTasks(t *testing.T) {
	service, repo, provider, _, _, topic, _ := practiceFixture(t, 15)

	if _, err := service.StartPractice(context.Background(), uuid.New(), PracticeInput{TopicID: &topic.ID}); err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if repo.items[0].TaskID != provider.tasks[len(provider.tasks)-1].ID {
		t.Fatal("задания должны перемешиваться внедряемым shuffler")
	}
}
