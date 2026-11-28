package learning

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"oge-ege/internal/content"
)

// ReviewStartData сообщает, есть ли задания для работы над ошибками.
func (s *Service) ReviewStartData(ctx context.Context, userID uuid.UUID) (ReviewStartData, error) {
	exam, err := s.resolveExam(ctx, userID, "")
	if err != nil {
		return ReviewStartData{Available: false, Reason: "Задайте учебную цель, чтобы работать над ошибками."}, nil
	}

	active, err := s.repo.GetInProgressSession(ctx, s.db, userID, exam.ID, SessionKindReview)
	if err == nil {
		return ReviewStartData{Available: true, HasActive: true, ActiveSessionID: active.ID}, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return ReviewStartData{}, err
	}

	tasks, err := s.collectReviewTasks(ctx, userID, exam.ID)
	if err != nil {
		return ReviewStartData{}, err
	}
	if len(tasks) == 0 {
		return ReviewStartData{
			Available: false,
			Reason:    "Пока нет неверно решённых заданий и тем с наступившим сроком повторения. Перейдите к практике или диагностике.",
		}, nil
	}

	return ReviewStartData{Available: true}, nil
}

// StartReview создаёт или возобновляет сессию работы над ошибками.
func (s *Service) StartReview(ctx context.Context, userID uuid.UUID) (*Session, error) {
	exam, err := s.resolveExam(ctx, userID, "")
	if err != nil {
		return nil, err
	}

	active, err := s.repo.GetInProgressSession(ctx, s.db, userID, exam.ID, SessionKindReview)
	if err == nil {
		return active, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	tasks, err := s.collectReviewTasks(ctx, userID, exam.ID)
	if err != nil {
		return nil, err
	}
	if len(tasks) == 0 {
		return nil, ErrNoReviewTasks
	}
	if len(tasks) > defaultPracticeSize {
		tasks = tasks[:defaultPracticeSize]
	}

	structure, err := s.loadStructure(ctx, exam.ID)
	if err != nil {
		return nil, err
	}

	return s.createSessionFromTasks(ctx, userID, exam.ID, SessionKindReview, tasks, structure, nil)
}

func (s *Service) collectReviewTasks(ctx context.Context, userID, examID uuid.UUID) ([]content.Task, error) {
	structure, err := s.loadStructure(ctx, examID)
	if err != nil {
		return nil, err
	}

	dueTopics, err := s.repo.ListDueTopicIDs(ctx, s.db, userID, s.clock.Now())
	if err != nil {
		return nil, err
	}

	var dueTasks []content.Task
	for _, topicID := range dueTopics {
		topicIDCopy := topicID
		tasks, err := s.listPracticeTasks(ctx, structure.Year.ID, &topicIDCopy, nil)
		if err != nil {
			return nil, err
		}
		dueTasks = append(dueTasks, tasks...)
	}
	s.shuffler.Shuffle(len(dueTasks), func(i, j int) {
		dueTasks[i], dueTasks[j] = dueTasks[j], dueTasks[i]
	})

	incorrectIDs, err := s.repo.ListIncorrectTaskIDs(ctx, s.db, userID)
	if err != nil {
		return nil, err
	}
	var incorrectTasks []content.Task
	for _, taskID := range incorrectIDs {
		task, err := s.content.GetTask(ctx, taskID)
		if err != nil {
			if errors.Is(err, content.ErrNotFound) {
				continue
			}
			return nil, err
		}
		if task.Status != content.StatusPublished || !task.IsPublic || task.ExamYearID != structure.Year.ID {
			continue
		}
		if !answerSpecSupports(task.AnswerSpec) {
			continue
		}
		incorrectTasks = append(incorrectTasks, *task)
	}
	s.shuffler.Shuffle(len(incorrectTasks), func(i, j int) {
		incorrectTasks[i], incorrectTasks[j] = incorrectTasks[j], incorrectTasks[i]
	})

	seen := make(map[uuid.UUID]struct{}, len(dueTasks)+len(incorrectTasks))
	result := make([]content.Task, 0, len(dueTasks)+len(incorrectTasks))
	for _, task := range append(dueTasks, incorrectTasks...) {
		if _, ok := seen[task.ID]; ok {
			continue
		}
		seen[task.ID] = struct{}{}
		result = append(result, task)
	}

	return result, nil
}
