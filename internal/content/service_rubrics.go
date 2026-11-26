package content

import (
	"context"
	"fmt"
	"sort"

	"github.com/google/uuid"

	"oge-ege/internal/shared/db"
	"oge-ege/internal/shared/ids"
)

// CreateRubric создаёт черновик новой версии рубрики.
func (s *Service) CreateRubric(ctx context.Context, in RubricInput) (*Rubric, error) {
	kim, err := s.validateRubric(ctx, in)
	if err != nil {
		return nil, err
	}
	_ = kim

	return s.newRubricVersion(ctx, in, StatusDraft)
}

// UpdateRubric создаёт новую версию действующей рубрики.
func (s *Service) UpdateRubric(ctx context.Context, id uuid.UUID, criteria RubricCriteria) (*Rubric, error) {
	existing, err := s.repo.GetRubric(ctx, s.db, id)
	if err != nil {
		return nil, err
	}

	in := RubricInput{
		KimNumberID: existing.KimNumberID,
		ExamYearID:  existing.ExamYearID,
		Criteria:    criteria,
	}
	if _, err := s.validateRubric(ctx, in); err != nil {
		return nil, err
	}

	status := StatusDraft
	if existing.Status == StatusPublished {
		status = StatusPublished
	}

	return s.newRubricVersion(ctx, in, status)
}

// PublishRubric публикует версию рубрики.
func (s *Service) PublishRubric(ctx context.Context, id uuid.UUID) (*Rubric, error) {
	rubric, err := s.repo.GetRubric(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	in := RubricInput{KimNumberID: rubric.KimNumberID, ExamYearID: rubric.ExamYearID, Criteria: rubric.Criteria}
	if _, err := s.validateRubric(ctx, in); err != nil {
		return nil, err
	}

	now := s.clock.Now()
	if err := s.repo.SetRubricStatus(ctx, s.db, id, StatusPublished, now); err != nil {
		return nil, err
	}
	rubric.Status = StatusPublished
	rubric.UpdatedAt = now

	return rubric, nil
}

// GetRubric возвращает версию рубрики.
func (s *Service) GetRubric(ctx context.Context, id uuid.UUID) (*Rubric, error) {
	return s.repo.GetRubric(ctx, s.db, id)
}

// GetActiveRubric возвращает действующую рубрику.
func (s *Service) GetActiveRubric(ctx context.Context, kimNumberID, examYearID uuid.UUID) (*Rubric, error) {
	return s.repo.GetActiveRubric(ctx, s.db, kimNumberID, examYearID)
}

// ListRubrics возвращает страницу рубрик.
func (s *Service) ListRubrics(ctx context.Context, filter RubricFilter) ([]Rubric, error) {
	return s.repo.ListRubrics(ctx, s.db, filter)
}

func (s *Service) validateRubric(ctx context.Context, in RubricInput) (*KimNumber, error) {
	kim, err := s.repo.GetKimNumber(ctx, s.db, in.KimNumberID)
	if err != nil {
		return nil, invalid("kim_number_id", "номер КИМ не найден")
	}
	if kim.ExamYearID != in.ExamYearID {
		return nil, invalid("exam_year_id", "номер КИМ принадлежит другому учебному году")
	}
	if len(in.Criteria) == 0 {
		return nil, invalid("criteria", "критерии обязательны")
	}

	seen := make(map[string]struct{}, len(in.Criteria))
	sum := 0
	for _, criterion := range in.Criteria {
		code := criterion.Code
		if code == "" {
			return nil, invalid("criteria", "код критерия обязателен")
		}
		if _, ok := seen[code]; ok {
			return nil, invalid("criteria", "код критерия повторяется")
		}
		seen[code] = struct{}{}
		if criterion.MaxPoints <= 0 {
			return nil, invalid("criteria", "максимальный балл критерия должен быть положительным")
		}
		sum += criterion.MaxPoints
	}

	if sum != kim.MaxPrimaryScore {
		return nil, invalid("criteria", fmt.Sprintf("сумма баллов критериев %d не совпадает с максимальным баллом номера КИМ %d", sum, kim.MaxPrimaryScore))
	}

	return kim, nil
}

func (s *Service) newRubricVersion(ctx context.Context, in RubricInput, status Status) (*Rubric, error) {
	latest, err := s.repo.GetLatestRubricVersion(ctx, s.db, in.KimNumberID, in.ExamYearID)
	if err != nil {
		return nil, err
	}

	id, err := ids.New()
	if err != nil {
		return nil, err
	}
	now := s.clock.Now()
	rubric := &Rubric{
		ID:          id,
		KimNumberID: in.KimNumberID,
		ExamYearID:  in.ExamYearID,
		Version:     latest + 1,
		Criteria:    in.Criteria,
		Status:      status,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.CreateRubric(ctx, s.db, rubric); err != nil {
		return nil, err
	}

	return rubric, nil
}

// ReplaceScoreScales сохраняет шкалу перевода учебного года.
func (s *Service) ReplaceScoreScales(ctx context.Context, in ScoreScaleInput) ([]ScoreScale, error) {
	if _, err := s.repo.GetExamYear(ctx, s.db, in.ExamYearID); err != nil {
		return nil, invalid("exam_year_id", "учебный год не найден")
	}
	if len(in.Entries) == 0 {
		return nil, invalid("entries", "шкала перевода обязательна")
	}

	entries := append([]ScoreScaleEntry(nil), in.Entries...)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].PrimaryScore < entries[j].PrimaryScore
	})

	maxPrimary, err := s.repo.SumMaxPrimaryScore(ctx, s.db, in.ExamYearID)
	if err != nil {
		return nil, err
	}

	for i, entry := range entries {
		if entry.PrimaryScore != i {
			return nil, invalid("entries", fmt.Sprintf("шкала не покрывает первичный балл %d", i))
		}
		if entry.TestScore < 0 {
			return nil, invalid("test_score", "тестовый балл не может быть отрицательным")
		}
		if entry.Grade != nil && (*entry.Grade < 1 || *entry.Grade > 5) {
			return nil, invalid("grade", "оценка должна быть от 1 до 5")
		}
	}
	if len(entries) != maxPrimary+1 {
		return nil, invalid("entries", fmt.Sprintf("шкала должна покрывать первичные баллы от 0 до %d", maxPrimary))
	}

	scales := make([]ScoreScale, 0, len(entries))
	for _, entry := range entries {
		id, err := ids.New()
		if err != nil {
			return nil, err
		}
		scales = append(scales, ScoreScale{
			ID:           id,
			ExamYearID:   in.ExamYearID,
			PrimaryScore: entry.PrimaryScore,
			TestScore:    entry.TestScore,
			Grade:        entry.Grade,
			MinThreshold: entry.MinThreshold,
		})
	}

	if err := s.tx.WithTx(ctx, func(q db.Querier) error {
		return s.repo.ReplaceScoreScales(ctx, q, in.ExamYearID, scales)
	}); err != nil {
		return nil, err
	}

	return s.repo.ListScoreScales(ctx, s.db, in.ExamYearID)
}

// ListScoreScales возвращает шкалу перевода учебного года.
func (s *Service) ListScoreScales(ctx context.Context, examYearID uuid.UUID) ([]ScoreScale, error) {
	return s.repo.ListScoreScales(ctx, s.db, examYearID)
}
