package publicsite

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"

	"oge-ege/internal/content"
)

func TestHomePageViewUsesConfiguredBrand(t *testing.T) {
	page := &HomePage{
		Brand: "Экзамен",
		SEO:   SEOMetadata{Title: "Главная"},
	}

	var buf bytes.Buffer
	if err := HomePageView(page).Render(context.Background(), &buf); err != nil {
		t.Fatalf("отрендерить главную: %v", err)
	}

	html := buf.String()
	for _, part := range []string{"<title>Главная — Экзамен</title>", `class="v-brand"`, `class="v-site-footer__brand">Экзамен`} {
		if !strings.Contains(html, part) {
			t.Fatalf("страница не содержит %q: %s", part, html)
		}
	}
	if strings.Contains(html, "ОГЭ и ЕГЭ</a>") {
		t.Fatalf("в шапке остался литерал бренда: %s", html)
	}
}

func TestPageLayoutHeaderForAnonymous(t *testing.T) {
	page := &HomePage{SEO: SEOMetadata{Title: "Главная"}}

	var buf bytes.Buffer
	if err := HomePageView(page).Render(context.Background(), &buf); err != nil {
		t.Fatalf("отрендерить главную: %v", err)
	}

	html := buf.String()
	for _, part := range []string{`href="/login"`, `href="/register"`} {
		if !strings.Contains(html, part) {
			t.Fatalf("шапка анонима не содержит %q: %s", part, html)
		}
	}
	if strings.Contains(html, `href="/account"`) {
		t.Fatalf("шапка анонима содержит ссылку на кабинет: %s", html)
	}
}

func TestPageLayoutHeaderForSignedIn(t *testing.T) {
	page := &HomePage{
		SEO:    SEOMetadata{Title: "Главная"},
		Viewer: Viewer{SignedIn: true, Label: "student@example.com"},
	}

	var buf bytes.Buffer
	if err := HomePageView(page).Render(context.Background(), &buf); err != nil {
		t.Fatalf("отрендерить главную: %v", err)
	}

	html := buf.String()
	for _, part := range []string{`href="/account"`, "student@example.com"} {
		if !strings.Contains(html, part) {
			t.Fatalf("шапка пользователя не содержит %q: %s", part, html)
		}
	}
	for _, part := range []string{`href="/login"`, `href="/register"`} {
		if strings.Contains(html, part) {
			t.Fatalf("шапка пользователя содержит %q: %s", part, html)
		}
	}
}

func TestHomePageViewFallsBackToDefaultBrand(t *testing.T) {
	page := &HomePage{SEO: SEOMetadata{Title: "Главная"}}

	var buf bytes.Buffer
	if err := HomePageView(page).Render(context.Background(), &buf); err != nil {
		t.Fatalf("отрендерить главную: %v", err)
	}

	if !strings.Contains(buf.String(), "<title>Главная — Верно</title>") {
		t.Fatalf("нет бренда по умолчанию: %s", buf.String())
	}
}

func TestLandingViewWithoutTaskHidesCheckBlock(t *testing.T) {
	page := &LandingPage{
		SEO:  SEOMetadata{Title: "Трек"},
		Exam: content.Exam{Code: "oge_math", Title: "ОГЭ математика"},
	}

	var buf bytes.Buffer
	if err := LandingView(page).Render(context.Background(), &buf); err != nil {
		t.Fatalf("отрендерить лендинг: %v", err)
	}
	if strings.Contains(buf.String(), "Попробуйте задание") {
		t.Fatal("блок проверки показан без задания")
	}
}

func TestLandingViewWithTaskHasCSRFAndFragmentTarget(t *testing.T) {
	task := content.Task{ID: uuid.Must(uuid.NewV7()), StatementMD: "Вычислите 1 + 1"}
	page := &LandingPage{
		SEO:      SEOMetadata{Title: "Трек"},
		Exam:     content.Exam{Code: "oge_math", Title: "ОГЭ математика"},
		Task:     &task,
		TaskHTML: "<p>Вычислите 1 + 1</p>",
	}

	var buf bytes.Buffer
	if err := LandingView(page).Render(context.Background(), &buf); err != nil {
		t.Fatalf("отрендерить лендинг: %v", err)
	}
	html := buf.String()
	for _, part := range []string{
		"Попробуйте задание",
		`hx-post="/hx/tasks/` + task.ID.String() + `/check"`,
		`hx-target="#landing-check-result"`,
		`name="csrf_token" value="__CSRF__"`,
		"Вычислите 1 + 1",
	} {
		if !strings.Contains(html, part) {
			t.Fatalf("лендинг не содержит %q", part)
		}
	}
}

func TestDemoStepViewHasProgressAndAutofocus(t *testing.T) {
	taskID := uuid.Must(uuid.NewV7())
	step := &DemoStep{
		Step:     2,
		Total:    5,
		Task:     content.Task{ID: taskID, StatementMD: "Задание"},
		TaskHTML: "<p>Задание</p>",
		TaskIDs:  []uuid.UUID{taskID, uuid.Must(uuid.NewV7())},
		Answers:  map[uuid.UUID]string{taskID: "2"},
		Filled:   1,
	}

	var buf bytes.Buffer
	if err := DemoStepFormView(step, "token").Render(context.Background(), &buf); err != nil {
		t.Fatalf("отрендерить шаг: %v", err)
	}
	html := buf.String()
	for _, part := range []string{
		"<progress",
		"Шаг",
		"2 из 5",
		"Заполнено: 1 из 5",
		`data-autofocus`,
		`tabindex="-1"`,
		`hx-post="/hx/demo/step"`,
		`hx-target="#demo-step"`,
	} {
		if !strings.Contains(html, part) {
			t.Fatalf("шаг не содержит %q: %s", part, html)
		}
	}
}
