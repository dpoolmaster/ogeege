package auth_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"oge-ege/internal/auth"
	"oge-ege/internal/notify"
	"oge-ege/internal/profile"
	"oge-ege/internal/shared/clock"
	"oge-ege/internal/shared/db"
	"oge-ege/internal/shared/mailer"
)

type noopSender struct{}

func (noopSender) Send(context.Context, mailer.Message) error { return nil }

type failingEnqueuer struct{}

func (failingEnqueuer) Enqueue(context.Context, db.Querier, notify.Message) error {
	return os.ErrDeadlineExceeded
}

func TestAuthIntegration(t *testing.T) {
	databaseURL, ok := os.LookupEnv("TEST_DATABASE_URL")
	if !ok || databaseURL == "" {
		t.Skip("TEST_DATABASE_URL не задан")
	}

	ctx := context.Background()
	pool, err := db.Open(ctx, databaseURL, 0, 2)
	if err != nil {
		t.Fatalf("открыть пул: %v", err)
	}
	defer pool.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	clk := clock.New()
	tx := db.NewTxManager(pool)

	notifyService := notify.NewService(notify.NewPgRepository(), tx, noopSender{}, clk, logger)
	authRepo := auth.NewPgRepository()
	authService := auth.NewService(authRepo, pool, tx, notifyService, clk, "https://example.ru", logger)
	profileService := profile.NewService(profile.NewPgRepository(), pool, tx, clk, authService, logger)

	email := "it-" + uuid.Must(uuid.NewV7()).String() + "@example.ru"
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM notify.outbox WHERE recipient = $1`, email)
		_, _ = pool.Exec(context.Background(), `DELETE FROM profile.profiles WHERE user_id IN (SELECT id FROM auth.users WHERE email = $1)`, email)
		_, _ = pool.Exec(context.Background(), `DELETE FROM profile.consents WHERE user_id IN (SELECT id FROM auth.users WHERE email = $1)`, email)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.login_attempts WHERE lower(email) = lower($1)`, email)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.users WHERE email = $1`, email)
	})

	user, err := authService.Register(ctx, auth.RegisterInput{
		Email:    email,
		Password: "strong-password",
		Role:     auth.RoleStudent,
		Consents: []auth.ConsentGrant{{Kind: auth.ConsentPersonalData, Granted: true}},
		IP:       "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("регистрация: %v", err)
	}
	if user.Status != auth.StatusUnverified {
		t.Fatalf("статус = %s", user.Status)
	}

	if err := profileService.RecordRegistrationConsents(ctx, user.ID, []auth.ConsentGrant{
		{Kind: auth.ConsentPersonalData, Granted: true},
	}, "127.0.0.1"); err != nil {
		t.Fatalf("согласия: %v", err)
	}

	var outboxCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM notify.outbox WHERE recipient = $1 AND template = $2`, email, notify.TemplateCodeVerification).Scan(&outboxCount); err != nil {
		t.Fatalf("посчитать outbox: %v", err)
	}
	if outboxCount != 1 {
		t.Fatalf("письмо подтверждения в outbox = %d", outboxCount)
	}

	verificationToken := outboxToken(t, pool, email, notify.TemplateCodeVerification)
	if err := authService.VerifyEmail(ctx, verificationToken); err != nil {
		t.Fatalf("подтверждение email: %v", err)
	}

	_, sessionToken, err := authService.Login(ctx, email, "strong-password", "127.0.0.1", "agent")
	if err != nil {
		t.Fatalf("вход: %v", err)
	}
	if _, _, err := authService.Authenticate(ctx, sessionToken); err != nil {
		t.Fatalf("аутентификация: %v", err)
	}

	t.Run("RBAC на защищённом маршруте", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.GET("/admin", auth.Auth(authService), auth.RequireRole(auth.RoleAdmin), func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: sessionToken})
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("ученик получил статус %d", rec.Code)
		}

		if _, err := pool.Exec(ctx, `UPDATE auth.users SET role = 'admin' WHERE id = $1`, user.ID); err != nil {
			t.Fatalf("сменить роль: %v", err)
		}

		rec = httptest.NewRecorder()
		req = httptest.NewRequest(http.MethodGet, "/admin", nil)
		req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: sessionToken})
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("администратор получил статус %d", rec.Code)
		}
	})

	t.Run("сброс пароля отзывает сессии", func(t *testing.T) {
		if err := authService.RequestPasswordReset(ctx, email); err != nil {
			t.Fatalf("запрос сброса: %v", err)
		}

		resetToken := outboxToken(t, pool, email, notify.TemplateCodePasswordReset)
		if err := authService.ResetPassword(ctx, resetToken, "new-strong-password"); err != nil {
			t.Fatalf("сброс пароля: %v", err)
		}

		if _, _, err := authService.Authenticate(ctx, sessionToken); err == nil {
			t.Fatal("сессия не отозвана после сброса")
		}
		if _, _, err := authService.Login(ctx, email, "new-strong-password", "127.0.0.1", "agent"); err != nil {
			t.Fatalf("вход с новым паролем: %v", err)
		}
	})
}

func TestRegisterRollbackOnEnqueueFailure(t *testing.T) {
	databaseURL, ok := os.LookupEnv("TEST_DATABASE_URL")
	if !ok || databaseURL == "" {
		t.Skip("TEST_DATABASE_URL не задан")
	}

	ctx := context.Background()
	pool, err := db.Open(ctx, databaseURL, 0, 2)
	if err != nil {
		t.Fatalf("открыть пул: %v", err)
	}
	defer pool.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tx := db.NewTxManager(pool)
	authRepo := auth.NewPgRepository()
	authService := auth.NewService(authRepo, pool, tx, failingEnqueuer{}, clock.New(), "https://example.ru", logger)

	email := "rollback-" + uuid.Must(uuid.NewV7()).String() + "@example.ru"
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.users WHERE email = $1`, email)
	})

	_, err = authService.Register(ctx, auth.RegisterInput{
		Email:    email,
		Password: "strong-password",
		Role:     auth.RoleStudent,
		Consents: []auth.ConsentGrant{{Kind: auth.ConsentPersonalData, Granted: true}},
		IP:       "127.0.0.1",
	})
	if err == nil {
		t.Fatal("ожидалась ошибка постановки письма")
	}

	var users int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM auth.users WHERE email = $1`, email).Scan(&users); err != nil {
		t.Fatalf("посчитать пользователей: %v", err)
	}
	if users != 0 {
		t.Fatalf("транзакция не откатилась, пользователей = %d", users)
	}
}

func outboxToken(t *testing.T, pool *pgxpool.Pool, recipient, templateCode string) string {
	t.Helper()

	var payload []byte
	err := pool.QueryRow(
		context.Background(),
		`SELECT payload_json FROM notify.outbox WHERE recipient = $1 AND template = $2 ORDER BY created_at DESC LIMIT 1`,
		recipient,
		templateCode,
	).Scan(&payload)
	if err != nil {
		t.Fatalf("прочитать письмо из outbox: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(payload, &data); err != nil {
		t.Fatalf("разобрать payload: %v", err)
	}
	link, _ := data["link"].(string)
	parsed, err := url.Parse(link)
	if err != nil {
		t.Fatalf("разобрать ссылку: %v", err)
	}
	token := parsed.Query().Get("token")
	if token == "" {
		t.Fatal("в ссылке нет токена")
	}

	return token
}

func TestMandatoryEmailsIgnoreNotificationSettings(t *testing.T) {
	databaseURL, ok := os.LookupEnv("TEST_DATABASE_URL")
	if !ok || databaseURL == "" {
		t.Skip("TEST_DATABASE_URL не задан")
	}

	ctx := context.Background()
	pool, err := db.Open(ctx, databaseURL, 0, 2)
	if err != nil {
		t.Fatalf("открыть пул: %v", err)
	}
	defer pool.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	clk := clock.New()
	tx := db.NewTxManager(pool)
	notifyService := notify.NewService(notify.NewPgRepository(), tx, noopSender{}, clk, logger)
	authService := auth.NewService(auth.NewPgRepository(), pool, tx, notifyService, clk, "https://example.ru", logger)
	profileService := profile.NewService(profile.NewPgRepository(), pool, tx, clk, authService, logger)

	email := "settings-" + uuid.Must(uuid.NewV7()).String() + "@example.ru"
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM notify.outbox WHERE recipient = $1`, email)
		_, _ = pool.Exec(context.Background(), `DELETE FROM profile.settings WHERE user_id IN (SELECT id FROM auth.users WHERE email = $1)`, email)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.users WHERE email = $1`, email)
	})

	user, err := authService.Register(ctx, auth.RegisterInput{
		Email:    email,
		Password: "strong-password",
		Role:     auth.RoleStudent,
		Consents: []auth.ConsentGrant{{Kind: auth.ConsentPersonalData, Granted: true}},
	})
	if err != nil {
		t.Fatalf("регистрация: %v", err)
	}
	if _, err := profileService.SaveSettings(ctx, user.ID, false, false); err != nil {
		t.Fatalf("настройки: %v", err)
	}
	if err := authService.RequestPasswordReset(ctx, email); err != nil {
		t.Fatalf("запрос сброса: %v", err)
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM notify.outbox WHERE recipient = $1 AND template = $2`, email, notify.TemplateCodePasswordReset).Scan(&count); err != nil {
		t.Fatalf("посчитать письма: %v", err)
	}
	if count != 1 {
		t.Fatalf("обязательное письмо не отправлено при отключённых уведомлениях: %d", count)
	}
}

func TestAccountDeletionIntegration(t *testing.T) {
	databaseURL, ok := os.LookupEnv("TEST_DATABASE_URL")
	if !ok || databaseURL == "" {
		t.Skip("TEST_DATABASE_URL не задан")
	}

	ctx := context.Background()
	pool, err := db.Open(ctx, databaseURL, 0, 2)
	if err != nil {
		t.Fatalf("открыть пул: %v", err)
	}
	defer pool.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	clk := clock.New()
	tx := db.NewTxManager(pool)
	notifyService := notify.NewService(notify.NewPgRepository(), tx, noopSender{}, clk, logger)
	authService := auth.NewService(auth.NewPgRepository(), pool, tx, notifyService, clk, "https://example.ru", logger)
	profileService := profile.NewService(profile.NewPgRepository(), pool, tx, clk, authService, logger)

	email := "delete-" + uuid.Must(uuid.NewV7()).String() + "@example.ru"
	var userID uuid.UUID
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM notify.outbox WHERE recipient = $1`, email)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.login_attempts WHERE lower(email) = lower($1)`, email)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.users WHERE email = $1 OR id = $2`, email, userID)
	})

	user, err := authService.Register(ctx, auth.RegisterInput{
		Email:    email,
		Password: "strong-password",
		Role:     auth.RoleStudent,
		Consents: []auth.ConsentGrant{{Kind: auth.ConsentPersonalData, Granted: true}},
	})
	if err != nil {
		t.Fatalf("регистрация: %v", err)
	}
	userID = user.ID
	if err := authService.VerifyEmail(ctx, outboxToken(t, pool, email, notify.TemplateCodeVerification)); err != nil {
		t.Fatalf("подтверждение: %v", err)
	}
	_, sessionToken, err := authService.Login(ctx, email, "strong-password", "127.0.0.1", "agent")
	if err != nil {
		t.Fatalf("вход: %v", err)
	}

	if err := profileService.DeleteAccount(ctx, user.ID); err != nil {
		t.Fatalf("удаление аккаунта: %v", err)
	}

	if _, _, err := authService.Authenticate(ctx, sessionToken); err == nil {
		t.Fatal("сессия не отозвана после удаления")
	}
	if _, _, err := authService.Login(ctx, email, "strong-password", "127.0.0.1", "agent"); err == nil {
		t.Fatal("вход по прежним данным после удаления разрешён")
	}
}
