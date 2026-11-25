package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"oge-ege/internal/shared/logging"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	m.Run()
}

func TestRequestID(t *testing.T) {
	t.Run("генерирует UUID v7", func(t *testing.T) {
		router := newTestRouter()

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

		id := rec.Header().Get(requestIDHeader)
		parsed, err := uuid.Parse(id)
		if err != nil {
			t.Fatalf("некорректный идентификатор %q: %v", id, err)
		}
		if parsed.Version() != 7 {
			t.Fatalf("версия идентификатора = %d, ожидалась 7", parsed.Version())
		}
	})

	t.Run("принимает валидный входящий идентификатор", func(t *testing.T) {
		router := newTestRouter()
		incoming := uuid.New().String()

		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		req.Header.Set(requestIDHeader, incoming)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if got := rec.Header().Get(requestIDHeader); got != incoming {
			t.Fatalf("идентификатор = %q, ожидался %q", got, incoming)
		}
	})

	t.Run("заменяет невалидный идентификатор", func(t *testing.T) {
		router := newTestRouter()

		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		req.Header.Set(requestIDHeader, "not-a-uuid")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if got := rec.Header().Get(requestIDHeader); got == "not-a-uuid" {
			t.Fatal("невалидный идентификатор не был заменён")
		}
	})
}

func TestRecovery(t *testing.T) {
	var logs bytes.Buffer
	router := NewRouter(Options{
		Env:          "local",
		Logger:       logging.NewWithWriter(&logs, "debug"),
		MaxBodyBytes: 1024,
	})
	router.GET("/panic", func(*gin.Context) {
		panic("boom-secret")
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/panic", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("статус = %d", rec.Code)
	}

	var envelope errorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("ответ не JSON: %v", err)
	}
	if envelope.Error.Code != "internal_error" {
		t.Fatalf("код ошибки = %q", envelope.Error.Code)
	}
	if envelope.Error.RequestID == "" {
		t.Fatal("в ответе нет request_id")
	}
	if strings.Contains(rec.Body.String(), "boom-secret") {
		t.Fatal("ответ раскрывает детали паники")
	}
	if !strings.Contains(logs.String(), "паника") {
		t.Fatal("паника не попала в лог")
	}
}

func TestRequestLogger(t *testing.T) {
	var logs bytes.Buffer
	logger := logging.NewWithWriter(&logs, "debug")

	router := gin.New()
	router.Use(RequestID(), RequestLogger(logger))
	router.POST("/log", func(c *gin.Context) {
		c.Status(http.StatusCreated)
	})

	req := httptest.NewRequest(http.MethodPost, "/log?token=query-secret", strings.NewReader(`{"password":"body-secret"}`))
	req.Header.Set("Authorization", "Bearer header-secret")
	req.Header.Set("Cookie", "sid=cookie-secret")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var entry map[string]any
	if err := json.Unmarshal(logs.Bytes(), &entry); err != nil {
		t.Fatalf("лог не JSON: %v (%q)", err, logs.String())
	}
	if entry["method"] != http.MethodPost || entry["path"] != "/log" {
		t.Fatalf("некорректная запись лога: %v", entry)
	}
	if entry["status"] != float64(http.StatusCreated) {
		t.Fatalf("status = %v", entry["status"])
	}
	if entry["request_id"] == "" {
		t.Fatal("в логе нет request_id")
	}
	for _, secret := range []string{"body-secret", "header-secret", "cookie-secret", "query-secret"} {
		if strings.Contains(logs.String(), secret) {
			t.Fatalf("секрет %q попал в лог", secret)
		}
	}
}

func TestSecurityHeaders(t *testing.T) {
	t.Run("local без HSTS", func(t *testing.T) {
		router := newTestRouter()
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

		assertHeader(t, rec, "X-Content-Type-Options", "nosniff")
		assertHeader(t, rec, "Referrer-Policy", "strict-origin-when-cross-origin")
		assertHeader(t, rec, "X-Frame-Options", "DENY")
		if rec.Header().Get("Content-Security-Policy") == "" {
			t.Fatal("CSP не установлен")
		}
		if rec.Header().Get("Strict-Transport-Security") != "" {
			t.Fatal("HSTS не должен устанавливаться вне production")
		}
	})

	t.Run("production по HTTPS с HSTS", func(t *testing.T) {
		router := NewRouter(Options{
			Env:          "production",
			Logger:       discardLogger(),
			MaxBodyBytes: 1024,
		})

		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		req.Header.Set("X-Forwarded-Proto", "https")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Header().Get("Strict-Transport-Security") == "" {
			t.Fatal("HSTS не установлен в production")
		}
	})
}

func TestBodyLimit(t *testing.T) {
	router := NewRouter(Options{
		Env:          "local",
		Logger:       discardLogger(),
		MaxBodyBytes: 10,
	})

	body := strings.NewReader(strings.Repeat("x", 20))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/healthz", body))

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("статус = %d", rec.Code)
	}

	var envelope errorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("ответ не JSON: %v", err)
	}
	if envelope.Error.Code != "bad_request" {
		t.Fatalf("код ошибки = %q", envelope.Error.Code)
	}
}

func TestHealthAndReadiness(t *testing.T) {
	tests := []struct {
		name       string
		pinger     Pinger
		checker    ReadinessChecker
		wantStatus int
		reasonPart string
	}{
		{
			name:       "готов",
			pinger:     fakePinger{},
			checker:    fakeChecker{},
			wantStatus: http.StatusOK,
		},
		{
			name:       "БД недоступна",
			pinger:     fakePinger{err: errors.New("нет связи")},
			checker:    fakeChecker{},
			wantStatus: http.StatusServiceUnavailable,
			reasonPart: "нет связи",
		},
		{
			name:       "миграции не применены",
			pinger:     fakePinger{},
			checker:    fakeChecker{err: errors.New("применена версия 0")},
			wantStatus: http.StatusServiceUnavailable,
			reasonPart: "применена версия 0",
		},
		{
			name:       "проверки не настроены",
			wantStatus: http.StatusServiceUnavailable,
			reasonPart: "не настроены",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := NewRouter(Options{
				Env:              "local",
				Logger:           discardLogger(),
				MaxBodyBytes:     1024,
				Pinger:           tt.pinger,
				MigrationChecker: tt.checker,
			})

			health := httptest.NewRecorder()
			router.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/healthz", nil))
			if health.Code != http.StatusOK {
				t.Fatalf("healthz статус = %d", health.Code)
			}

			ready := httptest.NewRecorder()
			router.ServeHTTP(ready, httptest.NewRequest(http.MethodGet, "/readyz", nil))
			if ready.Code != tt.wantStatus {
				t.Fatalf("readyz статус = %d, тело %s", ready.Code, ready.Body.String())
			}
			if tt.reasonPart != "" && !strings.Contains(ready.Body.String(), tt.reasonPart) {
				t.Fatalf("причина не указана: %s", ready.Body.String())
			}
		})
	}
}

type fakePinger struct {
	err error
}

func (f fakePinger) Ping(context.Context) error {
	return f.err
}

type fakeChecker struct {
	err error
}

func (f fakeChecker) Check(context.Context) error {
	return f.err
}

func newTestRouter() *gin.Engine {
	return NewRouter(Options{
		Env:          "local",
		Logger:       discardLogger(),
		MaxBodyBytes: 1024,
	})
}

func discardLogger() *slog.Logger {
	return logging.NewWithWriter(io.Discard, "error")
}

func assertHeader(t *testing.T, rec *httptest.ResponseRecorder, name, want string) {
	t.Helper()
	if got := rec.Header().Get(name); got != want {
		t.Fatalf("%s = %q, ожидалось %q", name, got, want)
	}
}
