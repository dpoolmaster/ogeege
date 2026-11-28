package learning

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"oge-ege/internal/content"
	"oge-ege/internal/grading"
	"oge-ege/internal/profile"
	"oge-ege/internal/shared/clock"
	"oge-ege/internal/shared/db"
	"oge-ege/internal/shared/ids"
)

const dateLayout = "2006-01-02"

const (
	defaultIntervalDays = 1
	defaultEase         = 2.0
	replanInterval      = 7 * 24 * time.Hour
	contentPageLimit    = 200
)

// GoalProvider предоставляет учебную цель из модуля profile.
type GoalProvider interface {
	GetCurrentGoal(ctx context.Context, userID uuid.UUID) (*profile.StudyGoal, error)
	SaveGoal(ctx context.Context, userID uuid.UUID, in profile.GoalInput) (*profile.StudyGoal, error)
}

// ContentProvider предоставляет опубликованный контент из модуля content.
type ContentProvider interface {
	GetExam(ctx context.Context, id uuid.UUID) (*content.Exam, error)
	GetExamByCode(ctx context.Context, code string) (*content.Exam, error)
	ListPublishedExams(ctx context.Context) ([]content.Exam, error)
	GetLatestPublishedExamYear(ctx context.Context, examID uuid.UUID) (*content.ExamYear, error)
	ListKimNumbers(ctx context.Context, filter content.KimNumberFilter) ([]content.KimNumber, error)
	ListTopics(ctx context.Context, filter content.TopicFilter) ([]content.Topic, error)
	ListKimTopicsByExamYear(ctx context.Context, examYearID uuid.UUID) ([]content.KimTopic, error)
	ListPublicTasks(ctx context.Context, filter content.TaskFilter) ([]content.Task, error)
	GetTask(ctx context.Context, id uuid.UUID) (*content.Task, error)
	ListTaskAttachments(ctx context.Context, taskID uuid.UUID) ([]content.TaskAttachment, error)
	ListTaskOptions(ctx context.Context, taskID uuid.UUID) ([]content.TaskOption, error)
}

// Shuffler перемешивает задания; подменяется в тестах.
type Shuffler interface {
	Shuffle(n int, swap func(i, j int))
}

type lockedShuffler struct {
	mu sync.Mutex
	r  *rand.Rand
}

func (s *lockedShuffler) Shuffle(n int, swap func(i, j int)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.r.Shuffle(n, swap)
}

// TxManager выполняет функцию в транзакции.
type TxManager interface {
	WithTx(ctx context.Context, fn func(q db.Querier) error) error
}

// Service содержит сценарии модуля learning.
type Service struct {
	repo     Repository
	db       db.Querier
	tx       TxManager
	clock    clock.Clock
	goals    GoalProvider
	content  ContentProvider
	shuffler Shuffler
	logger   *slog.Logger
}

// NewService создаёт сервис learning.
func NewService(repo Repository, database db.Querier, tx TxManager, clk clock.Clock, goals GoalProvider, contentProvider ContentProvider, shuffler Shuffler, logger *slog.Logger) *Service {
	if shuffler == nil {
		shuffler = &lockedShuffler{r: rand.New(rand.NewSource(time.Now().UnixNano()))}
	}

	return &Service{
		repo:     repo,
		db:       database,
		tx:       tx,
		clock:    clk,
		goals:    goals,
		content:  contentProvider,
		shuffler: shuffler,
		logger:   logger,
	}
}

type examStructure struct {
	Exam   content.Exam
	Year   content.ExamYear
	Topics []content.Topic
	Kims   []content.KimNumber
	Links  []content.KimTopic
}

// SetupGoal сохраняет учебную цель и создаёт первый недельный план.
func (s *Service) SetupGoal(ctx context.Context, userID uuid.UUID, in GoalInput) (*Plan, error) {
	examCode := strings.TrimSpace(in.ExamCode)
	if examCode == "" {
		return nil, ErrInvalidExamCode
	}
	examDate, err := time.Parse(dateLayout, strings.TrimSpace(in.ExamDate))
	if err != nil {
		return nil, ErrInvalidExamDate
	}
	now := s.clock.Now()
	if DateOnly(examDate).Before(DateOnly(now)) {
		return nil, ErrInvalidExamDate
	}
	if in.TargetScore < 0 || in.TargetScore > 100 {
		return nil, ErrInvalidTargetScore
	}
	if in.WeeklyMinutes < 1 || in.WeeklyMinutes > 10080 {
		return nil, ErrInvalidWeeklyMinutes
	}

	exam, err := s.content.GetExamByCode(ctx, examCode)
	if err != nil {
		return nil, ErrExamUnavailable
	}
	if exam.Status != content.StatusPublished {
		return nil, ErrExamUnavailable
	}

	if _, err := s.goals.SaveGoal(ctx, userID, profile.GoalInput{
		ExamID:        exam.ID.String(),
		ExamDate:      examDate.Format(dateLayout),
		TargetScore:   in.TargetScore,
		WeeklyMinutes: in.WeeklyMinutes,
	}); err != nil {
		return nil, err
	}

	structure, err := s.loadStructure(ctx, exam.ID)
	if err != nil {
		return nil, err
	}
	topicMastery, _, err := s.loadMastery(ctx, userID)
	if err != nil {
		return nil, err
	}
	diagnosticDone, err := s.repo.HasCompletedDiagnostic(ctx, s.db, userID, exam.ID)
	if err != nil {
		return nil, err
	}

	priorities := BuildPriorities(structure.Topics, ComputeTopicWeights(structure.Topics, structure.Kims, structure.Links), topicMastery, now)
	blueprints := BuildWeekBlueprints(now, examDate, in.WeeklyMinutes, priorities, diagnosticDone)

	version := 1
	var oldWeeks []Week
	var oldItems []PlanItem
	activePlan, err := s.repo.GetActivePlan(ctx, s.db, userID)
	if err == nil {
		version = activePlan.Version + 1
		oldWeeks, oldItems, err = s.loadPlanTree(ctx, activePlan.ID)
		if err != nil {
			return nil, err
		}
	} else if !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	plan := &Plan{
		ID:              uuid.Nil,
		UserID:          userID,
		ExamID:          exam.ID,
		ExamDate:        DateOnly(examDate),
		TargetScore:     in.TargetScore,
		WeeklyMinutes:   in.WeeklyMinutes,
		Status:          PlanActive,
		Version:         version,
		LastReplannedAt: &now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	plan.ID, err = newID()
	if err != nil {
		return nil, err
	}

	weeks, items, err := s.materializePlan(*plan, oldWeeks, oldItems, blueprints)
	if err != nil {
		return nil, err
	}

	if err := s.tx.WithTx(ctx, func(q db.Querier) error {
		if err := s.repo.SupersedeActivePlans(ctx, q, userID, now); err != nil {
			return err
		}
		if err := s.repo.CreatePlan(ctx, q, plan); err != nil {
			return err
		}
		if err := s.repo.CreateWeeks(ctx, q, weeks); err != nil {
			return err
		}

		return s.repo.CreateItems(ctx, q, items)
	}); err != nil {
		return nil, err
	}

	return plan, nil
}

// Today собирает данные экрана «Сегодня».
func (s *Service) Today(ctx context.Context, userID uuid.UUID) (TodayData, error) {
	goal, err := s.goals.GetCurrentGoal(ctx, userID)
	if errors.Is(err, profile.ErrNotFound) {
		return TodayData{
			HasGoal:  false,
			Empty:    true,
			NextStep: "Задайте цель: выберите трек и дату экзамена.",
			NextHref: "/app/goal",
		}, nil
	}
	if err != nil {
		return TodayData{}, err
	}

	data := TodayData{
		HasGoal:  true,
		Goal:     fmt.Sprintf("Цель: %d баллов к %s", goal.TargetScore, goal.ExamDate.Format("02.01.2006")),
		GoalMax:  100,
		NextHref: "/app/plan",
	}

	structure, err := s.loadStructure(ctx, goal.ExamID)
	if err != nil {
		return TodayData{}, err
	}
	topicMastery, kimMastery, err := s.loadMastery(ctx, userID)
	if err != nil {
		return TodayData{}, err
	}
	gaps := BuildGapMap(structure.Topics, structure.Kims, structure.Links, topicMastery, kimMastery, s.clock.Now())
	data.Gaps = gaps.Recommendations

	plan, err := s.repo.GetActivePlan(ctx, s.db, userID)
	if errors.Is(err, ErrNotFound) {
		data.NextStep = "План ещё не создан. Задайте цель заново."
		data.NextHref = "/app/goal"
		return data, nil
	}
	if err != nil {
		return TodayData{}, err
	}
	data.HasPlan = true

	plan, err = s.maybeReplan(ctx, userID, plan)
	if err != nil {
		return TodayData{}, err
	}
	weeks, items, err := s.loadPlanTree(ctx, plan.ID)
	if err != nil {
		return TodayData{}, err
	}
	data.NextStep, data.NextHref = nextStep(plan, weeks, items)
	done, total := doneCount(items)
	if total > 0 {
		data.GoalCurrent = done * 100 / total
	}

	return data, nil
}

// GoalPageData возвращает данные формы учебной цели.
func (s *Service) GoalPageData(ctx context.Context, userID uuid.UUID) (GoalPageData, error) {
	exams, err := s.content.ListPublishedExams(ctx)
	if err != nil {
		return GoalPageData{}, err
	}
	data := GoalPageData{Exams: make([]ExamOption, 0, len(exams))}
	for _, exam := range exams {
		data.Exams = append(data.Exams, ExamOption{Code: exam.Code, Title: exam.Title})
	}

	goal, err := s.goals.GetCurrentGoal(ctx, userID)
	if errors.Is(err, profile.ErrNotFound) {
		return data, nil
	}
	if err != nil {
		return GoalPageData{}, err
	}

	data.AlreadyHasGoal = true
	data.Input = GoalInput{
		ExamDate:      goal.ExamDate.Format(dateLayout),
		TargetScore:   goal.TargetScore,
		WeeklyMinutes: goal.WeeklyMinutes,
	}
	if exam, err := s.content.GetExam(ctx, goal.ExamID); err == nil {
		data.Input.ExamCode = exam.Code
	}

	return data, nil
}

// PrepareDiagnostic рассчитывает выборку без создания сессии.
func (s *Service) PrepareDiagnostic(ctx context.Context, userID uuid.UUID, examCode string) (DiagnosticStartData, error) {
	exam, err := s.resolveExam(ctx, userID, examCode)
	if err != nil {
		return DiagnosticStartData{}, err
	}

	active, err := s.repo.GetInProgressSession(ctx, s.db, userID, exam.ID, SessionKindDiagnostic)
	if err == nil {
		return DiagnosticStartData{Available: true, HasActive: true, ActiveSessionID: active.ID}, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return DiagnosticStartData{}, err
	}

	sample, duration, err := s.buildSample(ctx, exam.ID)
	if errors.Is(err, ErrInsufficientContent) {
		return DiagnosticStartData{
			Available: false,
			Reason:    "Недостаточно опубликованных заданий, чтобы покрыть все номера КИМ. Попробуйте позже.",
		}, nil
	}
	if err != nil {
		return DiagnosticStartData{}, err
	}

	return DiagnosticStartData{
		Available:       true,
		DurationMinutes: duration,
		TaskCount:       len(sample),
	}, nil
}

// StartDiagnostic создаёт или возобновляет диагностическую сессию.
func (s *Service) StartDiagnostic(ctx context.Context, userID uuid.UUID, examCode string) (*Session, error) {
	exam, err := s.resolveExam(ctx, userID, examCode)
	if err != nil {
		return nil, err
	}

	active, err := s.repo.GetInProgressSession(ctx, s.db, userID, exam.ID, SessionKindDiagnostic)
	if err == nil {
		return active, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	sample, _, err := s.buildSample(ctx, exam.ID)
	if err != nil {
		return nil, err
	}

	structure, err := s.loadStructure(ctx, exam.ID)
	if err != nil {
		return nil, err
	}
	kimByID := make(map[uuid.UUID]content.KimNumber, len(structure.Kims))
	for _, kim := range structure.Kims {
		kimByID[kim.ID] = kim
	}

	now := s.clock.Now()
	session := &Session{
		UserID:    userID,
		ExamID:    exam.ID,
		Kind:      SessionKindDiagnostic,
		Status:    SessionInProgress,
		StartedAt: now,
		CreatedAt: now,
	}
	session.ID, err = newID()
	if err != nil {
		return nil, err
	}

	items := make([]SessionItem, 0, len(sample))
	for index, task := range sample {
		kim := kimByID[task.KimNumberID]
		attachments, err := s.content.ListTaskAttachments(ctx, task.ID)
		if err != nil {
			return nil, err
		}
		options, err := s.content.ListTaskOptions(ctx, task.ID)
		if err != nil {
			return nil, err
		}
		snapshot := buildTaskSnapshot(task, kim, attachments, options)
		raw, err := json.Marshal(snapshot)
		if err != nil {
			return nil, fmt.Errorf("собрать снимок задания: %w", err)
		}

		itemID, err := newID()
		if err != nil {
			return nil, err
		}
		items = append(items, SessionItem{
			ID:           itemID,
			SessionID:    session.ID,
			Position:     index + 1,
			TaskID:       task.ID,
			TaskSnapshot: raw,
			MaxPoints:    snapshot.MaxPoints,
			Status:       SessionItemPending,
		})
	}

	if err := s.tx.WithTx(ctx, func(q db.Querier) error {
		if err := s.repo.CreateSession(ctx, q, session); err != nil {
			return err
		}

		return s.repo.CreateSessionItems(ctx, q, items)
	}); err != nil {
		return nil, err
	}

	return session, nil
}

// SessionView собирает экран прохождения диагностики.
func (s *Service) SessionView(ctx context.Context, userID, sessionID uuid.UUID) (SessionView, error) {
	session, err := s.repo.GetSessionByUser(ctx, s.db, sessionID, userID)
	if err != nil {
		return SessionView{}, err
	}
	items, err := s.repo.ListSessionItems(ctx, s.db, sessionID)
	if err != nil {
		return SessionView{}, err
	}
	answers, err := s.repo.ListAnswers(ctx, s.db, sessionID)
	if err != nil {
		return SessionView{}, err
	}
	byItem := make(map[uuid.UUID]Answer, len(answers))
	for _, answer := range answers {
		byItem[answer.SessionItemID] = answer
	}

	view := SessionView{Session: *session, Items: make([]SessionItemView, 0, len(items))}
	current := 0
	foundCurrent := false
	for index, item := range items {
		var snapshot TaskSnapshot
		if err := json.Unmarshal(item.TaskSnapshot, &snapshot); err != nil {
			return SessionView{}, fmt.Errorf("разобрать снимок задания: %w", err)
		}
		entry := SessionItemView{Item: item, Snapshot: snapshot}
		if answer, ok := byItem[item.ID]; ok {
			copied := answer
			entry.Answer = &copied
		} else if !foundCurrent {
			current = index
			foundCurrent = true
		}
		view.Items = append(view.Items, entry)
	}
	if !foundCurrent && len(items) > 0 {
		current = len(items) - 1
	}
	view.CurrentIndex = current
	view.Completed = session.Status == SessionCompleted
	if view.Completed {
		result, err := s.repo.GetSessionResult(ctx, s.db, sessionID)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return SessionView{}, err
		}
		view.Result = result
	}

	return view, nil
}

// SessionKind возвращает вид сессии владельца.
func (s *Service) SessionKind(ctx context.Context, userID, sessionID uuid.UUID) (string, error) {
	session, err := s.repo.GetSessionByUser(ctx, s.db, sessionID, userID)
	if err != nil {
		return "", err
	}

	return session.Kind, nil
}

// SaveAnswer проверяет и сохраняет ответ ученика.
func (s *Service) SaveAnswer(ctx context.Context, userID, sessionID, itemID uuid.UUID, raw string) (AnswerResult, error) {
	session, err := s.repo.GetSessionByUser(ctx, s.db, sessionID, userID)
	if err != nil {
		return AnswerResult{}, err
	}
	if session.Status == SessionCompleted {
		return AnswerResult{}, ErrSessionCompleted
	}

	item, err := s.repo.GetSessionItem(ctx, s.db, sessionID, itemID)
	if err != nil {
		return AnswerResult{}, err
	}
	var snapshot TaskSnapshot
	if err := json.Unmarshal(item.TaskSnapshot, &snapshot); err != nil {
		return AnswerResult{}, fmt.Errorf("разобрать снимок задания: %w", err)
	}

	checked, awarded, err := evaluateAnswer(snapshot, raw)
	if err != nil {
		return AnswerResult{}, err
	}
	now := s.clock.Now()
	answer := &Answer{
		SessionItemID:    item.ID,
		RawAnswer:        strings.TrimSpace(raw),
		NormalizedAnswer: checked.NormalizedAnswer,
		IsCorrect:        checked.Correct,
		AwardedPoints:    awarded,
		CheckKind:        snapshotDifficultyKind(snapshot.AnswerSpec),
		CheckedAt:        now,
		CreatedAt:        now,
	}
	answer.ID, err = newID()
	if err != nil {
		return AnswerResult{}, err
	}

	if err := s.tx.WithTx(ctx, func(q db.Querier) error {
		if err := s.repo.UpsertAnswer(ctx, q, answer); err != nil {
			return err
		}

		return s.repo.MarkSessionItemAnswered(ctx, q, item.ID)
	}); err != nil {
		return AnswerResult{}, err
	}

	result := AnswerResult{
		Correct:    checked.Correct,
		Reason:     reasonLabel(checked.Correct),
		Normalized: checked.NormalizedAnswer,
		Expected:   checked.Expected,
		NextHref:   s.nextItemHref(ctx, sessionID, session.Kind, item.Position),
	}

	return result, nil
}

// CompleteSession завершает сессию, сохраняет результат и пересчитывает план.
func (s *Service) CompleteSession(ctx context.Context, userID, sessionID uuid.UUID) (*SessionResult, error) {
	session, err := s.repo.GetSessionByUser(ctx, s.db, sessionID, userID)
	if err != nil {
		return nil, err
	}
	if session.Status == SessionCompleted {
		result, err := s.repo.GetSessionResult(ctx, s.db, sessionID)
		if err != nil {
			return nil, err
		}

		return result, nil
	}

	items, err := s.repo.ListSessionItems(ctx, s.db, sessionID)
	if err != nil {
		return nil, err
	}
	answers, err := s.repo.ListAnswers(ctx, s.db, sessionID)
	if err != nil {
		return nil, err
	}
	answerByItem := make(map[uuid.UUID]Answer, len(answers))
	for _, answer := range answers {
		answerByItem[answer.SessionItemID] = answer
	}

	now := s.clock.Now()
	result, payload, touchedTopics, touchedKims := buildResult(sessionID, items, answerByItem)

	topicMastery, kimMastery, err := s.loadMastery(ctx, userID)
	if err != nil {
		return nil, err
	}
	applyMastery(userID, topicMastery, kimMastery, items, answerByItem, now)
	if session.Kind == SessionKindPractice || session.Kind == SessionKindReview {
		applyIntervals(userID, topicMastery, items, answerByItem, now)
	}

	var (
		newPlan   *Plan
		weeks     []Week
		planItems []PlanItem
	)
	if session.Kind == SessionKindDiagnostic {
		structure, err := s.loadStructure(ctx, session.ExamID)
		if err != nil {
			return nil, err
		}
		priorities := BuildPriorities(structure.Topics, ComputeTopicWeights(structure.Topics, structure.Kims, structure.Links), topicMastery, now)

		activePlan, err := s.repo.GetActivePlan(ctx, s.db, userID)
		hasPlan := true
		if errors.Is(err, ErrNotFound) {
			hasPlan = false
		} else if err != nil {
			return nil, err
		}
		if hasPlan {
			oldWeeks, oldItems, err := s.loadPlanTree(ctx, activePlan.ID)
			if err != nil {
				return nil, err
			}
			newPlan, err = clonePlan(*activePlan, now)
			if err != nil {
				return nil, err
			}
			blueprints := BuildWeekBlueprints(now, newPlan.ExamDate, newPlan.WeeklyMinutes, priorities, true)
			weeks, planItems, err = s.materializePlan(*newPlan, oldWeeks, oldItems, blueprints)
			if err != nil {
				return nil, err
			}
		}
	}

	if err := s.tx.WithTx(ctx, func(q db.Querier) error {
		if err := s.repo.CompleteSession(ctx, q, sessionID, now, result.PrimaryScore); err != nil {
			return err
		}
		if err := s.repo.UpsertSessionResult(ctx, q, result); err != nil {
			return err
		}
		for _, topicID := range touchedTopics {
			record := topicMastery[topicID]
			if err := s.repo.UpsertTopicMastery(ctx, q, &record); err != nil {
				return err
			}
		}
		for _, kimID := range touchedKims {
			record := kimMastery[kimID]
			if err := s.repo.UpsertKimMastery(ctx, q, &record); err != nil {
				return err
			}
		}
		if session.PlanItemID != nil {
			if err := s.repo.MarkItemDone(ctx, q, *session.PlanItemID, userID); err != nil && !errors.Is(err, ErrNotFound) {
				return err
			}
		}
		if newPlan == nil {
			return nil
		}
		if err := s.repo.SupersedeActivePlans(ctx, q, userID, now); err != nil {
			return err
		}
		if err := s.repo.CreatePlan(ctx, q, newPlan); err != nil {
			return err
		}
		if err := s.repo.CreateWeeks(ctx, q, weeks); err != nil {
			return err
		}

		return s.repo.CreateItems(ctx, q, planItems)
	}); err != nil {
		return nil, err
	}

	result.Breakdown = payload

	return result, nil
}

// Result возвращает сохранённый результат завершённой сессии.
func (s *Service) Result(ctx context.Context, userID, sessionID uuid.UUID) (*ResultView, error) {
	session, err := s.repo.GetSessionByUser(ctx, s.db, sessionID, userID)
	if err != nil {
		return nil, err
	}
	if session.Status != SessionCompleted {
		return nil, ErrResultNotReady
	}
	result, err := s.repo.GetSessionResult(ctx, s.db, sessionID)
	if err != nil {
		return nil, err
	}

	var payload breakdownPayload
	if err := json.Unmarshal(result.Breakdown, &payload); err != nil {
		return nil, fmt.Errorf("разобрать результат сессии: %w", err)
	}

	structure, err := s.loadStructure(ctx, session.ExamID)
	if err != nil {
		return nil, err
	}
	topicTitles := make(map[uuid.UUID]string, len(structure.Topics))
	for _, topic := range structure.Topics {
		topicTitles[topic.ID] = topic.Title
	}
	kimNumbers := make(map[uuid.UUID]int, len(structure.Kims))
	for _, kim := range structure.Kims {
		kimNumbers[kim.ID] = kim.Number
	}

	view := &ResultView{Session: *session, Result: *result}
	view.Items = make([]BreakdownItem, 0, len(payload.Items))
	for _, entry := range payload.Items {
		title := ""
		if entry.TopicID != nil {
			title = topicTitles[*entry.TopicID]
		}
		view.Items = append(view.Items, BreakdownItem{
			Position:   entry.Position,
			KimNumber:  kimNumbers[entry.KimNumberID],
			TopicTitle: title,
			Correct:    entry.Correct,
			Awarded:    entry.Awarded,
			MaxPoints:  entry.MaxPoints,
			Reason:     entry.Reason,
			RawAnswer:  entry.RawAnswer,
		})
	}
	for _, entry := range payload.Topics {
		view.Topics = append(view.Topics, TopicBreakdown{TopicID: entry.TopicID, Title: topicTitles[entry.TopicID], Correct: entry.Correct, Total: entry.Total})
	}
	for _, entry := range payload.Kims {
		view.Kims = append(view.Kims, KimBreakdown{KimNumberID: entry.KimNumberID, Number: kimNumbers[entry.KimNumberID], Correct: entry.Correct, Total: entry.Total})
	}

	return view, nil
}

// Gaps собирает карту пробелов ученика.
func (s *Service) Gaps(ctx context.Context, userID uuid.UUID) (GapMapView, error) {
	goal, err := s.goals.GetCurrentGoal(ctx, userID)
	if errors.Is(err, profile.ErrNotFound) {
		return GapMapView{}, ErrNoGoal
	}
	if err != nil {
		return GapMapView{}, err
	}

	structure, err := s.loadStructure(ctx, goal.ExamID)
	if err != nil {
		return GapMapView{}, err
	}
	topicMastery, kimMastery, err := s.loadMastery(ctx, userID)
	if err != nil {
		return GapMapView{}, err
	}

	return BuildGapMap(structure.Topics, structure.Kims, structure.Links, topicMastery, kimMastery, s.clock.Now()), nil
}

// PlanView собирает экран недельного плана.
func (s *Service) PlanView(ctx context.Context, userID uuid.UUID) (PlanView, error) {
	goal, err := s.goals.GetCurrentGoal(ctx, userID)
	if errors.Is(err, profile.ErrNotFound) {
		return PlanView{HasGoal: false, Empty: true, Reason: "Цель не задана. Задайте цель, чтобы получить план."}, nil
	}
	if err != nil {
		return PlanView{}, err
	}

	if DateOnly(goal.ExamDate).Before(DateOnly(s.clock.Now())) {
		return PlanView{HasGoal: true, Empty: true, Reason: "Дата экзамена уже прошла. Обновите цель."}, nil
	}

	plan, err := s.repo.GetActivePlan(ctx, s.db, userID)
	if errors.Is(err, ErrNotFound) {
		return PlanView{HasGoal: true, Empty: true, Reason: "План ещё не создан. Задайте цель заново."}, nil
	}
	if err != nil {
		return PlanView{}, err
	}

	plan, err = s.maybeReplan(ctx, userID, plan)
	if err != nil {
		return PlanView{}, err
	}

	weeks, items, err := s.loadPlanTree(ctx, plan.ID)
	if err != nil {
		return PlanView{}, err
	}
	structure, err := s.loadStructure(ctx, plan.ExamID)
	if err != nil {
		return PlanView{}, err
	}
	topicTitles := make(map[uuid.UUID]string, len(structure.Topics))
	for _, topic := range structure.Topics {
		topicTitles[topic.ID] = topic.Title
	}
	kimNumbers := make(map[uuid.UUID]int, len(structure.Kims))
	for _, kim := range structure.Kims {
		kimNumbers[kim.ID] = kim.Number
	}

	itemsByWeek := make(map[uuid.UUID][]PlanItem, len(weeks))
	for _, item := range items {
		itemsByWeek[item.WeekID] = append(itemsByWeek[item.WeekID], item)
	}

	view := PlanView{HasGoal: true, Plan: plan, Weeks: make([]WeekView, 0, len(weeks))}
	for _, week := range weeks {
		weekView := WeekView{Week: week, Items: make([]PlanItemView, 0, len(itemsByWeek[week.ID]))}
		for _, item := range itemsByWeek[week.ID] {
			entry := PlanItemView{Item: item, Done: item.Status == ItemDone}
			if item.TopicID != nil {
				entry.TopicTitle = topicTitles[*item.TopicID]
			}
			if item.KimNumberID != nil {
				entry.KimNumber = kimNumbers[*item.KimNumberID]
			}
			weekView.TotalLoad += item.TaskCount
			weekView.Items = append(weekView.Items, entry)
		}
		view.Weeks = append(view.Weeks, weekView)
	}
	view.Empty = len(view.Weeks) == 0

	return view, nil
}

// Replan принудительно пересобирает план.
func (s *Service) Replan(ctx context.Context, userID uuid.UUID) (*Plan, error) {
	plan, err := s.repo.GetActivePlan(ctx, s.db, userID)
	if errors.Is(err, ErrNotFound) {
		return nil, ErrNoPlan
	}
	if err != nil {
		return nil, err
	}

	return s.replan(ctx, userID, plan, true)
}

// MarkItemDone отмечает пункт плана выполненным.
func (s *Service) MarkItemDone(ctx context.Context, userID, itemID uuid.UUID) error {
	return s.repo.MarkItemDone(ctx, s.db, itemID, userID)
}

func (s *Service) maybeReplan(ctx context.Context, userID uuid.UUID, plan *Plan) (*Plan, error) {
	if plan.LastReplannedAt != nil && s.clock.Now().Sub(*plan.LastReplannedAt) < replanInterval {
		return plan, nil
	}

	replanned, err := s.replan(ctx, userID, plan, false)
	if err != nil {
		return nil, err
	}

	return replanned, nil
}

func (s *Service) replan(ctx context.Context, userID uuid.UUID, plan *Plan, force bool) (*Plan, error) {
	now := s.clock.Now()
	if !force && plan.LastReplannedAt != nil && now.Sub(*plan.LastReplannedAt) < replanInterval {
		return plan, nil
	}

	structure, err := s.loadStructure(ctx, plan.ExamID)
	if err != nil {
		return nil, err
	}
	topicMastery, _, err := s.loadMastery(ctx, userID)
	if err != nil {
		return nil, err
	}
	diagnosticDone, err := s.repo.HasCompletedDiagnostic(ctx, s.db, userID, plan.ExamID)
	if err != nil {
		return nil, err
	}

	oldWeeks, oldItems, err := s.loadPlanTree(ctx, plan.ID)
	if err != nil {
		return nil, err
	}

	priorities := BuildPriorities(structure.Topics, ComputeTopicWeights(structure.Topics, structure.Kims, structure.Links), topicMastery, now)
	blueprints := BuildWeekBlueprints(now, plan.ExamDate, plan.WeeklyMinutes, priorities, diagnosticDone)

	newPlan, err := clonePlan(*plan, now)
	if err != nil {
		return nil, err
	}
	weeks, items, err := s.materializePlan(*newPlan, oldWeeks, oldItems, blueprints)
	if err != nil {
		return nil, err
	}

	currentWeek := WeekStart(now)
	if planSignature(oldWeeks, oldItems, currentWeek) == planSignature(weeks, items, currentWeek) {
		if err := s.repo.UpdatePlanLastReplanned(ctx, s.db, plan.ID, now); err != nil {
			return nil, err
		}
		plan.LastReplannedAt = &now
		plan.UpdatedAt = now

		return plan, nil
	}

	if err := s.tx.WithTx(ctx, func(q db.Querier) error {
		if err := s.repo.SupersedeActivePlans(ctx, q, userID, now); err != nil {
			return err
		}
		if err := s.repo.CreatePlan(ctx, q, newPlan); err != nil {
			return err
		}
		if err := s.repo.CreateWeeks(ctx, q, weeks); err != nil {
			return err
		}

		return s.repo.CreateItems(ctx, q, items)
	}); err != nil {
		return nil, err
	}

	return newPlan, nil
}

func (s *Service) resolveExam(ctx context.Context, userID uuid.UUID, examCode string) (*content.Exam, error) {
	code := strings.TrimSpace(examCode)
	var (
		exam *content.Exam
		err  error
	)
	if code != "" {
		exam, err = s.content.GetExamByCode(ctx, code)
	} else {
		goal, goalErr := s.goals.GetCurrentGoal(ctx, userID)
		if goalErr != nil {
			return nil, ErrNoGoal
		}
		exam, err = s.content.GetExam(ctx, goal.ExamID)
	}
	if err != nil || exam.Status != content.StatusPublished {
		return nil, ErrExamUnavailable
	}

	return exam, nil
}

func (s *Service) buildSample(ctx context.Context, examID uuid.UUID) ([]content.Task, int, error) {
	year, err := s.content.GetLatestPublishedExamYear(ctx, examID)
	if err != nil {
		return nil, 0, ErrInsufficientContent
	}
	yearID := year.ID
	numbers, err := s.content.ListKimNumbers(ctx, content.KimNumberFilter{ExamYearID: &yearID, Limit: contentPageLimit})
	if err != nil {
		return nil, 0, err
	}
	tasks, err := s.listAllPublicTasks(ctx, yearID)
	if err != nil {
		return nil, 0, err
	}

	return BuildDiagnosticSample(numbers, tasks)
}

func (s *Service) listAllPublicTasks(ctx context.Context, examYearID uuid.UUID) ([]content.Task, error) {
	var tasks []content.Task
	cursor := uuid.Nil
	for {
		page, err := s.content.ListPublicTasks(ctx, content.TaskFilter{
			Cursor:               cursor,
			ExamYearID:           &examYearID,
			Kinds:                supportedKinds(),
			ExcludePublishedMock: true,
			Limit:                contentPageLimit,
		})
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, page...)
		if len(page) < contentPageLimit {
			return tasks, nil
		}
		cursor = page[len(page)-1].ID
	}
}

func supportedKinds() []string {
	return grading.CheckableKinds()
}

func (s *Service) loadStructure(ctx context.Context, examID uuid.UUID) (*examStructure, error) {
	exam, err := s.content.GetExam(ctx, examID)
	if err != nil {
		return nil, ErrExamUnavailable
	}
	year, err := s.content.GetLatestPublishedExamYear(ctx, examID)
	if err != nil {
		return nil, ErrInsufficientContent
	}
	examIDCopy := examID
	topics, err := s.content.ListTopics(ctx, content.TopicFilter{ExamID: &examIDCopy, Limit: contentPageLimit})
	if err != nil {
		return nil, err
	}
	yearID := year.ID
	kims, err := s.content.ListKimNumbers(ctx, content.KimNumberFilter{ExamYearID: &yearID, Limit: contentPageLimit})
	if err != nil {
		return nil, err
	}
	links, err := s.content.ListKimTopicsByExamYear(ctx, yearID)
	if err != nil {
		return nil, err
	}

	return &examStructure{Exam: *exam, Year: *year, Topics: topics, Kims: kims, Links: links}, nil
}

func (s *Service) loadMastery(ctx context.Context, userID uuid.UUID) (map[uuid.UUID]TopicMastery, map[uuid.UUID]KimMastery, error) {
	topicList, err := s.repo.ListTopicMastery(ctx, s.db, userID)
	if err != nil {
		return nil, nil, err
	}
	kimList, err := s.repo.ListKimMastery(ctx, s.db, userID)
	if err != nil {
		return nil, nil, err
	}

	topics := make(map[uuid.UUID]TopicMastery, len(topicList))
	for _, item := range topicList {
		topics[item.TopicID] = item
	}
	kims := make(map[uuid.UUID]KimMastery, len(kimList))
	for _, item := range kimList {
		kims[item.KimNumberID] = item
	}

	return topics, kims, nil
}

func (s *Service) loadPlanTree(ctx context.Context, planID uuid.UUID) ([]Week, []PlanItem, error) {
	weeks, err := s.repo.ListWeeks(ctx, s.db, planID)
	if err != nil {
		return nil, nil, err
	}
	items, err := s.repo.ListItemsByPlan(ctx, s.db, planID)
	if err != nil {
		return nil, nil, err
	}

	return weeks, items, nil
}

func (s *Service) materializePlan(plan Plan, oldWeeks []Week, oldItems []PlanItem, blueprints []WeekBlueprint) ([]Week, []PlanItem, error) {
	currentWeek := WeekStart(plan.CreatedAt)
	startByWeek := make(map[uuid.UUID]time.Time, len(oldWeeks))
	for _, week := range oldWeeks {
		startByWeek[week.ID] = week.WeekStart
	}
	doneByStart := make(map[time.Time][]PlanItem)
	for _, item := range oldItems {
		start, ok := startByWeek[item.WeekID]
		if !ok || item.Status != ItemDone || start.Before(currentWeek) {
			continue
		}
		doneByStart[start] = append(doneByStart[start], item)
	}

	var weeks []Week
	var items []PlanItem
	for _, week := range oldWeeks {
		if !week.WeekStart.Before(currentWeek) {
			continue
		}
		id, err := newID()
		if err != nil {
			return nil, nil, err
		}
		copied := Week{ID: id, PlanID: plan.ID, WeekStart: week.WeekStart, Phase: week.Phase, Goals: week.Goals}
		weeks = append(weeks, copied)
		for _, item := range oldItems {
			if item.WeekID != week.ID {
				continue
			}
			copiedItem, err := copyItem(item, copied.ID)
			if err != nil {
				return nil, nil, err
			}
			items = append(items, copiedItem)
		}
	}

	for _, blueprint := range blueprints {
		id, err := newID()
		if err != nil {
			return nil, nil, err
		}
		goals, err := json.Marshal(map[string]string{"phase": string(blueprint.Phase)})
		if err != nil {
			return nil, nil, err
		}
		week := Week{ID: id, PlanID: plan.ID, WeekStart: blueprint.WeekStart, Phase: blueprint.Phase, Goals: goals}
		weeks = append(weeks, week)

		occupied := make(map[int]struct{})
		for _, item := range doneByStart[blueprint.WeekStart] {
			copiedItem, err := copyItem(item, week.ID)
			if err != nil {
				return nil, nil, err
			}
			items = append(items, copiedItem)
			occupied[item.DayOfWeek] = struct{}{}
		}
		for _, entry := range blueprint.Items {
			if _, ok := occupied[entry.DayOfWeek]; ok {
				continue
			}
			itemID, err := newID()
			if err != nil {
				return nil, nil, err
			}
			items = append(items, PlanItem{
				ID:          itemID,
				WeekID:      week.ID,
				DayOfWeek:   entry.DayOfWeek,
				Kind:        entry.Kind,
				TopicID:     entry.TopicID,
				KimNumberID: entry.KimNumberID,
				TaskCount:   entry.TaskCount,
				Status:      ItemPending,
			})
		}
	}

	return weeks, items, nil
}

func (s *Service) nextItemHref(ctx context.Context, sessionID uuid.UUID, kind string, position int) string {
	base := sessionPath(kind)
	items, err := s.repo.ListSessionItems(ctx, s.db, sessionID)
	if err != nil {
		return fmt.Sprintf("%s/%s", base, sessionID)
	}
	for _, item := range items {
		if item.Position > position && item.Status != SessionItemAnswered {
			return fmt.Sprintf("%s/%s?item=%s", base, sessionID, item.ID)
		}
	}

	return fmt.Sprintf("%s/%s", base, sessionID)
}

func sessionPath(kind string) string {
	switch kind {
	case SessionKindPractice:
		return "/app/practice"
	case SessionKindReview:
		return "/app/review"
	default:
		return "/app/diagnostics"
	}
}

func buildResult(sessionID uuid.UUID, items []SessionItem, answers map[uuid.UUID]Answer) (*SessionResult, json.RawMessage, []uuid.UUID, []uuid.UUID) {
	payload := breakdownPayload{}
	topicTotals := make(map[uuid.UUID]*topicEntry)
	kimTotals := make(map[uuid.UUID]*kimEntry)
	touchedTopics := make(map[uuid.UUID]struct{})
	touchedKims := make(map[uuid.UUID]struct{})

	var primary, maxScore float64
	for _, item := range items {
		var snapshot TaskSnapshot
		_ = json.Unmarshal(item.TaskSnapshot, &snapshot)
		answer, answered := answers[item.ID]
		correct := answered && answer.IsCorrect
		awarded := 0.0
		raw := ""
		reason := "нет ответа"
		if answered {
			awarded = answer.AwardedPoints
			raw = answer.RawAnswer
			reason = reasonLabel(answer.IsCorrect)
		}
		primary += awarded
		maxScore += item.MaxPoints

		payload.Items = append(payload.Items, breakdownEntry{
			Position:    item.Position,
			TaskID:      item.TaskID,
			KimNumberID: snapshot.KimNumberID,
			TopicID:     snapshot.TopicID,
			Correct:     correct,
			Awarded:     awarded,
			MaxPoints:   item.MaxPoints,
			Reason:      reason,
			RawAnswer:   raw,
		})

		kimTotal := kimTotals[snapshot.KimNumberID]
		if kimTotal == nil {
			kimTotal = &kimEntry{KimNumberID: snapshot.KimNumberID}
			kimTotals[snapshot.KimNumberID] = kimTotal
		}
		kimTotal.Total++
		if correct {
			kimTotal.Correct++
		}
		touchedKims[snapshot.KimNumberID] = struct{}{}

		if snapshot.TopicID != nil {
			topicTotal := topicTotals[*snapshot.TopicID]
			if topicTotal == nil {
				topicTotal = &topicEntry{TopicID: *snapshot.TopicID}
				topicTotals[*snapshot.TopicID] = topicTotal
			}
			topicTotal.Total++
			if correct {
				topicTotal.Correct++
			}
			touchedTopics[*snapshot.TopicID] = struct{}{}
		}
	}

	for _, entry := range topicTotals {
		payload.Topics = append(payload.Topics, *entry)
	}
	sort.Slice(payload.Topics, func(i, j int) bool { return payload.Topics[i].TopicID.String() < payload.Topics[j].TopicID.String() })
	for _, entry := range kimTotals {
		payload.Kims = append(payload.Kims, *entry)
	}
	sort.Slice(payload.Kims, func(i, j int) bool {
		return payload.Kims[i].KimNumberID.String() < payload.Kims[j].KimNumberID.String()
	})

	breakdown, _ := json.Marshal(payload)
	result := &SessionResult{
		SessionID:    sessionID,
		PrimaryScore: primary,
		MaxScore:     maxScore,
		Breakdown:    breakdown,
	}

	return result, breakdown, keys(touchedTopics), keys(touchedKims)
}

func applyMastery(
	userID uuid.UUID,
	topicMastery map[uuid.UUID]TopicMastery,
	kimMastery map[uuid.UUID]KimMastery,
	items []SessionItem,
	answers map[uuid.UUID]Answer,
	now time.Time,
) {
	for _, item := range items {
		answer, ok := answers[item.ID]
		if !ok {
			continue
		}
		var snapshot TaskSnapshot
		if err := json.Unmarshal(item.TaskSnapshot, &snapshot); err != nil {
			continue
		}

		if snapshot.TopicID != nil {
			record := topicMastery[*snapshot.TopicID]
			record.UserID = userID
			record.TopicID = *snapshot.TopicID
			if record.IntervalDays == 0 {
				record.IntervalDays = defaultIntervalDays
			}
			if record.Ease == 0 {
				record.Ease = defaultEase
			}
			record.Mastery = NextMastery(record.Mastery, answer.IsCorrect, snapshot.Difficulty)
			record.Attempts++
			if answer.IsCorrect {
				record.Correct++
			}
			record.LastPracticedAt = &now
			topicMastery[*snapshot.TopicID] = record
		}

		record := kimMastery[snapshot.KimNumberID]
		record.UserID = userID
		record.KimNumberID = snapshot.KimNumberID
		record.Mastery = NextMastery(record.Mastery, answer.IsCorrect, snapshot.Difficulty)
		record.Attempts++
		if answer.IsCorrect {
			record.Correct++
		}
		record.LastPracticedAt = &now
		kimMastery[snapshot.KimNumberID] = record
	}
}

func clonePlan(plan Plan, now time.Time) (*Plan, error) {
	cloned := plan
	cloned.Status = PlanActive
	cloned.Version = plan.Version + 1
	cloned.LastReplannedAt = &now
	cloned.CreatedAt = now
	cloned.UpdatedAt = now
	id, err := newID()
	if err != nil {
		return nil, err
	}
	cloned.ID = id

	return &cloned, nil
}

func copyItem(item PlanItem, weekID uuid.UUID) (PlanItem, error) {
	id, err := newID()
	if err != nil {
		return PlanItem{}, err
	}
	copied := item
	copied.ID = id
	copied.WeekID = weekID

	return copied, nil
}

func buildTaskSnapshot(task content.Task, kim content.KimNumber, attachments []content.TaskAttachment, options []content.TaskOption) TaskSnapshot {
	return TaskSnapshot{
		TaskID:        task.ID,
		Version:       task.Version,
		StatementMD:   task.StatementMD,
		AnswerSpec:    task.AnswerSpec,
		AnswerDisplay: task.AnswerDisplay,
		Difficulty:    string(task.Difficulty),
		KimNumberID:   task.KimNumberID,
		TopicID:       task.TopicID,
		MaxPoints:     float64(kim.MaxPrimaryScore),
		Attachments:   attachmentSnapshots(attachments),
		Options:       optionSnapshots(options),
	}
}

func optionSnapshots(options []content.TaskOption) []OptionSnapshot {
	result := make([]OptionSnapshot, 0, len(options))
	for _, option := range options {
		result = append(result, OptionSnapshot{ID: option.ID, Position: option.Position, ContentMD: option.ContentMD})
	}

	return result
}

func evaluateAnswer(snapshot TaskSnapshot, raw string) (grading.Result, float64, error) {
	checked, err := grading.Check(snapshot.AnswerSpec, raw)
	if err != nil {
		return grading.Result{}, 0, invalid("answer", "Не удалось проверить ответ. Проверьте формат.")
	}
	awarded := 0.0
	if checked.Correct {
		awarded = snapshot.MaxPoints
	}

	return checked, awarded, nil
}

func attachmentSnapshots(attachments []content.TaskAttachment) []AttachmentSnapshot {
	result := make([]AttachmentSnapshot, 0, len(attachments))
	for _, attachment := range attachments {
		result = append(result, AttachmentSnapshot{Kind: attachment.Kind, URL: attachment.URL, Alt: attachment.Alt})
	}

	return result
}

func planSignature(weeks []Week, items []PlanItem, currentWeek time.Time) string {
	startByWeek := make(map[uuid.UUID]time.Time, len(weeks))
	phaseByWeek := make(map[uuid.UUID]Phase, len(weeks))
	for _, week := range weeks {
		startByWeek[week.ID] = week.WeekStart
		phaseByWeek[week.ID] = week.Phase
	}

	var parts []string
	for _, item := range items {
		start, ok := startByWeek[item.WeekID]
		if !ok || start.Before(currentWeek) {
			continue
		}
		topic := ""
		if item.TopicID != nil {
			topic = item.TopicID.String()
		}
		kim := ""
		if item.KimNumberID != nil {
			kim = item.KimNumberID.String()
		}
		parts = append(parts, fmt.Sprintf("%s|%s|%d|%s|%s|%s|%d|%s",
			start.Format(dateLayout), phaseByWeek[item.WeekID], item.DayOfWeek, item.Kind, topic, kim, item.TaskCount, item.Status))
	}
	sort.Strings(parts)

	return strings.Join(parts, ";")
}

func nextStep(plan *Plan, weeks []Week, items []PlanItem) (string, string) {
	if plan == nil {
		return "Задайте цель.", "/app/goal"
	}
	weekStart := make(map[uuid.UUID]time.Time, len(weeks))
	weekPhase := make(map[uuid.UUID]Phase, len(weeks))
	for _, week := range weeks {
		weekStart[week.ID] = week.WeekStart
		weekPhase[week.ID] = week.Phase
	}

	sort.SliceStable(items, func(i, j int) bool {
		leftStart := weekStart[items[i].WeekID]
		rightStart := weekStart[items[j].WeekID]
		if !leftStart.Equal(rightStart) {
			return leftStart.Before(rightStart)
		}

		return items[i].DayOfWeek < items[j].DayOfWeek
	})

	for _, item := range items {
		if item.Status == ItemDone {
			continue
		}
		switch item.Kind {
		case KindDiagnostic:
			return "Пройдите диагностику, чтобы построить карту пробелов.", "/app/diagnostics"
		case KindReview:
			return "Повторите темы с наибольшим приоритетом.", "/app/review"
		case KindPractice:
			return "Практика по темам из плана.", "/app/practice?plan=" + item.ID.String()
		default:
			phase := weekPhase[item.WeekID]
			if phase == PhasePractice {
				return "Практика по темам из плана.", "/app/practice"
			}

			return "Изучите теорию и закрепите материал.", "/app/plan"
		}
	}

	return "Все пункты плана выполнены. Поддерживайте форму практикой.", "/app/gaps"
}

func doneCount(items []PlanItem) (int, int) {
	done := 0
	for _, item := range items {
		if item.Status == ItemDone {
			done++
		}
	}

	return done, len(items)
}

func reasonLabel(correct bool) string {
	if correct {
		return "совпало"
	}

	return "не совпало"
}

func snapshotDifficultyKind(spec json.RawMessage) string {
	var payload struct {
		Kind string `json:"kind"`
	}
	_ = json.Unmarshal(spec, &payload)

	return payload.Kind
}

func keys[V any](items map[uuid.UUID]V) []uuid.UUID {
	result := make([]uuid.UUID, 0, len(items))
	for key := range items {
		result = append(result, key)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].String() < result[j].String() })

	return result
}

func newID() (uuid.UUID, error) {
	id, err := ids.New()
	if err != nil {
		return uuid.Nil, fmt.Errorf("сгенерировать идентификатор: %w", err)
	}

	return id, nil
}
