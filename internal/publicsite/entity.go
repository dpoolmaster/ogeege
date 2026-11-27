// Package publicsite реализует публичный контур: лендинги треков, банк заданий,
// демо-диагностику, статьи, лид-форму, sitemap и SEO-редиректы.
package publicsite

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ArticleStatus — статус публикации статьи.
type ArticleStatus string

const (
	ArticleDraft     ArticleStatus = "draft"
	ArticlePublished ArticleStatus = "published"
	ArticleArchived  ArticleStatus = "archived"
)

// LeadStatus — состояние заявки.
type LeadStatus string

const (
	LeadNew        LeadStatus = "new"
	LeadInProgress LeadStatus = "in_progress"
	LeadProcessed  LeadStatus = "processed"
	LeadRejected   LeadStatus = "rejected"
)

// Article — публичная статья или разбор.
type Article struct {
	ID          uuid.UUID
	Slug        string
	Title       string
	Excerpt     string
	ContentMD   string
	Status      ArticleStatus
	PublishedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Lead — заявка с публичной формы.
type Lead struct {
	ID        uuid.UUID
	Name      string
	Contact   string
	Comment   string
	Source    string
	Status    LeadStatus
	CreatedAt time.Time
}

// SEOPage — метаданные публичной страницы.
type SEOPage struct {
	ID            uuid.UUID
	Path          string
	Title         string
	Description   string
	OGTitle       string
	OGDescription string
	OGImage       string
	CanonicalPath string
	NoIndex       bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Redirect — SEO-редирект прежнего адреса.
type Redirect struct {
	ID         uuid.UUID
	FromPath   string
	ToPath     string
	StatusCode int
	CreatedAt  time.Time
}

// ArticleInput содержит данные статьи для seed и тестов.
type ArticleInput struct {
	Slug        string
	Title       string
	Excerpt     string
	ContentMD   string
	Status      ArticleStatus
	PublishedAt *time.Time
}

// LeadInput содержит данные публичной формы.
type LeadInput struct {
	Name    string
	Contact string
	Comment string
	Source  string
	Website string
}

// SEOPageInput содержит метаданные страницы.
type SEOPageInput struct {
	Path          string
	Title         string
	Description   string
	OGTitle       string
	OGDescription string
	OGImage       string
	CanonicalPath string
	NoIndex       bool
}

// RedirectInput содержит данные SEO-редиректа.
type RedirectInput struct {
	FromPath   string
	ToPath     string
	StatusCode int
}

// Ошибки модуля publicsite.
var (
	ErrNotFound        = errors.New("публичный объект не найден")
	ErrConflict        = errors.New("конфликт публичных данных")
	ErrInsufficient    = errors.New("недостаточно заданий для демо")
	ErrUnsupported     = errors.New("тип ответа не поддерживается публичной проверкой")
	ErrInvalidSession  = errors.New("данные демо-диагностики истекли или изменены")
	ErrInvalidRedirect = errors.New("небезопасный адрес перенаправления")
)

// ValidationError описывает ошибку проверки поля формы.
type ValidationError struct {
	Field   string
	Message string
}

// Error возвращает текст ошибки.
func (e *ValidationError) Error() string {
	if e.Field == "" {
		return e.Message
	}

	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

func invalid(field, message string) error {
	return &ValidationError{Field: field, Message: message}
}
