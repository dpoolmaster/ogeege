package content

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// ImportTasks создаёт черновики заданий из структурированного батча.
// Импорт никогда не публикует задания автоматически.
func (s *Service) ImportTasks(ctx context.Context, items []ImportItem) (*ImportReport, error) {
	report := &ImportReport{
		Created:  []ImportCreated{},
		Skipped:  []ImportSkipped{},
		Rejected: []ImportRejected{},
	}
	if len(items) == 0 {
		return nil, invalid("items", "батч импорта пуст")
	}

	for index, item := range items {
		task, err := s.importOne(ctx, item)
		if err != nil {
			var validationErr *ValidationError
			switch {
			case errors.Is(err, ErrConflict):
				report.Skipped = append(report.Skipped, ImportSkipped{Index: index, Reason: "условие уже существует в банке"})
			case errors.As(err, &validationErr):
				report.Rejected = append(report.Rejected, ImportRejected{Index: index, Reason: err.Error()})
			default:
				return nil, fmt.Errorf("импортировать задание %d: %w", index, err)
			}
			continue
		}

		if task == nil {
			existing, err := s.repo.GetTaskByHash(ctx, s.db, ContentHash(item.StatementMD))
			if err != nil {
				return nil, fmt.Errorf("найти дубликат задания %d: %w", index, err)
			}
			report.Skipped = append(report.Skipped, ImportSkipped{
				Index:  index,
				TaskID: existing.ID,
				Reason: "условие уже существует в банке",
			})
			continue
		}

		report.Created = append(report.Created, ImportCreated{Index: index, TaskID: task.ID})
	}

	return report, nil
}

func (s *Service) importOne(ctx context.Context, item ImportItem) (*Task, error) {
	input := TaskInput{
		ExamYearID:    item.ExamYearID,
		KimNumberID:   item.KimNumberID,
		TopicID:       item.TopicID,
		Difficulty:    item.Difficulty,
		StatementMD:   strings.TrimSpace(item.StatementMD),
		AnswerSpec:    item.AnswerSpec,
		AnswerDisplay: item.AnswerDisplay,
		Source:        item.Source,
		SourceRef:     item.SourceRef,
		CreatedBy:     item.CreatedBy,
		Options:       item.Options,
	}

	hash := ContentHash(input.StatementMD)
	if _, err := s.repo.GetTaskByHash(ctx, s.db, hash); err == nil {
		return nil, nil
	} else if !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	return s.CreateTask(ctx, input)
}
