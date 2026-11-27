package publicsite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"oge-ege/internal/content"
	"oge-ege/internal/grading"
	"oge-ege/internal/notify"
	"oge-ege/internal/shared/clock"
	"oge-ege/internal/shared/db"
	"oge-ege/internal/shared/ids"
	"oge-ege/internal/shared/markdown"
)

// ContentProvider — чтение опубликованного контента, которое объявляет publicsite.
type ContentProvider interface {
	GetExamByCode(ctx context.Context, code string) (*content.Exam, error)
	GetLatestPublishedExamYear(ctx context.Context, examID uuid.UUID) (*content.ExamYear, error)
	ListKimNumbers(ctx context.Context, filter content.KimNumberFilter) ([]content.KimNumber, error)
	GetKimNumberByNumber(ctx context.Context, examYearID uuid.UUID, number int) (*content.KimNumber, error)
	ListTopics(ctx context.Context, filter content.TopicFilter) ([]content.Topic, error)
	GetTopic(ctx context.Context, id uuid.UUID) (*content.Topic, error)
	ListPublicTasks(ctx context.Context, filter content.TaskFilter) ([]content.Task, error)
	ListDemoTasks(ctx context.Context, examYearID uuid.UUID, limit int) ([]content.Task, error)
	GetTask(ctx context.Context, id uuid.UUID) (*content.Task, error)
	ListTaskAttachments(ctx context.Context, taskID uuid.UUID) ([]content.TaskAttachment, error)
	ListPublishedExams(ctx context.Context) ([]content.Exam, error)
}

// RecipientProvider возвращает email активных администраторов.
type RecipientProvider interface {
	ListActiveAdminEmails(ctx context.Context) ([]string, error)
}

// Notifier ставит письмо в outbox.
type Notifier interface {
	Enqueue(ctx context.Context, q db.Querier, msg notify.Message) error
}

// TxManager выполняет функцию в транзакции.
type TxManager interface {
	WithTx(ctx context.Context, fn func(q db.Querier) error) error
}

// Service содержит сценарии публичного контура.
type Service struct {
	repo       Repository
	db         db.Querier
	tx         TxManager
	clock      clock.Clock
	content    ContentProvider
	recipients RecipientProvider
	notifier   Notifier
	logger     *slog.Logger
}

// NewService создаёт сервис publicsite.
func NewService(
	repo Repository,
	database db.Querier,
	tx TxManager,
	clk clock.Clock,
	contentProvider ContentProvider,
	recipients RecipientProvider,
	notifier Notifier,
	logger *slog.Logger,
) *Service {
	return &Service{
		repo:       repo,
		db:         database,
		tx:         tx,
		clock:      clk,
		content:    contentProvider,
		recipients: recipients,
		notifier:   notifier,
		logger:     logger,
	}
}

// SEOMetadata содержит метаданные публичной страницы.
type SEOMetadata struct {
	Title         string
	Description   string
	OGTitle       string
	OGDescription string
	OGImage       string
	Canonical     string
	NoIndex       bool
}

// Breadcrumb — элемент хлебных крошек.
type Breadcrumb struct {
	Title string
	Path  string
}

// HomePage — данные главной страницы.
type HomePage struct {
	Exams []content.Exam
	SEO   SEOMetadata
}

// LandingPage — данные лендинга трека.
type LandingPage struct {
	Exam       content.Exam
	Year       *content.ExamYear
	KimNumbers []content.KimNumber
	Topics     []content.Topic
	SEO        SEOMetadata
}

// BankFilter — фильтр публичного банка.
type BankFilter struct {
	ExamCode  string
	KimNumber int
	TopicID   *uuid.UUID
	Cursor    uuid.UUID
	Limit     int
}

// BankPage — данные страницы банка.
type BankPage struct {
	Exam       *content.Exam
	Year       *content.ExamYear
	Tasks      []content.Task
	KimNumbers []content.KimNumber
	Topics     []content.Topic
	NextCursor uuid.UUID
	HasNext    bool
	Filter     BankFilter
	SEO        SEOMetadata
}

// TaskPage — данные карточки задания.
type TaskPage struct {
	Task          content.Task
	StatementHTML string
	Topic         *content.Topic
	Attachments   []content.TaskAttachment
	Result        *CheckResult
	SEO           SEOMetadata
}

// CheckResult — результат проверки ответа.
type CheckResult struct {
	Correct       bool
	Reason        grading.Reason
	CorrectAnswer string
	Explanation   string
}

// DemoStart — данные запуска демо-диагностики.
type DemoStart struct {
	Exam  content.Exam
	Year  content.ExamYear
	Tasks []content.Task
	SEO   SEOMetadata
}

// TopicResult — разбивка результата демо по теме.
type TopicResult struct {
	Title   string
	Total   int
	Correct int
}

// DemoResult — результат демо-диагностики.
type DemoResult struct {
	Total   int
	Correct int
	Topics  []TopicResult
}

// ArticleView — данные страницы статьи.
type ArticleView struct {
	Article     Article
	ContentHTML string
	SEO         SEOMetadata
	Breadcrumbs []Breadcrumb
}

// SitemapEntry — адрес для sitemap.
type SitemapEntry struct {
	Loc     string
	LastMod time.Time
}

// Home возвращает данные главной страницы.
func (s *Service) Home(ctx context.Context) (*HomePage, error) {
	exams, err := s.content.ListPublishedExams(ctx)
	if err != nil {
		return nil, err
	}

	return &HomePage{
		Exams: exams,
		SEO:   s.seoFor(ctx, "/", "Подготовка к ОГЭ и ЕГЭ", "Публичный банк заданий, демо-диагностика и разборы."),
	}, nil
}

// Landing собирает лендинг опубликованного трека.
func (s *Service) Landing(ctx context.Context, code string) (*LandingPage, error) {
	exam, err := s.publishedExam(ctx, code)
	if err != nil {
		return nil, err
	}

	var year *content.ExamYear
	latest, err := s.content.GetLatestPublishedExamYear(ctx, exam.ID)
	if err == nil {
		year = latest
	} else if !errors.Is(err, content.ErrNotFound) {
		return nil, err
	}

	page := &LandingPage{Exam: *exam, Year: year}
	if year != nil {
		page.KimNumbers, err = s.content.ListKimNumbers(ctx, content.KimNumberFilter{
			ExamYearID: &year.ID,
			Limit:      200,
		})
		if err != nil {
			return nil, err
		}
	}

	page.Topics, err = s.content.ListTopics(ctx, content.TopicFilter{
		ExamID: &exam.ID,
		Limit:  200,
	})
	if err != nil {
		return nil, err
	}

	page.SEO = s.seoFor(ctx, "/tracks/"+exam.Code, exam.Title, "Подготовка к экзамену: структура, темы и демо-диагностика.")

	return page, nil
}

// Bank возвращает страницу публичного банка заданий.
func (s *Service) Bank(ctx context.Context, filter BankFilter) (*BankPage, error) {
	limit := normalizeLimit(filter.Limit)
	page := &BankPage{Filter: filter}

	taskFilter := content.TaskFilter{
		Cursor:               filter.Cursor,
		Limit:                limit,
		Kinds:                supportedKinds(),
		ExcludePublishedMock: true,
	}

	if strings.TrimSpace(filter.ExamCode) != "" {
		exam, err := s.publishedExam(ctx, filter.ExamCode)
		if err != nil {
			return nil, err
		}
		page.Exam = exam
		taskFilter.ExamID = &exam.ID

		year, err := s.content.GetLatestPublishedExamYear(ctx, exam.ID)
		if err == nil {
			page.Year = year
		} else if !errors.Is(err, content.ErrNotFound) {
			return nil, err
		}

		if page.Year != nil {
			page.KimNumbers, err = s.content.ListKimNumbers(ctx, content.KimNumberFilter{
				ExamYearID: &page.Year.ID,
				Limit:      200,
			})
			if err != nil {
				return nil, err
			}
		}

		page.Topics, err = s.content.ListTopics(ctx, content.TopicFilter{
			ExamID: &exam.ID,
			Limit:  200,
		})
		if err != nil {
			return nil, err
		}

		if filter.KimNumber > 0 {
			if page.Year == nil {
				return nil, ErrNotFound
			}
			kim, err := s.content.GetKimNumberByNumber(ctx, page.Year.ID, filter.KimNumber)
			if err != nil {
				if errors.Is(err, content.ErrNotFound) {
					return nil, ErrNotFound
				}
				return nil, err
			}
			taskFilter.KimNumberID = &kim.ID
		}
	}

	taskFilter.TopicID = filter.TopicID

	tasks, err := s.content.ListPublicTasks(ctx, taskFilter)
	if err != nil {
		return nil, err
	}
	page.Tasks = tasks
	if len(tasks) == limit {
		page.NextCursor = tasks[len(tasks)-1].ID
		page.HasNext = true
	}

	title := "Открытый банк заданий"
	if page.Exam != nil {
		title = "Банк заданий: " + page.Exam.Title
	}
	page.SEO = s.seoFor(ctx, "/bank", title, "Открытый банк заданий с автоматической проверкой ответа.")

	return page, nil
}

// Task возвращает карточку опубликованного публичного задания.
func (s *Service) Task(ctx context.Context, taskID uuid.UUID) (*TaskPage, error) {
	task, err := s.publicTask(ctx, taskID)
	if err != nil {
		return nil, err
	}

	statementHTML, err := markdown.Render(task.StatementMD)
	if err != nil {
		return nil, err
	}
	page := &TaskPage{Task: *task, StatementHTML: statementHTML}
	page.Attachments, err = s.content.ListTaskAttachments(ctx, task.ID)
	if err != nil {
		return nil, err
	}
	if task.TopicID != nil {
		topic, err := s.content.GetTopic(ctx, *task.TopicID)
		if err == nil {
			page.Topic = topic
		} else if !errors.Is(err, content.ErrNotFound) {
			return nil, err
		}
	}
	page.SEO = s.seoFor(ctx, "/bank/"+task.ID.String(), "Задание банка", "Задание открытого банка с проверкой ответа.")

	return page, nil
}

// CheckTask проверяет ответ на задание банка без сохранения результата.
func (s *Service) CheckTask(ctx context.Context, taskID uuid.UUID, answer string) (*CheckResult, error) {
	task, err := s.publicTask(ctx, taskID)
	if err != nil {
		return nil, err
	}

	kind, err := answerKind(task.AnswerSpec)
	if err != nil {
		return nil, ErrUnsupported
	}
	if !grading.Supports(kind) {
		return nil, ErrUnsupported
	}

	result, err := grading.Check(task.AnswerSpec, answer)
	if err != nil {
		return nil, fmt.Errorf("проверить ответ: %w", err)
	}

	explanation := "Правильный ответ: " + task.AnswerDisplay
	if task.TopicID != nil {
		if topic, err := s.content.GetTopic(ctx, *task.TopicID); err == nil {
			explanation += ". Тема: " + topic.Title
		}
	}

	return &CheckResult{
		Correct:       result.Correct,
		Reason:        result.Reason,
		CorrectAnswer: task.AnswerDisplay,
		Explanation:   explanation,
	}, nil
}

// StartDemo запускает демо-диагностику по опубликованному треку.
func (s *Service) StartDemo(ctx context.Context, examCode string) (*DemoStart, error) {
	exam, err := s.publishedExam(ctx, examCode)
	if err != nil {
		return nil, err
	}

	year, err := s.content.GetLatestPublishedExamYear(ctx, exam.ID)
	if err != nil {
		if errors.Is(err, content.ErrNotFound) {
			return nil, ErrInsufficient
		}
		return nil, err
	}

	tasks, err := s.content.ListDemoTasks(ctx, year.ID, 7)
	if err != nil {
		if errors.Is(err, content.ErrInsufficient) {
			return nil, ErrInsufficient
		}
		return nil, err
	}

	return &DemoStart{
		Exam:  *exam,
		Year:  *year,
		Tasks: tasks,
		SEO:   s.seoFor(ctx, "/diagnostics/demo", "Демо-диагностика", "Пройдите демо-диагностику без регистрации."),
	}, nil
}

// CheckDemo проверяет ответы демо-диагностики по подписанной сессии.
func (s *Service) CheckDemo(ctx context.Context, payload *DemoPayload, answers map[uuid.UUID]string) (*DemoResult, error) {
	if payload == nil || len(payload.TaskIDs) == 0 {
		return nil, ErrInvalidSession
	}

	result := &DemoResult{Total: len(payload.TaskIDs)}
	byTopic := make(map[string]*TopicResult)
	order := make([]string, 0)

	for _, taskID := range payload.TaskIDs {
		task, err := s.publicTask(ctx, taskID)
		if err != nil {
			return nil, ErrInvalidSession
		}

		kind, err := answerKind(task.AnswerSpec)
		if err != nil || !grading.Supports(kind) {
			return nil, ErrInvalidSession
		}

		checked, err := grading.Check(task.AnswerSpec, answers[taskID])
		if err != nil {
			return nil, ErrInvalidSession
		}

		title := "Без темы"
		if task.TopicID != nil {
			if topic, err := s.content.GetTopic(ctx, *task.TopicID); err == nil {
				title = topic.Title
			}
		}

		item, ok := byTopic[title]
		if !ok {
			item = &TopicResult{Title: title}
			byTopic[title] = item
			order = append(order, title)
		}
		item.Total++
		if checked.Correct {
			item.Correct++
			result.Correct++
		}
	}

	for _, title := range order {
		result.Topics = append(result.Topics, *byTopic[title])
	}

	return result, nil
}

// Articles возвращает опубликованные статьи.
func (s *Service) Articles(ctx context.Context, cursor *time.Time, limit int) ([]Article, error) {
	return s.repo.ListPublishedArticles(ctx, s.db, cursor, normalizeLimit(limit))
}

// Article возвращает опубликованную статью по slug.
func (s *Service) Article(ctx context.Context, slug string) (*ArticleView, error) {
	article, err := s.repo.GetArticleBySlug(ctx, s.db, strings.TrimSpace(slug))
	if err != nil {
		return nil, err
	}
	if article.Status != ArticlePublished || article.PublishedAt == nil {
		return nil, ErrNotFound
	}

	html, err := markdown.Render(article.ContentMD)
	if err != nil {
		return nil, err
	}

	path := "/articles/" + article.Slug
	seo := SEOMetadata{
		Title:         article.Title,
		Description:   article.Excerpt,
		OGTitle:       article.Title,
		OGDescription: article.Excerpt,
		Canonical:     path,
	}
	if page, err := s.repo.GetSEOPageByPath(ctx, s.db, path); err == nil {
		seo = mergeSEO(seo, page)
	}

	return &ArticleView{
		Article:     *article,
		ContentHTML: html,
		SEO:         seo,
		Breadcrumbs: []Breadcrumb{
			{Title: "Главная", Path: "/"},
			{Title: "Статьи", Path: "/articles"},
			{Title: article.Title, Path: path},
		},
	}, nil
}

// SubmitLead валидирует и сохраняет заявку, затем ставит уведомления администраторам.
func (s *Service) SubmitLead(ctx context.Context, in LeadInput) error {
	if strings.TrimSpace(in.Website) != "" {
		return nil
	}

	name := strings.TrimSpace(in.Name)
	if len([]rune(name)) < 2 || len([]rune(name)) > 100 {
		return invalid("name", "имя должно быть от 2 до 100 символов")
	}
	contact := strings.TrimSpace(in.Contact)
	if !validContact(contact) {
		return invalid("contact", "укажите корректный email или телефон")
	}
	comment := strings.TrimSpace(in.Comment)
	if len([]rune(comment)) > 2000 {
		return invalid("comment", "комментарий не должен превышать 2000 символов")
	}
	source := strings.TrimSpace(in.Source)
	if len([]rune(source)) > 100 {
		return invalid("source", "источник не должен превышать 100 символов")
	}

	id, err := ids.New()
	if err != nil {
		return err
	}
	lead := &Lead{
		ID:        id,
		Name:      name,
		Contact:   contact,
		Comment:   comment,
		Source:    source,
		Status:    LeadNew,
		CreatedAt: s.clock.Now(),
	}

	if err := s.tx.WithTx(ctx, func(q db.Querier) error {
		return s.repo.CreateLead(ctx, q, lead)
	}); err != nil {
		return err
	}

	s.notifyAdmins(ctx, lead)

	return nil
}

func (s *Service) notifyAdmins(ctx context.Context, lead *Lead) {
	recipients, err := s.recipients.ListActiveAdminEmails(ctx)
	if err != nil {
		s.logger.Error("не удалось получить получателей уведомления о заявке", "lead_id", lead.ID, "error", err)
		return
	}
	if len(recipients) == 0 {
		s.logger.Warn("нет активных администраторов для уведомления о заявке", "lead_id", lead.ID)
		return
	}

	for _, recipient := range recipients {
		msg := notify.Message{
			Channel:   notify.ChannelEmail,
			Recipient: recipient,
			Template:  notify.TemplateCodeLeadNotification,
			Payload: map[string]any{
				"name":    lead.Name,
				"contact": lead.Contact,
				"comment": lead.Comment,
				"source":  lead.Source,
			},
			DedupKey: fmt.Sprintf("lead:%s:%s", lead.ID, recipient),
		}
		if err := s.notifier.Enqueue(ctx, s.db, msg); err != nil {
			s.logger.Error("не удалось поставить уведомление о заявке", "lead_id", lead.ID, "error", err)
		}
	}
}

// Sitemap собирает канонические публичные адреса.
func (s *Service) Sitemap(ctx context.Context) ([]SitemapEntry, error) {
	seen := make(map[string]struct{})
	var entries []SitemapEntry
	add := func(loc string, lastMod time.Time) {
		loc = strings.TrimSpace(loc)
		if loc == "" || !strings.HasPrefix(loc, "/") || strings.HasPrefix(loc, "//") {
			return
		}
		if _, ok := seen[loc]; ok {
			return
		}
		seen[loc] = struct{}{}
		entries = append(entries, SitemapEntry{Loc: loc, LastMod: lastMod})
	}

	exams, err := s.content.ListPublishedExams(ctx)
	if err != nil {
		return nil, err
	}
	for _, exam := range exams {
		add("/tracks/"+exam.Code, exam.UpdatedAt)
	}

	pages, err := s.repo.ListSEOPages(ctx, s.db)
	if err != nil {
		return nil, err
	}
	for _, page := range pages {
		path := page.CanonicalPath
		if path == "" {
			path = page.Path
		}
		add(path, page.UpdatedAt)
	}

	cursor := (*time.Time)(nil)
	for {
		articles, err := s.repo.ListPublishedArticles(ctx, s.db, cursor, 50)
		if err != nil {
			return nil, err
		}
		for _, article := range articles {
			lastMod := article.UpdatedAt
			if article.PublishedAt != nil {
				lastMod = *article.PublishedAt
			}
			add("/articles/"+article.Slug, lastMod)
		}
		if len(articles) < 50 {
			break
		}
		last := articles[len(articles)-1]
		if last.PublishedAt == nil {
			break
		}
		cursor = last.PublishedAt
	}

	return entries, nil
}

// ResolveRedirect возвращает безопасный SEO-редирект.
func (s *Service) ResolveRedirect(ctx context.Context, path string) (*Redirect, error) {
	redirect, err := s.repo.GetRedirectByFromPath(ctx, s.db, path)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(redirect.ToPath, "/") || strings.HasPrefix(redirect.ToPath, "//") {
		return nil, ErrInvalidRedirect
	}
	switch redirect.StatusCode {
	case 301, 302, 307, 308:
	default:
		redirect.StatusCode = 301
	}

	return redirect, nil
}

// UpsertArticle сохраняет статью для seed.
func (s *Service) UpsertArticle(ctx context.Context, in ArticleInput) (*Article, error) {
	slug := strings.TrimSpace(in.Slug)
	if slug == "" {
		return nil, invalid("slug", "slug обязателен")
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return nil, invalid("title", "заголовок обязателен")
	}
	if strings.TrimSpace(in.ContentMD) == "" {
		return nil, invalid("content_md", "содержимое обязательно")
	}

	id, err := ids.New()
	if err != nil {
		return nil, err
	}
	now := s.clock.Now()
	article := &Article{
		ID:          id,
		Slug:        slug,
		Title:       title,
		Excerpt:     strings.TrimSpace(in.Excerpt),
		ContentMD:   in.ContentMD,
		Status:      in.Status,
		PublishedAt: in.PublishedAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if article.Status == "" {
		article.Status = ArticleDraft
	}
	if article.Status == ArticlePublished && article.PublishedAt == nil {
		article.PublishedAt = &now
	}

	if err := s.repo.UpsertArticle(ctx, s.db, article); err != nil {
		return nil, err
	}

	return article, nil
}

// UpsertSEOPage сохраняет метаданные страницы для seed.
func (s *Service) UpsertSEOPage(ctx context.Context, in SEOPageInput) error {
	if !strings.HasPrefix(in.Path, "/") || strings.HasPrefix(in.Path, "//") {
		return invalid("path", "путь должен начинаться с /")
	}
	if strings.TrimSpace(in.Title) == "" {
		return invalid("title", "заголовок обязателен")
	}
	id, err := ids.New()
	if err != nil {
		return err
	}
	now := s.clock.Now()

	return s.repo.UpsertSEOPage(ctx, s.db, &SEOPage{
		ID:            id,
		Path:          in.Path,
		Title:         strings.TrimSpace(in.Title),
		Description:   strings.TrimSpace(in.Description),
		OGTitle:       strings.TrimSpace(in.OGTitle),
		OGDescription: strings.TrimSpace(in.OGDescription),
		OGImage:       strings.TrimSpace(in.OGImage),
		CanonicalPath: strings.TrimSpace(in.CanonicalPath),
		NoIndex:       in.NoIndex,
		CreatedAt:     now,
		UpdatedAt:     now,
	})
}

// UpsertRedirect сохраняет SEO-редирект для seed.
func (s *Service) UpsertRedirect(ctx context.Context, in RedirectInput) error {
	if !strings.HasPrefix(in.FromPath, "/") || strings.HasPrefix(in.FromPath, "//") {
		return invalid("from_path", "исходный путь должен начинаться с /")
	}
	if !strings.HasPrefix(in.ToPath, "/") || strings.HasPrefix(in.ToPath, "//") {
		return invalid("to_path", "целевой путь должен начинаться с / и не с //")
	}
	status := in.StatusCode
	if status == 0 {
		status = 301
	}
	switch status {
	case 301, 302, 307, 308:
	default:
		return invalid("status_code", "неподдерживаемый код перенаправления")
	}
	id, err := ids.New()
	if err != nil {
		return err
	}

	return s.repo.UpsertRedirect(ctx, s.db, &Redirect{
		ID:         id,
		FromPath:   in.FromPath,
		ToPath:     in.ToPath,
		StatusCode: status,
		CreatedAt:  s.clock.Now(),
	})
}

func (s *Service) publishedExam(ctx context.Context, code string) (*content.Exam, error) {
	exam, err := s.content.GetExamByCode(ctx, strings.TrimSpace(code))
	if err != nil {
		if errors.Is(err, content.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if exam.Status != content.StatusPublished {
		return nil, ErrNotFound
	}

	return exam, nil
}

func (s *Service) publicTask(ctx context.Context, taskID uuid.UUID) (*content.Task, error) {
	task, err := s.content.GetTask(ctx, taskID)
	if err != nil {
		if errors.Is(err, content.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if task.Status != content.StatusPublished || !task.IsPublic {
		return nil, ErrNotFound
	}
	kind, err := answerKind(task.AnswerSpec)
	if err != nil || !grading.Supports(kind) {
		return nil, ErrUnsupported
	}

	return task, nil
}

func (s *Service) seoFor(ctx context.Context, path, title, description string) SEOMetadata {
	seo := SEOMetadata{
		Title:         title,
		Description:   description,
		OGTitle:       title,
		OGDescription: description,
		Canonical:     path,
	}
	page, err := s.repo.GetSEOPageByPath(ctx, s.db, path)
	if err != nil {
		return seo
	}

	return mergeSEO(seo, page)
}

func mergeSEO(base SEOMetadata, page *SEOPage) SEOMetadata {
	if page.Title != "" {
		base.Title = page.Title
	}
	if page.Description != "" {
		base.Description = page.Description
	}
	if page.OGTitle != "" {
		base.OGTitle = page.OGTitle
	} else {
		base.OGTitle = base.Title
	}
	if page.OGDescription != "" {
		base.OGDescription = page.OGDescription
	} else {
		base.OGDescription = base.Description
	}
	if page.OGImage != "" {
		base.OGImage = page.OGImage
	}
	if page.CanonicalPath != "" {
		base.Canonical = page.CanonicalPath
	} else if base.Canonical == "" {
		base.Canonical = page.Path
	}
	base.NoIndex = page.NoIndex

	return base
}

func supportedKinds() []string {
	kinds := []string{"number", "text", "digits", "interval"}
	supported := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		if grading.Supports(kind) {
			supported = append(supported, kind)
		}
	}

	return supported
}

func answerKind(spec json.RawMessage) (string, error) {
	var payload struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(spec, &payload); err != nil {
		return "", err
	}
	if payload.Kind == "" {
		return "", errors.New("пустой kind")
	}

	return payload.Kind, nil
}

func intString(value int) string {
	return strconv.Itoa(value)
}

func validContact(contact string) bool {
	if len([]rune(contact)) < 3 || len([]rune(contact)) > 200 {
		return false
	}
	if strings.Contains(contact, "@") {
		parts := strings.Split(contact, "@")
		return len(parts) == 2 && parts[0] != "" && strings.Contains(parts[1], ".") && !strings.HasPrefix(parts[1], ".") && !strings.HasSuffix(parts[1], ".")
	}

	digits := 0
	for _, r := range contact {
		switch {
		case r >= '0' && r <= '9':
			digits++
		case r == '+', r == '-', r == ' ', r == '(', r == ')':
		default:
			return false
		}
	}

	return digits >= 10
}
