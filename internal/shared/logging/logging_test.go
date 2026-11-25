package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestNewWithWriterWritesJSON(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWithWriter(&buf, "debug")

	logger.Debug("проверка", "request_id", "abc")

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("лог не является JSON: %v (%q)", err, buf.String())
	}
	if entry["msg"] != "проверка" {
		t.Fatalf("msg = %v", entry["msg"])
	}
	if entry["request_id"] != "abc" {
		t.Fatalf("request_id = %v", entry["request_id"])
	}
	if entry["level"] != "DEBUG" {
		t.Fatalf("level = %v", entry["level"])
	}
}

func TestLevelFiltersEntries(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWithWriter(&buf, "warn")

	logger.Info("не должно попасть")
	if buf.Len() != 0 {
		t.Fatalf("info-запись попала в лог уровня warn: %q", buf.String())
	}
}

func TestRedaction(t *testing.T) {
	tests := []struct {
		key   string
		value string
		want  string
	}{
		{key: "password", value: "secret-value", want: "[REDACTED]"},
		{key: "Authorization", value: "Bearer token", want: "[REDACTED]"},
		{key: "Set-Cookie", value: "sid=1", want: "[REDACTED]"},
		{key: "token", value: "abc", want: "[REDACTED]"},
		{key: "request_id", value: "abc", want: "abc"},
	}

	for _, tt := range tests {
		if got := Redact(tt.key, tt.value); got != tt.want {
			t.Errorf("Redact(%q) = %q, ожидалось %q", tt.key, got, tt.want)
		}
	}

	attr := RedactAttr(slog.String("secret", "value"))
	if !strings.Contains(attr.Value.String(), "[REDACTED]") {
		t.Fatalf("атрибут не отредактирован: %v", attr.Value)
	}
}
