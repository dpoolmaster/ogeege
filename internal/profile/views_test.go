package profile

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestProfilePageViewUsesDesignSystem(t *testing.T) {
	data := PageViewData{
		Brand:       "Экзамен",
		Title:       "Профиль",
		Email:       "student@example.ru",
		FirstName:   "Иван",
		LastName:    "Петров",
		RoleLabel:   "Ученик",
		StatusLabel: "Активен",
	}

	var buf bytes.Buffer
	if err := ProfilePageView(data).Render(context.Background(), &buf); err != nil {
		t.Fatalf("отрендерить профиль: %v", err)
	}

	html := buf.String()
	for _, part := range []string{"<title>Профиль — Экзамен</title>", "v-card", "v-field", "v-btn", "Ученик"} {
		if !strings.Contains(html, part) {
			t.Fatalf("профиль не содержит %q: %s", part, html)
		}
	}
}
