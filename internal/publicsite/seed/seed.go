// Package seed создаёт стартовые публичные статьи, SEO-страницы и редиректы.
package seed

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"oge-ege/internal/publicsite"
)

// Run создаёт публичный контент; повторный запуск не создаёт дубликатов.
func Run(ctx context.Context, service *publicsite.Service, logger *slog.Logger) error {
	if err := seedArticles(ctx, service); err != nil {
		return err
	}
	if err := seedSEOPages(ctx, service); err != nil {
		return err
	}
	if err := seedRedirects(ctx, service); err != nil {
		return err
	}

	logger.Info("публичный контент подготовлен")

	return nil
}

func seedArticles(ctx context.Context, service *publicsite.Service) error {
	now := time.Now()
	articles := []publicsite.ArticleInput{
		{
			Slug:    "kak-gotovitsya-k-oge",
			Title:   "Как готовиться к ОГЭ по математике",
			Excerpt: "Пошаговый план подготовки и разбор типичных ошибок.",
			ContentMD: "## С чего начать\n\n" +
				"Начните с диагностики: она покажет темы, которые нужно повторить.\n\n" +
				"## План подготовки\n\n" +
				"1. Проверьте базовые вычисления.\n" +
				"2. Разберите геометрию.\n" +
				"3. Решайте варианты целиком.\n",
			Status:      publicsite.ArticlePublished,
			PublishedAt: &now,
		},
		{
			Slug:    "kak-reshat-zadachi-s-parametrom",
			Title:   "Как решать задачи с параметром",
			Excerpt: "Основные приёмы и логика разбора задач с параметром.",
			ContentMD: "## Анализ условия\n\n" +
				"Задачи с параметром требуют разбора случаев. Начните с области допустимых значений.\n\n" +
				"## Проверка\n\n" +
				"Подставьте найденные значения и проверьте ограничения.\n",
			Status:      publicsite.ArticlePublished,
			PublishedAt: &now,
		},
	}

	for _, article := range articles {
		if _, err := service.UpsertArticle(ctx, article); err != nil {
			return fmt.Errorf("сохранить статью %s: %w", article.Slug, err)
		}
	}

	return nil
}

func seedSEOPages(ctx context.Context, service *publicsite.Service) error {
	pages := []publicsite.SEOPageInput{
		{
			Path:        "/",
			Title:       "Подготовка к ОГЭ и ЕГЭ по математике",
			Description: "Открытый банк заданий, демо-диагностика и разборы для подготовки к экзаменам.",
		},
		{
			Path:        "/bank",
			Title:       "Открытый банк заданий",
			Description: "Задания ОГЭ и ЕГЭ с автоматической проверкой ответа.",
		},
		{
			Path:        "/diagnostics/demo",
			Title:       "Демо-диагностика",
			Description: "Пройдите демо-диагностику и узнайте, какие темы стоит повторить.",
		},
		{
			Path:        "/articles",
			Title:       "Статьи и разборы",
			Description: "Материалы о подготовке к ОГЭ и ЕГЭ.",
		},
		{
			Path:        "/tracks/oge_math",
			Title:       "ОГЭ математика",
			Description: "Структура экзамена, темы и демо-диагностика по математике ОГЭ.",
		},
		{
			Path:        "/tracks/ege_math_profile",
			Title:       "ЕГЭ математика (профиль)",
			Description: "Структура профильного ЕГЭ по математике и демо-диагностика.",
		},
		{
			Path:        "/tracks/ege_math_base",
			Title:       "ЕГЭ математика (база)",
			Description: "Структура базового ЕГЭ по математике и демо-диагностика.",
		},
	}

	for _, page := range pages {
		if err := service.UpsertSEOPage(ctx, page); err != nil {
			return fmt.Errorf("сохранить SEO-страницу %s: %w", page.Path, err)
		}
	}

	return nil
}

func seedRedirects(ctx context.Context, service *publicsite.Service) error {
	redirects := []publicsite.RedirectInput{
		{FromPath: "/oge", ToPath: "/tracks/oge_math"},
		{FromPath: "/ege-profil", ToPath: "/tracks/ege_math_profile"},
		{FromPath: "/ege-baza", ToPath: "/tracks/ege_math_base"},
	}

	for _, redirect := range redirects {
		if err := service.UpsertRedirect(ctx, redirect); err != nil {
			return fmt.Errorf("сохранить редирект %s: %w", redirect.FromPath, err)
		}
	}

	return nil
}
