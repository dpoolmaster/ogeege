package publicsite

import (
	"bytes"
	"context"
	"strings"
	"testing"
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
