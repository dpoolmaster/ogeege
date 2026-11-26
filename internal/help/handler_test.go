package help

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"oge-ege/internal/auth"
)

func newRouter(user *auth.User) *gin.Engine {
	gin.SetMode(gin.TestMode)

	handler := NewHandler(NewService())
	router := gin.New()
	if user != nil {
		router.Use(func(c *gin.Context) {
			auth.WithUser(c, user)
			c.Next()
		})
	}

	handler.Register(router.Group("/api"))
	return router
}

func perform(t *testing.T, router *gin.Engine, path string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	return recorder
}

func TestHandlerGet(t *testing.T) {
	methodist := &auth.User{ID: uuid.Must(uuid.NewV7()), Email: "methodist@example.ru", Role: auth.RoleMethodist, Status: auth.StatusActive}
	admin := &auth.User{ID: uuid.Must(uuid.NewV7()), Email: "admin@example.ru", Role: auth.RoleAdmin, Status: auth.StatusActive}

	tests := []struct {
		name       string
		user       *auth.User
		path       string
		wantStatus int
	}{
		{name: "методист открывает справку заданий", user: methodist, path: "/api/help/admin-tasks", wantStatus: http.StatusOK},
		{name: "администратор открывает справку пользователей", user: admin, path: "/api/help/admin-users", wantStatus: http.StatusOK},
		{name: "неизвестный раздел", user: methodist, path: "/api/help/unknown", wantStatus: http.StatusNotFound},
		{name: "методист запрашивает справку пользователей", user: methodist, path: "/api/help/admin-users", wantStatus: http.StatusForbidden},
		{name: "неаутентифицированный запрос", user: nil, path: "/api/help/admin-tasks", wantStatus: http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := perform(t, newRouter(tt.user), tt.path)
			if recorder.Code != tt.wantStatus {
				t.Fatalf("статус = %d, ожидался %d, тело = %s", recorder.Code, tt.wantStatus, recorder.Body.String())
			}
		})
	}
}

func TestHandlerGetReturnsDocument(t *testing.T) {
	methodist := &auth.User{ID: uuid.Must(uuid.NewV7()), Role: auth.RoleMethodist, Status: auth.StatusActive}
	recorder := perform(t, newRouter(methodist), "/api/help/admin-tasks")
	if recorder.Code != http.StatusOK {
		t.Fatalf("статус = %d", recorder.Code)
	}

	var body struct {
		Title string `json:"title"`
		HTML  string `json:"html"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("ответ не JSON: %v", err)
	}
	if body.Title != "Задания" {
		t.Fatalf("заголовок = %q", body.Title)
	}
	if body.HTML == "" {
		t.Fatal("HTML документа пуст")
	}
}
