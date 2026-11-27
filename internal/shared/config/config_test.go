package config

import (
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestLoadFromEnv(t *testing.T) {
	tests := []struct {
		name        string
		override    map[string]string
		wantErr     bool
		errContains []string
		check       func(t *testing.T, cfg *Config)
	}{
		{
			name:     "полная конфигурация",
			override: nil,
			check: func(t *testing.T, cfg *Config) {
				t.Helper()
				if cfg.App.Env != "local" {
					t.Fatalf("APP_ENV = %q", cfg.App.Env)
				}
				if cfg.DB.MaxConns != 20 || cfg.DB.MinConns != 2 {
					t.Fatalf("размеры пула = %d/%d", cfg.DB.MaxConns, cfg.DB.MinConns)
				}
				if cfg.HTTP.ReadTimeout.String() != "10s" {
					t.Fatalf("HTTP_READ_TIMEOUT = %s", cfg.HTTP.ReadTimeout)
				}
				if cfg.SMTP.TLS {
					t.Fatal("SMTP_TLS должен быть false")
				}
				if cfg.RateLimit.LoginPerMin != 10 || cfg.RateLimit.APIPerMin != 120 {
					t.Fatalf("лимиты = %d/%d", cfg.RateLimit.LoginPerMin, cfg.RateLimit.APIPerMin)
				}
				if cfg.App.BrandName != "Верно" {
					t.Fatalf("BRAND_NAME = %q", cfg.App.BrandName)
				}
			},
		},
		{
			name:     "своё название продукта",
			override: map[string]string{"BRAND_NAME": "Экзамен"},
			check: func(t *testing.T, cfg *Config) {
				t.Helper()
				if cfg.App.BrandName != "Экзамен" {
					t.Fatalf("BRAND_NAME = %q", cfg.App.BrandName)
				}
			},
		},
		{
			name:        "отсутствует RATE_LIMIT_LOGIN_PER_MIN",
			override:    map[string]string{"RATE_LIMIT_LOGIN_PER_MIN": ""},
			wantErr:     true,
			errContains: []string{"RATE_LIMIT_LOGIN_PER_MIN", "обязательна"},
		},
		{
			name:        "нечисловой RATE_LIMIT_API_PER_MIN",
			override:    map[string]string{"RATE_LIMIT_API_PER_MIN": "many"},
			wantErr:     true,
			errContains: []string{"RATE_LIMIT_API_PER_MIN", "целым числом"},
		},
		{
			name:        "нулевой RATE_LIMIT_LOGIN_PER_MIN",
			override:    map[string]string{"RATE_LIMIT_LOGIN_PER_MIN": "0"},
			wantErr:     true,
			errContains: []string{"RATE_LIMIT_LOGIN_PER_MIN", "диапазоне"},
		},
		{
			name:        "отсутствует DATABASE_URL",
			override:    map[string]string{"DATABASE_URL": ""},
			wantErr:     true,
			errContains: []string{"DATABASE_URL", "обязательна"},
		},
		{
			name:        "отсутствует DB_MAX_CONNS",
			override:    map[string]string{"DB_MAX_CONNS": ""},
			wantErr:     true,
			errContains: []string{"DB_MAX_CONNS", "обязательна"},
		},
		{
			name:        "отсутствует SMTP_PORT",
			override:    map[string]string{"SMTP_PORT": ""},
			wantErr:     true,
			errContains: []string{"SMTP_PORT", "обязательна"},
		},
		{
			name:        "нечисловой DB_MAX_CONNS",
			override:    map[string]string{"DB_MAX_CONNS": "many"},
			wantErr:     true,
			errContains: []string{"DB_MAX_CONNS", "целым числом"},
		},
		{
			name:        "DB_MIN_CONNS больше DB_MAX_CONNS",
			override:    map[string]string{"DB_MIN_CONNS": "30"},
			wantErr:     true,
			errContains: []string{"DB_MIN_CONNS", "DB_MAX_CONNS"},
		},
		{
			name:        "некорректный APP_BASE_URL",
			override:    map[string]string{"APP_BASE_URL": "localhost:8080"},
			wantErr:     true,
			errContains: []string{"APP_BASE_URL", "URL"},
		},
		{
			name:        "некорректный LOG_LEVEL",
			override:    map[string]string{"LOG_LEVEL": "verbose"},
			wantErr:     true,
			errContains: []string{"LOG_LEVEL", "debug"},
		},
		{
			name:        "некорректный SMTP_TLS",
			override:    map[string]string{"SMTP_TLS": "yes"},
			wantErr:     true,
			errContains: []string{"SMTP_TLS", "true или false"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setFullEnv(t, tt.override)

			cfg, err := loadFromEnv()
			if tt.wantErr {
				if err == nil {
					t.Fatal("ожидалась ошибка")
				}
				for _, want := range tt.errContains {
					if !strings.Contains(err.Error(), want) {
						t.Fatalf("ошибка %q не содержит %q", err.Error(), want)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("неожиданная ошибка: %v", err)
			}
			if tt.check != nil {
				tt.check(t, cfg)
			}
		})
	}
}

func TestEnvExampleContainsAllSettings(t *testing.T) {
	data, err := os.ReadFile("../../../.env.example")
	if err != nil {
		t.Fatalf("прочитать .env.example: %v", err)
	}

	for _, name := range envNames() {
		pattern := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(name) + `=`)
		if !pattern.Match(data) {
			t.Errorf("в .env.example отсутствует %s", name)
		}
	}
}

func TestLoadDatabaseURL(t *testing.T) {
	t.Run("значение задано", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db")
		got, err := loadDatabaseURLFromEnv()
		if err != nil {
			t.Fatalf("неожиданная ошибка: %v", err)
		}
		if got != "postgres://user:pass@localhost:5432/db" {
			t.Fatalf("получено %q", got)
		}
	})

	t.Run("значение отсутствует", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "")
		_, err := loadDatabaseURLFromEnv()
		if err == nil {
			t.Fatal("ожидалась ошибка")
		}
		if !strings.Contains(err.Error(), "DATABASE_URL") {
			t.Fatalf("ошибка %q не содержит DATABASE_URL", err.Error())
		}
	})
}

func envNames() []string {
	typ := reflect.TypeOf(rawConfig{})
	names := make([]string, 0, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		if name := typ.Field(i).Tag.Get("env"); name != "" {
			names = append(names, name)
		}
	}

	return names
}

func setFullEnv(t *testing.T, override map[string]string) {
	t.Helper()

	values := map[string]string{
		"APP_ENV":                       "local",
		"APP_BASE_URL":                  "http://localhost:8080",
		"SESSION_SECRET":                "test-session-secret",
		"MIGRATIONS_DIR":                "migrations",
		"HTTP_ADDR":                     ":8080",
		"HTTP_READ_TIMEOUT_SECONDS":     "10",
		"HTTP_WRITE_TIMEOUT_SECONDS":    "15",
		"HTTP_SHUTDOWN_TIMEOUT_SECONDS": "10",
		"HTTP_MAX_BODY_BYTES":           "1048576",
		"DATABASE_URL":                  "postgres://user:pass@localhost:5432/db",
		"DB_MAX_CONNS":                  "20",
		"DB_MIN_CONNS":                  "2",
		"LOG_LEVEL":                     "info",
		"SMTP_HOST":                     "localhost",
		"SMTP_PORT":                     "1025",
		"SMTP_USER":                     "",
		"SMTP_PASSWORD":                 "",
		"SMTP_FROM":                     "no-reply@example.ru",
		"SMTP_TLS":                      "false",
		"S3_ENDPOINT":                   "http://localhost:9000",
		"S3_BUCKET":                     "oge-ege",
		"S3_ACCESS_KEY":                 "minio",
		"S3_SECRET_KEY":                 "minio123",
		"S3_REGION":                     "us-east-1",
		"RATE_LIMIT_LOGIN_PER_MIN":      "10",
		"RATE_LIMIT_API_PER_MIN":        "120",
		"BRAND_NAME":                    "Верно",
	}

	for name, value := range values {
		t.Setenv(name, value)
	}
	for name, value := range override {
		t.Setenv(name, value)
	}
}
