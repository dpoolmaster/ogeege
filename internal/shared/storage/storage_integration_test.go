package storage

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestMinIOIntegration(t *testing.T) {
	endpoint, ok := os.LookupEnv("TEST_S3_ENDPOINT")
	if !ok || endpoint == "" {
		t.Skip("TEST_S3_ENDPOINT не задан")
	}

	store, err := NewMinIO(Options{
		Endpoint:  endpoint,
		Bucket:    envOrDefault("TEST_S3_BUCKET", "oge-ege"),
		AccessKey: envOrDefault("TEST_S3_ACCESS_KEY", "minio"),
		SecretKey: envOrDefault("TEST_S3_SECRET_KEY", "minio123"),
		Region:    envOrDefault("TEST_S3_REGION", "us-east-1"),
	})
	if err != nil {
		t.Fatalf("создать клиент: %v", err)
	}

	ctx := context.Background()
	key := "tests/" + uuid.Must(uuid.NewV7()).String() + "/object.txt"
	t.Cleanup(func() {
		_ = store.Delete(context.Background(), key)
	})

	content := []byte("storage integration")
	if err := store.Put(ctx, key, "text/plain", bytes.NewReader(content), int64(len(content))); err != nil {
		t.Fatalf("Put: %v", err)
	}

	object, contentType, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer func() {
		_ = object.Close()
	}()

	data, err := io.ReadAll(object)
	if err != nil {
		t.Fatalf("прочитать объект: %v", err)
	}
	if !bytes.Equal(data, content) {
		t.Fatalf("содержимое = %q", data)
	}
	if !strings.HasPrefix(contentType, "text/plain") {
		t.Fatalf("Content-Type = %q", contentType)
	}

	if err := store.Delete(ctx, key); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, _, err := store.Get(ctx, key); err == nil {
		t.Fatal("ожидалась ошибка для удалённого объекта")
	}
}

func envOrDefault(name, fallback string) string {
	if value, ok := os.LookupEnv(name); ok && value != "" {
		return value
	}

	return fallback
}
