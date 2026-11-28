package learning

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"oge-ege/internal/content"
	"oge-ege/internal/grading"
	"oge-ege/internal/shared/db"
)

const (
	defaultPracticeSize = 10
	maxPracticeSize     = 20
)

// PracticeStartData собирает варианты запуска практики.
func (s *Service) PracticeStartData(ctx context.Context, userID uuid.UUID) (PracticeStartData, error) {
	exam, err := s.resolveExam(ctx, userID, "")
	if err != nil {
		return PracticeStartData{}, err
	}

	active, err := s.repo.GetInProgressSession(ctx, s.db, userID, exam.ID, SessionKindPractice)
	if err == nil {
		return PracticeStartData{Available: true, HasActive: true, ActiveSessionID: active.ID}, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return PracticeStartData{}, err
	}

	structure, err := s.loadStructure(ctx, exam.ID)
	if err != nil {
		return PracticeStartData{}, err
	}

	data := PracticeStartData{Available: true, Choices: make([]PracticeChoice, 0, len(structure.Topics)+len(structure.Kims))}
	if plan, err := s.repo.GetActivePlan(ctx, s.db, userID); err == nil {
		weeks, items, err := s.loadPlanTree(ctx, plan.ID)
		if err != nil {
			return PracticeStartData{}, err
		}
		weekStart := make(map[uuid.UUID]string, len(weeks))
		for _, week := range weeks {
			weekStart[week.ID] = formatDate(week.WeekStart)
		}
		topicTitles := make(map[uuid.UUID]string, len(structure.Topics))
		for _, topic := range structure.Topics {
			topicTitles[topic.ID] = topic.Title
		}
		kimNumbers := make(map[uuid.UUID]int, len(structure.Kims))
		for _, kim := range structure.Kims {
			kimNumbers[kim.ID] = kim.Number
		}
		for _, item := range items {
			if item.Status == ItemDone || item.TaskCount <= 0 {
				continue
			}
			if item.Kind != KindPractice && item.Kind != KindTheory {
				continue
			}
			label := fmt.Sprintf("План: %s — %s", weekStart[item.WeekID], planItemTarget(item, topicTitles, kimNumbers))
			data.Choices = append(data.Choices, PracticeChoice{Value: "plan:" + item.ID.String(), Label: label})
		}
	} else if !errors.Is(err, ErrNotFound) {
		return PracticeStartData{}, err
	}

	for _, topic := range structure.Topics {
		data.Choices = append(data.Choices, PracticeChoice{Value: "topic:" + topic.ID.String(), Label: "Тема: " + topic.Title})
	}
	for _, kim := range structure.Kims {
		data.Choices = append(data.Choices, PracticeChoice{Value: "kim:" + kim.ID.String(), Label: fmt.Sprintf("Номер КИМ: %d", kim.Number)})
	}

	return data, nil
}

// StartPractice создаёт или возобновляет сессию практики.
func (s *Service) StartPractice(ctx context.Context, userID uuid.UUID, in PracticeInput) (*Session, error) {
	var examID uuid.UUID
	var planItem *PlanItem
	if in.PlanItemID != nil {
		item, itemExamID, err := s.repo.GetPlanItemForUser(ctx, s.db, *in.PlanItemID, userID)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return nil, ErrNotFound
			}
			return nil, err
		}
		planItem = item
		examID = itemExamID
	} else {
		exam, err := s.resolveExam(ctx, userID, "")
		if err != nil {
			return nil, err
		}
		examID = exam.ID
	}

	exam, err := s.content.GetExam(ctx, examID)
	if err != nil || exam.Status != content.StatusPublished {
		return nil, ErrExamUnavailable
	}

	active, err := s.repo.GetInProgressSession(ctx, s.db, userID, examID, SessionKindPractice)
	if err == nil {
		return active, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	target := defaultPracticeSize
	var (
		topicID *uuid.UUID
		kimID   *uuid.UUID
	)
	if planItem != nil {
		topicID = planItem.TopicID
		kimID = planItem.KimNumberID
		target = planItem.TaskCount
		if target <= 0 {
			target = defaultPracticeSize
		}
	} else {
		if (in.TopicID == nil) == (in.KimNumberID == nil) {
			return nil, invalid("filter", "выберите тему или номер КИМ")
		}
		topicID = in.TopicID
		kimID = in.KimNumberID
	}
	if target > maxPracticeSize {
		target = maxPracticeSize
	}

	structure, err := s.loadStructure(ctx, exam.ID)
	if err != nil {
		return nil, err
	}
	tasks, err := s.listPracticeTasks(ctx, structure.Year.ID, topicID, kimID)
	if err != nil {
		return nil, err
	}
	if len(tasks) == 0 {
		return nil, ErrContentUnavailable
	}
	s.shuffler.Shuffle(len(tasks), func(i, j int) {
		tasks[i], tasks[j] = tasks[j], tasks[i]
	})
	if target > len(tasks) {
		target = len(tasks)
	}
	tasks = tasks[:target]

	var planItemID *uuid.UUID
	if planItem != nil {
		id := planItem.ID
		planItemID = &id
	}

	return s.createSessionFromTasks(ctx, userID, exam.ID, SessionKindPractice, tasks, structure, planItemID)
}

func (s *Service) listPracticeTasks(ctx context.Context, examYearID uuid.UUID, topicID, kimID *uuid.UUID) ([]content.Task, error) {
	return s.content.ListPublicTasks(ctx, content.TaskFilter{
		ExamYearID:           &examYearID,
		TopicID:              topicID,
		KimNumberID:          kimID,
		Kinds:                supportedKinds(),
		ExcludePublishedMock: true,
		Limit:                contentPageLimit,
	})
}

func (s *Service) createSessionFromTasks(
	ctx context.Context,
	userID, examID uuid.UUID,
	kind string,
	tasks []content.Task,
	structure *examStructure,
	planItemID *uuid.UUID,
) (*Session, error) {
	kimByID := make(map[uuid.UUID]content.KimNumber, len(structure.Kims))
	for _, kim := range structure.Kims {
		kimByID[kim.ID] = kim
	}

	now := s.clock.Now()
	session := &Session{
		UserID:     userID,
		ExamID:     examID,
		Kind:       kind,
		Status:     SessionInProgress,
		StartedAt:  now,
		PlanItemID: planItemID,
		CreatedAt:  now,
	}
	var err error
	session.ID, err = newID()
	if err != nil {
		return nil, err
	}

	items := make([]SessionItem, 0, len(tasks))
	for index, task := range tasks {
		kim, ok := kimByID[task.KimNumberID]
		if !ok {
			return nil, ErrContentUnavailable
		}
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

func planItemTarget(item PlanItem, topicTitles map[uuid.UUID]string, kimNumbers map[uuid.UUID]int) string {
	if item.TopicID != nil {
		if title := topicTitles[*item.TopicID]; title != "" {
			return title
		}
	}
	if item.KimNumberID != nil {
		return fmt.Sprintf("номер КИМ %d", kimNumbers[*item.KimNumberID])
	}

	return "общая подготовка"
}

func answerSpecSupports(spec json.RawMessage) bool {
	var payload struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(spec, &payload); err != nil {
		return false
	}

	return grading.Supports(payload.Kind)
}
