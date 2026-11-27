package ui

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func renderComponent(t *testing.T, component templ.Component) string {
	t.Helper()

	var buf bytes.Buffer
	if err := component.Render(context.Background(), &buf); err != nil {
		t.Fatalf("отрендерить компонент: %v", err)
	}

	return buf.String()
}

func TestSubjectTone(t *testing.T) {
	tests := []struct {
		name string
		code string
		want Tone
	}{
		{name: "пустой код нейтрален", code: "", want: ToneNeutral},
		{name: "пробелы нейтральны", code: "   ", want: ToneNeutral},
		{name: "oge_math стабилен", code: "oge_math", want: SubjectTone("oge_math")},
		{name: "ege_math_profile стабилен", code: "ege_math_profile", want: SubjectTone("ege_math_profile")},
		{name: "физика стабильна", code: "ege_physics", want: SubjectTone("ege_physics")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SubjectTone(tt.code); got != tt.want {
				t.Fatalf("SubjectTone(%q) = %q, want %q", tt.code, got, tt.want)
			}
		})
	}

	first := SubjectTone("oge_math")
	second := SubjectTone("oge_math")
	if first != second {
		t.Fatal("цвет трека должен быть детерминированным")
	}
}

func TestButtonRendersStates(t *testing.T) {
	html := renderComponent(t, Button(ButtonProps{
		Label:    "Проверить",
		Type:     "submit",
		Variant:  ButtonPrimary,
		Disabled: true,
	}))

	for _, part := range []string{"Проверить", `type="submit"`, "v-btn--primary", "disabled"} {
		if !strings.Contains(html, part) {
			t.Fatalf("кнопка не содержит %q: %s", part, html)
		}
	}
}

func TestFieldRendersValidation(t *testing.T) {
	html := renderComponent(t, Field(FieldProps{
		Label: "Email",
		Name:  "email",
		Type:  "email",
		Error: "Введите корректный email",
	}))

	for _, part := range []string{`id="field-email"`, `for="field-email"`, `aria-invalid="true"`, `id="field-email-error"`, "Введите корректный email"} {
		if !strings.Contains(html, part) {
			t.Fatalf("поле не содержит %q: %s", part, html)
		}
	}
}

func TestEmptyStateHasAction(t *testing.T) {
	html := renderComponent(t, EmptyState(EmptyStateProps{
		Title:       "Ничего нет",
		Text:        "Попробуйте позже.",
		ActionLabel: "Повторить",
		ActionHref:  "/bank",
	}))

	for _, part := range []string{"Ничего нет", "Попробуйте позже", "Повторить", `href="/bank"`} {
		if !strings.Contains(html, part) {
			t.Fatalf("пустое состояние не содержит %q: %s", part, html)
		}
	}
}

func TestProgressClampsValues(t *testing.T) {
	html := renderComponent(t, Progress(ProgressProps{Value: 9, Max: 5, Label: "Заполнено"}))

	if !strings.Contains(html, `value="5"`) || !strings.Contains(html, `max="5"`) {
		t.Fatalf("прогресс не ограничен: %s", html)
	}
	if !strings.Contains(html, "5 из 5") {
		t.Fatalf("прогресс не содержит подпись: %s", html)
	}
}

func TestResultPanelBranches(t *testing.T) {
	tests := []struct {
		name    string
		correct bool
		role    string
		title   string
	}{
		{name: "верно", correct: true, role: `role="status"`, title: "Верно"},
		{name: "неверно", correct: false, role: `role="alert"`, title: "Неверно"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html := renderComponent(t, ResultPanel(ResultPanelProps{
				Correct: tt.correct,
				Text:    "Объяснение",
			}))
			for _, part := range []string{tt.role, tt.title, "Объяснение"} {
				if !strings.Contains(html, part) {
					t.Fatalf("панель результата не содержит %q: %s", part, html)
				}
			}
		})
	}
}

func TestGapHeatmapHasTextState(t *testing.T) {
	html := renderComponent(t, GapHeatmap(GapHeatmapProps{
		Topics:     []string{"Алгебра"},
		KimNumbers: []int{1, 2},
		Cells:      [][]GapCell{{{State: GapCorrect}, {State: GapWrong}}},
	}))

	for _, part := range []string{"Алгебра", "КИМ 1: верно", "КИМ 2: ошибка", "✓", "✕"} {
		if !strings.Contains(html, part) {
			t.Fatalf("хитмап не содержит %q: %s", part, html)
		}
	}
}

func TestAppShellUsesGroups(t *testing.T) {
	html := renderComponent(t, AppShell(AppShellProps{
		Brand:     "Верно",
		PageTitle: "Сегодня",
		Nav: []NavGroup{
			{Title: "Учёба", Items: []NavItem{{Label: "Сегодня", Href: "/app", Active: true}}},
			{Title: "Аккаунт", Items: []NavItem{{Label: "Профиль", Href: "/account"}}},
		},
	}))

	for _, part := range []string{"Верно", "Сегодня", "Учёба", "Аккаунт", `aria-current="page"`} {
		if !strings.Contains(html, part) {
			t.Fatalf("каркас не содержит %q: %s", part, html)
		}
	}
}

func TestTodayScreenUsesComponents(t *testing.T) {
	html := renderComponent(t, TodayScreen(TodayScreenProps{
		NextStep:    "Повторить алгебру",
		NextHref:    "/app/practice",
		Gaps:        []GapSummary{{Title: "Алгебра", Count: 3, Tone: ToneWarn}},
		Goal:        "Набрать 18 баллов",
		GoalCurrent: 7,
		GoalMax:     18,
	}))

	for _, part := range []string{"Ближайший шаг", "Пробелы", "Цель", "v-badge", "v-progress", "7 из 18"} {
		if !strings.Contains(html, part) {
			t.Fatalf("экран «Сегодня» не содержит %q: %s", part, html)
		}
	}
}
