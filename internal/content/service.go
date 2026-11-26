package content

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"oge-ege/internal/shared/clock"
	"oge-ege/internal/shared/db"
	"oge-ege/internal/shared/ids"
	"oge-ege/internal/shared/markdown"
)

// TxManager выполняет функцию в транзакции.
type TxManager interface {
	WithTx(ctx context.Context, fn func(q db.Querier) error) error
}

// Shuffler перемешивает срез заданий; подменяется в тестах.
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

// Service содержит сценарии модуля content.
type Service struct {
	repo     Repository
	db       db.Querier
	tx       TxManager
	clock    clock.Clock
	shuffler Shuffler
	logger   *slog.Logger
}

// NewService создаёт сервис content.
func NewService(repo Repository, database db.Querier, tx TxManager, clk clock.Clock, shuffler Shuffler, logger *slog.Logger) *Service {
	if shuffler == nil {
		shuffler = &lockedShuffler{r: rand.New(rand.NewSource(time.Now().UnixNano()))}
	}

	return &Service{
		repo:     repo,
		db:       database,
		tx:       tx,
		clock:    clk,
		shuffler: shuffler,
		logger:   logger,
	}
}

// CreateExam создаёт черновик трека.
func (s *Service) CreateExam(ctx context.Context, in ExamInput) (*Exam, error) {
	code, err := required("code", in.Code)
	if err != nil {
		return nil, err
	}
	title, err := required("title", in.Title)
	if err != nil {
		return nil, err
	}
	subject, err := required("subject", in.Subject)
	if err != nil {
		return nil, err
	}

	id, err := ids.New()
	if err != nil {
		return nil, err
	}
	now := s.clock.Now()
	exam := &Exam{
		ID:        id,
		Code:      code,
		Title:     title,
		Subject:   subject,
		Status:    StatusDraft,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.CreateExam(ctx, s.db, exam); err != nil {
		return nil, err
	}

	return exam, nil
}

// UpdateExam изменяет данные трека.
func (s *Service) UpdateExam(ctx context.Context, id uuid.UUID, in ExamInput) (*Exam, error) {
	code, err := required("code", in.Code)
	if err != nil {
		return nil, err
	}
	title, err := required("title", in.Title)
	if err != nil {
		return nil, err
	}
	subject, err := required("subject", in.Subject)
	if err != nil {
		return nil, err
	}

	exam, err := s.repo.GetExam(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	exam.Code = code
	exam.Title = title
	exam.Subject = subject
	exam.UpdatedAt = s.clock.Now()

	if err := s.repo.UpdateExam(ctx, s.db, exam); err != nil {
		return nil, err
	}

	return exam, nil
}

// GetExam возвращает трек по идентификатору.
func (s *Service) GetExam(ctx context.Context, id uuid.UUID) (*Exam, error) {
	return s.repo.GetExam(ctx, s.db, id)
}

// GetExamByCode возвращает трек по коду.
func (s *Service) GetExamByCode(ctx context.Context, code string) (*Exam, error) {
	return s.repo.GetExamByCode(ctx, s.db, strings.TrimSpace(code))
}

// ListExams возвращает страницу треков.
func (s *Service) ListExams(ctx context.Context, filter ExamFilter) ([]Exam, error) {
	return s.repo.ListExams(ctx, s.db, filter)
}

// ListPublishedExams возвращает опубликованные треки.
func (s *Service) ListPublishedExams(ctx context.Context) ([]Exam, error) {
	return s.repo.ListPublishedExams(ctx, s.db)
}

// PublishExam публикует трек.
func (s *Service) PublishExam(ctx context.Context, id uuid.UUID) (*Exam, error) {
	exam, err := s.repo.GetExam(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	now := s.clock.Now()
	if err := s.repo.SetExamStatus(ctx, s.db, id, StatusPublished, now); err != nil {
		return nil, err
	}
	exam.Status = StatusPublished
	exam.UpdatedAt = now

	return exam, nil
}

// DeleteExam удаляет черновик трека.
func (s *Service) DeleteExam(ctx context.Context, id uuid.UUID) error {
	exam, err := s.repo.GetExam(ctx, s.db, id)
	if err != nil {
		return err
	}
	if exam.Status == StatusPublished {
		return fmt.Errorf("удалить опубликованный трек: %w", ErrConflict)
	}

	return s.repo.DeleteExam(ctx, s.db, id)
}

// IsExamAvailable сообщает, существует ли трек и опубликован ли он.
func (s *Service) IsExamAvailable(ctx context.Context, id uuid.UUID) (bool, error) {
	exam, err := s.repo.GetExam(ctx, s.db, id)
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return exam.Status == StatusPublished, nil
}

// CreateExamYear создаёт черновик учебного года.
func (s *Service) CreateExamYear(ctx context.Context, in ExamYearInput) (*ExamYear, error) {
	if _, err := s.repo.GetExam(ctx, s.db, in.ExamID); err != nil {
		return nil, invalid("exam_id", "трек не найден")
	}
	if err := validateYear(in.Year); err != nil {
		return nil, err
	}
	if err := validateDuration(in.DurationMinutes); err != nil {
		return nil, err
	}

	id, err := ids.New()
	if err != nil {
		return nil, err
	}
	now := s.clock.Now()
	year := &ExamYear{
		ID:              id,
		ExamID:          in.ExamID,
		Year:            in.Year,
		DurationMinutes: in.DurationMinutes,
		Status:          StatusDraft,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.repo.CreateExamYear(ctx, s.db, year); err != nil {
		return nil, err
	}

	return year, nil
}

// UpdateExamYear изменяет учебный год.
func (s *Service) UpdateExamYear(ctx context.Context, id uuid.UUID, in ExamYearInput) (*ExamYear, error) {
	if err := validateYear(in.Year); err != nil {
		return nil, err
	}
	if err := validateDuration(in.DurationMinutes); err != nil {
		return nil, err
	}

	year, err := s.repo.GetExamYear(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	year.Year = in.Year
	year.DurationMinutes = in.DurationMinutes
	year.UpdatedAt = s.clock.Now()

	if err := s.repo.UpdateExamYear(ctx, s.db, year); err != nil {
		return nil, err
	}

	return year, nil
}

// GetExamYear возвращает учебный год.
func (s *Service) GetExamYear(ctx context.Context, id uuid.UUID) (*ExamYear, error) {
	return s.repo.GetExamYear(ctx, s.db, id)
}

// GetExamYearByYear возвращает учебный год трека.
func (s *Service) GetExamYearByYear(ctx context.Context, examID uuid.UUID, year int) (*ExamYear, error) {
	return s.repo.GetExamYearByYear(ctx, s.db, examID, year)
}

// ListExamYears возвращает страницу учебных годов.
func (s *Service) ListExamYears(ctx context.Context, filter ExamYearFilter) ([]ExamYear, error) {
	return s.repo.ListExamYears(ctx, s.db, filter)
}

// PublishExamYear публикует учебный год.
func (s *Service) PublishExamYear(ctx context.Context, id uuid.UUID) (*ExamYear, error) {
	year, err := s.repo.GetExamYear(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	now := s.clock.Now()
	if err := s.repo.SetExamYearStatus(ctx, s.db, id, StatusPublished, now); err != nil {
		return nil, err
	}
	year.Status = StatusPublished
	year.UpdatedAt = now

	return year, nil
}

// DeleteExamYear удаляет черновик учебного года.
func (s *Service) DeleteExamYear(ctx context.Context, id uuid.UUID) error {
	year, err := s.repo.GetExamYear(ctx, s.db, id)
	if err != nil {
		return err
	}
	if year.Status == StatusPublished {
		return fmt.Errorf("удалить опубликованный учебный год: %w", ErrConflict)
	}

	return s.repo.DeleteExamYear(ctx, s.db, id)
}

// CreateKimNumber создаёт номер КИМ.
func (s *Service) CreateKimNumber(ctx context.Context, in KimNumberInput) (*KimNumber, error) {
	if _, err := s.repo.GetExamYear(ctx, s.db, in.ExamYearID); err != nil {
		return nil, invalid("exam_year_id", "учебный год не найден")
	}
	if in.Number <= 0 {
		return nil, invalid("number", "номер КИМ должен быть положительным")
	}
	if in.Part != 1 && in.Part != 2 {
		return nil, invalid("part", "часть должна быть 1 или 2")
	}
	if !in.AnswerType.Valid() {
		return nil, invalid("answer_type", "неизвестный тип ответа")
	}
	if in.MaxPrimaryScore < 0 {
		return nil, invalid("max_primary_score", "максимальный балл не может быть отрицательным")
	}

	id, err := ids.New()
	if err != nil {
		return nil, err
	}
	now := s.clock.Now()
	number := &KimNumber{
		ID:              id,
		ExamYearID:      in.ExamYearID,
		Number:          in.Number,
		Part:            in.Part,
		AnswerType:      in.AnswerType,
		MaxPrimaryScore: in.MaxPrimaryScore,
		Title:           strings.TrimSpace(in.Title),
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.repo.CreateKimNumber(ctx, s.db, number); err != nil {
		return nil, err
	}

	return number, nil
}

// UpdateKimNumber изменяет номер КИМ.
func (s *Service) UpdateKimNumber(ctx context.Context, id uuid.UUID, in KimNumberInput) (*KimNumber, error) {
	if in.Number <= 0 {
		return nil, invalid("number", "номер КИМ должен быть положительным")
	}
	if in.Part != 1 && in.Part != 2 {
		return nil, invalid("part", "часть должна быть 1 или 2")
	}
	if !in.AnswerType.Valid() {
		return nil, invalid("answer_type", "неизвестный тип ответа")
	}
	if in.MaxPrimaryScore < 0 {
		return nil, invalid("max_primary_score", "максимальный балл не может быть отрицательным")
	}

	number, err := s.repo.GetKimNumber(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	number.Number = in.Number
	number.Part = in.Part
	number.AnswerType = in.AnswerType
	number.MaxPrimaryScore = in.MaxPrimaryScore
	number.Title = strings.TrimSpace(in.Title)
	number.UpdatedAt = s.clock.Now()

	if err := s.repo.UpdateKimNumber(ctx, s.db, number); err != nil {
		return nil, err
	}

	return number, nil
}

// GetKimNumber возвращает номер КИМ.
func (s *Service) GetKimNumber(ctx context.Context, id uuid.UUID) (*KimNumber, error) {
	return s.repo.GetKimNumber(ctx, s.db, id)
}

// GetKimNumberByNumber возвращает номер КИМ учебного года.
func (s *Service) GetKimNumberByNumber(ctx context.Context, examYearID uuid.UUID, number int) (*KimNumber, error) {
	return s.repo.GetKimNumberByNumber(ctx, s.db, examYearID, number)
}

// ListKimNumbers возвращает страницу номеров КИМ.
func (s *Service) ListKimNumbers(ctx context.Context, filter KimNumberFilter) ([]KimNumber, error) {
	return s.repo.ListKimNumbers(ctx, s.db, filter)
}

// DeleteKimNumber удаляет номер КИМ.
func (s *Service) DeleteKimNumber(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteKimNumber(ctx, s.db, id)
}

// CreateTopic создаёт тему.
func (s *Service) CreateTopic(ctx context.Context, in TopicInput) (*Topic, error) {
	exam, err := s.repo.GetExam(ctx, s.db, in.ExamID)
	if err != nil {
		return nil, invalid("exam_id", "трек не найден")
	}
	code, err := required("code", in.Code)
	if err != nil {
		return nil, err
	}
	title, err := required("title", in.Title)
	if err != nil {
		return nil, err
	}
	if in.OrderIndex < 0 {
		return nil, invalid("order_index", "порядок не может быть отрицательным")
	}
	if err := s.validateParent(ctx, exam.ID, in.ParentID); err != nil {
		return nil, err
	}

	id, err := ids.New()
	if err != nil {
		return nil, err
	}
	now := s.clock.Now()
	topic := &Topic{
		ID:         id,
		ExamID:     in.ExamID,
		ParentID:   in.ParentID,
		Code:       code,
		Title:      title,
		OrderIndex: in.OrderIndex,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.repo.CreateTopic(ctx, s.db, topic); err != nil {
		return nil, err
	}

	return topic, nil
}

// UpdateTopic изменяет тему.
func (s *Service) UpdateTopic(ctx context.Context, id uuid.UUID, in TopicInput) (*Topic, error) {
	topic, err := s.repo.GetTopic(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	code, err := required("code", in.Code)
	if err != nil {
		return nil, err
	}
	title, err := required("title", in.Title)
	if err != nil {
		return nil, err
	}
	if in.OrderIndex < 0 {
		return nil, invalid("order_index", "порядок не может быть отрицательным")
	}
	if in.ParentID != nil && *in.ParentID == id {
		return nil, invalid("parent_id", "тема не может быть родителем самой себя")
	}
	if err := s.validateParent(ctx, topic.ExamID, in.ParentID); err != nil {
		return nil, err
	}

	topic.ParentID = in.ParentID
	topic.Code = code
	topic.Title = title
	topic.OrderIndex = in.OrderIndex
	topic.UpdatedAt = s.clock.Now()

	if err := s.repo.UpdateTopic(ctx, s.db, topic); err != nil {
		return nil, err
	}

	return topic, nil
}

// GetTopic возвращает тему.
func (s *Service) GetTopic(ctx context.Context, id uuid.UUID) (*Topic, error) {
	return s.repo.GetTopic(ctx, s.db, id)
}

// GetTopicByCode возвращает тему трека по коду.
func (s *Service) GetTopicByCode(ctx context.Context, examID uuid.UUID, code string) (*Topic, error) {
	return s.repo.GetTopicByCode(ctx, s.db, examID, strings.TrimSpace(code))
}

// ListTopics возвращает страницу тем.
func (s *Service) ListTopics(ctx context.Context, filter TopicFilter) ([]Topic, error) {
	return s.repo.ListTopics(ctx, s.db, filter)
}

// DeleteTopic удаляет тему.
func (s *Service) DeleteTopic(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteTopic(ctx, s.db, id)
}

// SetKimTopics заменяет связи номера КИМ с темами.
func (s *Service) SetKimTopics(ctx context.Context, kimNumberID uuid.UUID, topics []KimTopicInput) ([]KimTopic, error) {
	number, err := s.repo.GetKimNumber(ctx, s.db, kimNumberID)
	if err != nil {
		return nil, err
	}
	year, err := s.repo.GetExamYear(ctx, s.db, number.ExamYearID)
	if err != nil {
		return nil, err
	}

	seen := make(map[uuid.UUID]struct{}, len(topics))
	links := make([]KimTopic, 0, len(topics))
	for _, item := range topics {
		if item.Weight < 0 {
			return nil, invalid("weight", "вес темы не может быть отрицательным")
		}
		if _, ok := seen[item.TopicID]; ok {
			return nil, invalid("topic_id", "тема указана повторно")
		}
		seen[item.TopicID] = struct{}{}

		topic, err := s.repo.GetTopic(ctx, s.db, item.TopicID)
		if err != nil {
			return nil, invalid("topic_id", "тема не найдена")
		}
		if topic.ExamID != year.ExamID {
			return nil, invalid("topic_id", "тема принадлежит другому треку")
		}

		links = append(links, KimTopic{KimNumberID: kimNumberID, TopicID: item.TopicID, Weight: item.Weight})
	}

	if err := s.tx.WithTx(ctx, func(q db.Querier) error {
		return s.repo.ReplaceKimTopics(ctx, q, kimNumberID, links)
	}); err != nil {
		return nil, err
	}

	return links, nil
}

// ListKimTopics возвращает связи номера КИМ с темами.
func (s *Service) ListKimTopics(ctx context.Context, kimNumberID uuid.UUID) ([]KimTopic, error) {
	return s.repo.ListKimTopics(ctx, s.db, kimNumberID)
}

func (s *Service) validateParent(ctx context.Context, examID uuid.UUID, parentID *uuid.UUID) error {
	if parentID == nil {
		return nil
	}

	parent, err := s.repo.GetTopic(ctx, s.db, *parentID)
	if err != nil {
		return invalid("parent_id", "родительская тема не найдена")
	}
	if parent.ExamID != examID {
		return invalid("parent_id", "родительская тема принадлежит другому треку")
	}

	return nil
}

func required(field, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", invalid(field, "поле обязательно")
	}

	return value, nil
}

func validateYear(year int) error {
	if year < 2000 || year > 2100 {
		return invalid("year", "учебный год должен быть в диапазоне 2000–2100")
	}

	return nil
}

func validateDuration(minutes int) error {
	if minutes <= 0 {
		return invalid("duration_minutes", "длительность должна быть положительной")
	}

	return nil
}

// NormalizeStatement нормализует условие задания для расчёта хеша.
func NormalizeStatement(statement string) string {
	statement = strings.ReplaceAll(statement, "\r\n", "\n")
	statement = strings.ReplaceAll(statement, "\r", "\n")

	return strings.Join(strings.Fields(statement), " ")
}

// ContentHash рассчитывает SHA-256 от нормализованного условия.
func ContentHash(statement string) []byte {
	sum := sha256.Sum256([]byte(NormalizeStatement(statement)))

	return sum[:]
}

func validateAnswerSpec(spec []byte) error {
	if len(spec) == 0 {
		return invalid("answer_spec", "правила проверки обязательны")
	}

	var payload struct {
		Kind string `json:"kind"`
	}
	if err := jsonUnmarshal(spec, &payload); err != nil {
		return invalid("answer_spec", "правила проверки должны быть корректным JSON")
	}

	switch payload.Kind {
	case "number", "interval", "digits", "extended", "text", "choice":
		return nil
	default:
		return invalid("answer_spec", "неизвестный тип правил проверки")
	}
}

func jsonUnmarshal(data []byte, value any) error {
	if err := json.Unmarshal(data, value); err != nil {
		return fmt.Errorf("разобрать JSON: %w", err)
	}

	return nil
}

func validateLaTeX(source string) error {
	if err := markdown.ValidateFormulas(source); err != nil {
		return invalid("statement_md", err.Error())
	}

	return nil
}
