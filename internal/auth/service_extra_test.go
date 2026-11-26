package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestServiceErrorPaths(t *testing.T) {
	t.Run("регистрация при ошибке outbox", func(t *testing.T) {
		env := newTestEnv()
		env.enqueuer.err = errors.New("outbox недоступен")

		_, err := env.service.Register(context.Background(), RegisterInput{
			Email:    "student@example.ru",
			Password: "strong-password",
			Role:     RoleStudent,
			Consents: []ConsentGrant{{Kind: ConsentPersonalData, Granted: true}},
		})
		if err == nil {
			t.Fatal("ожидалась ошибка постановки письма")
		}
	})

	t.Run("повторная отправка при ошибке outbox", func(t *testing.T) {
		env := newTestEnv()
		env.register(t)
		env.clock.Advance(EmailResendCooldown + time.Second)
		env.enqueuer.err = errors.New("outbox недоступен")

		if err := env.service.ResendVerification(context.Background(), "student@example.ru"); err == nil {
			t.Fatal("ожидалась ошибка постановки письма")
		}
	})

	t.Run("сброс при ошибке outbox", func(t *testing.T) {
		env := newTestEnv()
		user := env.register(t)
		env.verify(t, user)
		env.enqueuer.err = errors.New("outbox недоступен")

		if err := env.service.RequestPasswordReset(context.Background(), "student@example.ru"); err == nil {
			t.Fatal("ожидалась ошибка постановки письма")
		}
	})

	t.Run("вход при ошибке подсчёта попыток", func(t *testing.T) {
		env := newTestEnv()
		env.repo.countErr = errors.New("БД недоступна")

		_, _, err := env.service.Login(context.Background(), "student@example.ru", "strong-password", "127.0.0.1", "agent")
		if err == nil || errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("ошибка = %v", err)
		}
	})

	t.Run("аутентификация при ошибке продления", func(t *testing.T) {
		env := newTestEnv()
		user := env.register(t)
		env.verify(t, user)
		_, token, err := env.service.Login(context.Background(), "student@example.ru", "strong-password", "127.0.0.1", "agent")
		if err != nil {
			t.Fatalf("вход: %v", err)
		}
		env.repo.touchErr = errors.New("БД недоступна")

		if _, _, err := env.service.Authenticate(context.Background(), token); err != nil {
			t.Fatalf("аутентификация должна продолжаться: %v", err)
		}
	})

	t.Run("смена пароля неизвестному пользователю", func(t *testing.T) {
		env := newTestEnv()

		err := env.service.ChangePassword(context.Background(), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), "current", "new-strong-password")
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("ошибка = %v", err)
		}
	})

	t.Run("сброс по неизвестному токену", func(t *testing.T) {
		env := newTestEnv()

		err := env.service.ResetPassword(context.Background(), "unknown-token", "new-strong-password")
		if !errors.Is(err, ErrInvalidToken) {
			t.Fatalf("ошибка = %v", err)
		}
	})

	t.Run("обезличивание при ошибке", func(t *testing.T) {
		env := newTestEnv()
		env.repo.anonymizeErr = errors.New("БД недоступна")

		if err := env.service.AnonymizeAccount(context.Background(), uuid.Must(uuid.NewV7())); err == nil {
			t.Fatal("ожидалась ошибка")
		}
	})

	t.Run("выход при ошибке отзыва", func(t *testing.T) {
		env := newTestEnv()
		env.repo.revokeSessionErr = errors.New("БД недоступна")

		if err := env.service.Logout(context.Background(), uuid.Must(uuid.NewV7())); err == nil {
			t.Fatal("ожидалась ошибка")
		}
	})
}

func TestVerifyPasswordMalformed(t *testing.T) {
	valid, err := HashPassword("strong-password")
	if err != nil {
		t.Fatalf("хеширование: %v", err)
	}

	tests := []struct {
		name  string
		value string
	}{
		{name: "пустой", value: ""},
		{name: "неверное число частей", value: "$argon2id$v=19$m=1,t=1,p=1$salt"},
		{name: "другой алгоритм", value: "$argon2i$v=19$m=65536,t=1,p=4$c2FsdA$aGFzaA"},
		{name: "другая версия", value: "$argon2id$v=18$m=65536,t=1,p=4$c2FsdA$aGFzaA"},
		{name: "без равно", value: "$argon2id$v=19$m65536,t=1,p=4$c2FsdA$aGFzaA"},
		{name: "нечисловые параметры", value: "$argon2id$v=19$m=x,t=1,p=4$c2FsdA$aGFzaA"},
		{name: "слишком много потоков", value: "$argon2id$v=19$m=65536,t=1,p=300$c2FsdA$aGFzaA"},
		{name: "нулевые параметры", value: "$argon2id$v=19$m=0,t=0,p=0$c2FsdA$aGFzaA"},
		{name: "битая соль", value: "$argon2id$v=19$m=65536,t=1,p=4$%%%$aGFzaA"},
		{name: "битый хеш", value: "$argon2id$v=19$m=65536,t=1,p=4$c2FsdA$%%%"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, err := VerifyPassword("strong-password", tt.value)
			if err == nil {
				t.Fatalf("ожидалась ошибка, ok=%t", ok)
			}
		})
	}

	other, err := HashPassword("strong-password")
	if err != nil {
		t.Fatalf("хеширование: %v", err)
	}
	if other == valid {
		t.Fatal("соль не уникальна")
	}
}

func TestServiceEarlyValidation(t *testing.T) {
	env := newTestEnv()

	if err := env.service.VerifyEmail(context.Background(), "   "); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("пустой токен подтверждения: %v", err)
	}
	if err := env.service.ResendVerification(context.Background(), "bad"); err != nil {
		t.Fatalf("некорректный email при повторной отправке: %v", err)
	}
	if err := env.service.RequestPasswordReset(context.Background(), "bad"); err != nil {
		t.Fatalf("некорректный email при сбросе: %v", err)
	}
	if err := env.service.ResetPassword(context.Background(), "   ", "new-strong-password"); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("пустой токен сброса: %v", err)
	}
	if err := env.service.ResetPassword(context.Background(), "token", "short"); !errors.Is(err, ErrWeakPassword) {
		t.Fatalf("короткий пароль при сбросе: %v", err)
	}
	if err := env.service.ChangePassword(context.Background(), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), "current", "short"); !errors.Is(err, ErrWeakPassword) {
		t.Fatalf("короткий пароль при смене: %v", err)
	}
}
