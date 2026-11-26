package content

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"oge-ege/internal/shared/ids"
)

// CreateTheoryVersion создаёт новую версию теории темы.
func (s *Service) CreateTheoryVersion(ctx context.Context, in TheoryInput) (*Theory, error) {
	if _, err := s.repo.GetTopic(ctx, s.db, in.TopicID); err != nil {
		return nil, invalid("topic_id", "тема не найдена")
	}

	content, err := required("content_md", in.ContentMD)
	if err != nil {
		return nil, err
	}
	if err := validateLaTeX(content); err != nil {
		return nil, err
	}

	latest, err := s.repo.GetLatestTheoryVersion(ctx, s.db, in.TopicID)
	if err != nil {
		return nil, err
	}

	id, err := ids.New()
	if err != nil {
		return nil, err
	}
	now := s.clock.Now()
	theory := &Theory{
		ID:        id,
		TopicID:   in.TopicID,
		ContentMD: content,
		Version:   latest + 1,
		Status:    StatusDraft,
		UpdatedBy: in.UpdatedBy,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.CreateTheory(ctx, s.db, theory); err != nil {
		return nil, err
	}

	return theory, nil
}

// UpdateTheory изменяет текст черновой версии теории.
func (s *Service) UpdateTheory(ctx context.Context, id uuid.UUID, in TheoryInput) (*Theory, error) {
	theory, err := s.repo.GetTheory(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	if theory.Status != StatusDraft {
		return nil, fmt.Errorf("изменить опубликованную версию теории: %w", ErrConflict)
	}

	content, err := required("content_md", in.ContentMD)
	if err != nil {
		return nil, err
	}
	if err := validateLaTeX(content); err != nil {
		return nil, err
	}

	theory.ContentMD = content
	theory.UpdatedBy = in.UpdatedBy
	theory.UpdatedAt = s.clock.Now()

	if err := s.repo.UpdateTheory(ctx, s.db, theory); err != nil {
		return nil, err
	}

	return theory, nil
}

// PublishTheory публикует версию теории.
func (s *Service) PublishTheory(ctx context.Context, id uuid.UUID) (*Theory, error) {
	theory, err := s.repo.GetTheory(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	if err := validateLaTeX(theory.ContentMD); err != nil {
		return nil, err
	}

	now := s.clock.Now()
	if err := s.repo.SetTheoryStatus(ctx, s.db, id, StatusPublished, now); err != nil {
		return nil, err
	}
	theory.Status = StatusPublished
	theory.UpdatedAt = now

	return theory, nil
}

// GetTheory возвращает версию теории.
func (s *Service) GetTheory(ctx context.Context, id uuid.UUID) (*Theory, error) {
	return s.repo.GetTheory(ctx, s.db, id)
}

// GetActiveTheory возвращает действующую теорию темы.
func (s *Service) GetActiveTheory(ctx context.Context, topicID uuid.UUID) (*Theory, error) {
	theory, err := s.repo.GetActiveTheory(ctx, s.db, topicID)
	if err != nil {
		return nil, fmt.Errorf("действующая теория темы: %w", err)
	}

	return theory, nil
}

// ListTheoryVersions возвращает историю версий теории.
func (s *Service) ListTheoryVersions(ctx context.Context, topicID uuid.UUID) ([]Theory, error) {
	return s.repo.ListTheoryVersions(ctx, s.db, topicID)
}
