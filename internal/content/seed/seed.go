// Package seed создаёт стартовый учебный контент.
package seed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"oge-ege/internal/content"
)

type trackSpec struct {
	code            string
	title           string
	durationMinutes int
	part1Count      int
	lastNumber      int
}

var tracks = []trackSpec{
	{code: "oge_math", title: "ОГЭ математика", durationMinutes: 235, part1Count: 19, lastNumber: 25},
	{code: "ege_math_profile", title: "ЕГЭ математика (профиль)", durationMinutes: 235, part1Count: 11, lastNumber: 18},
	{code: "ege_math_base", title: "ЕГЭ математика (база)", durationMinutes: 180, part1Count: 21, lastNumber: 21},
}

// Run создаёт демонстрационный контент; повторный запуск не создаёт дубликатов.
func Run(ctx context.Context, service *content.Service, logger *slog.Logger) error {
	year := time.Now().Year()

	for _, spec := range tracks {
		if err := seedTrack(ctx, service, logger, spec, year); err != nil {
			return fmt.Errorf("заполнить трек %s: %w", spec.code, err)
		}
	}

	return nil
}

func seedTrack(ctx context.Context, service *content.Service, logger *slog.Logger, spec trackSpec, year int) error {
	exam, err := service.GetExamByCode(ctx, spec.code)
	if errors.Is(err, content.ErrNotFound) {
		exam, err = service.CreateExam(ctx, content.ExamInput{
			Code: spec.code, Title: spec.title, Subject: "math",
		})
		if err != nil {
			return fmt.Errorf("создать трек: %w", err)
		}
	} else if err != nil {
		return err
	}

	if exam.Status != content.StatusPublished {
		if _, err := service.PublishExam(ctx, exam.ID); err != nil {
			return fmt.Errorf("опубликовать трек: %w", err)
		}
	}

	examYear, err := service.GetExamYearByYear(ctx, exam.ID, year)
	if errors.Is(err, content.ErrNotFound) {
		examYear, err = service.CreateExamYear(ctx, content.ExamYearInput{
			ExamID: exam.ID, Year: year, DurationMinutes: spec.durationMinutes,
		})
		if err != nil {
			return fmt.Errorf("создать учебный год: %w", err)
		}
	} else if err != nil {
		return err
	}
	if examYear.Status != content.StatusPublished {
		if _, err := service.PublishExamYear(ctx, examYear.ID); err != nil {
			return fmt.Errorf("опубликовать учебный год: %w", err)
		}
	}

	parent, err := ensureTopic(ctx, service, exam.ID, nil, "algebra", "Алгебра", 1)
	if err != nil {
		return err
	}
	child, err := ensureTopic(ctx, service, exam.ID, &parent.ID, "numbers", "Числа и вычисления", 1)
	if err != nil {
		return err
	}

	kimNumbers, err := ensureKimNumbers(ctx, service, examYear.ID, spec)
	if err != nil {
		return err
	}

	for _, kim := range kimNumbers {
		if _, err := service.SetKimTopics(ctx, kim.ID, []content.KimTopicInput{{TopicID: child.ID, Weight: 1}}); err != nil {
			return fmt.Errorf("связать тему с номером КИМ %d: %w", kim.Number, err)
		}
	}

	for _, topic := range []content.Topic{parent, child} {
		if err := ensureTheory(ctx, service, topic); err != nil {
			return err
		}
	}

	published, err := ensureTasks(ctx, service, spec.code, examYear.ID, kimNumbers, child.ID)
	if err != nil {
		return err
	}

	if err := ensureRubrics(ctx, service, examYear.ID, kimNumbers); err != nil {
		return err
	}

	if err := ensureVariant(ctx, service, examYear, published); err != nil {
		return err
	}

	if err := ensureScoreScale(ctx, service, examYear, kimNumbers, spec.code == "ege_math_profile"); err != nil {
		return err
	}

	logger.Info("контент трека подготовлен", "code", spec.code, "year", year, "kim_numbers", len(kimNumbers), "tasks", len(published))

	return nil
}

func ensureTopic(ctx context.Context, service *content.Service, examID uuid.UUID, parentID *uuid.UUID, code, title string, order int) (content.Topic, error) {
	topic, err := service.GetTopicByCode(ctx, examID, code)
	if err == nil {
		return *topic, nil
	}
	if !errors.Is(err, content.ErrNotFound) {
		return content.Topic{}, err
	}

	created, err := service.CreateTopic(ctx, content.TopicInput{
		ExamID: examID, ParentID: parentID, Code: code, Title: title, OrderIndex: order,
	})
	if err != nil {
		return content.Topic{}, fmt.Errorf("создать тему %s: %w", code, err)
	}

	return *created, nil
}

func ensureKimNumbers(ctx context.Context, service *content.Service, examYearID uuid.UUID, spec trackSpec) ([]content.KimNumber, error) {
	numbers := make([]content.KimNumber, 0, spec.lastNumber)
	for number := 1; number <= spec.lastNumber; number++ {
		part := 1
		answerType := content.AnswerTypeShort
		maxScore := 1
		if number > spec.part1Count {
			part = 2
			answerType = content.AnswerTypeExtended
			maxScore = 2
		}

		kim, err := service.GetKimNumberByNumber(ctx, examYearID, number)
		if errors.Is(err, content.ErrNotFound) {
			kim, err = service.CreateKimNumber(ctx, content.KimNumberInput{
				ExamYearID: examYearID, Number: number, Part: part,
				AnswerType: answerType, MaxPrimaryScore: maxScore,
				Title: fmt.Sprintf("Задание %d", number),
			})
			if err != nil {
				return nil, fmt.Errorf("создать номер КИМ %d: %w", number, err)
			}
		} else if err != nil {
			return nil, err
		}
		numbers = append(numbers, *kim)
	}

	return numbers, nil
}

func ensureTheory(ctx context.Context, service *content.Service, topic content.Topic) error {
	versions, err := service.ListTheoryVersions(ctx, topic.ID)
	if err != nil {
		return err
	}
	if len(versions) > 0 {
		return nil
	}

	body := fmt.Sprintf("## %s\n\nБазовый теоретический материал по теме. Формула: $a^2 + b^2 = c^2$.\n", topic.Title)
	theory, err := service.CreateTheoryVersion(ctx, content.TheoryInput{TopicID: topic.ID, ContentMD: body})
	if err != nil {
		return fmt.Errorf("создать теорию темы %s: %w", topic.Code, err)
	}
	if _, err := service.PublishTheory(ctx, theory.ID); err != nil {
		return fmt.Errorf("опубликовать теорию темы %s: %w", topic.Code, err)
	}

	return nil
}

func ensureTasks(ctx context.Context, service *content.Service, code string, examYearID uuid.UUID, kimNumbers []content.KimNumber, topicID uuid.UUID) ([]content.Task, error) {
	published := make([]content.Task, 0, len(kimNumbers))
	for _, kim := range kimNumbers {
		statement := fmt.Sprintf("Демонстрационное задание трека %s №%d. Вычислите $%d + %d$.", code, kim.Number, kim.Number, kim.Number)
		hash := content.ContentHash(statement)

		task, err := service.GetTaskByHash(ctx, hash)
		if errors.Is(err, content.ErrNotFound) {
			task, err = service.CreateTask(ctx, content.TaskInput{
				ExamYearID: examYearID, KimNumberID: kim.ID, TopicID: &topicID,
				Difficulty: content.DifficultyEasy, StatementMD: statement,
				AnswerSpec: answerSpec(kim.AnswerType), AnswerDisplay: "1",
				Source: "seed", SourceRef: "https://fipi.ru",
			})
			if err != nil {
				if errors.Is(err, content.ErrConflict) {
					continue
				}
				return nil, fmt.Errorf("создать задание №%d: %w", kim.Number, err)
			}
		} else if err != nil {
			return nil, err
		}

		if task.Status != content.StatusPublished {
			task, err = service.PublishTask(ctx, task.ID)
			if err != nil {
				return nil, fmt.Errorf("опубликовать задание №%d: %w", kim.Number, err)
			}
		}
		published = append(published, *task)
	}

	return published, nil
}

func answerSpec(answerType content.AnswerType) json.RawMessage {
	if answerType == content.AnswerTypeExtended {
		return json.RawMessage(`{"kind":"extended"}`)
	}

	return json.RawMessage(`{"kind":"number","accepted":["1"]}`)
}

func ensureRubrics(ctx context.Context, service *content.Service, examYearID uuid.UUID, kimNumbers []content.KimNumber) error {
	for _, kim := range kimNumbers {
		if kim.Part != 2 {
			continue
		}

		_, err := service.GetActiveRubric(ctx, kim.ID, examYearID)
		if err == nil {
			continue
		}
		if !errors.Is(err, content.ErrNotFound) {
			return err
		}

		rubric, err := service.CreateRubric(ctx, content.RubricInput{
			KimNumberID: kim.ID, ExamYearID: examYearID,
			Criteria: content.RubricCriteria{{
				Code: "C1", Description: "Обоснованное решение", MaxPoints: kim.MaxPrimaryScore,
			}},
		})
		if err != nil {
			return fmt.Errorf("создать рубрику №%d: %w", kim.Number, err)
		}
		if _, err := service.PublishRubric(ctx, rubric.ID); err != nil {
			return fmt.Errorf("опубликовать рубрику №%d: %w", kim.Number, err)
		}
	}

	return nil
}

func ensureVariant(ctx context.Context, service *content.Service, examYear *content.ExamYear, published []content.Task) error {
	count, err := service.CountVariants(ctx, examYear.ID)
	if err != nil {
		return err
	}
	if count > 0 || len(published) == 0 {
		return nil
	}

	limit := len(published)
	if limit > 10 {
		limit = 10
	}
	tasks := make([]content.VariantTaskInput, 0, limit)
	for i := 0; i < limit; i++ {
		tasks = append(tasks, content.VariantTaskInput{TaskID: published[i].ID, Points: 1})
	}

	variant, err := service.CreateVariant(ctx, content.VariantInput{
		ExamYearID: examYear.ID, Title: "Демонстрационный вариант",
		Kind: content.VariantMock, DurationMinutes: examYear.DurationMinutes, Tasks: tasks,
	})
	if err != nil {
		return fmt.Errorf("создать вариант: %w", err)
	}
	if _, err := service.PublishVariant(ctx, variant.ID); err != nil {
		return fmt.Errorf("опубликовать вариант: %w", err)
	}

	return nil
}

func ensureScoreScale(ctx context.Context, service *content.Service, examYear *content.ExamYear, kimNumbers []content.KimNumber, profile bool) error {
	scales, err := service.ListScoreScales(ctx, examYear.ID)
	if err != nil {
		return err
	}
	if len(scales) > 0 {
		return nil
	}

	maxPrimary := 0
	for _, kim := range kimNumbers {
		maxPrimary += kim.MaxPrimaryScore
	}

	entries := make([]content.ScoreScaleEntry, 0, maxPrimary+1)
	for primary := 0; primary <= maxPrimary; primary++ {
		testScore := primary
		var grade *int
		if profile {
			testScore = primary * 4
			if testScore > 100 {
				testScore = 100
			}
		} else {
			value := gradeFor(primary, maxPrimary)
			grade = &value
		}
		entries = append(entries, content.ScoreScaleEntry{
			PrimaryScore: primary, TestScore: testScore, Grade: grade,
		})
	}

	if _, err := service.ReplaceScoreScales(ctx, content.ScoreScaleInput{
		ExamYearID: examYear.ID, Entries: entries,
	}); err != nil {
		return fmt.Errorf("сохранить шкалу перевода: %w", err)
	}

	return nil
}

func gradeFor(primary, maxPrimary int) int {
	if maxPrimary == 0 {
		return 2
	}

	ratio := float64(primary) / float64(maxPrimary)
	switch {
	case ratio < 0.3:
		return 2
	case ratio < 0.5:
		return 3
	case ratio < 0.7:
		return 4
	default:
		return 5
	}
}
