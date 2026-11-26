package content

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"oge-ege/internal/shared/db"
	"oge-ege/internal/shared/ids"
)

// CreateVariant создаёт вариант из ручного выбора заданий.
func (s *Service) CreateVariant(ctx context.Context, in VariantInput) (*Variant, error) {
	if err := s.validateVariantHeader(ctx, in.ExamYearID, in.Title, in.Kind, in.DurationMinutes); err != nil {
		return nil, err
	}
	if len(in.Tasks) == 0 {
		return nil, invalid("tasks", "вариант должен содержать задания")
	}

	seen := make(map[uuid.UUID]struct{}, len(in.Tasks))
	tasks := make([]VariantTask, 0, len(in.Tasks))
	for i, item := range in.Tasks {
		if item.Points < 0 {
			return nil, invalid("points", "баллы не могут быть отрицательными")
		}
		if _, ok := seen[item.TaskID]; ok {
			return nil, invalid("task_id", "задание повторяется в варианте")
		}
		seen[item.TaskID] = struct{}{}

		task, err := s.repo.GetTask(ctx, s.db, item.TaskID)
		if err != nil {
			return nil, invalid("task_id", "задание не найдено")
		}
		if task.ExamYearID != in.ExamYearID {
			return nil, invalid("task_id", "задание принадлежит другому учебному году")
		}

		tasks = append(tasks, VariantTask{
			Position: i + 1,
			TaskID:   item.TaskID,
			Points:   item.Points,
		})
	}

	return s.persistVariant(ctx, in.ExamYearID, in.Title, in.Kind, in.DurationMinutes, tasks)
}

// GenerateVariant генерирует вариант по правилам отбора.
func (s *Service) GenerateVariant(ctx context.Context, in GenerateVariantInput) (*Variant, error) {
	if err := s.validateVariantHeader(ctx, in.ExamYearID, in.Title, in.Kind, in.DurationMinutes); err != nil {
		return nil, err
	}
	if len(in.Rules) == 0 {
		return nil, invalid("rules", "правила генерации обязательны")
	}

	used := make(map[uuid.UUID]struct{})
	var tasks []VariantTask

	for _, rule := range in.Rules {
		if rule.Count <= 0 {
			return nil, invalid("count", "количество заданий должно быть положительным")
		}

		kim, err := s.repo.GetKimNumber(ctx, s.db, rule.KimNumberID)
		if err != nil {
			return nil, invalid("kim_number_id", "номер КИМ не найден")
		}
		if kim.ExamYearID != in.ExamYearID {
			return nil, invalid("kim_number_id", "номер КИМ принадлежит другому учебному году")
		}

		candidates, err := s.repo.ListCandidateTasks(ctx, s.db, in.ExamYearID, rule.KimNumberID, rule.Difficulty, rule.TopicID)
		if err != nil {
			return nil, err
		}

		available := make([]Task, 0, len(candidates))
		for _, candidate := range candidates {
			if _, ok := used[candidate.ID]; ok {
				continue
			}
			available = append(available, candidate)
		}
		if len(available) < rule.Count {
			return nil, fmt.Errorf("номер КИМ %d: %w", kim.Number, ErrInsufficient)
		}

		s.shuffler.Shuffle(len(available), func(i, j int) {
			available[i], available[j] = available[j], available[i]
		})

		for _, task := range available[:rule.Count] {
			used[task.ID] = struct{}{}
			tasks = append(tasks, VariantTask{
				Position: len(tasks) + 1,
				TaskID:   task.ID,
				Points:   kim.MaxPrimaryScore,
			})
		}
	}

	return s.persistVariant(ctx, in.ExamYearID, in.Title, in.Kind, in.DurationMinutes, tasks)
}

// GetVariant возвращает вариант.
func (s *Service) GetVariant(ctx context.Context, id uuid.UUID) (*Variant, error) {
	return s.repo.GetVariant(ctx, s.db, id)
}

// ListVariants возвращает страницу вариантов.
func (s *Service) ListVariants(ctx context.Context, filter VariantFilter) ([]Variant, error) {
	return s.repo.ListVariants(ctx, s.db, filter)
}

// PublishVariant публикует вариант.
func (s *Service) PublishVariant(ctx context.Context, id uuid.UUID) (*Variant, error) {
	variant, err := s.repo.GetVariant(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	now := s.clock.Now()
	if err := s.repo.SetVariantStatus(ctx, s.db, id, StatusPublished, now); err != nil {
		return nil, err
	}
	variant.Status = StatusPublished
	variant.UpdatedAt = now

	return variant, nil
}

// CountVariants считает варианты учебного года.
func (s *Service) CountVariants(ctx context.Context, examYearID uuid.UUID) (int, error) {
	return s.repo.CountVariants(ctx, s.db, examYearID)
}

func (s *Service) validateVariantHeader(ctx context.Context, examYearID uuid.UUID, title string, kind VariantKind, duration int) error {
	if _, err := s.repo.GetExamYear(ctx, s.db, examYearID); err != nil {
		return invalid("exam_year_id", "учебный год не найден")
	}
	if _, err := required("title", title); err != nil {
		return err
	}
	if !kind.Valid() {
		return invalid("kind", "неизвестное назначение варианта")
	}

	return validateDuration(duration)
}

func (s *Service) persistVariant(ctx context.Context, examYearID uuid.UUID, title string, kind VariantKind, duration int, tasks []VariantTask) (*Variant, error) {
	variantID, err := ids.New()
	if err != nil {
		return nil, err
	}

	now := s.clock.Now()
	variant := &Variant{
		ID:              variantID,
		ExamYearID:      examYearID,
		Title:           strings.TrimSpace(title),
		Kind:            kind,
		DurationMinutes: duration,
		Status:          StatusDraft,
		CreatedAt:       now,
		UpdatedAt:       now,
		Tasks:           tasks,
	}

	for i := range variant.Tasks {
		taskID, err := ids.New()
		if err != nil {
			return nil, err
		}
		variant.Tasks[i].ID = taskID
		variant.Tasks[i].VariantID = variantID
	}

	if err := s.tx.WithTx(ctx, func(q db.Querier) error {
		return s.repo.CreateVariant(ctx, q, variant)
	}); err != nil {
		return nil, err
	}

	return variant, nil
}
