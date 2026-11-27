package publicsite

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"oge-ege/internal/shared/db"
)

// Repository описывает хранилище модуля publicsite.
type Repository interface {
	CreateArticle(ctx context.Context, q db.Querier, article *Article) error
	UpdateArticle(ctx context.Context, q db.Querier, article *Article) error
	UpsertArticle(ctx context.Context, q db.Querier, article *Article) error
	GetArticleBySlug(ctx context.Context, q db.Querier, slug string) (*Article, error)
	ListPublishedArticles(ctx context.Context, q db.Querier, cursor *time.Time, limit int) ([]Article, error)

	CreateLead(ctx context.Context, q db.Querier, lead *Lead) error
	GetLead(ctx context.Context, q db.Querier, id uuid.UUID) (*Lead, error)

	UpsertSEOPage(ctx context.Context, q db.Querier, page *SEOPage) error
	GetSEOPageByPath(ctx context.Context, q db.Querier, path string) (*SEOPage, error)
	ListSEOPages(ctx context.Context, q db.Querier) ([]SEOPage, error)

	UpsertRedirect(ctx context.Context, q db.Querier, redirect *Redirect) error
	GetRedirectByFromPath(ctx context.Context, q db.Querier, path string) (*Redirect, error)
}

// PgRepository работает с PostgreSQL.
type PgRepository struct{}

// NewPgRepository создаёт репозиторий publicsite.
func NewPgRepository() *PgRepository {
	return &PgRepository{}
}

const articleColumns = `
    id,
    slug,
    title,
    excerpt,
    content_md,
    status,
    published_at,
    created_at,
    updated_at
`

const createArticleQuery = `
INSERT INTO public.articles (
    id, slug, title, excerpt, content_md, status, published_at, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
`

const updateArticleQuery = `
UPDATE public.articles
SET slug = $2,
    title = $3,
    excerpt = $4,
    content_md = $5,
    status = $6,
    published_at = $7,
    updated_at = $8
WHERE id = $1
`

const upsertArticleQuery = `
INSERT INTO public.articles (
    id, slug, title, excerpt, content_md, status, published_at, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
ON CONFLICT (slug) DO UPDATE
SET title = EXCLUDED.title,
    excerpt = EXCLUDED.excerpt,
    content_md = EXCLUDED.content_md,
    status = EXCLUDED.status,
    published_at = EXCLUDED.published_at,
    updated_at = EXCLUDED.updated_at
`

const getArticleBySlugQuery = `
SELECT ` + articleColumns + `
FROM public.articles
WHERE slug = $1
`

const listPublishedArticlesQuery = `
SELECT ` + articleColumns + `
FROM public.articles
WHERE status = 'published' AND published_at IS NOT NULL
  AND ($1::timestamptz IS NULL OR published_at < $1)
ORDER BY published_at DESC, id DESC
LIMIT $2
`

// CreateArticle сохраняет статью.
func (r *PgRepository) CreateArticle(ctx context.Context, q db.Querier, article *Article) error {
	if _, err := q.Exec(ctx, createArticleQuery,
		article.ID, article.Slug, article.Title, article.Excerpt, article.ContentMD,
		string(article.Status), article.PublishedAt, article.CreatedAt,
	); err != nil {
		return wrapWriteError("создать статью", err)
	}

	return nil
}

// UpdateArticle изменяет статью.
func (r *PgRepository) UpdateArticle(ctx context.Context, q db.Querier, article *Article) error {
	if _, err := q.Exec(ctx, updateArticleQuery,
		article.ID, article.Slug, article.Title, article.Excerpt, article.ContentMD,
		string(article.Status), article.PublishedAt, article.UpdatedAt,
	); err != nil {
		return wrapWriteError("обновить статью", err)
	}

	return nil
}

// UpsertArticle создаёт или обновляет статью по slug.
func (r *PgRepository) UpsertArticle(ctx context.Context, q db.Querier, article *Article) error {
	if _, err := q.Exec(ctx, upsertArticleQuery,
		article.ID, article.Slug, article.Title, article.Excerpt, article.ContentMD,
		string(article.Status), article.PublishedAt, article.CreatedAt,
	); err != nil {
		return wrapWriteError("сохранить статью", err)
	}

	return nil
}

// GetArticleBySlug возвращает статью по slug.
func (r *PgRepository) GetArticleBySlug(ctx context.Context, q db.Querier, slug string) (*Article, error) {
	return scanArticle(q.QueryRow(ctx, getArticleBySlugQuery, slug))
}

// ListPublishedArticles возвращает опубликованные статьи с курсором по дате.
func (r *PgRepository) ListPublishedArticles(ctx context.Context, q db.Querier, cursor *time.Time, limit int) ([]Article, error) {
	rows, err := q.Query(ctx, listPublishedArticlesQuery, cursor, normalizeLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("выбрать статьи: %w", err)
	}
	defer rows.Close()

	var articles []Article
	for rows.Next() {
		article, err := scanArticle(rows)
		if err != nil {
			return nil, err
		}
		articles = append(articles, *article)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("прочитать статьи: %w", err)
	}

	return articles, nil
}

const createLeadQuery = `
INSERT INTO public.leads (id, name, contact, comment, source, status, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
`

const getLeadQuery = `
SELECT id, name, contact, comment, source, status, created_at
FROM public.leads
WHERE id = $1
`

// CreateLead сохраняет заявку.
func (r *PgRepository) CreateLead(ctx context.Context, q db.Querier, lead *Lead) error {
	if _, err := q.Exec(ctx, createLeadQuery,
		lead.ID, lead.Name, lead.Contact, lead.Comment, lead.Source, string(lead.Status), lead.CreatedAt,
	); err != nil {
		return wrapWriteError("создать заявку", err)
	}

	return nil
}

// GetLead возвращает заявку по идентификатору.
func (r *PgRepository) GetLead(ctx context.Context, q db.Querier, id uuid.UUID) (*Lead, error) {
	var lead Lead
	err := q.QueryRow(ctx, getLeadQuery, id).Scan(
		&lead.ID, &lead.Name, &lead.Contact, &lead.Comment, &lead.Source, &lead.Status, &lead.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("прочитать заявку: %w", err)
	}

	return &lead, nil
}

const seoPageColumns = `
    id,
    path,
    title,
    description,
    og_title,
    og_description,
    og_image,
    canonical_path,
    noindex,
    created_at,
    updated_at
`

const upsertSEOPageQuery = `
INSERT INTO public.seo_pages (
    id, path, title, description, og_title, og_description, og_image,
    canonical_path, noindex, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)
ON CONFLICT (path) DO UPDATE
SET title = EXCLUDED.title,
    description = EXCLUDED.description,
    og_title = EXCLUDED.og_title,
    og_description = EXCLUDED.og_description,
    og_image = EXCLUDED.og_image,
    canonical_path = EXCLUDED.canonical_path,
    noindex = EXCLUDED.noindex,
    updated_at = EXCLUDED.updated_at
`

const getSEOPageByPathQuery = `
SELECT ` + seoPageColumns + `
FROM public.seo_pages
WHERE path = $1
`

const listSEOPagesQuery = `
SELECT ` + seoPageColumns + `
FROM public.seo_pages
WHERE noindex = false
ORDER BY path
`

// UpsertSEOPage создаёт или обновляет SEO-страницу по пути.
func (r *PgRepository) UpsertSEOPage(ctx context.Context, q db.Querier, page *SEOPage) error {
	if _, err := q.Exec(ctx, upsertSEOPageQuery,
		page.ID, page.Path, page.Title, page.Description, page.OGTitle, page.OGDescription,
		page.OGImage, page.CanonicalPath, page.NoIndex, page.CreatedAt,
	); err != nil {
		return wrapWriteError("сохранить SEO-страницу", err)
	}

	return nil
}

// GetSEOPageByPath возвращает SEO-страницу по пути.
func (r *PgRepository) GetSEOPageByPath(ctx context.Context, q db.Querier, path string) (*SEOPage, error) {
	return scanSEOPage(q.QueryRow(ctx, getSEOPageByPathQuery, path))
}

// ListSEOPages возвращает индексируемые SEO-страницы.
func (r *PgRepository) ListSEOPages(ctx context.Context, q db.Querier) ([]SEOPage, error) {
	rows, err := q.Query(ctx, listSEOPagesQuery)
	if err != nil {
		return nil, fmt.Errorf("выбрать SEO-страницы: %w", err)
	}
	defer rows.Close()

	var pages []SEOPage
	for rows.Next() {
		page, err := scanSEOPage(rows)
		if err != nil {
			return nil, err
		}
		pages = append(pages, *page)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("прочитать SEO-страницы: %w", err)
	}

	return pages, nil
}

const redirectColumns = `
    id,
    from_path,
    to_path,
    status_code,
    created_at
`

const upsertRedirectQuery = `
INSERT INTO public.redirects (id, from_path, to_path, status_code, created_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (from_path) DO UPDATE
SET to_path = EXCLUDED.to_path,
    status_code = EXCLUDED.status_code
`

const getRedirectByFromPathQuery = `
SELECT ` + redirectColumns + `
FROM public.redirects
WHERE from_path = $1
`

// UpsertRedirect создаёт или обновляет SEO-редирект.
func (r *PgRepository) UpsertRedirect(ctx context.Context, q db.Querier, redirect *Redirect) error {
	if _, err := q.Exec(ctx, upsertRedirectQuery,
		redirect.ID, redirect.FromPath, redirect.ToPath, redirect.StatusCode, redirect.CreatedAt,
	); err != nil {
		return wrapWriteError("сохранить редирект", err)
	}

	return nil
}

// GetRedirectByFromPath возвращает редирект по исходному пути.
func (r *PgRepository) GetRedirectByFromPath(ctx context.Context, q db.Querier, path string) (*Redirect, error) {
	var redirect Redirect
	err := q.QueryRow(ctx, getRedirectByFromPathQuery, path).Scan(
		&redirect.ID, &redirect.FromPath, &redirect.ToPath, &redirect.StatusCode, &redirect.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("прочитать редирект: %w", err)
	}

	return &redirect, nil
}

func scanArticle(row rowScanner) (*Article, error) {
	var article Article
	err := row.Scan(
		&article.ID, &article.Slug, &article.Title, &article.Excerpt, &article.ContentMD,
		&article.Status, &article.PublishedAt, &article.CreatedAt, &article.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("прочитать статью: %w", err)
	}

	return &article, nil
}

func scanSEOPage(row rowScanner) (*SEOPage, error) {
	var page SEOPage
	err := row.Scan(
		&page.ID, &page.Path, &page.Title, &page.Description, &page.OGTitle, &page.OGDescription,
		&page.OGImage, &page.CanonicalPath, &page.NoIndex, &page.CreatedAt, &page.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("прочитать SEO-страницу: %w", err)
	}

	return &page, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func wrapWriteError(action string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return fmt.Errorf("%s: %w", action, ErrConflict)
	}

	return fmt.Errorf("%s: %w", action, err)
}

func normalizeLimit(limit int) int {
	const (
		defaultLimit = 20
		maxLimit     = 50
	)

	if limit <= 0 {
		return defaultLimit
	}
	if limit > maxLimit {
		return maxLimit
	}

	return limit
}
