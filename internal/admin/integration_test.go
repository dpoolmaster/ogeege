package admin

import (
	"context"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"oge-ege/internal/auth"
	"oge-ege/internal/content"
	"oge-ege/internal/shared/clock"
	"oge-ege/internal/shared/db"
)

func TestAdminIntegration(t *testing.T) {
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

	authRepo := auth.NewPgRepository()
	authService := auth.NewService(authRepo, pool, tx, nil, clk, "https://example.ru", logger)
	contentService := content.NewService(content.NewPgRepository(), pool, tx, clk, nil, logger)
	adminRepo := NewPgRepository()
	service := NewService(contentService, authService, adminRepo, pool, tx, clk, logger)

	adminID := uuid.Must(uuid.NewV7())
	user := &auth.User{
		ID:           uuid.Must(uuid.NewV7()),
		Email:        "admin-it-" + uuid.Must(uuid.NewV7()).String() + "@example.ru",
		PasswordHash: "hash",
		Role:         auth.RoleStudent,
		Status:       auth.StatusActive,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := authRepo.CreateUser(ctx, pool, user); err != nil {
		t.Fatalf("создать пользователя: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM admin.audit_log WHERE actor_id = $1`, adminID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.sessions WHERE user_id = $1`, user.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.users WHERE id = $1`, user.ID)
	})

	actor := Actor{ID: adminID, Role: auth.RoleAdmin, IP: "127.0.0.1"}
	if _, err := service.SetUserRole(ctx, actor, user.ID, auth.RoleMethodist); err != nil {
		t.Fatalf("сменить роль: %v", err)
	}

	stored, err := authRepo.GetUserByID(ctx, pool, user.ID)
	if err != nil {
		t.Fatalf("прочитать пользователя: %v", err)
	}
	if stored.Role != auth.RoleMethodist {
		t.Fatalf("роль = %s", stored.Role)
	}

	entries, err := service.ListAudit(ctx, actor, AuditFilter{ActorID: &adminID})
	if err != nil {
		t.Fatalf("прочитать аудит: %v", err)
	}
	found := false
	for _, entry := range entries {
		if entry.Action == "update_role" && entry.Entity == "user" && entry.EntityID != nil && *entry.EntityID == user.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("запись о смене роли не найдена: %+v", entries)
	}

	tokenHash := []byte("session-" + uuid.Must(uuid.NewV7()).String())
	now := time.Now()
	session := &auth.Session{
		ID:         uuid.Must(uuid.NewV7()),
		UserID:     user.ID,
		TokenHash:  tokenHash,
		ExpiresAt:  now.Add(time.Hour),
		LastSeenAt: now,
		CreatedAt:  now,
	}
	if err := authRepo.CreateSession(ctx, pool, session); err != nil {
		t.Fatalf("создать сессию: %v", err)
	}

	if _, err := service.SetUserStatus(ctx, actor, user.ID, auth.StatusBlocked); err != nil {
		t.Fatalf("заблокировать пользователя: %v", err)
	}
	revoked, err := authRepo.GetSessionByTokenHash(ctx, pool, tokenHash)
	if err != nil {
		t.Fatalf("прочитать сессию: %v", err)
	}
	if revoked.RevokedAt == nil {
		t.Fatal("при блокировке сессия не отозвана")
	}
}
