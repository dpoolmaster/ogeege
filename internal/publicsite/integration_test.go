package publicsite

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"oge-ege/internal/notify"
	"oge-ege/internal/shared/clock"
	"oge-ege/internal/shared/db"
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
