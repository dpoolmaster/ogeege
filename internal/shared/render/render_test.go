package render

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
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
	if err := Component(context.Background(), &buf, Layout("Вход")); err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}

	html := buf.String()
	for _, part := range []string{"<!doctype html>", `lang="ru"`, "<title>Вход</title>", "pico.min.css", stylesPath, `class="container auth-page"`} {
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
