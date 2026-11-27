package publicsite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"oge-ege/internal/content"
	"oge-ege/internal/notify"
	"oge-ege/internal/shared/clock"
	"oge-ege/internal/shared/db"
	"oge-ege/internal/shared/httpserver"
)

type integrationTx struct {
	tx pgx.Tx
}

func (m integrationTx) WithTx(_ context.Context, fn func(q db.Querier) error) error {
	return fn(m.tx)
}

func openIntegrationTx(t *testing.T) (context.Context, pgx.Tx, func()) {
	t.Helper()

	databaseURL, ok := os.LookupEnv("TEST_DATABASE_URL")
	if !ok || databaseURL == "" {
		t.Skip("TEST_DATABASE_URL не задан")
	}

	ctx := context.Background()
	pool, err := db.Open(ctx, databaseURL, 0, 2)
	if err != nil {
		t.Fatalf("открыть пул: %v", err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		pool.Close()
		t.Fatalf("начать транзакцию: %v", err)
	}

	return ctx, tx, func() {
		_ = tx.Rollback(context.Background())
		pool.Close()
	}
}

func TestRepositoryIntegration(t *testing.T) {
	ctx, tx, cleanup := openIntegrationTx(t)
	defer cleanup()

	repo := NewPgRepository()
	now := time.Now().UTC()
	suffix := uuid.Must(uuid.NewV7()).String()

	article := &Article{
		ID: uuid.Must(uuid.NewV7()), Slug: "draft-" + suffix, Title: "Черновик",
		ContentMD: "# Черновик", Status: ArticleDraft, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateArticle(ctx, tx, article); err != nil {
		t.Fatalf("создать статью: %v", err)
	}
	if _, err := repo.GetArticleBySlug(ctx, tx, article.Slug); err != nil {
		t.Fatalf("найти черновик: %v", err)
	}
	articles, err := repo.ListPublishedArticles(ctx, tx, nil, 10)
	if err != nil {
		t.Fatalf("опубликованные статьи: %v", err)
	}
	for _, item := range articles {
		if item.ID == article.ID {
			t.Fatal("черновик попал в опубликованный список")
		}
	}

	article.Status = ArticlePublished
	article.PublishedAt = &now
	if err := repo.UpdateArticle(ctx, tx, article); err != nil {
		t.Fatalf("опубликовать статью: %v", err)
	}
	articles, err = repo.ListPublishedArticles(ctx, tx, nil, 10)
	if err != nil {
		t.Fatalf("опубликованные статьи: %v", err)
	}
	found := false
	for _, item := range articles {
		if item.ID == article.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("опубликованная статья не попала в список")
	}

	lead := &Lead{
		ID: uuid.Must(uuid.NewV7()), Name: "Иван", Contact: "ivan@example.ru",
		Comment: "Комментарий", Source: "/oge", Status: LeadNew, CreatedAt: now,
	}
	if err := repo.CreateLead(ctx, tx, lead); err != nil {
		t.Fatalf("создать заявку: %v", err)
	}
	stored, err := repo.GetLead(ctx, tx, lead.ID)
	if err != nil {
		t.Fatalf("найти заявку: %v", err)
	}
	if stored.Status != LeadNew || stored.Name != lead.Name {
		t.Fatalf("заявка = %+v", stored)
	}

	page := &SEOPage{
		ID: uuid.Must(uuid.NewV7()), Path: "/seo-" + suffix, Title: "Заголовок",
		Description: "Описание", CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.UpsertSEOPage(ctx, tx, page); err != nil {
		t.Fatalf("сохранить SEO-страницу: %v", err)
	}
	page.Title = "Обновлённый заголовок"
	if err := repo.UpsertSEOPage(ctx, tx, page); err != nil {
		t.Fatalf("повторно сохранить SEO-страницу: %v", err)
	}
	storedPage, err := repo.GetSEOPageByPath(ctx, tx, page.Path)
	if err != nil || storedPage.Title != "Обновлённый заголовок" {
		t.Fatalf("SEO-страница = %+v, %v", storedPage, err)
	}

	redirect := &Redirect{
		ID: uuid.Must(uuid.NewV7()), FromPath: "/old-" + suffix, ToPath: "/new", StatusCode: 301, CreatedAt: now,
	}
	if err := repo.UpsertRedirect(ctx, tx, redirect); err != nil {
		t.Fatalf("сохранить редирект: %v", err)
	}
	redirect.ToPath = "/updated"
	if err := repo.UpsertRedirect(ctx, tx, redirect); err != nil {
		t.Fatalf("повторно сохранить редирект: %v", err)
	}
	storedRedirect, err := repo.GetRedirectByFromPath(ctx, tx, redirect.FromPath)
	if err != nil || storedRedirect.ToPath != "/updated" {
		t.Fatalf("редирект = %+v, %v", storedRedirect, err)
	}
}

type integrationRecipients struct{}

func (integrationRecipients) ListActiveAdminEmails(context.Context) ([]string, error) {
	return []string{"admin@example.ru"}, nil
}

func TestSubmitLeadIntegrationWritesOutbox(t *testing.T) {
	ctx, tx, cleanup := openIntegrationTx(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	txManager := integrationTx{tx: tx}
	notifyService := notify.NewService(notify.NewPgRepository(), txManager, nil, clock.New(), logger)
	service := NewService(NewPgRepository(), tx, txManager, clock.New(), nil, integrationRecipients{}, notifyService, logger)

	contact := "lead-" + uuid.Must(uuid.NewV7()).String() + "@example.ru"
	if err := service.SubmitLead(ctx, LeadInput{
		Name: "Иван", Contact: contact, Comment: "Нужна помощь", Source: "/oge",
	}); err != nil {
		t.Fatalf("сохранить заявку: %v", err)
	}

	var count int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM notify.outbox WHERE template = $1 AND payload_json->>'contact' = $2`, notify.TemplateCodeLeadNotification, contact).Scan(&count); err != nil {
		t.Fatalf("посчитать outbox: %v", err)
	}
	if count != 1 {
		t.Fatalf("писем в outbox: %d", count)
	}

	if _, err := service.ResolveRedirect(ctx, "/missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("отсутствующий редирект: %v", err)
	}
}

func TestDemoStepFlowIntegration(t *testing.T) {
	ctx, tx, cleanup := openIntegrationTx(t)
	defer cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	clk := clock.New()
	contentService := content.NewService(content.NewPgRepository(), tx, integrationTx{tx: tx}, clk, nil, logger)

	suffix := uuid.Must(uuid.NewV7()).String()
	exam, err := contentService.CreateExam(ctx, content.ExamInput{
		Code: "it_demo_" + suffix, Title: "Интеграционное демо", Subject: "math",
	})
	if err != nil {
		t.Fatalf("создать трек: %v", err)
	}
	if _, err := contentService.PublishExam(ctx, exam.ID); err != nil {
		t.Fatalf("опубликовать трек: %v", err)
	}
	year, err := contentService.CreateExamYear(ctx, content.ExamYearInput{ExamID: exam.ID, Year: 2026, DurationMinutes: 235})
	if err != nil {
		t.Fatalf("создать учебный год: %v", err)
	}
	if _, err := contentService.PublishExamYear(ctx, year.ID); err != nil {
		t.Fatalf("опубликовать учебный год: %v", err)
	}

	topic, err := contentService.CreateTopic(ctx, content.TopicInput{
		ExamID: exam.ID, Code: "it_topic", Title: "Интеграционная тема",
	})
	if err != nil {
		t.Fatalf("создать тему: %v", err)
	}

	for i := 1; i <= 7; i++ {
		kim, err := contentService.CreateKimNumber(ctx, content.KimNumberInput{
			ExamYearID: year.ID, Number: i, Part: 1, AnswerType: content.AnswerTypeShort, MaxPrimaryScore: 1,
		})
		if err != nil {
			t.Fatalf("создать номер КИМ: %v", err)
		}
		if _, err := contentService.SetKimTopics(ctx, kim.ID, []content.KimTopicInput{{TopicID: topic.ID, Weight: 1}}); err != nil {
			t.Fatalf("связать тему: %v", err)
		}
		task, err := contentService.CreateTask(ctx, content.TaskInput{
			ExamYearID: year.ID, KimNumberID: kim.ID, TopicID: &topic.ID, Difficulty: content.DifficultyEasy,
			StatementMD:   fmt.Sprintf("Интеграционное задание %d", i),
			AnswerSpec:    json.RawMessage(`{"kind":"number","accepted":["2"]}`),
			AnswerDisplay: "2", Source: "seed",
		})
		if err != nil {
			t.Fatalf("создать задание: %v", err)
		}
		if _, err := contentService.PublishTask(ctx, task.ID); err != nil {
			t.Fatalf("опубликовать задание: %v", err)
		}
	}

	service := NewService(NewPgRepository(), tx, integrationTx{tx: tx}, clk, contentService, integrationRecipients{}, &fakeNotifier{}, logger)
	csrf := httpserver.NewCSRF("secret", false, nil)
	handler := NewHandler(service, csrf, NewDemoSigner("secret", DemoTTL, clk), NewCache(time.Minute, 10), logger, false, "https://example.ru", "Верно")
	router := gin.New()
	router.Use(csrf.Middleware())
	handler.Register(router, httpserver.NewRateLimiter(100, time.Minute))

	rec := perform(router, http.MethodGet, "/diagnostics/demo?exam="+exam.Code, nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("запуск демо: %d %s", rec.Code, rec.Body.String())
	}
	token := demoToken(rec)
	csrfValue := csrfFrom(rec)
	if token == "" || csrfValue == "" {
		t.Fatal("демо не выдало токен или CSRF")
	}

	form := url.Values{"csrf_token": {csrfValue}, "demo_token": {token}}
	for step := 2; step <= 7; step++ {
		form.Set("step", fmt.Sprintf("%d", step))
		rec = perform(router, http.MethodPost, "/diagnostics/demo/step", form, map[string]string{"csrf_token": csrfValue, DemoCookieName: token})
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), fmt.Sprintf("Задание %d из 7", step)) {
			t.Fatalf("шаг %d: %d %s", step, rec.Code, rec.Body.String())
		}
	}

	checkForm := url.Values{"csrf_token": {csrfValue}, "demo_token": {token}}
	rec = perform(router, http.MethodGet, "/diagnostics/demo?exam="+exam.Code, nil, nil)
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == DemoCookieName {
			token = cookie.Value
		}
	}
	payload, err := handler.signer.Verify(token)
	if err != nil {
		t.Fatalf("проверить токен: %v", err)
	}
	checkForm.Set("demo_token", token)
	for _, taskID := range payload.TaskIDs {
		checkForm.Set("answer_"+taskID.String(), "2")
	}
	rec = perform(router, http.MethodPost, "/diagnostics/demo/check", checkForm, map[string]string{"csrf_token": csrfValue, DemoCookieName: token})
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Результат демо-диагностики") {
		t.Fatalf("итог демо: %d %s", rec.Code, rec.Body.String())
	}

	var taskCount int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM content.tasks WHERE exam_year_id = $1`, year.ID).Scan(&taskCount); err != nil {
		t.Fatalf("посчитать задания: %v", err)
	}
	if taskCount != 7 {
		t.Fatalf("демо изменило задания: %d", taskCount)
	}

	var sessionTable *string
	if err := tx.QueryRow(ctx, `SELECT to_regclass('learning.sessions')::text`).Scan(&sessionTable); err != nil {
		t.Fatalf("проверить таблицу сессий: %v", err)
	}
	if sessionTable != nil {
		var sessions int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM learning.sessions`).Scan(&sessions); err != nil {
			t.Fatalf("посчитать учебные сессии: %v", err)
		}
		if sessions != 0 {
			t.Fatalf("демо создало учебные сессии: %d", sessions)
		}
	}
}
