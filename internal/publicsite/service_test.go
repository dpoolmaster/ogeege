package publicsite

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"oge-ege/internal/content"
	"oge-ege/internal/notify"
	"oge-ege/internal/shared/db"
)

type testClock struct {
	now time.Time
}

func (c *testClock) Now() time.Time { return c.now }

type testTx struct{}

func (testTx) WithTx(_ context.Context, fn func(q db.Querier) error) error {
	return fn(nil)
}

type fakeRepo struct {
	Repository
	articles   map[string]*Article
	published  []Article
	seo        map[string]*SEOPage
	redirects  map[string]*Redirect
	leads      []Lead
	listCalls  int
	lastCursor *time.Time
}

func (r *fakeRepo) GetArticleBySlug(_ context.Context, _ db.Querier, slug string) (*Article, error) {
	article, ok := r.articles[slug]
	if !ok {
		return nil, ErrNotFound
	}

	return article, nil
}

func (r *fakeRepo) ListPublishedArticles(_ context.Context, _ db.Querier, cursor *time.Time, _ int) ([]Article, error) {
	r.listCalls++
	r.lastCursor = cursor

	return r.published, nil
}

func (r *fakeRepo) CreateLead(_ context.Context, _ db.Querier, lead *Lead) error {
	r.leads = append(r.leads, *lead)
	return nil
}

func (r *fakeRepo) GetSEOPageByPath(_ context.Context, _ db.Querier, path string) (*SEOPage, error) {
	page, ok := r.seo[path]
	if !ok {
		return nil, ErrNotFound
	}

	return page, nil
}

func (r *fakeRepo) ListSEOPages(_ context.Context, _ db.Querier) ([]SEOPage, error) {
	pages := make([]SEOPage, 0, len(r.seo))
	for _, page := range r.seo {
		pages = append(pages, *page)
	}

	return pages, nil
}

func (r *fakeRepo) GetRedirectByFromPath(_ context.Context, _ db.Querier, path string) (*Redirect, error) {
	redirect, ok := r.redirects[path]
	if !ok {
		return nil, ErrNotFound
	}

	return redirect, nil
}

type fakeContent struct {
	ContentProvider
	exams     []content.Exam
	exam      *content.Exam
	year      *content.ExamYear
	kim       *content.KimNumber
	kimList   []content.KimNumber
	topics    []content.Topic
	topic     *content.Topic
	tasks     []content.Task
	task      *content.Task
	attaches  []content.TaskAttachment
	lastTaskF content.TaskFilter
	demoCalls int
}

func (c *fakeContent) GetExamByCode(_ context.Context, _ string) (*content.Exam, error) {
	if c.exam == nil {
		return nil, content.ErrNotFound
	}

	return c.exam, nil
}

func (c *fakeContent) GetLatestPublishedExamYear(_ context.Context, _ uuid.UUID) (*content.ExamYear, error) {
	if c.year == nil {
		return nil, content.ErrNotFound
	}

	return c.year, nil
}

func (c *fakeContent) ListKimNumbers(_ context.Context, _ content.KimNumberFilter) ([]content.KimNumber, error) {
	return c.kimList, nil
}

func (c *fakeContent) GetKimNumberByNumber(_ context.Context, _ uuid.UUID, _ int) (*content.KimNumber, error) {
	if c.kim == nil {
		return nil, content.ErrNotFound
	}

	return c.kim, nil
}

func (c *fakeContent) ListTopics(_ context.Context, _ content.TopicFilter) ([]content.Topic, error) {
	return c.topics, nil
}

func (c *fakeContent) GetTopic(_ context.Context, _ uuid.UUID) (*content.Topic, error) {
	if c.topic == nil {
		return nil, content.ErrNotFound
	}

	return c.topic, nil
}

func (c *fakeContent) ListPublicTasks(_ context.Context, filter content.TaskFilter) ([]content.Task, error) {
	c.lastTaskF = filter
	return c.tasks, nil
}

func (c *fakeContent) ListDemoTasks(_ context.Context, _ uuid.UUID, _ int) ([]content.Task, error) {
	c.demoCalls++
	if len(c.tasks) < 5 {
		return nil, content.ErrInsufficient
	}

	return c.tasks, nil
}

func (c *fakeContent) GetTask(_ context.Context, id uuid.UUID) (*content.Task, error) {
	if c.task != nil && c.task.ID == id {
		return c.task, nil
	}
	for i := range c.tasks {
		if c.tasks[i].ID == id {
			return &c.tasks[i], nil
		}
	}
	if c.task != nil {
		return c.task, nil
	}

	return nil, content.ErrNotFound
}

func (c *fakeContent) ListTaskAttachments(_ context.Context, _ uuid.UUID) ([]content.TaskAttachment, error) {
	return c.attaches, nil
}

func (c *fakeContent) ListPublishedExams(_ context.Context) ([]content.Exam, error) {
	return c.exams, nil
}

type fakeRecipients struct {
	emails []string
	err    error
}

func (r fakeRecipients) ListActiveAdminEmails(context.Context) ([]string, error) {
	return r.emails, r.err
}

type fakeNotifier struct {
	messages []notify.Message
	err      error
}

func (n *fakeNotifier) Enqueue(_ context.Context, _ db.Querier, msg notify.Message) error {
	n.messages = append(n.messages, msg)
	return n.err
}

func newTestService(repo Repository, provider ContentProvider, recipients RecipientProvider, notifier Notifier) *Service {
	return NewService(
		repo, nil, testTx{}, &testClock{now: time.Date(2026, time.November, 27, 12, 0, 0, 0, time.UTC)},
		provider, recipients, notifier, slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
}

func TestLanding(t *testing.T) {
	exam := content.Exam{ID: uuid.Must(uuid.NewV7()), Code: "oge_math", Title: "ОГЭ математика", Status: content.StatusPublished}
	year := content.ExamYear{ID: uuid.Must(uuid.NewV7()), ExamID: exam.ID, Year: 2026, Status: content.StatusPublished}
	provider := &fakeContent{
		exam:    &exam,
		year:    &year,
		kimList: []content.KimNumber{{ID: uuid.Must(uuid.NewV7()), Number: 1}},
		topics:  []content.Topic{{ID: uuid.Must(uuid.NewV7()), Title: "Алгебра"}},
	}
	service := newTestService(&fakeRepo{}, provider, fakeRecipients{}, &fakeNotifier{})

	page, err := service.Landing(context.Background(), "oge_math")
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if page.Exam.Code != "oge_math" || page.Year == nil || len(page.KimNumbers) != 1 || len(page.Topics) != 1 {
		t.Fatalf("неполный лендинг: %+v", page)
	}

	draft := exam
	draft.Status = content.StatusDraft
	provider.exam = &draft
	if _, err := service.Landing(context.Background(), "oge_math"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("черновик должен давать ErrNotFound, получено %v", err)
	}

	provider.exam = nil
	if _, err := service.Landing(context.Background(), "unknown"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("неизвестный трек должен давать ErrNotFound, получено %v", err)
	}
}

func TestLandingWithoutYear(t *testing.T) {
	exam := content.Exam{ID: uuid.Must(uuid.NewV7()), Code: "oge_math", Title: "ОГЭ математика", Status: content.StatusPublished}
	provider := &fakeContent{exam: &exam, topics: []content.Topic{{ID: uuid.Must(uuid.NewV7()), Title: "Алгебра"}}}
	service := newTestService(&fakeRepo{}, provider, fakeRecipients{}, &fakeNotifier{})

	page, err := service.Landing(context.Background(), "oge_math")
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if page.Year != nil || len(page.KimNumbers) != 0 {
		t.Fatalf("без года не должно быть структуры: %+v", page)
	}
}

func TestBankFilters(t *testing.T) {
	exam := content.Exam{ID: uuid.Must(uuid.NewV7()), Code: "oge_math", Title: "ОГЭ математика", Status: content.StatusPublished}
	year := content.ExamYear{ID: uuid.Must(uuid.NewV7()), ExamID: exam.ID, Year: 2026, Status: content.StatusPublished}
	kim := content.KimNumber{ID: uuid.Must(uuid.NewV7()), ExamYearID: year.ID, Number: 1}
	topicID := uuid.Must(uuid.NewV7())
	tasks := []content.Task{{ID: uuid.Must(uuid.NewV7())}}
	provider := &fakeContent{exam: &exam, year: &year, kim: &kim, tasks: tasks}
	service := newTestService(&fakeRepo{}, provider, fakeRecipients{}, &fakeNotifier{})

	page, err := service.Bank(context.Background(), BankFilter{ExamCode: "oge_math", KimNumber: 1, TopicID: &topicID, Limit: 1})
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if !provider.lastTaskF.ExcludePublishedMock || len(provider.lastTaskF.Kinds) == 0 {
		t.Fatalf("фильтр банка не содержит ограничений: %+v", provider.lastTaskF)
	}
	if provider.lastTaskF.KimNumberID == nil || *provider.lastTaskF.KimNumberID != kim.ID {
		t.Fatal("фильтр по номеру КИМ не применён")
	}
	if page.NextCursor != tasks[0].ID {
		t.Fatal("курсор следующей страницы не установлен")
	}
}

func TestBankEmpty(t *testing.T) {
	service := newTestService(&fakeRepo{}, &fakeContent{}, fakeRecipients{}, &fakeNotifier{})
	page, err := service.Bank(context.Background(), BankFilter{})
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if len(page.Tasks) != 0 {
		t.Fatalf("ожидалось пустое состояние, получено %d", len(page.Tasks))
	}
}

func TestCheckTask(t *testing.T) {
	task := content.Task{
		ID: uuid.Must(uuid.NewV7()), Status: content.StatusPublished, IsPublic: true,
		AnswerSpec: json.RawMessage(`{"kind":"number","accepted":["2"]}`), AnswerDisplay: "2",
	}
	provider := &fakeContent{task: &task}
	service := newTestService(&fakeRepo{}, provider, fakeRecipients{}, &fakeNotifier{})

	result, err := service.CheckTask(context.Background(), task.ID, "2")
	if err != nil || !result.Correct {
		t.Fatalf("верный ответ: %+v, %v", result, err)
	}
	result, err = service.CheckTask(context.Background(), task.ID, "3")
	if err != nil || result.Correct {
		t.Fatalf("неверный ответ: %+v, %v", result, err)
	}

	unsupported := task
	unsupported.AnswerSpec = json.RawMessage(`{"kind":"choice","accepted":["2"]}`)
	provider.task = &unsupported
	if _, err := service.CheckTask(context.Background(), task.ID, "2"); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("ожидалась ErrUnsupported, получено %v", err)
	}

	draft := task
	draft.Status = content.StatusDraft
	provider.task = &draft
	if _, err := service.CheckTask(context.Background(), task.ID, "2"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("черновик должен давать ErrNotFound, получено %v", err)
	}
}

func TestDemoSigner(t *testing.T) {
	clk := &testClock{now: time.Date(2026, time.November, 27, 12, 0, 0, 0, time.UTC)}
	signer := NewDemoSigner("secret", 30*time.Minute, clk)
	payload := &DemoPayload{
		ExamYearID: uuid.Must(uuid.NewV7()),
		TaskIDs:    []uuid.UUID{uuid.Must(uuid.NewV7())},
	}

	token, err := signer.Sign(payload)
	if err != nil {
		t.Fatalf("подписать: %v", err)
	}
	verified, err := signer.Verify(token)
	if err != nil {
		t.Fatalf("проверить: %v", err)
	}
	if verified.ExamYearID != payload.ExamYearID {
		t.Fatal("данные не совпали")
	}

	clk.now = clk.now.Add(31 * time.Minute)
	if _, err := signer.Verify(token); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("истёкший токен должен отклоняться, получено %v", err)
	}

	clk.now = clk.now.Add(-31 * time.Minute)
	parts := strings.Split(token, ".")
	tampered := parts[0] + "." + strings.Repeat("a", len(parts[1]))
	if _, err := signer.Verify(tampered); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("изменённый токен должен отклоняться, получено %v", err)
	}
}

func TestCheckDemo(t *testing.T) {
	topicID := uuid.Must(uuid.NewV7())
	taskOne := content.Task{ID: uuid.Must(uuid.NewV7()), Status: content.StatusPublished, IsPublic: true, TopicID: &topicID, AnswerSpec: json.RawMessage(`{"kind":"number","accepted":["2"]}`)}
	taskTwo := content.Task{ID: uuid.Must(uuid.NewV7()), Status: content.StatusPublished, IsPublic: true, TopicID: &topicID, AnswerSpec: json.RawMessage(`{"kind":"number","accepted":["3"]}`)}

	provider := &taskProvider{tasks: map[uuid.UUID]content.Task{taskOne.ID: taskOne, taskTwo.ID: taskTwo}, topic: content.Topic{ID: topicID, Title: "Алгебра"}}
	service := newTestService(&fakeRepo{}, provider, fakeRecipients{}, &fakeNotifier{})

	result, err := service.CheckDemo(context.Background(), &DemoPayload{TaskIDs: []uuid.UUID{taskOne.ID, taskTwo.ID}}, map[uuid.UUID]string{
		taskOne.ID: "2",
		taskTwo.ID: "0",
	})
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if result.Total != 2 || result.Correct != 1 || len(result.Topics) != 1 || result.Topics[0].Title != "Алгебра" {
		t.Fatalf("неверный результат: %+v", result)
	}
}

type taskProvider struct {
	fakeContent
	tasks map[uuid.UUID]content.Task
	topic content.Topic
}

func (p *taskProvider) GetTask(_ context.Context, id uuid.UUID) (*content.Task, error) {
	task, ok := p.tasks[id]
	if !ok {
		return nil, content.ErrNotFound
	}

	return &task, nil
}

func (p *taskProvider) GetTopic(_ context.Context, _ uuid.UUID) (*content.Topic, error) {
	return &p.topic, nil
}

func TestSubmitLead(t *testing.T) {
	repo := &fakeRepo{}
	notifier := &fakeNotifier{}
	service := newTestService(repo, &fakeContent{}, fakeRecipients{emails: []string{"admin@example.ru"}}, notifier)

	err := service.SubmitLead(context.Background(), LeadInput{
		Name: "Иван", Contact: "ivan@example.ru", Comment: "Нужна помощь", Source: "/oge",
	})
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if len(repo.leads) != 1 || repo.leads[0].Status != LeadNew {
		t.Fatalf("заявка не сохранена: %+v", repo.leads)
	}
	if len(notifier.messages) != 1 || notifier.messages[0].Template != notify.TemplateCodeLeadNotification {
		t.Fatalf("уведомление не поставлено: %+v", notifier.messages)
	}
	if notifier.messages[0].DedupKey == "" {
		t.Fatal("dedup_key не задан")
	}
}

func TestSubmitLeadValidationAndHoneypot(t *testing.T) {
	repo := &fakeRepo{}
	notifier := &fakeNotifier{}
	service := newTestService(repo, &fakeContent{}, fakeRecipients{}, notifier)

	err := service.SubmitLead(context.Background(), LeadInput{Name: "И", Contact: "x"})
	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("ожидалась ошибка валидации, получено %v", err)
	}
	if len(repo.leads) != 0 {
		t.Fatal("некорректная заявка не должна сохраняться")
	}

	if err := service.SubmitLead(context.Background(), LeadInput{Name: "Иван", Contact: "ivan@example.ru", Website: "spam"}); err != nil {
		t.Fatalf("honeypot неожиданная ошибка: %v", err)
	}
	if len(repo.leads) != 0 || len(notifier.messages) != 0 {
		t.Fatal("honeypot не должен создавать заявку")
	}
}

func TestSubmitLeadNotificationErrorKeepsLead(t *testing.T) {
	repo := &fakeRepo{}
	notifier := &fakeNotifier{err: errors.New("outbox недоступен")}
	service := newTestService(repo, &fakeContent{}, fakeRecipients{emails: []string{"admin@example.ru"}}, notifier)

	if err := service.SubmitLead(context.Background(), LeadInput{Name: "Иван", Contact: "+7 999 123-45-67"}); err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if len(repo.leads) != 1 {
		t.Fatal("заявка должна остаться сохранённой")
	}
}

func TestArticleVisibilityAndSEO(t *testing.T) {
	published := time.Date(2026, time.November, 20, 10, 0, 0, 0, time.UTC)
	draft := &Article{ID: uuid.Must(uuid.NewV7()), Slug: "draft", Title: "Черновик", ContentMD: "x", Status: ArticleDraft}
	publishedArticle := &Article{ID: uuid.Must(uuid.NewV7()), Slug: "post", Title: "Статья", Excerpt: "Кратко", ContentMD: "## Текст", Status: ArticlePublished, PublishedAt: &published}
	repo := &fakeRepo{
		articles: map[string]*Article{"draft": draft, "post": publishedArticle},
		seo: map[string]*SEOPage{"/articles/post": {
			ID: uuid.Must(uuid.NewV7()), Path: "/articles/post", Title: "SEO заголовок", Description: "SEO описание",
		}},
	}
	service := newTestService(repo, &fakeContent{}, fakeRecipients{}, &fakeNotifier{})

	if _, err := service.Article(context.Background(), "draft"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("черновик должен быть недоступен, получено %v", err)
	}
	view, err := service.Article(context.Background(), "post")
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if !strings.Contains(view.ContentHTML, "<h2") {
		t.Fatalf("Markdown не отрендерен: %q", view.ContentHTML)
	}
	if view.SEO.Title != "SEO заголовок" || view.SEO.Description != "SEO описание" {
		t.Fatalf("SEO не переопределён: %+v", view.SEO)
	}
}

func TestSitemap(t *testing.T) {
	exam := content.Exam{ID: uuid.Must(uuid.NewV7()), Code: "oge_math", Status: content.StatusPublished, UpdatedAt: time.Now()}
	published := time.Now()
	repo := &fakeRepo{
		published: []Article{{ID: uuid.Must(uuid.NewV7()), Slug: "post", Status: ArticlePublished, PublishedAt: &published}},
		seo: map[string]*SEOPage{
			"/":     {ID: uuid.Must(uuid.NewV7()), Path: "/", UpdatedAt: time.Now()},
			"/bank": {ID: uuid.Must(uuid.NewV7()), Path: "/bank", UpdatedAt: time.Now()},
		},
	}
	service := newTestService(repo, &fakeContent{exams: []content.Exam{exam}}, fakeRecipients{}, &fakeNotifier{})

	entries, err := service.Sitemap(context.Background())
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	locations := make(map[string]bool)
	for _, entry := range entries {
		locations[entry.Loc] = true
	}
	for _, want := range []string{"/tracks/oge_math", "/", "/bank", "/articles/post"} {
		if !locations[want] {
			t.Fatalf("в sitemap нет %s: %+v", want, locations)
		}
	}
	if locations["/articles/draft"] || locations["/account"] {
		t.Fatal("в sitemap попали личные или черновые пути")
	}
}

func TestResolveRedirect(t *testing.T) {
	repo := &fakeRepo{redirects: map[string]*Redirect{
		"/oge":    {ID: uuid.Must(uuid.NewV7()), FromPath: "/oge", ToPath: "/tracks/oge_math", StatusCode: 301},
		"/danger": {ID: uuid.Must(uuid.NewV7()), FromPath: "/danger", ToPath: "//evil.example", StatusCode: 301},
	}}
	service := newTestService(repo, &fakeContent{}, fakeRecipients{}, &fakeNotifier{})

	redirect, err := service.ResolveRedirect(context.Background(), "/oge")
	if err != nil || redirect.ToPath != "/tracks/oge_math" {
		t.Fatalf("валидный редирект: %+v, %v", redirect, err)
	}
	if _, err := service.ResolveRedirect(context.Background(), "/missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("отсутствующий редирект: %v", err)
	}
	if _, err := service.ResolveRedirect(context.Background(), "/danger"); !errors.Is(err, ErrInvalidRedirect) {
		t.Fatalf("опасный редирект: %v", err)
	}
}

func TestCache(t *testing.T) {
	now := time.Date(2026, time.November, 27, 12, 0, 0, 0, time.UTC)
	cache := NewCache(time.Minute, 2)
	cache.now = func() time.Time { return now }

	if _, ok := cache.Get("/"); ok {
		t.Fatal("пустой кэш не должен попадать")
	}
	cache.Set("/", []byte("body"))
	if body, ok := cache.Get("/"); !ok || string(body) != "body" {
		t.Fatalf("промах после записи: %q, %v", body, ok)
	}
	now = now.Add(61 * time.Second)
	if _, ok := cache.Get("/"); ok {
		t.Fatal("истёкшая запись должна удаляться")
	}

	cache.Set("/a", []byte("a"))
	cache.Set("/b", []byte("b"))
	cache.Set("/c", []byte("c"))
	if _, ok := cache.Get("/a"); ok {
		t.Fatal("самая старая запись должна вытесняться")
	}
}
