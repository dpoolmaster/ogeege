// Package logging настраивает структурированное логирование приложения.
package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

var sensitiveKeys = map[string]struct{}{
	"password":       {},
	"token":          {},
	"secret":         {},
	"session_secret": {},
	"api_key":        {},
	"access_key":     {},
	"secret_key":     {},
	"authorization":  {},
	"cookie":         {},
	"set-cookie":     {},
}

// New создаёт JSON-логгер с уровнем из конфигурации и выводом в stdout.
func New(level string) *slog.Logger {
	return NewWithWriter(os.Stdout, level)
}

// NewWithWriter создаёт JSON-логгер с заданным выводом.
func NewWithWriter(w io.Writer, level string) *slog.Logger {
	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: parseLevel(level)})

	return slog.New(handler)
}

// IsSensitiveKey сообщает, относится ли поле к секретным данным.
func IsSensitiveKey(key string) bool {
	_, ok := sensitiveKeys[strings.ToLower(strings.TrimSpace(key))]

	return ok
}

// Redact заменяет значение секретного поля на маркер.
func Redact(key, value string) string {
	if IsSensitiveKey(key) {
		return "[REDACTED]"
	}

	return value
}

// RedactAttr редактирует секретный атрибут лога.
func RedactAttr(attr slog.Attr) slog.Attr {
	if IsSensitiveKey(attr.Key) {
		return slog.String(attr.Key, "[REDACTED]")
	}

	return attr
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
