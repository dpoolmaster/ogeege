package admin

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"

	"oge-ege/internal/auth"
	"oge-ege/internal/content"
	"oge-ege/internal/shared/clock"
	"oge-ege/internal/shared/db"
	"oge-ege/internal/shared/ids"
)

// TxManager выполняет функцию в транзакции.
type TxManager interface {
	WithTx(ctx context.Context, fn func(q db.Querier) error) error
}

// ContentAdmin описывает административные операции над контентом.
type ContentAdmin interface {
	ListExams(ctx context.Context, filter content.ExamFilter) ([]content.Exam, error)
	GetExam(ctx context.Context, id uuid.UUID) (*content.Exam, error)
	CreateExam(ctx context.Context, in content.ExamInput) (*content.Exam, error)
	UpdateExam(ctx context.Context, id uuid.UUID, in content.ExamInput) (*content.Exam, error)
	PublishExam(ctx context.Context, id uuid.UUID) (*content.Exam, error)
	DeleteExam(ctx context.Context, id uuid.UUID) error

	ListExamYears(ctx context.Context, filter content.ExamYearFilter) ([]content.ExamYear, error)
	GetExamYear(ctx context.Context, id uuid.UUID) (*content.ExamYear, error)
	CreateExamYear(ctx context.Context, in content.ExamYearInput) (*content.ExamYear, error)
	UpdateExamYear(ctx context.Context, id uuid.UUID, in content.ExamYearInput) (*content.ExamYear, error)
	PublishExamYear(ctx context.Context, id uuid.UUID) (*content.ExamYear, error)
	DeleteExamYear(ctx context.Context, id uuid.UUID) error

	ListKimNumbers(ctx context.Context, filter content.KimNumberFilter) ([]content.KimNumber, error)
	GetKimNumber(ctx context.Context, id uuid.UUID) (*content.KimNumber, error)
	CreateKimNumber(ctx context.Context, in content.KimNumberInput) (*content.KimNumber, error)
	UpdateKimNumber(ctx context.Context, id uuid.UUID, in content.KimNumberInput) (*content.KimNumber, error)
	DeleteKimNumber(ctx context.Context, id uuid.UUID) error
	ListKimTopics(ctx context.Context, kimNumberID uuid.UUID) ([]content.KimTopic, error)
	SetKimTopics(ctx context.Context, kimNumberID uuid.UUID, topics []content.KimTopicInput) ([]content.KimTopic, error)

	ListTopics(ctx context.Context, filter content.TopicFilter) ([]content.Topic, error)
	GetTopic(ctx context.Context, id uuid.UUID) (*content.Topic, error)
	CreateTopic(ctx context.Context, in content.TopicInput) (*content.Topic, error)
	UpdateTopic(ctx context.Context, id uuid.UUID, in content.TopicInput) (*content.Topic, error)
	DeleteTopic(ctx context.Context, id uuid.UUID) error

	ListTheoryVersions(ctx context.Context, topicID uuid.UUID) ([]content.Theory, error)
	GetActiveTheory(ctx context.Context, topicID uuid.UUID) (*content.Theory, error)
	CreateTheoryVersion(ctx context.Context, in content.TheoryInput) (*content.Theory, error)
	PublishTheory(ctx context.Context, id uuid.UUID) (*content.Theory, error)

	ListTasks(ctx context.Context, filter content.TaskFilter) ([]content.Task, error)
	GetTask(ctx context.Context, id uuid.UUID) (*content.Task, error)
	CreateTask(ctx context.Context, in content.TaskInput) (*content.Task, error)
	UpdateTask(ctx context.Context, id uuid.UUID, in content.TaskInput) (*content.Task, error)
	PublishTask(ctx context.Context, id uuid.UUID) (*content.Task, error)
	ArchiveTask(ctx context.Context, id uuid.UUID) (*content.Task, error)
	DeleteTask(ctx context.Context, id uuid.UUID) error
	ImportTasks(ctx context.Context, items []content.ImportItem) (*content.ImportReport, error)
	ListTaskVersions(ctx context.Context, taskID uuid.UUID) ([]content.TaskVersion, error)
	ListTaskAttachments(ctx context.Context, taskID uuid.UUID) ([]content.TaskAttachment, error)
	AddTaskAttachment(ctx context.Context, taskID uuid.UUID, in content.TaskAttachmentInput) (*content.TaskAttachment, error)

	ListVariants(ctx context.Context, filter content.VariantFilter) ([]content.Variant, error)
	GetVariant(ctx context.Context, id uuid.UUID) (*content.Variant, error)
	CreateVariant(ctx context.Context, in content.VariantInput) (*content.Variant, error)
	GenerateVariant(ctx context.Context, in content.GenerateVariantInput) (*content.Variant, error)
	PublishVariant(ctx context.Context, id uuid.UUID) (*content.Variant, error)

	ListRubrics(ctx context.Context, filter content.RubricFilter) ([]content.Rubric, error)
	GetRubric(ctx context.Context, id uuid.UUID) (*content.Rubric, error)
	CreateRubric(ctx context.Context, in content.RubricInput) (*content.Rubric, error)
	UpdateRubric(ctx context.Context, id uuid.UUID, criteria content.RubricCriteria) (*content.Rubric, error)
	PublishRubric(ctx context.Context, id uuid.UUID) (*content.Rubric, error)

	ListScoreScales(ctx context.Context, examYearID uuid.UUID) ([]content.ScoreScale, error)
	ReplaceScoreScales(ctx context.Context, in content.ScoreScaleInput) ([]content.ScoreScale, error)
}

// UserAdmin описывает административные операции над пользователями.
type UserAdmin interface {
	GetUser(ctx context.Context, id uuid.UUID) (*auth.User, error)
	SearchUsers(ctx context.Context, query string, cursor uuid.UUID, limit int) ([]auth.User, error)
	SetUserRole(ctx context.Context, id uuid.UUID, role auth.Role) (*auth.User, error)
	SetUserStatus(ctx context.Context, id uuid.UUID, status auth.Status) (*auth.User, error)
}

// Service содержит сценарии администрирования и аудита.
type Service struct {
	content  ContentAdmin
	users    UserAdmin
	repo     Repository
	database db.Querier
	tx       TxManager
	clock    clock.Clock
	logger   *slog.Logger
}

// NewService создаёт сервис admin.
func NewService(contentAdmin ContentAdmin, users UserAdmin, repo Repository, database db.Querier, tx TxManager, clk clock.Clock, logger *slog.Logger) *Service {
	return &Service{
		content:  contentAdmin,
		users:    users,
		repo:     repo,
		database: database,
		tx:       tx,
		clock:    clk,
		logger:   logger,
	}
}

// ListAudit возвращает журнал аудита.
func (s *Service) ListAudit(ctx context.Context, actor Actor, filter AuditFilter) ([]AuditEntry, error) {
	if err := requireAdmin(actor); err != nil {
		return nil, err
	}

	return s.repo.ListAudit(ctx, s.database, filter)
}

// SearchUsers ищет пользователей.
func (s *Service) SearchUsers(ctx context.Context, actor Actor, query string, cursor uuid.UUID, limit int) ([]UserSummary, error) {
	if err := requireAdmin(actor); err != nil {
		return nil, err
	}

	users, err := s.users.SearchUsers(ctx, query, cursor, limit)
	if err != nil {
		return nil, err
	}

	summaries := make([]UserSummary, 0, len(users))
	for i := range users {
		summaries = append(summaries, NewUserSummary(&users[i]))
	}

	return summaries, nil
}

// SetUserRole меняет роль пользователя.
func (s *Service) SetUserRole(ctx context.Context, actor Actor, id uuid.UUID, role auth.Role) (*UserSummary, error) {
	if err := requireAdmin(actor); err != nil {
		return nil, err
	}

	before, err := s.users.GetUser(ctx, id)
	if err != nil {
		return nil, err
	}
	updated, err := s.users.SetUserRole(ctx, id, role)
	if err != nil {
		return nil, err
	}

	summary := NewUserSummary(updated)
	s.recordAudit(ctx, actor, "update_role", "user", &id, NewUserSummary(before), summary)

	return &summary, nil
}

// SetUserStatus меняет статус пользователя.
func (s *Service) SetUserStatus(ctx context.Context, actor Actor, id uuid.UUID, status auth.Status) (*UserSummary, error) {
	if err := requireAdmin(actor); err != nil {
		return nil, err
	}

	before, err := s.users.GetUser(ctx, id)
	if err != nil {
		return nil, err
	}
	updated, err := s.users.SetUserStatus(ctx, id, status)
	if err != nil {
		return nil, err
	}

	summary := NewUserSummary(updated)
	s.recordAudit(ctx, actor, "update_status", "user", &id, NewUserSummary(before), summary)

	return &summary, nil
}

// CreateExam создаёт трек и пишет аудит.
func (s *Service) CreateExam(ctx context.Context, actor Actor, in content.ExamInput) (*content.Exam, error) {
	if err := requireContent(actor); err != nil {
		return nil, err
	}

	exam, err := s.content.CreateExam(ctx, in)
	if err != nil {
		return nil, err
	}
	s.recordAudit(ctx, actor, "create", "exam", &exam.ID, nil, exam)

	return exam, nil
}

// UpdateExam изменяет трек и пишет аудит.
func (s *Service) UpdateExam(ctx context.Context, actor Actor, id uuid.UUID, in content.ExamInput) (*content.Exam, error) {
	if err := requireContent(actor); err != nil {
		return nil, err
	}

	before, err := s.content.GetExam(ctx, id)
	if err != nil {
		return nil, err
	}
	exam, err := s.content.UpdateExam(ctx, id, in)
	if err != nil {
		return nil, err
	}
	s.recordAudit(ctx, actor, "update", "exam", &id, before, exam)

	return exam, nil
}

// PublishExam публикует трек и пишет аудит.
func (s *Service) PublishExam(ctx context.Context, actor Actor, id uuid.UUID) (*content.Exam, error) {
	if err := requireContent(actor); err != nil {
		return nil, err
	}

	before, err := s.content.GetExam(ctx, id)
	if err != nil {
		return nil, err
	}
	exam, err := s.content.PublishExam(ctx, id)
	if err != nil {
		return nil, err
	}
	s.recordAudit(ctx, actor, "publish", "exam", &id, before, exam)

	return exam, nil
}

// DeleteExam удаляет трек и пишет аудит.
func (s *Service) DeleteExam(ctx context.Context, actor Actor, id uuid.UUID) error {
	if err := requireContent(actor); err != nil {
		return err
	}

	before, err := s.content.GetExam(ctx, id)
	if err != nil {
		return err
	}
	if err := s.content.DeleteExam(ctx, id); err != nil {
		return err
	}
	s.recordAudit(ctx, actor, "delete", "exam", &id, before, nil)

	return nil
}

// CreateExamYear создаёт учебный год и пишет аудит.
func (s *Service) CreateExamYear(ctx context.Context, actor Actor, in content.ExamYearInput) (*content.ExamYear, error) {
	if err := requireContent(actor); err != nil {
		return nil, err
	}

	year, err := s.content.CreateExamYear(ctx, in)
	if err != nil {
		return nil, err
	}
	s.recordAudit(ctx, actor, "create", "exam_year", &year.ID, nil, year)

	return year, nil
}

// UpdateExamYear изменяет учебный год и пишет аудит.
func (s *Service) UpdateExamYear(ctx context.Context, actor Actor, id uuid.UUID, in content.ExamYearInput) (*content.ExamYear, error) {
	if err := requireContent(actor); err != nil {
		return nil, err
	}

	before, err := s.content.GetExamYear(ctx, id)
	if err != nil {
		return nil, err
	}
	year, err := s.content.UpdateExamYear(ctx, id, in)
	if err != nil {
		return nil, err
	}
	s.recordAudit(ctx, actor, "update", "exam_year", &id, before, year)

	return year, nil
}

// PublishExamYear публикует учебный год и пишет аудит.
func (s *Service) PublishExamYear(ctx context.Context, actor Actor, id uuid.UUID) (*content.ExamYear, error) {
	if err := requireContent(actor); err != nil {
		return nil, err
	}

	before, err := s.content.GetExamYear(ctx, id)
	if err != nil {
		return nil, err
	}
	year, err := s.content.PublishExamYear(ctx, id)
	if err != nil {
		return nil, err
	}
	s.recordAudit(ctx, actor, "publish", "exam_year", &id, before, year)

	return year, nil
}

// DeleteExamYear удаляет учебный год и пишет аудит.
func (s *Service) DeleteExamYear(ctx context.Context, actor Actor, id uuid.UUID) error {
	if err := requireContent(actor); err != nil {
		return err
	}

	before, err := s.content.GetExamYear(ctx, id)
	if err != nil {
		return err
	}
	if err := s.content.DeleteExamYear(ctx, id); err != nil {
		return err
	}
	s.recordAudit(ctx, actor, "delete", "exam_year", &id, before, nil)

	return nil
}

// CreateKimNumber создаёт номер КИМ и пишет аудит.
func (s *Service) CreateKimNumber(ctx context.Context, actor Actor, in content.KimNumberInput) (*content.KimNumber, error) {
	if err := requireContent(actor); err != nil {
		return nil, err
	}

	number, err := s.content.CreateKimNumber(ctx, in)
	if err != nil {
		return nil, err
	}
	s.recordAudit(ctx, actor, "create", "kim_number", &number.ID, nil, number)

	return number, nil
}

// UpdateKimNumber изменяет номер КИМ и пишет аудит.
func (s *Service) UpdateKimNumber(ctx context.Context, actor Actor, id uuid.UUID, in content.KimNumberInput) (*content.KimNumber, error) {
	if err := requireContent(actor); err != nil {
		return nil, err
	}

	before, err := s.content.GetKimNumber(ctx, id)
	if err != nil {
		return nil, err
	}
	number, err := s.content.UpdateKimNumber(ctx, id, in)
	if err != nil {
		return nil, err
	}
	s.recordAudit(ctx, actor, "update", "kim_number", &id, before, number)

	return number, nil
}

// DeleteKimNumber удаляет номер КИМ и пишет аудит.
func (s *Service) DeleteKimNumber(ctx context.Context, actor Actor, id uuid.UUID) error {
	if err := requireContent(actor); err != nil {
		return err
	}

	before, err := s.content.GetKimNumber(ctx, id)
	if err != nil {
		return err
	}
	if err := s.content.DeleteKimNumber(ctx, id); err != nil {
		return err
	}
	s.recordAudit(ctx, actor, "delete", "kim_number", &id, before, nil)

	return nil
}

// SetKimTopics заменяет связи тем и пишет аудит.
func (s *Service) SetKimTopics(ctx context.Context, actor Actor, kimNumberID uuid.UUID, topics []content.KimTopicInput) ([]content.KimTopic, error) {
	if err := requireContent(actor); err != nil {
		return nil, err
	}

	before, err := s.content.ListKimTopics(ctx, kimNumberID)
	if err != nil {
		return nil, err
	}
	links, err := s.content.SetKimTopics(ctx, kimNumberID, topics)
	if err != nil {
		return nil, err
	}
	s.recordAudit(ctx, actor, "set_topics", "kim_number", &kimNumberID, before, links)

	return links, nil
}

// CreateTopic создаёт тему и пишет аудит.
func (s *Service) CreateTopic(ctx context.Context, actor Actor, in content.TopicInput) (*content.Topic, error) {
	if err := requireContent(actor); err != nil {
		return nil, err
	}

	topic, err := s.content.CreateTopic(ctx, in)
	if err != nil {
		return nil, err
	}
	s.recordAudit(ctx, actor, "create", "topic", &topic.ID, nil, topic)

	return topic, nil
}

// UpdateTopic изменяет тему и пишет аудит.
func (s *Service) UpdateTopic(ctx context.Context, actor Actor, id uuid.UUID, in content.TopicInput) (*content.Topic, error) {
	if err := requireContent(actor); err != nil {
		return nil, err
	}

	before, err := s.content.GetTopic(ctx, id)
	if err != nil {
		return nil, err
	}
	topic, err := s.content.UpdateTopic(ctx, id, in)
	if err != nil {
		return nil, err
	}
	s.recordAudit(ctx, actor, "update", "topic", &id, before, topic)

	return topic, nil
}

// DeleteTopic удаляет тему и пишет аудит.
func (s *Service) DeleteTopic(ctx context.Context, actor Actor, id uuid.UUID) error {
	if err := requireContent(actor); err != nil {
		return err
	}

	before, err := s.content.GetTopic(ctx, id)
	if err != nil {
		return err
	}
	if err := s.content.DeleteTopic(ctx, id); err != nil {
		return err
	}
	s.recordAudit(ctx, actor, "delete", "topic", &id, before, nil)

	return nil
}

// CreateTheoryVersion создаёт версию теории и пишет аудит.
func (s *Service) CreateTheoryVersion(ctx context.Context, actor Actor, in content.TheoryInput) (*content.Theory, error) {
	if err := requireContent(actor); err != nil {
		return nil, err
	}

	theory, err := s.content.CreateTheoryVersion(ctx, in)
	if err != nil {
		return nil, err
	}
	s.recordAudit(ctx, actor, "create", "theory", &theory.ID, nil, theory)

	return theory, nil
}

// PublishTheory публикует теорию и пишет аудит.
func (s *Service) PublishTheory(ctx context.Context, actor Actor, id uuid.UUID) (*content.Theory, error) {
	if err := requireContent(actor); err != nil {
		return nil, err
	}

	theory, err := s.content.PublishTheory(ctx, id)
	if err != nil {
		return nil, err
	}
	s.recordAudit(ctx, actor, "publish", "theory", &id, nil, theory)

	return theory, nil
}

// CreateTask создаёт задание и пишет аудит.
func (s *Service) CreateTask(ctx context.Context, actor Actor, in content.TaskInput) (*content.Task, error) {
	if err := requireContent(actor); err != nil {
		return nil, err
	}

	task, err := s.content.CreateTask(ctx, in)
	if err != nil {
		return nil, err
	}
	s.recordAudit(ctx, actor, "create", "task", &task.ID, nil, task)

	return task, nil
}

// UpdateTask изменяет задание и пишет аудит.
func (s *Service) UpdateTask(ctx context.Context, actor Actor, id uuid.UUID, in content.TaskInput) (*content.Task, error) {
	if err := requireContent(actor); err != nil {
		return nil, err
	}

	before, err := s.content.GetTask(ctx, id)
	if err != nil {
		return nil, err
	}
	task, err := s.content.UpdateTask(ctx, id, in)
	if err != nil {
		return nil, err
	}
	s.recordAudit(ctx, actor, "update", "task", &id, before, task)

	return task, nil
}

// PublishTask публикует задание и пишет аудит.
func (s *Service) PublishTask(ctx context.Context, actor Actor, id uuid.UUID) (*content.Task, error) {
	if err := requireContent(actor); err != nil {
		return nil, err
	}

	before, err := s.content.GetTask(ctx, id)
	if err != nil {
		return nil, err
	}
	task, err := s.content.PublishTask(ctx, id)
	if err != nil {
		return nil, err
	}
	s.recordAudit(ctx, actor, "publish", "task", &id, before, task)

	return task, nil
}

// ArchiveTask архивирует задание и пишет аудит.
func (s *Service) ArchiveTask(ctx context.Context, actor Actor, id uuid.UUID) (*content.Task, error) {
	if err := requireContent(actor); err != nil {
		return nil, err
	}

	before, err := s.content.GetTask(ctx, id)
	if err != nil {
		return nil, err
	}
	task, err := s.content.ArchiveTask(ctx, id)
	if err != nil {
		return nil, err
	}
	s.recordAudit(ctx, actor, "archive", "task", &id, before, task)

	return task, nil
}

// DeleteTask удаляет задание и пишет аудит.
func (s *Service) DeleteTask(ctx context.Context, actor Actor, id uuid.UUID) error {
	if err := requireContent(actor); err != nil {
		return err
	}

	before, err := s.content.GetTask(ctx, id)
	if err != nil {
		return err
	}
	if err := s.content.DeleteTask(ctx, id); err != nil {
		return err
	}
	s.recordAudit(ctx, actor, "delete", "task", &id, before, nil)

	return nil
}

// ImportTasks импортирует задания и пишет один аудит-отчёт.
func (s *Service) ImportTasks(ctx context.Context, actor Actor, items []content.ImportItem) (*content.ImportReport, error) {
	if err := requireContent(actor); err != nil {
		return nil, err
	}

	report, err := s.content.ImportTasks(ctx, items)
	if err != nil {
		return nil, err
	}
	s.recordAudit(ctx, actor, "import", "task", nil, nil, report)

	return report, nil
}

// AddTaskAttachment сохраняет изображение задания и пишет аудит.
func (s *Service) AddTaskAttachment(ctx context.Context, actor Actor, taskID uuid.UUID, in content.TaskAttachmentInput) (*content.TaskAttachment, error) {
	if err := requireContent(actor); err != nil {
		return nil, err
	}

	attachment, err := s.content.AddTaskAttachment(ctx, taskID, in)
	if err != nil {
		return nil, err
	}
	s.recordAudit(ctx, actor, "attach", "task", &taskID, nil, attachment)

	return attachment, nil
}

// CreateVariant создаёт вариант и пишет аудит.
func (s *Service) CreateVariant(ctx context.Context, actor Actor, in content.VariantInput) (*content.Variant, error) {
	if err := requireContent(actor); err != nil {
		return nil, err
	}

	variant, err := s.content.CreateVariant(ctx, in)
	if err != nil {
		return nil, err
	}
	s.recordAudit(ctx, actor, "create", "variant", &variant.ID, nil, variant)

	return variant, nil
}

// GenerateVariant генерирует вариант и пишет аудит.
func (s *Service) GenerateVariant(ctx context.Context, actor Actor, in content.GenerateVariantInput) (*content.Variant, error) {
	if err := requireContent(actor); err != nil {
		return nil, err
	}

	variant, err := s.content.GenerateVariant(ctx, in)
	if err != nil {
		return nil, err
	}
	s.recordAudit(ctx, actor, "generate", "variant", &variant.ID, nil, variant)

	return variant, nil
}

// PublishVariant публикует вариант и пишет аудит.
func (s *Service) PublishVariant(ctx context.Context, actor Actor, id uuid.UUID) (*content.Variant, error) {
	if err := requireContent(actor); err != nil {
		return nil, err
	}

	before, err := s.content.GetVariant(ctx, id)
	if err != nil {
		return nil, err
	}
	variant, err := s.content.PublishVariant(ctx, id)
	if err != nil {
		return nil, err
	}
	s.recordAudit(ctx, actor, "publish", "variant", &id, before, variant)

	return variant, nil
}

// CreateRubric создаёт рубрику и пишет аудит.
func (s *Service) CreateRubric(ctx context.Context, actor Actor, in content.RubricInput) (*content.Rubric, error) {
	if err := requireContent(actor); err != nil {
		return nil, err
	}

	rubric, err := s.content.CreateRubric(ctx, in)
	if err != nil {
		return nil, err
	}
	s.recordAudit(ctx, actor, "create", "rubric", &rubric.ID, nil, rubric)

	return rubric, nil
}

// UpdateRubric создаёт новую версию рубрики и пишет аудит.
func (s *Service) UpdateRubric(ctx context.Context, actor Actor, id uuid.UUID, criteria content.RubricCriteria) (*content.Rubric, error) {
	if err := requireContent(actor); err != nil {
		return nil, err
	}

	before, err := s.content.GetRubric(ctx, id)
	if err != nil {
		return nil, err
	}
	rubric, err := s.content.UpdateRubric(ctx, id, criteria)
	if err != nil {
		return nil, err
	}
	s.recordAudit(ctx, actor, "update", "rubric", &rubric.ID, before, rubric)

	return rubric, nil
}

// PublishRubric публикует рубрику и пишет аудит.
func (s *Service) PublishRubric(ctx context.Context, actor Actor, id uuid.UUID) (*content.Rubric, error) {
	if err := requireContent(actor); err != nil {
		return nil, err
	}

	before, err := s.content.GetRubric(ctx, id)
	if err != nil {
		return nil, err
	}
	rubric, err := s.content.PublishRubric(ctx, id)
	if err != nil {
		return nil, err
	}
	s.recordAudit(ctx, actor, "publish", "rubric", &id, before, rubric)

	return rubric, nil
}

// ReplaceScoreScales сохраняет шкалу и пишет аудит.
func (s *Service) ReplaceScoreScales(ctx context.Context, actor Actor, in content.ScoreScaleInput) ([]content.ScoreScale, error) {
	if err := requireContent(actor); err != nil {
		return nil, err
	}

	before, err := s.content.ListScoreScales(ctx, in.ExamYearID)
	if err != nil {
		return nil, err
	}
	scales, err := s.content.ReplaceScoreScales(ctx, in)
	if err != nil {
		return nil, err
	}
	s.recordAudit(ctx, actor, "replace", "score_scale", &in.ExamYearID, before, scales)

	return scales, nil
}

func (s *Service) recordAudit(ctx context.Context, actor Actor, action, entity string, entityID *uuid.UUID, before, after any) {
	id, err := ids.New()
	if err != nil {
		s.logger.Error("не удалось создать идентификатор записи аудита", "error", err)
		return
	}

	entry := &AuditEntry{
		ID:        id,
		ActorID:   actor.ID,
		ActorRole: string(actor.Role),
		Action:    action,
		Entity:    entity,
		EntityID:  entityID,
		IP:        actor.IP,
		CreatedAt: s.clock.Now(),
	}
	if before != nil {
		entry.Before = mustJSON(before, s.logger)
	}
	if after != nil {
		entry.After = mustJSON(after, s.logger)
	}

	if err := s.tx.WithTx(ctx, func(q db.Querier) error {
		return s.repo.InsertAudit(ctx, q, entry)
	}); err != nil {
		s.logger.Error("не удалось записать аудит", "error", err, "action", action, "entity", entity)
	}
}

func requireContent(actor Actor) error {
	if actor.Role != auth.RoleMethodist && actor.Role != auth.RoleAdmin {
		return ErrForbidden
	}

	return nil
}

func requireAdmin(actor Actor) error {
	if actor.Role != auth.RoleAdmin {
		return ErrForbidden
	}

	return nil
}

func mustJSON(value any, logger *slog.Logger) json.RawMessage {
	data, err := json.Marshal(value)
	if err != nil {
		logger.Error("не удалось сериализовать аудит", "error", err)
		return nil
	}

	return data
}
