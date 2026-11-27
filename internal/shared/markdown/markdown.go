// Package markdown рендерит Markdown в безопасный HTML и проверяет формулы LaTeX.
package markdown

import (
	"errors"
	"fmt"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

var (
	renderer = goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithRendererOptions(html.WithUnsafe()),
	)
	policy = newPolicy()
)

// Render преобразует Markdown в HTML и санитизирует результат по allowlist.
func Render(source string) (string, error) {
	var buf strings.Builder
	if err := renderer.Convert([]byte(source), &buf); err != nil {
		return "", fmt.Errorf("отрендерить Markdown: %w", err)
	}

	return policy.Sanitize(buf.String()), nil
}

// Sanitize очищает HTML по тому же allowlist, что и Render.
func Sanitize(source string) string {
	return policy.Sanitize(source)
}

// ValidateFormulas проверяет сбалансированность формул LaTeX в Markdown.
func ValidateFormulas(source string) error {
	for _, segment := range mathSegments(source) {
		if err := validateFormula(segment); err != nil {
			return err
		}
	}

	return nil
}

func newPolicy() *bluemonday.Policy {
	p := bluemonday.NewPolicy()

	p.AllowStandardURLs()
	p.AllowRelativeURLs(true)
	p.RequireNoFollowOnLinks(true)

	p.AllowElements(
		"p", "br", "hr",
		"h1", "h2", "h3", "h4", "h5", "h6",
		"strong", "b", "em", "i", "u", "s", "del", "sub", "sup", "mark",
		"ul", "ol", "li",
		"blockquote",
		"code", "pre",
		"table", "thead", "tbody", "tfoot", "tr", "th", "td",
		"div", "span",
	)
	p.AllowAttrs("class").OnElements("div", "span", "code", "pre", "table", "th", "td")
	p.AllowAttrs("href").OnElements("a")
	p.AllowAttrs("title").OnElements("a", "code", "span", "div")
	p.AllowAttrs("src", "alt", "title").OnElements("img")
	p.AllowAttrs("colspan", "rowspan").OnElements("th", "td")
	p.AllowElements("img")

	return p
}

func mathSegments(source string) []string {
	var segments []string

	for i := 0; i < len(source); i++ {
		if source[i] != '$' {
			continue
		}

		delimiter := "$"
		start := i + 1
		if i+1 < len(source) && source[i+1] == '$' {
			delimiter = "$$"
			start = i + 2
		}

		end := strings.Index(source[start:], delimiter)
		if end < 0 {
			// Незакрытый $$ — ошибка разметки. Одиночный $ без пары считаем
			// литералом (цена, символ) и продолжаем поиск, чтобы не ломать
			// сохранение условий в треках без формул.
			if delimiter == "$$" {
				segments = append(segments, source[start:])
				break
			}
			continue
		}

		segments = append(segments, source[start:start+end])
		i = start + end + len(delimiter) - 1
	}

	return segments
}

func validateFormula(formula string) error {
	if strings.TrimSpace(formula) == "" {
		return errors.New("пустая формула LaTeX")
	}

	depth := 0
	for _, r := range formula {
		switch r {
		case '{':
			depth++
		case '}':
			depth--
			if depth < 0 {
				return errors.New("несбалансированные скобки в формуле LaTeX")
			}
		}
	}
	if depth != 0 {
		return errors.New("несбалансированные скобки в формуле LaTeX")
	}

	if strings.Count(formula, `\begin{`) != strings.Count(formula, `\end{`) {
		return errors.New("несбалансированные окружения LaTeX")
	}
	if strings.Count(formula, `\left`) != strings.Count(formula, `\right`) {
		return errors.New("несбалансированные \\left и \\right в формуле LaTeX")
	}

	return nil
}
