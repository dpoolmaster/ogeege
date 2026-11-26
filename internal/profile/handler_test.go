package profile

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"oge-ege/internal/auth"
)

func newProfileRouter(env *testEnv, user *auth.User) *gin.Engine {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(env.service, slog.New(slog.NewTextHandler(io.Discard, nil)))
	router := gin.New()
	router.Use(func(c *gin.Context) {
		auth.WithUser(c, user)
		c.Next()
	})
	router.GET("/api/me", handler.Me)
	router.GET("/api/me/profile", handler.GetProfile)
	router.PATCH("/api/me/profile", handler.UpdateProfile)
	router.GET("/api/me/goals", handler.GetGoals)
	router.PUT("/api/me/goals", handler.SaveGoals)
	router.GET("/api/me/settings", handler.GetSettings)
	router.PATCH("/api/me/settings", handler.UpdateSettings)
	router.POST("/api/me/delete-request", handler.DeleteRequest)

	return router
}

func TestProfileHandlers(t *testing.T) {
	env := newTestEnv()
	user := &auth.User{ID: uuid.Must(uuid.NewV7()), Email: "student@example.ru", Role: auth.RoleStudent, Status: auth.StatusActive}
	router := newProfileRouter(env, user)

	t.Run("пустой профиль", func(t *testing.T) {
		rec := performJSONRequest(t, router, http.MethodGet, "/api/me", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("статус = %d", rec.Code)
		}
		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("ответ не JSON: %v", err)
		}
		if body["profile"] != nil {
			t.Fatalf("профиль = %v", body["profile"])
		}
	})

	t.Run("без даты рождения", func(t *testing.T) {
		rec := performJSONRequest(t, router, http.MethodPatch, "/api/me/profile", `{"first_name":"Иван","last_name":"Петров"}`)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("статус = %d, тело %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("сохранение профиля", func(t *testing.T) {
		rec := performJSONRequest(t, router, http.MethodPatch, "/api/me/profile", `{
			"first_name": "Иван",
			"last_name": "Петров",
			"birth_date": "2010-05-01",
			"grade": 9,
			"timezone": "Europe/Moscow"
		}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("статус = %d, тело %s", rec.Code, rec.Body.String())
		}

		rec = performJSONRequest(t, router, http.MethodGet, "/api/me/profile", "")
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Иван") {
			t.Fatalf("профиль не сохранён: %s", rec.Body.String())
		}
	})

	t.Run("учебные цели", func(t *testing.T) {
		examID := uuid.Must(uuid.NewV7()).String()
		rec := performJSONRequest(t, router, http.MethodPut, "/api/me/goals", `{
			"exam_id": "`+examID+`",
			"exam_date": "2027-06-01",
			"target_score": 80,
			"weekly_minutes": 300
		}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("статус = %d, тело %s", rec.Code, rec.Body.String())
		}

		rec = performJSONRequest(t, router, http.MethodGet, "/api/me/goals", "")
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"target_score":80`) {
			t.Fatalf("цель не сохранена: %s", rec.Body.String())
		}
	})

	t.Run("настройки", func(t *testing.T) {
		rec := performJSONRequest(t, router, http.MethodPatch, "/api/me/settings", `{"email_notifications": false}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("статус = %d, тело %s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), `"email_notifications":false`) {
			t.Fatalf("настройки = %s", rec.Body.String())
		}
	})

	t.Run("удаление аккаунта", func(t *testing.T) {
		rec := performJSONRequest(t, router, http.MethodPost, "/api/me/delete-request", "{}")
		if rec.Code != http.StatusOK {
			t.Fatalf("статус = %d, тело %s", rec.Code, rec.Body.String())
		}
		if len(env.anonymizer.calls) != 1 {
			t.Fatalf("обезличивание не вызвано: %v", env.anonymizer.calls)
		}
	})
}

func performJSONRequest(t *testing.T, router *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	return rec
}
