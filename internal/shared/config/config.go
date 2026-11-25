package config

import (
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

// Config содержит все настройки приложения, прочитанные при старте.
type Config struct {
	App  AppConfig
	HTTP HTTPConfig
	DB   DBConfig
	Log  LogConfig
	SMTP SMTPConfig
	S3   S3Config
}

// AppConfig содержит общие настройки приложения.
type AppConfig struct {
	Env           string
	BaseURL       string
	SessionSecret string
	MigrationsDir string
}

// HTTPConfig содержит настройки HTTP-сервера.
type HTTPConfig struct {
	Addr            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
	MaxBodyBytes    int64
}

// DBConfig содержит настройки подключения к PostgreSQL.
type DBConfig struct {
	URL      string
	MinConns int32
	MaxConns int32
}

// LogConfig содержит настройки логирования.
type LogConfig struct {
	Level string
}

// SMTPConfig содержит настройки отправки писем.
type SMTPConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
	TLS      bool
}

// S3Config содержит настройки объектного хранилища.
type S3Config struct {
	Endpoint  string
	Bucket    string
	AccessKey string
	SecretKey string
	Region    string
}

// Load читает конфигурацию из окружения и локального .env и валидирует её.
func Load() (*Config, error) {
	if err := loadDotEnv(); err != nil {
		return nil, err
	}

	return loadFromEnv()
}

func loadFromEnv() (*Config, error) {
	var raw rawConfig
	if err := cleanenv.ReadEnv(&raw); err != nil {
		return nil, fmt.Errorf("прочитать переменные окружения: %w", err)
	}

	return parseConfig(&raw)
}

// LoadDatabaseURL читает только DATABASE_URL; используется утилитой миграций.
func LoadDatabaseURL() (string, error) {
	if err := loadDotEnv(); err != nil {
		return "", err
	}

	return loadDatabaseURLFromEnv()
}

func loadDatabaseURLFromEnv() (string, error) {
	var raw struct {
		DatabaseURL string `env:"DATABASE_URL"`
	}
	if err := cleanenv.ReadEnv(&raw); err != nil {
		return "", fmt.Errorf("прочитать переменные окружения: %w", err)
	}

	value := strings.TrimSpace(raw.DatabaseURL)
	if value == "" {
		return "", errors.New("переменная DATABASE_URL обязательна")
	}

	return value, nil
}

func loadDotEnv() error {
	info, err := os.Stat(".env")
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("проверить .env: %w", err)
	}
	if info.IsDir() {
		return errors.New(".env должен быть файлом")
	}
	if err := godotenv.Load(".env"); err != nil {
		return fmt.Errorf("загрузить .env: %w", err)
	}

	return nil
}

type rawConfig struct {
	AppEnv                  string `env:"APP_ENV"`
	AppBaseURL              string `env:"APP_BASE_URL"`
	AppSessionSecret        string `env:"SESSION_SECRET"`
	AppMigrationsDir        string `env:"MIGRATIONS_DIR"`
	HTTPAddr                string `env:"HTTP_ADDR"`
	HTTPReadTimeoutSeconds  string `env:"HTTP_READ_TIMEOUT_SECONDS" env-default:"10"`
	HTTPWriteTimeoutSeconds string `env:"HTTP_WRITE_TIMEOUT_SECONDS" env-default:"15"`
	HTTPShutdownTimeoutSecs string `env:"HTTP_SHUTDOWN_TIMEOUT_SECONDS" env-default:"10"`
	HTTPMaxBodyBytes        string `env:"HTTP_MAX_BODY_BYTES" env-default:"1048576"`
	DBURL                   string `env:"DATABASE_URL"`
	DBMaxConns              string `env:"DB_MAX_CONNS"`
	DBMinConns              string `env:"DB_MIN_CONNS"`
	LogLevel                string `env:"LOG_LEVEL"`
	SMTPHost                string `env:"SMTP_HOST"`
	SMTPPort                string `env:"SMTP_PORT"`
	SMTPUser                string `env:"SMTP_USER"`
	SMTPPassword            string `env:"SMTP_PASSWORD"`
	SMTPFrom                string `env:"SMTP_FROM"`
	SMTPTLS                 string `env:"SMTP_TLS"`
	S3Endpoint              string `env:"S3_ENDPOINT"`
	S3Bucket                string `env:"S3_BUCKET"`
	S3AccessKey             string `env:"S3_ACCESS_KEY"`
	S3SecretKey             string `env:"S3_SECRET_KEY"`
	S3Region                string `env:"S3_REGION"`
}

type validationErrors []string

func (e *validationErrors) addf(format string, args ...any) {
	*e = append(*e, fmt.Sprintf(format, args...))
}

func (e validationErrors) err() error {
	if len(e) == 0 {
		return nil
	}

	return errors.New(strings.Join(e, "; "))
}

func parseConfig(raw *rawConfig) (*Config, error) {
	var errs validationErrors

	appEnv := normalizeEnum(requireString(&errs, "APP_ENV", raw.AppEnv))
	if appEnv != "" && !isOneOf(appEnv, "local", "test", "development", "staging", "production") {
		errs.addf("переменная APP_ENV должна быть одной из: local, test, development, staging, production")
	}

	baseURL := requireString(&errs, "APP_BASE_URL", raw.AppBaseURL)
	if baseURL != "" {
		validateURL(&errs, "APP_BASE_URL", baseURL)
	}

	sessionSecret := requireString(&errs, "SESSION_SECRET", raw.AppSessionSecret)
	migrationsDir := requireString(&errs, "MIGRATIONS_DIR", raw.AppMigrationsDir)

	httpAddr := requireString(&errs, "HTTP_ADDR", raw.HTTPAddr)
	readTimeout := parseInt(&errs, "HTTP_READ_TIMEOUT_SECONDS", raw.HTTPReadTimeoutSeconds, 1, 3600)
	writeTimeout := parseInt(&errs, "HTTP_WRITE_TIMEOUT_SECONDS", raw.HTTPWriteTimeoutSeconds, 1, 3600)
	shutdownTimeout := parseInt(&errs, "HTTP_SHUTDOWN_TIMEOUT_SECONDS", raw.HTTPShutdownTimeoutSecs, 1, 3600)
	maxBodyBytes := parseInt64(&errs, "HTTP_MAX_BODY_BYTES", raw.HTTPMaxBodyBytes, 1, 1<<30)

	dbURL := requireString(&errs, "DATABASE_URL", raw.DBURL)
	dbMaxConns := parseRequiredInt(&errs, "DB_MAX_CONNS", raw.DBMaxConns, 1, 10000)
	dbMinConns := parseRequiredInt(&errs, "DB_MIN_CONNS", raw.DBMinConns, 0, 10000)
	if dbMaxConns > 0 && dbMinConns > dbMaxConns {
		errs.addf("переменная DB_MIN_CONNS не может быть больше DB_MAX_CONNS")
	}

	logLevel := normalizeEnum(requireString(&errs, "LOG_LEVEL", raw.LogLevel))
	if logLevel != "" && !isOneOf(logLevel, "debug", "info", "warn", "error") {
		errs.addf("переменная LOG_LEVEL должна быть одной из: debug, info, warn, error")
	}

	smtpHost := requireString(&errs, "SMTP_HOST", raw.SMTPHost)
	smtpPort := parseRequiredInt(&errs, "SMTP_PORT", raw.SMTPPort, 1, 65535)
	smtpFrom := requireString(&errs, "SMTP_FROM", raw.SMTPFrom)
	smtpTLS := parseRequiredBool(&errs, "SMTP_TLS", raw.SMTPTLS)

	s3Endpoint := requireString(&errs, "S3_ENDPOINT", raw.S3Endpoint)
	if s3Endpoint != "" {
		validateURL(&errs, "S3_ENDPOINT", s3Endpoint)
	}
	s3Bucket := requireString(&errs, "S3_BUCKET", raw.S3Bucket)
	s3AccessKey := requireString(&errs, "S3_ACCESS_KEY", raw.S3AccessKey)
	s3SecretKey := requireString(&errs, "S3_SECRET_KEY", raw.S3SecretKey)
	s3Region := requireString(&errs, "S3_REGION", raw.S3Region)

	if err := errs.err(); err != nil {
		return nil, fmt.Errorf("некорректная конфигурация: %w", err)
	}

	return &Config{
		App: AppConfig{
			Env:           appEnv,
			BaseURL:       baseURL,
			SessionSecret: sessionSecret,
			MigrationsDir: migrationsDir,
		},
		HTTP: HTTPConfig{
			Addr:            httpAddr,
			ReadTimeout:     time.Duration(readTimeout) * time.Second,
			WriteTimeout:    time.Duration(writeTimeout) * time.Second,
			ShutdownTimeout: time.Duration(shutdownTimeout) * time.Second,
			MaxBodyBytes:    maxBodyBytes,
		},
		DB: DBConfig{
			URL:      dbURL,
			MinConns: int32(dbMinConns),
			MaxConns: int32(dbMaxConns),
		},
		Log: LogConfig{
			Level: logLevel,
		},
		SMTP: SMTPConfig{
			Host:     smtpHost,
			Port:     smtpPort,
			User:     raw.SMTPUser,
			Password: raw.SMTPPassword,
			From:     smtpFrom,
			TLS:      smtpTLS,
		},
		S3: S3Config{
			Endpoint:  s3Endpoint,
			Bucket:    s3Bucket,
			AccessKey: s3AccessKey,
			SecretKey: s3SecretKey,
			Region:    s3Region,
		},
	}, nil
}

func requireString(errs *validationErrors, name, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		errs.addf("переменная %s обязательна", name)
	}

	return value
}

func parseRequiredInt(errs *validationErrors, name, value string, min, max int) int {
	if strings.TrimSpace(value) == "" {
		errs.addf("переменная %s обязательна", name)
		return 0
	}

	return parseInt(errs, name, value, min, max)
}

func parseInt(errs *validationErrors, name, value string, min, max int) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		errs.addf("переменная %s должна быть целым числом, получено %q", name, value)
		return 0
	}
	if parsed < min || parsed > max {
		errs.addf("переменная %s должна быть в диапазоне от %d до %d, получено %d", name, min, max, parsed)
		return 0
	}

	return parsed
}

func parseInt64(errs *validationErrors, name, value string, min, max int64) int64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		errs.addf("переменная %s должна быть целым числом, получено %q", name, value)
		return 0
	}
	if parsed < min || parsed > max {
		errs.addf("переменная %s должна быть в диапазоне от %d до %d, получено %d", name, min, max, parsed)
		return 0
	}

	return parsed
}

func parseRequiredBool(errs *validationErrors, name, value string) bool {
	if strings.TrimSpace(value) == "" {
		errs.addf("переменная %s обязательна", name)
		return false
	}

	return parseBool(errs, name, value)
}

func parseBool(errs *validationErrors, name, value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		errs.addf("переменная %s должна быть true или false, получено %q", name, value)
		return false
	}

	return parsed
}

func validateURL(errs *validationErrors, name, value string) {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		errs.addf("переменная %s должна быть абсолютным URL с http или https", name)
	}
}

func normalizeEnum(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func isOneOf(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}

	return false
}
