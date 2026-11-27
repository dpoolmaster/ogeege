package content

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"oge-ege/internal/grading"
	"oge-ege/internal/shared/db"
	"oge-ege/internal/shared/ids"
	"oge-ege/internal/shared/storage"
)

// CreateTask создаёт черновик задания.
func (s *Service) CreateTask(ctx context.Context, in TaskInput) (*Task, error) {
	if err := s.validateTask(ctx, in, false); err != nil {
		return nil, err
	}

	id, err := ids.New()
	if err != nil {
		return nil, err
	}
	now := s.clock.Now()
	task := &Task{
		ID:            id,
		ExamYearID:    in.ExamYearID,
		KimNumberID:   in.KimNumberID,
		TopicID:       in.TopicID,
		Difficulty:    in.Difficulty,
		StatementMD:   strings.TrimSpace(in.StatementMD),
		AnswerSpec:    in.AnswerSpec,
		AnswerDisplay: strings.TrimSpace(in.AnswerDisplay),
		Source:        strings.TrimSpace(in.Source),
		SourceRef:     strings.TrimSpace(in.SourceRef),
		Status:        StatusDraft,
		IsPublic:      false,
		ContentHash:   ContentHash(in.StatementMD),
		Version:       1,
		CreatedBy:     in.CreatedBy,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.repo.CreateTask(ctx, s.db, task); err != nil {
		return nil, err
	}

	return task, nil
}

// UpdateTask изменяет задание, сохраняя версию опубликованного задания.
func (s *Service) UpdateTask(ctx context.Context, id uuid.UUID, in TaskInput) (*Task, error) {
	task, err := s.repo.GetTask(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	if err := s.validateTask(ctx, in, task.Status == StatusPublished); err != nil {
		return nil, err
	}

	now := s.clock.Now()
	updated := *task
	updated.KimNumberID = in.KimNumberID
	updated.TopicID = in.TopicID
	updated.Difficulty = in.Difficulty
	updated.StatementMD = strings.TrimSpace(in.StatementMD)
	updated.AnswerSpec = in.AnswerSpec
	updated.AnswerDisplay = strings.TrimSpace(in.AnswerDisplay)
	updated.Source = strings.TrimSpace(in.Source)
	updated.SourceRef = strings.TrimSpace(in.SourceRef)
	updated.ContentHash = ContentHash(in.StatementMD)
	updated.UpdatedAt = now

	if task.Status == StatusPublished {
		versionID, err := ids.New()
		if err != nil {
			return nil, err
		}
		updated.Version = task.Version + 1

		err = s.tx.WithTx(ctx, func(q db.Querier) error {
			if err := s.repo.CreateTaskVersion(ctx, q, &TaskVersion{
				ID:            versionID,
				TaskID:        task.ID,
				Version:       task.Version,
				StatementMD:   task.StatementMD,
				AnswerSpec:    task.AnswerSpec,
				AnswerDisplay: task.AnswerDisplay,
				Difficulty:    task.Difficulty,
				KimNumberID:   task.KimNumberID,
				TopicID:       task.TopicID,
				Source:        task.Source,
				SourceRef:     task.SourceRef,
				CreatedBy:     task.CreatedBy,
				CreatedAt:     now,
			}); err != nil {
				return err
			}

			return s.repo.UpdateTask(ctx, q, &updated)
		})
		if err != nil {
			return nil, err
		}

		return &updated, nil
	}

	if err := s.repo.UpdateTask(ctx, s.db, &updated); err != nil {
		return nil, err
	}

	return &updated, nil
}

// GetTask возвращает задание.
func (s *Service) GetTask(ctx context.Context, id uuid.UUID) (*Task, error) {
	return s.repo.GetTask(ctx, s.db, id)
}

// GetTaskByHash возвращает задание по хешу условия.
func (s *Service) GetTaskByHash(ctx context.Context, hash []byte) (*Task, error) {
	return s.repo.GetTaskByHash(ctx, s.db, hash)
}

// ListTasks возвращает страницу заданий.
func (s *Service) ListTasks(ctx context.Context, filter TaskFilter) ([]Task, error) {
	return s.repo.ListTasks(ctx, s.db, filter)
}

// ListPublicTasks возвращает опубликованные публичные задания.
func (s *Service) ListPublicTasks(ctx context.Context, filter TaskFilter) ([]Task, error) {
	return s.repo.ListPublicTasks(ctx, s.db, filter)
}

// ListDemoTasks выбирает от 5 до 7 проверяемых заданий демо-диагностики.
// На каждый номер КИМ попадает не более одного задания; задания опубликованных
// пробников исключаются.
func (s *Service) ListDemoTasks(ctx context.Context, examYearID uuid.UUID, limit int) ([]Task, error) {
	if limit < 5 {
		limit = 5
	}
	if limit > 7 {
		limit = 7
	}

	tasks, err := s.repo.ListPublicTasks(ctx, s.db, TaskFilter{
		ExamYearID:           &examYearID,
		Kinds:                supportedGradingKinds(),
		ExcludePublishedMock: true,
		Limit:                200,
	})
	if err != nil {
		return nil, err
	}

	s.shuffler.Shuffle(len(tasks), func(i, j int) {
		tasks[i], tasks[j] = tasks[j], tasks[i]
	})

	seen := make(map[uuid.UUID]struct{}, len(tasks))
	result := make([]Task, 0, limit)
	for _, task := range tasks {
		if _, ok := seen[task.KimNumberID]; ok {
			continue
		}
		seen[task.KimNumberID] = struct{}{}
		result = append(result, task)
		if len(result) == limit {
			break
		}
	}

	if len(result) < limit {
		return nil, ErrInsufficient
	}

	return result, nil
}

func supportedGradingKinds() []string {
	kinds := []string{"number", "text", "digits", "interval"}
	supported := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		if grading.Supports(kind) {
			supported = append(supported, kind)
		}
	}

	return supported
}

// PublishTask публикует задание после проверки.
func (s *Service) PublishTask(ctx context.Context, id uuid.UUID) (*Task, error) {
	task, err := s.repo.GetTask(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	in := TaskInput{
		ExamYearID:    task.ExamYearID,
		KimNumberID:   task.KimNumberID,
		TopicID:       task.TopicID,
		Difficulty:    task.Difficulty,
		StatementMD:   task.StatementMD,
		AnswerSpec:    task.AnswerSpec,
		AnswerDisplay: task.AnswerDisplay,
		Source:        task.Source,
		SourceRef:     task.SourceRef,
		CreatedBy:     task.CreatedBy,
	}
	if err := s.validateTask(ctx, in, true); err != nil {
		return nil, err
	}

	now := s.clock.Now()
	if err := s.repo.SetTaskStatus(ctx, s.db, id, StatusPublished, true, now); err != nil {
		return nil, err
	}
	task.Status = StatusPublished
	task.IsPublic = true
	task.UpdatedAt = now

	return task, nil
}

// ArchiveTask переводит задание в архив.
func (s *Service) ArchiveTask(ctx context.Context, id uuid.UUID) (*Task, error) {
	task, err := s.repo.GetTask(ctx, s.db, id)
	if err != nil {
		return nil, err
	}

	now := s.clock.Now()
	if err := s.repo.SetTaskStatus(ctx, s.db, id, StatusArchived, false, now); err != nil {
		return nil, err
	}
	task.Status = StatusArchived
	task.IsPublic = false
	task.UpdatedAt = now

	return task, nil
}

// DeleteTask удаляет черновик задания.
func (s *Service) DeleteTask(ctx context.Context, id uuid.UUID) error {
	task, err := s.repo.GetTask(ctx, s.db, id)
	if err != nil {
		return err
	}
	if task.Status != StatusDraft {
		return fmt.Errorf("удалить задание в статусе %q: %w", task.Status, ErrConflict)
	}

	return s.repo.DeleteTask(ctx, s.db, id)
}

// ListTaskVersions возвращает историю версий задания.
func (s *Service) ListTaskVersions(ctx context.Context, taskID uuid.UUID) ([]TaskVersion, error) {
	return s.repo.ListTaskVersions(ctx, s.db, taskID)
}

// AddTaskAttachment сохраняет изображение задания.
func (s *Service) AddTaskAttachment(ctx context.Context, taskID uuid.UUID, in TaskAttachmentInput) (*TaskAttachment, error) {
	if _, err := s.repo.GetTask(ctx, s.db, taskID); err != nil {
		return nil, err
	}
	if in.Kind != "image" {
		return nil, invalid("kind", "поддерживаются только изображения")
	}
	if !storage.ValidImageKey(in.StorageKey) {
		return nil, invalid("storage_key", "некорректный ключ изображения")
	}

	id, err := ids.New()
	if err != nil {
		return nil, err
	}
	attachment := &TaskAttachment{
		ID:         id,
		TaskID:     taskID,
		Kind:       in.Kind,
		StorageKey: in.StorageKey,
		URL:        "/media/" + in.StorageKey,
		Alt:        strings.TrimSpace(in.Alt),
		CreatedAt:  s.clock.Now(),
	}

	if err := s.repo.CreateTaskAttachment(ctx, s.db, attachment); err != nil {
		return nil, err
	}

	return attachment, nil
}

// ListTaskAttachments возвращает изображения задания.
func (s *Service) ListTaskAttachments(ctx context.Context, taskID uuid.UUID) ([]TaskAttachment, error) {
	return s.repo.ListTaskAttachments(ctx, s.db, taskID)
}

func (s *Service) validateTask(ctx context.Context, in TaskInput, forPublish bool) error {
	year, err := s.repo.GetExamYear(ctx, s.db, in.ExamYearID)
	if err != nil {
		return invalid("exam_year_id", "учебный год не найден")
	}

	kim, err := s.repo.GetKimNumber(ctx, s.db, in.KimNumberID)
	if err != nil {
		return invalid("kim_number_id", "номер КИМ не найден")
	}
	if kim.ExamYearID != in.ExamYearID {
		return invalid("kim_number_id", "номер КИМ принадлежит другому учебному году")
	}

	if !in.Difficulty.Valid() {
		return invalid("difficulty", "неизвестная сложность")
	}
	if _, err := required("statement_md", in.StatementMD); err != nil {
		return err
	}
	if _, err := required("source", in.Source); err != nil {
		return err
	}
	if err := validateAnswerSpec(in.AnswerSpec); err != nil {
		return err
	}
	if err := validateLaTeX(in.StatementMD); err != nil {
		return err
	}

	if in.TopicID == nil {
		if forPublish {
			return invalid("topic_id", "тема обязательна для публикации")
		}
		return nil
	}

	topic, err := s.repo.GetTopic(ctx, s.db, *in.TopicID)
	if err != nil {
		return invalid("topic_id", "тема не найдена")
	}
	if topic.ExamID != year.ExamID {
		return invalid("topic_id", "тема принадлежит другому треку")
	}

	if forPublish {
		if _, err := s.repo.GetKimTopic(ctx, s.db, kim.ID, topic.ID); err != nil {
			if errors.Is(err, ErrNotFound) {
				return invalid("topic_id", "тема не связана с номером КИМ")
			}
			return err
		}
	}

	return nil
}
