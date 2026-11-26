package storage

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestValidateImage(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		size        int64
		wantErr     bool
	}{
		{name: "png", contentType: "image/png", size: 1024},
		{name: "jpeg", contentType: "image/jpeg", size: 1024},
		{name: "недопустимый тип", contentType: "application/pdf", size: 1024, wantErr: true},
		{name: "пустой файл", contentType: "image/png", size: 0, wantErr: true},
		{name: "слишком большой", contentType: "image/png", size: MaxImageBytes + 1, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateImage(tt.contentType, tt.size)
			if tt.wantErr && err == nil {
				t.Fatal("ожидалась ошибка")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("неожиданная ошибка: %v", err)
			}
		})
	}
}

func TestDetectImageType(t *testing.T) {
	png := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0, 0}
	if got, err := DetectImageType(png); err != nil || got != "image/png" {
		t.Fatalf("DetectImageType = %q, %v", got, err)
	}
	if _, err := DetectImageType([]byte("plain text")); err == nil {
		t.Fatal("ожидалась ошибка для текстового файла")
	}
}

func TestImageKey(t *testing.T) {
	taskID := uuid.Must(uuid.NewV7())
	key, err := ImageKey(taskID, "image/png")
	if err != nil {
		t.Fatalf("ImageKey: %v", err)
	}
	if !strings.HasPrefix(key, "tasks/"+taskID.String()+"/") || !strings.HasSuffix(key, ".png") {
		t.Fatalf("некорректный ключ %q", key)
	}
	if !ValidImageKey(key) {
		t.Fatalf("ключ %q должен быть допустим", key)
	}
	if _, err := ImageKey(taskID, "text/plain"); err == nil {
		t.Fatal("ожидалась ошибка для недопустимого типа")
	}
}

func TestValidImageKeyRejectsForeignKeys(t *testing.T) {
	valid := "tasks/" + uuid.Must(uuid.NewV7()).String() + "/" + uuid.Must(uuid.NewV7()).String() + ".png"

	tests := []struct {
		name string
		key  string
		want bool
	}{
		{name: "допустимый", key: valid, want: true},
		{name: "другой префикс", key: strings.Replace(valid, "tasks/", "other/", 1)},
		{name: "обход пути", key: "tasks/../../etc/passwd"},
		{name: "недопустимое расширение", key: strings.TrimSuffix(valid, ".png") + ".exe"},
		{name: "без имени", key: "tasks/" + uuid.Must(uuid.NewV7()).String()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidImageKey(tt.key); got != tt.want {
				t.Fatalf("ValidImageKey(%q) = %v, ожидалось %v", tt.key, got, tt.want)
			}
		})
	}
}

type fakeStore struct {
	objects map[string][]byte
	types   map[string]string
}

func (f *fakeStore) Put(_ context.Context, key, contentType string, r io.Reader, _ int64) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	f.objects[key] = data
	f.types[key] = contentType
	return nil
}

func (f *fakeStore) Get(_ context.Context, key string) (io.ReadCloser, string, error) {
	data, ok := f.objects[key]
	if !ok {
		return nil, "", io.EOF
	}
	return io.NopCloser(strings.NewReader(string(data))), f.types[key], nil
}

func (f *fakeStore) Delete(_ context.Context, key string) error {
	delete(f.objects, key)
	return nil
}

func TestMediaHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	key := "tasks/" + uuid.Must(uuid.NewV7()).String() + "/" + uuid.Must(uuid.NewV7()).String() + ".png"
	store := &fakeStore{
		objects: map[string][]byte{key: []byte("image")},
		types:   map[string]string{key: "image/png"},
	}

	router := gin.New()
	router.GET("/media/*key", MediaHandler(store))

	t.Run("известный ключ", func(t *testing.T) {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/media/"+key, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("статус = %d", rec.Code)
		}
		if rec.Header().Get("Content-Type") != "image/png" {
			t.Fatalf("Content-Type = %q", rec.Header().Get("Content-Type"))
		}
	})

	t.Run("чужой ключ", func(t *testing.T) {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/media/other/file.png", nil))
		if rec.Code != http.StatusForbidden {
			t.Fatalf("статус = %d", rec.Code)
		}
	})

	t.Run("отсутствующий ключ", func(t *testing.T) {
		missing := "tasks/" + uuid.Must(uuid.NewV7()).String() + "/" + uuid.Must(uuid.NewV7()).String() + ".png"
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/media/"+missing, nil))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("статус = %d", rec.Code)
		}
	})
}
