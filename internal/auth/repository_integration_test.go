package auth

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"oge-ege/internal/shared/db"
)

func TestPgRepositoryIntegration(t *testing.T) {
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

	repo := NewPgRepository()
	email := "repo-" + uuid.Must(uuid.NewV7()).String() + "@example.ru"

	now := time.Now().UTC()
	user := &User{
		ID:           uuid.Must(uuid.NewV7()),
		Email:        email,
		PasswordHash: "hash",
		Role:         RoleStudent,
		Status:       StatusUnverified,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.login_attempts WHERE lower(email) = lower($1)`, email)
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth.users WHERE id = $1`, user.ID)
	})

	if err := repo.CreateUser(ctx, pool, user); err != nil {
		t.Fatalf("создать пользователя: %v", err)
	}
	if err := repo.CreateUser(ctx, pool, user); err == nil {
		t.Fatal("ожидалась ошибка дубликата email")
	}
	if _, err := repo.GetUserByEmail(ctx, pool, email); err != nil {
		t.Fatalf("найти по email: %v", err)
	}
	if _, err := repo.GetUserByID(ctx, pool, user.ID); err != nil {
		t.Fatalf("найти по id: %v", err)
	}
	if _, err := repo.GetUserByEmail(ctx, pool, "missing@example.ru"); err == nil {
		t.Fatal("ожидалась ошибка для отсутствующего email")
	}
	if _, err := repo.GetUserByID(ctx, pool, uuid.Must(uuid.NewV7())); err == nil {
		t.Fatal("ожидалась ошибка для отсутствующего id")
	}

	if err := repo.MarkEmailVerified(ctx, pool, user.ID, now); err != nil {
		t.Fatalf("подтвердить email: %v", err)
	}
	if err := repo.UpdatePasswordHash(ctx, pool, user.ID, "new-hash", now); err != nil {
		t.Fatalf("обновить пароль: %v", err)
	}

	verification := &Verification{
		ID:        uuid.Must(uuid.NewV7()),
		UserID:    user.ID,
		TokenHash: []byte("verification-" + uuid.Must(uuid.NewV7()).String()),
		ExpiresAt: now.Add(time.Hour),
		CreatedAt: now,
	}
	if err := repo.CreateEmailVerification(ctx, pool, verification); err != nil {
		t.Fatalf("создать токен подтверждения: %v", err)
	}
	if err := repo.CreateEmailVerification(ctx, pool, verification); err == nil {
		t.Fatal("ожидалась ошибка дубликата токена")
	}
	if _, err := repo.GetEmailVerificationByHash(ctx, pool, verification.TokenHash); err != nil {
		t.Fatalf("найти токен подтверждения: %v", err)
	}
	if _, err := repo.GetEmailVerificationByHash(ctx, pool, []byte("missing")); err == nil {
		t.Fatal("ожидалась ошибка для отсутствующего токена")
	}
	recent, err := repo.HasRecentEmailVerification(ctx, pool, user.ID, now.Add(-time.Minute))
	if err != nil || !recent {
		t.Fatalf("недавнее письмо подтверждения: recent=%t, err=%v", recent, err)
	}
	recent, err = repo.HasRecentEmailVerification(ctx, pool, user.ID, now.Add(time.Minute))
	if err != nil || recent {
		t.Fatalf("письмо подтверждения вне окна: recent=%t, err=%v", recent, err)
	}
	if err := repo.InvalidateEmailVerifications(ctx, pool, user.ID, now); err != nil {
		t.Fatalf("инвалидировать токены: %v", err)
	}
	if err := repo.ConsumeEmailVerification(ctx, pool, verification.ID, now); err != nil {
		t.Fatalf("использовать токен: %v", err)
	}

	reset := &Verification{
		ID:        uuid.Must(uuid.NewV7()),
		UserID:    user.ID,
		TokenHash: []byte("reset-" + uuid.Must(uuid.NewV7()).String()),
		ExpiresAt: now.Add(time.Hour),
		CreatedAt: now,
	}
	if err := repo.CreatePasswordReset(ctx, pool, reset); err != nil {
		t.Fatalf("создать токен сброса: %v", err)
	}
	if _, err := repo.GetPasswordResetByHash(ctx, pool, reset.TokenHash); err != nil {
		t.Fatalf("найти токен сброса: %v", err)
	}
	if _, err := repo.GetPasswordResetByHash(ctx, pool, []byte("missing")); err == nil {
		t.Fatal("ожидалась ошибка для отсутствующего токена сброса")
	}
	recentReset, err := repo.HasRecentPasswordReset(ctx, pool, user.ID, now.Add(-time.Minute))
	if err != nil || !recentReset {
		t.Fatalf("недавнее письмо сброса: recent=%t, err=%v", recentReset, err)
	}
	recentReset, err = repo.HasRecentPasswordReset(ctx, pool, user.ID, now.Add(time.Minute))
	if err != nil || recentReset {
		t.Fatalf("письмо сброса вне окна: recent=%t, err=%v", recentReset, err)
	}
	if err := repo.InvalidatePasswordResets(ctx, pool, user.ID, now); err != nil {
		t.Fatalf("инвалидировать токены сброса: %v", err)
	}
	if err := repo.ConsumePasswordReset(ctx, pool, reset.ID, now); err != nil {
		t.Fatalf("использовать токен сброса: %v", err)
	}

	for i := 0; i < 2; i++ {
		attempt := &LoginAttempt{
			ID:        uuid.Must(uuid.NewV7()),
			Email:     email,
			IP:        "127.0.0.1",
			Success:   false,
			CreatedAt: now,
		}
		if err := repo.InsertLoginAttempt(ctx, pool, attempt); err != nil {
			t.Fatalf("сохранить попытку: %v", err)
		}
	}
	count, err := repo.CountFailedLoginAttempts(ctx, pool, email, "127.0.0.1", now.Add(-time.Minute))
	if err != nil || count != 2 {
		t.Fatalf("неудачные попытки = %d, ошибка %v", count, err)
	}
	if err := repo.ClearFailedLoginAttempts(ctx, pool, email, "127.0.0.1"); err != nil {
		t.Fatalf("сбросить попытки: %v", err)
	}

	firstSession := &Session{
		ID:         uuid.Must(uuid.NewV7()),
		UserID:     user.ID,
		TokenHash:  []byte("session-" + uuid.Must(uuid.NewV7()).String()),
		UserAgent:  "agent",
		IP:         "127.0.0.1",
		ExpiresAt:  now.Add(time.Hour),
		LastSeenAt: now,
		CreatedAt:  now,
	}
	secondSession := &Session{
		ID:         uuid.Must(uuid.NewV7()),
		UserID:     user.ID,
		TokenHash:  []byte("session-" + uuid.Must(uuid.NewV7()).String()),
		ExpiresAt:  now.Add(time.Hour),
		LastSeenAt: now,
		CreatedAt:  now,
	}
	for _, session := range []*Session{firstSession, secondSession} {
		if err := repo.CreateSession(ctx, pool, session); err != nil {
			t.Fatalf("создать сессию: %v", err)
		}
	}
	if err := repo.CreateSession(ctx, pool, firstSession); err == nil {
		t.Fatal("ожидалась ошибка дубликата сессии")
	}
	if _, err := repo.GetSessionByTokenHash(ctx, pool, firstSession.TokenHash); err != nil {
		t.Fatalf("найти сессию: %v", err)
	}
	if _, err := repo.GetSessionByTokenHash(ctx, pool, []byte("missing")); err == nil {
		t.Fatal("ожидалась ошибка для отсутствующей сессии")
	}
	if err := repo.TouchSession(ctx, pool, firstSession.ID, now.Add(time.Minute), now.Add(time.Hour)); err != nil {
		t.Fatalf("продлить сессию: %v", err)
	}
	if err := repo.RevokeOtherUserSessions(ctx, pool, user.ID, firstSession.ID, now); err != nil {
		t.Fatalf("отозвать прочие сессии: %v", err)
	}
	if err := repo.RevokeSession(ctx, pool, firstSession.ID, now); err != nil {
		t.Fatalf("отозвать сессию: %v", err)
	}
	if err := repo.RevokeUserSessions(ctx, pool, user.ID, now); err != nil {
		t.Fatalf("отозвать сессии пользователя: %v", err)
	}

	if err := repo.AnonymizeUser(ctx, pool, user.ID, now); err != nil {
		t.Fatalf("обезличить пользователя: %v", err)
	}
	anonymized, err := repo.GetUserByID(ctx, pool, user.ID)
	if err != nil {
		t.Fatalf("прочитать обезличенного пользователя: %v", err)
	}
	if anonymized.Status != StatusAnonymized || anonymized.PasswordHash != "!" {
		t.Fatalf("пользователь = %+v", anonymized)
	}
}
