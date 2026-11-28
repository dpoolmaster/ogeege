package render

import (
	"bytes"
	"context"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
)

func TestComponent(t *testing.T) {
	page := templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		_, err := w.Write([]byte("<p>Проверка</p>"))
		return err
	})

	var buf bytes.Buffer
	if err := Component(context.Background(), &buf, page); err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if !strings.Contains(buf.String(), "Проверка") {
		t.Fatalf("некорректный вывод: %s", buf.String())
	}
}

func TestLayoutRendersWithoutUserData(t *testing.T) {
	var buf bytes.Buffer
	if err := Component(context.Background(), &buf, Layout(Chrome{Brand: "Верно"}, "Вход")); err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}

	html := buf.String()
	for _, part := range []string{"<!doctype html>", `lang="ru"`, "<title>Вход — Верно</title>", "pico.min.css", stylesPath, `class="container auth-page"`, faviconPath, FontsPath("golos-text-cyrillic.woff2")} {
		if !strings.Contains(html, part) {
			t.Fatalf("страница не содержит %q: %s", part, html)
		}
	}
}

func TestStylesHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET(stylesPath, StylesHandler())

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, stylesPath, nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("статус = %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/css") {
		t.Fatalf("Content-Type = %q", got)
	}
	if !strings.Contains(rec.Body.String(), ".auth-page") {
		t.Fatalf("CSS не содержит стили страниц: %s", rec.Body.String())
	}
}

var tokenPattern = regexp.MustCompile(`--v-([a-z0-9-]+)\s*:\s*(#[0-9a-fA-F]{6})\s*;`)

func TestTokenContrast(t *testing.T) {
	data, err := assetsFS.ReadFile("assets/app.css")
	if err != nil {
		t.Fatalf("прочитать app.css: %v", err)
	}

	tokens := make(map[string]string)
	for _, match := range tokenPattern.FindAllStringSubmatch(string(data), -1) {
		tokens["--v-"+match[1]] = match[2]
	}

	pairs := []struct {
		name string
		fg   string
		bg   string
	}{
		{name: "приглушённый на бумаге", fg: "--v-muted", bg: "--v-bg"},
		{name: "приглушённый на белой карточке", fg: "--v-muted", bg: "--v-surface"},
		{name: "приглушённый на мягкой акцентной подложке", fg: "--v-muted", bg: "--v-accent-soft"},
		{name: "приглушённый на мягкой подложке успеха", fg: "--v-muted", bg: "--v-ok-soft"},
		{name: "приглушённый на мягкой подложке ошибки", fg: "--v-muted", bg: "--v-err-soft"},
		{name: "приглушённый на мягкой подложке предупреждения", fg: "--v-muted", bg: "--v-warn-soft"},
		{name: "акцент на бумаге", fg: "--v-accent", bg: "--v-bg"},
		{name: "акцент на белой карточке", fg: "--v-accent", bg: "--v-surface"},
		{name: "акцент на мягкой акцентной подложке", fg: "--v-accent", bg: "--v-accent-soft"},
		{name: "сильный акцент на бумаге", fg: "--v-accent-strong", bg: "--v-bg"},
		{name: "вторичный текст на бумаге", fg: "--v-ink-2", bg: "--v-bg"},
		{name: "вторичный текст на белой карточке", fg: "--v-ink-2", bg: "--v-surface"},
		{name: "основной текст на бумаге", fg: "--v-ink", bg: "--v-bg"},
		{name: "основной текст на белой карточке", fg: "--v-ink", bg: "--v-surface"},
		{name: "бирюза на своей подложке", fg: "--v-teal", bg: "--v-teal-soft"},
		{name: "олива на своей подложке", fg: "--v-olive", bg: "--v-olive-soft"},
		{name: "коричневый на своей подложке", fg: "--v-brown", bg: "--v-brown-soft"},
		{name: "охра на своей подложке", fg: "--v-ochre", bg: "--v-ochre-soft"},
	}

	for _, pair := range pairs {
		t.Run(pair.name, func(t *testing.T) {
			fg := tokenValue(t, tokens, pair.fg)
			bg := tokenValue(t, tokens, pair.bg)
			if ratio := contrastRatio(t, fg, bg); ratio < 4.5 {
				t.Fatalf("контраст %s (%s) к %s (%s) = %.2f, требуется ≥ 4.5", pair.fg, fg, pair.bg, bg, ratio)
			}
		})
	}

	t.Run("белый текст CTA на акценте", func(t *testing.T) {
		ctaColor := ruleColor(t, string(data), `.v-cta p`)
		if ctaColor != "#fff" && ctaColor != "#ffffff" {
			t.Fatalf("текст .v-cta p = %q, ожидался непрозрачный белый #fff", ctaColor)
		}
		accent := tokenValue(t, tokens, "--v-accent")
		if ratio := contrastRatio(t, ctaColor, accent); ratio < 4.5 {
			t.Fatalf("контраст текста CTA (%s) к акценту (%s) = %.2f, требуется ≥ 4.5", ctaColor, accent, ratio)
		}
	})
}

func tokenValue(t *testing.T, tokens map[string]string, name string) string {
	t.Helper()

	value, ok := tokens[name]
	if !ok {
		t.Fatalf("токен %s не найден в app.css", name)
	}

	return value
}

func TestShellNavOverridesPico(t *testing.T) {
	data, err := assetsFS.ReadFile("assets/app.css")
	if err != nil {
		t.Fatalf("прочитать app.css: %v", err)
	}
	css := string(data)

	tests := []struct {
		selector string
		property string
		want     string
	}{
		{selector: ".v-shell__nav", property: "display", want: "block"},
		{selector: ".v-shell__nav ul", property: "display", want: "block"},
		{selector: ".v-shell__nav ul:first-of-type", property: "margin-left", want: "0"},
		{selector: ".v-shell__nav li", property: "display", want: "block"},
		{selector: ".v-shell__nav a", property: "margin", want: "0"},
	}

	for _, tt := range tests {
		t.Run(tt.selector, func(t *testing.T) {
			body, ok := cssRuleBody(css, tt.selector)
			if !ok {
				t.Fatalf("не найдено правило %s", tt.selector)
			}
			pattern := regexp.MustCompile(`(?:^|[;\s])` + tt.property + `:\s*([^;]+);`)
			match := pattern.FindStringSubmatch(body)
			if match == nil {
				t.Fatalf("в правиле %s не найдено свойство %s", tt.selector, tt.property)
			}
			if got := strings.TrimSpace(match[1]); got != tt.want {
				t.Fatalf("%s { %s } = %q, ожидалось %q", tt.selector, tt.property, got, tt.want)
			}
		})
	}
}

func cssRuleBody(css, selector string) (string, bool) {
	index := strings.Index(css, selector)
	if index < 0 {
		return "", false
	}
	rest := css[index:]
	open := strings.Index(rest, "{")
	if open < 0 {
		return "", false
	}
	rest = rest[open+1:]
	close := strings.Index(rest, "}")
	if close < 0 {
		return "", false
	}

	return rest[:close], true
}

func ruleColor(t *testing.T, css, selector string) string {
	t.Helper()

	pattern := regexp.MustCompile(regexp.QuoteMeta(selector) + `\s*\{[^}]*color:\s*([^;]+);`)
	match := pattern.FindStringSubmatch(css)
	if match == nil {
		t.Fatalf("в правиле %s не найден цвет", selector)
	}

	return strings.ToLower(strings.TrimSpace(match[1]))
}

func contrastRatio(t *testing.T, fg, bg string) float64 {
	t.Helper()

	fgLum := relativeLuminance(t, fg)
	bgLum := relativeLuminance(t, bg)
	lighter := math.Max(fgLum, bgLum)
	darker := math.Min(fgLum, bgLum)

	return (lighter + 0.05) / (darker + 0.05)
}

func relativeLuminance(t *testing.T, color string) float64 {
	t.Helper()

	hex := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(color)), "#")
	if len(hex) == 3 {
		hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
	}
	if len(hex) != 6 {
		t.Fatalf("неподдерживаемый цвет %q", color)
	}

	channels := make([]float64, 3)
	for i := range channels {
		value, err := strconv.ParseUint(hex[i*2:i*2+2], 16, 8)
		if err != nil {
			t.Fatalf("разобрать цвет %q: %v", color, err)
		}
		channels[i] = linearChannel(float64(value) / 255)
	}

	return 0.2126*channels[0] + 0.7152*channels[1] + 0.0722*channels[2]
}

func linearChannel(value float64) float64 {
	if value <= 0.04045 {
		return value / 12.92
	}

	return math.Pow((value+0.055)/1.055, 2.4)
}
