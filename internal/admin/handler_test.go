package admin

import (
	"context"
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
	"oge-ege/internal/content"
)

func (f *fakeContent) ListExams(_ context.Context, _ content.ExamFilter) ([]content.Exam, error) {
	return []content.Exam{{ID: uuid.Must(uuid.NewV7()), Code: "oge_math", Title: "ОГЭ", Subject: "math", Status: content.StatusPublished}}, nil
}

func newAdminRouter(user *auth.User, contentAdmin ContentAdmin, service *Service) *gin.Engine {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(contentAdmin, service, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	router := gin.New()
	router.Use(func(c *gin.Context) {
		auth.WithUser(c, user)
		c.Next()
	})

	group := router.Group("/api/admin")
	handler.Register(group)
	handler.RegisterAdminOnly(group)

	return router
}

func perform(t *testing.T, router *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	return recorder
}

func TestAdminHandlerContentAccess(t *testing.T) {
	methodist := &auth.User{ID: uuid.Must(uuid.NewV7()), Email: "methodist@example.ru", Role: auth.RoleMethodist, Status: auth.StatusActive}
	student := &auth.User{ID: uuid.Must(uuid.NewV7()), Email: "student@example.ru", Role: auth.RoleStudent, Status: auth.StatusActive}

	t.Run("создание трека методистом", func(t *testing.T) {
		repo := &fakeAuditRepo{}
		contentAdmin := &fakeContent{}
		service := newAdminService(contentAdmin, newFakeUsers(), repo)
		router := newAdminRouter(methodist, contentAdmin, service)

		rec := perform(t, router, http.MethodPost, "/api/admin/exams", `{"code":"oge_math","title":"ОГЭ","subject":"math"}`)
		if rec.Code != http.StatusCreated {
			t.Fatalf("статус = %d, тело = %s", rec.Code, rec.Body.String())
		}
		if len(repo.entries) != 1 {
			t.Fatalf("записей аудита = %d", len(repo.entries))
		}
	})

	t.Run("создание трека учеником", func(t *testing.T) {
		contentAdmin := &fakeContent{}
		service := newAdminService(contentAdmin, newFakeUsers(), &fakeAuditRepo{})
		router := newAdminRouter(student, contentAdmin, service)

		rec := perform(t, router, http.MethodPost, "/api/admin/exams", `{"code":"x","title":"x","subject":"math"}`)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("статус = %d", rec.Code)
		}
	})

	t.Run("некорректный JSON", func(t *testing.T) {
		contentAdmin := &fakeContent{}
		service := newAdminService(contentAdmin, newFakeUsers(), &fakeAuditRepo{})
		router := newAdminRouter(methodist, contentAdmin, service)

		rec := perform(t, router, http.MethodPost, "/api/admin/exams", `{`)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("статус = %d", rec.Code)
		}
	})

	t.Run("список треков", func(t *testing.T) {
		contentAdmin := &fakeContent{}
		service := newAdminService(contentAdmin, newFakeUsers(), &fakeAuditRepo{})
		router := newAdminRouter(methodist, contentAdmin, service)

		rec := perform(t, router, http.MethodGet, "/api/admin/exams", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("статус = %d", rec.Code)
		}
		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("ответ не JSON: %v", err)
		}
		if body["items"] == nil {
			t.Fatalf("нет items: %s", rec.Body.String())
		}
	})

	t.Run("импорт с некорректным идентификатором", func(t *testing.T) {
		contentAdmin := &fakeContent{}
		service := newAdminService(contentAdmin, newFakeUsers(), &fakeAuditRepo{})
		router := newAdminRouter(methodist, contentAdmin, service)

		rec := perform(t, router, http.MethodPost, "/api/admin/tasks/import", `{"items":[{"exam_year_id":"bad"}]}`)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("статус = %d", rec.Code)
		}
	})

	t.Run("загрузка без файла", func(t *testing.T) {
		contentAdmin := &fakeContent{}
		service := newAdminService(contentAdmin, newFakeUsers(), &fakeAuditRepo{})
		router := newAdminRouter(methodist, contentAdmin, service)

		taskID := uuid.Must(uuid.NewV7())
		request := httptest.NewRequest(http.MethodPost, "/api/admin/tasks/"+taskID.String()+"/attachments", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, request)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("статус = %d", rec.Code)
		}
	})
}

func TestAdminHandlerUsersAccess(t *testing.T) {
	methodist := &auth.User{ID: uuid.Must(uuid.NewV7()), Email: "methodist@example.ru", Role: auth.RoleMethodist, Status: auth.StatusActive}
	adminUser := &auth.User{ID: uuid.Must(uuid.NewV7()), Email: "admin@example.ru", Role: auth.RoleAdmin, Status: auth.StatusActive}

	t.Run("методист не видит пользователей", func(t *testing.T) {
		contentAdmin := &fakeContent{}
		service := newAdminService(contentAdmin, newFakeUsers(), &fakeAuditRepo{})
		router := newAdminRouter(methodist, contentAdmin, service)

		rec := perform(t, router, http.MethodGet, "/api/admin/users", "")
		if rec.Code != http.StatusForbidden {
			t.Fatalf("статус = %d", rec.Code)
		}
	})

	t.Run("администратор видит пользователей без секретов", func(t *testing.T) {
		users := newFakeUsers()
		userID := uuid.Must(uuid.NewV7())
		users.users[userID] = &auth.User{
			ID: userID, Email: "user@example.ru", PasswordHash: "secret-hash",
			Role: auth.RoleStudent, Status: auth.StatusActive,
		}
		contentAdmin := &fakeContent{}
		service := newAdminService(contentAdmin, users, &fakeAuditRepo{})
		router := newAdminRouter(adminUser, contentAdmin, service)

		rec := perform(t, router, http.MethodGet, "/api/admin/users", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("статус = %d, тело = %s", rec.Code, rec.Body.String())
		}
		if strings.Contains(rec.Body.String(), "secret-hash") || strings.Contains(rec.Body.String(), "password_hash") {
			t.Fatalf("ответ содержит секрет: %s", rec.Body.String())
		}
	})

	t.Run("некорректная роль", func(t *testing.T) {
		users := newFakeUsers()
		userID := uuid.Must(uuid.NewV7())
		users.users[userID] = &auth.User{ID: userID, Email: "user@example.ru", Role: auth.RoleStudent, Status: auth.StatusActive}
		contentAdmin := &fakeContent{}
		service := newAdminService(contentAdmin, users, &fakeAuditRepo{})
		router := newAdminRouter(adminUser, contentAdmin, service)

		rec := perform(t, router, http.MethodPatch, "/api/admin/users/"+userID.String(), `{"role":"unknown"}`)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("статус = %d", rec.Code)
		}
	})
}
