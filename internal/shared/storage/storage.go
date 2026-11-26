// Package storage реализует S3-совместимое хранилище и проверку изображений.
package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MaxImageBytes — максимальный размер загружаемого изображения.
const MaxImageBytes int64 = 5 << 20

// Store описывает объектное хранилище.
type Store interface {
	Put(ctx context.Context, key, contentType string, r io.Reader, size int64) error
	Get(ctx context.Context, key string) (io.ReadCloser, string, error)
	Delete(ctx context.Context, key string) error
}

// Options содержит настройки подключения к S3-совместимому хранилищу.
type Options struct {
	Endpoint  string
	Bucket    string
	AccessKey string
	SecretKey string
	Region    string
}

// MinIO хранит объекты в S3-совместимом бакете.
type MinIO struct {
	client *minio.Client
	bucket string
}

// NewMinIO создаёт клиент S3-совместимого хранилища.
func NewMinIO(opts Options) (*MinIO, error) {
	endpoint := strings.TrimSpace(opts.Endpoint)
	if endpoint == "" {
		return nil, errors.New("не задан адрес объектного хранилища")
	}
	if strings.TrimSpace(opts.Bucket) == "" {
		return nil, errors.New("не задан бакет объектного хранилища")
	}

	secure := strings.HasPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(strings.TrimPrefix(endpoint, "https://"), "http://")

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(opts.AccessKey, opts.SecretKey, ""),
		Secure: secure,
		Region: opts.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("создать клиент объектного хранилища: %w", err)
	}

	return &MinIO{client: client, bucket: opts.Bucket}, nil
}

// Put сохраняет объект.
func (s *MinIO) Put(ctx context.Context, key, contentType string, r io.Reader, size int64) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, r, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return fmt.Errorf("сохранить объект %q: %w", key, err)
	}

	return nil
}

// Get возвращает объект и его тип содержимого.
func (s *MinIO) Get(ctx context.Context, key string) (io.ReadCloser, string, error) {
	info, err := s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return nil, "", fmt.Errorf("прочитать метаданные объекта %q: %w", key, err)
	}

	object, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, "", fmt.Errorf("получить объект %q: %w", key, err)
	}

	contentType := info.ContentType
	if contentType == "" {
		contentType = http.DetectContentType(nil)
	}

	return object, contentType, nil
}

// Delete удаляет объект.
func (s *MinIO) Delete(ctx context.Context, key string) error {
	if err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("удалить объект %q: %w", key, err)
	}

	return nil
}

var imageExtensions = map[string]string{
	"image/png":  ".png",
	"image/jpeg": ".jpg",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

// DetectImageType определяет допустимый тип изображения по первым байтам.
func DetectImageType(header []byte) (string, error) {
	contentType := http.DetectContentType(header)
	if _, ok := imageExtensions[contentType]; !ok {
		return "", fmt.Errorf("недопустимый тип файла %q", contentType)
	}

	return contentType, nil
}

// ValidateImage проверяет тип и размер изображения.
func ValidateImage(contentType string, size int64) error {
	if _, ok := imageExtensions[contentType]; !ok {
		return fmt.Errorf("недопустимый тип файла %q", contentType)
	}
	if size <= 0 {
		return errors.New("файл пуст")
	}
	if size > MaxImageBytes {
		return fmt.Errorf("размер файла превышает %d байт", MaxImageBytes)
	}

	return nil
}

// ImageKey формирует серверный ключ изображения задания.
func ImageKey(taskID uuid.UUID, contentType string) (string, error) {
	extension, ok := imageExtensions[contentType]
	if !ok {
		return "", fmt.Errorf("недопустимый тип файла %q", contentType)
	}

	name, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("сгенерировать имя объекта: %w", err)
	}

	return "tasks/" + taskID.String() + "/" + name.String() + extension, nil
}

// ValidImageKey сообщает, соответствует ли ключ разрешённому формату.
func ValidImageKey(key string) bool {
	parts := strings.Split(key, "/")
	if len(parts) != 3 || parts[0] != "tasks" {
		return false
	}
	if _, err := uuid.Parse(parts[1]); err != nil {
		return false
	}

	extension := strings.ToLower(parts[2][strings.LastIndex(parts[2], ".")+1:])
	if !isAllowedExtension(extension) {
		return false
	}

	base := strings.TrimSuffix(parts[2], "."+extension)
	_, err := uuid.Parse(base)

	return err == nil
}

func isAllowedExtension(extension string) bool {
	for _, allowed := range imageExtensions {
		if strings.TrimPrefix(allowed, ".") == extension {
			return true
		}
	}

	return false
}
