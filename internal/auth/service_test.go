package auth

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"oge-ege/internal/notify"
	"oge-ege/internal/shared/db"
)

type testClock struct {
	now time.Time
}

func (c *testClock) Now() time.Time {
	return c.now
}

func (c *testClock) Advance(d time.Duration) {
	c.now = c.now.Add(d)
}

type fakeTx struct{}

func (fakeTx) WithTx(_ context.Context, fn func(q db.Querier) error) error {
	return fn(nil)
}

type fakeEnqueuer struct {
	messages []notify.Message
	err      error
}

func (f *fakeEnqueuer) Enqueue(_ context.Context, _ db.Querier, msg notify.Message) error {
	f.messages = append(f.messages, msg)
	return f.err
}

type fakeRepo struct {
	usersByEmail     map[string]*User
	usersByID        map[uuid.UUID]*User
	verifications    map[string]*Verification
	resets           map[string]*Verification
	sessions         map[string]*Session
	attempts         []*LoginAttempt
	countErr         error
	touchErr         error
	revokeSessionErr error
	anonymizeErr     error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		usersByEmail:  make(map[string]*User),
		usersByID:     make(map[uuid.UUID]*User),
		verifications: make(map[string]*Verification),
		resets:        make(map[string]*Verification),
		sessions:      make(map[string]*Session),
	}
}

func (f *fakeRepo) CreateUser(_ context.Context, _ db.Querier, user *User) error {
	key := strings.ToLower(user.Email)
	if _, ok := f.usersByEmail[key]; ok {
		return &pgconn.PgError{Code: "23505"}
	}
	f.usersByEmail[key] = user
	f.usersByID[user.ID] = user
	return nil
}

func (f *fakeRepo) GetUserByEmail(_ context.Context, _ db.Querier, email string) (*User, error) {
	user, ok := f.usersByEmail[strings.ToLower(email)]
	if !ok {
		return nil, errors.New("пользователь не найден")
	}
	return user, nil
}

func (f *fakeRepo) GetUserByID(_ context.Context, _ db.Querier, id uuid.UUID) (*User, error) {
	user, ok := f.usersByID[id]
	if !ok {
		return nil, errors.New("пользователь не найден")
	}
	return user, nil
}

func (f *fakeRepo) MarkEmailVerified(_ context.Context, _ db.Querier, id uuid.UUID, now time.Time) error {
	user, ok := f.usersByID[id]
	if !ok {
		return errors.New("пользователь не найден")
	}
	user.Status = StatusActive
	user.EmailVerifiedAt = &now
	return nil
}

func (f *fakeRepo) UpdatePasswordHash(_ context.Context, _ db.Querier, id uuid.UUID, hash string, now time.Time) error {
	user, ok := f.usersByID[id]
	if !ok {
		return errors.New("пользователь не найден")
	}
	user.PasswordHash = hash
	user.UpdatedAt = now
	return nil
}

func (f *fakeRepo) AnonymizeUser(_ context.Context, _ db.Querier, id uuid.UUID, now time.Time) error {
	if f.anonymizeErr != nil {
		return f.anonymizeErr
	}
	user, ok := f.usersByID[id]
	if !ok {
		return errors.New("пользователь не найден")
	}
	user.Email = fmt.Sprintf("deleted+%s@anonymized.invalid", id)
	user.PasswordHash = "!"
	user.Status = StatusAnonymized
	user.UpdatedAt = now
	return nil
}

func (f *fakeRepo) CreateEmailVerification(_ context.Context, _ db.Querier, verification *Verification) error {
	copied := *verification
	f.verifications[string(verification.TokenHash)] = &copied
	return nil
}

func (f *fakeRepo) GetEmailVerificationByHash(_ context.Context, _ db.Querier, tokenHash []byte) (*Verification, error) {
	verification, ok := f.verifications[string(tokenHash)]
	if !ok {
		return nil, errors.New("токен не найден")
	}
	return verification, nil
}

func (f *fakeRepo) ConsumeEmailVerification(_ context.Context, _ db.Querier, id uuid.UUID, now time.Time) error {
	for _, verification := range f.verifications {
		if verification.ID == id {
			verification.ConsumedAt = &now
			return nil
		}
	}
	return errors.New("токен не найден")
}

func (f *fakeRepo) InvalidateEmailVerifications(_ context.Context, _ db.Querier, userID uuid.UUID, now time.Time) error {
	for _, verification := range f.verifications {
		if verification.UserID == userID && verification.ConsumedAt == nil {
			verification.ConsumedAt = &now
		}
	}
	return nil
}

func (f *fakeRepo) HasRecentEmailVerification(_ context.Context, _ db.Querier, userID uuid.UUID, since time.Time) (bool, error) {
	for _, verification := range f.verifications {
		if verification.UserID == userID && verification.CreatedAt.After(since) {
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeRepo) CreatePasswordReset(_ context.Context, _ db.Querier, reset *Verification) error {
	copied := *reset
	f.resets[string(reset.TokenHash)] = &copied
	return nil
}

func (f *fakeRepo) GetPasswordResetByHash(_ context.Context, _ db.Querier, tokenHash []byte) (*Verification, error) {
	reset, ok := f.resets[string(tokenHash)]
	if !ok {
		return nil, errors.New("токен не найден")
	}
	return reset, nil
}

func (f *fakeRepo) ConsumePasswordReset(_ context.Context, _ db.Querier, id uuid.UUID, now time.Time) error {
	for _, reset := range f.resets {
		if reset.ID == id {
			reset.ConsumedAt = &now
			return nil
		}
	}
	return errors.New("токен не найден")
}

func (f *fakeRepo) InvalidatePasswordResets(_ context.Context, _ db.Querier, userID uuid.UUID, now time.Time) error {
	for _, reset := range f.resets {
		if reset.UserID == userID && reset.ConsumedAt == nil {
			reset.ConsumedAt = &now
		}
	}
	return nil
}

func (f *fakeRepo) HasRecentPasswordReset(_ context.Context, _ db.Querier, userID uuid.UUID, since time.Time) (bool, error) {
	for _, reset := range f.resets {
		if reset.UserID == userID && reset.CreatedAt.After(since) {
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeRepo) InsertLoginAttempt(_ context.Context, _ db.Querier, attempt *LoginAttempt) error {
	copied := *attempt
	f.attempts = append(f.attempts, &copied)
	return nil
}

func (f *fakeRepo) CountFailedLoginAttempts(_ context.Context, _ db.Querier, email, ip string, since time.Time) (int, error) {
	if f.countErr != nil {
		return 0, f.countErr
	}
	count := 0
	for _, attempt := range f.attempts {
		if strings.EqualFold(attempt.Email, email) && attempt.IP == ip && !attempt.Success && !attempt.CreatedAt.Before(since) {
			count++
		}
	}
	return count, nil
}

func (f *fakeRepo) ClearFailedLoginAttempts(_ context.Context, _ db.Querier, email, ip string) error {
	filtered := f.attempts[:0]
	for _, attempt := range f.attempts {
		if strings.EqualFold(attempt.Email, email) && attempt.IP == ip && !attempt.Success {
			continue
		}
		filtered = append(filtered, attempt)
	}
	f.attempts = filtered
	return nil
}

func (f *fakeRepo) CreateSession(_ context.Context, _ db.Querier, session *Session) error {
	copied := *session
	f.sessions[string(session.TokenHash)] = &copied
	return nil
}

func (f *fakeRepo) GetSessionByTokenHash(_ context.Context, _ db.Querier, tokenHash []byte) (*Session, error) {
	session, ok := f.sessions[string(tokenHash)]
	if !ok {
		return nil, errors.New("сессия не найдена")
	}
	return session, nil
}

func (f *fakeRepo) TouchSession(_ context.Context, _ db.Querier, id uuid.UUID, lastSeenAt, expiresAt time.Time) error {
	if f.touchErr != nil {
		return f.touchErr
	}
	for _, session := range f.sessions {
		if session.ID == id {
			session.LastSeenAt = lastSeenAt
			session.ExpiresAt = expiresAt
			return nil
		}
	}
	return errors.New("сессия не найдена")
}

func (f *fakeRepo) RevokeSession(_ context.Context, _ db.Querier, id uuid.UUID, now time.Time) error {
	if f.revokeSessionErr != nil {
		return f.revokeSessionErr
	}
	for _, session := range f.sessions {
		if session.ID == id {
			session.RevokedAt = &now
			return nil
		}
	}
	return errors.New("сессия не найдена")
}

func (f *fakeRepo) RevokeUserSessions(_ context.Context, _ db.Querier, userID uuid.UUID, now time.Time) error {
	for _, session := range f.sessions {
		if session.UserID == userID {
			session.RevokedAt = &now
		}
	}
	return nil
}

func (f *fakeRepo) RevokeOtherUserSessions(_ context.Context, _ db.Querier, userID, keepSessionID uuid.UUID, now time.Time) error {
	for _, session := range f.sessions {
		if session.UserID == userID && session.ID != keepSessionID {
			session.RevokedAt = &now
		}
	}
	return nil
}

type testEnv struct {
	service  *Service
	repo     *fakeRepo
	enqueuer *fakeEnqueuer
	clock    *testClock
}

func newTestEnv() *testEnv {
	repo := newFakeRepo()
	enqueuer := &fakeEnqueuer{}
	clk := &testClock{now: time.Date(2026, time.November, 26, 12, 0, 0, 0, time.UTC)}
	service := NewService(repo, nil, fakeTx{}, enqueuer, clk, "https://example.ru", discardLogger())

	return &testEnv{service: service, repo: repo, enqueuer: enqueuer, clock: clk}
}

func (e *testEnv) register(t *testing.T) *User {
	t.Helper()

	user, err := e.service.Register(context.Background(), RegisterInput{
		Email:    "student@example.ru",
		Password: "strong-password",
		Role:     RoleStudent,
		Consents: []ConsentGrant{{Kind: ConsentPersonalData, Granted: true}},
		IP:       "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("регистрация: %v", err)
	}

	return user
}

func (e *testEnv) verify(t *testing.T, user *User) {
	t.Helper()

	token := tokenFromLastMessage(t, e.enqueuer)
	if err := e.service.VerifyEmail(context.Background(), token); err != nil {
		t.Fatalf("подтверждение email: %v", err)
	}
}

func tokenFromLastMessage(t *testing.T, enqueuer *fakeEnqueuer) string {
	t.Helper()

	if len(enqueuer.messages) == 0 {
		t.Fatal("письмо не поставлено в outbox")
	}
	link, ok := enqueuer.messages[len(enqueuer.messages)-1].Payload["link"].(string)
	if !ok {
		t.Fatal("в письме нет ссылки")
	}
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

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "валидный", input: " Student@Example.RU ", want: "student@example.ru"},
		{name: "пустой", input: "", wantErr: true},
		{name: "без собаки", input: "student.example.ru", wantErr: true},
		{name: "с именем", input: "Иван <student@example.ru>", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateEmail(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("ожидалась ошибка")
				}
				return
			}
			if err != nil {
				t.Fatalf("неожиданная ошибка: %v", err)
			}
			if got != tt.want {
				t.Fatalf("получено %q, ожидалось %q", got, tt.want)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "ровно 10", input: "0123456789"},
		{name: "9 символов", input: "012345678", wantErr: true},
		{name: "пустой", input: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.input)
			if tt.wantErr && err == nil {
				t.Fatal("ожидалась ошибка")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("неожиданная ошибка: %v", err)
			}
		})
	}
}

func TestNormalizeConsents(t *testing.T) {
	t.Run("обязательное согласие", func(t *testing.T) {
		grants, err := NormalizeConsents([]ConsentGrant{{Kind: ConsentPersonalData, Granted: true}})
		if err != nil {
			t.Fatalf("неожиданная ошибка: %v", err)
		}
		if grants[0].DocumentVersion != DefaultConsentVersion {
			t.Fatalf("версия = %q", grants[0].DocumentVersion)
		}
	})

	t.Run("без согласия", func(t *testing.T) {
		_, err := NormalizeConsents([]ConsentGrant{{Kind: ConsentPersonalData, Granted: false}})
		if !errors.Is(err, ErrConsentRequired) {
			t.Fatalf("ошибка = %v", err)
		}
	})

	t.Run("неизвестный вид", func(t *testing.T) {
		_, err := NormalizeConsents([]ConsentGrant{{Kind: "unknown", Granted: true}})
		if err == nil {
			t.Fatal("ожидалась ошибка")
		}
	})
}

func TestPasswordHashRoundTrip(t *testing.T) {
	hash, err := HashPassword("strong-password")
	if err != nil {
		t.Fatalf("хеширование: %v", err)
	}
	if strings.Contains(hash, "strong-password") {
		t.Fatal("хеш содержит пароль")
	}

	ok, err := VerifyPassword("strong-password", hash)
	if err != nil || !ok {
		t.Fatalf("верный пароль не принят: %v", err)
	}

	ok, err = VerifyPassword("wrong-password", hash)
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if ok {
		t.Fatal("неверный пароль принят")
	}

	if _, err := VerifyPassword("x", "broken"); err == nil {
		t.Fatal("ожидалась ошибка формата")
	}
}

func TestRegisterSuccess(t *testing.T) {
	env := newTestEnv()
	user := env.register(t)

	if user.Status != StatusUnverified {
		t.Fatalf("статус = %s", user.Status)
	}
	if len(env.repo.verifications) != 1 {
		t.Fatalf("токенов подтверждения = %d", len(env.repo.verifications))
	}
	if len(env.enqueuer.messages) != 1 {
		t.Fatalf("писем = %d", len(env.enqueuer.messages))
	}
	link, _ := env.enqueuer.messages[0].Payload["link"].(string)
	if !strings.Contains(link, "/verify-email?token=") {
		t.Fatalf("ссылка = %q", link)
	}
}

func TestRegisterRejections(t *testing.T) {
	tests := []struct {
		name string
		in   RegisterInput
		want error
	}{
		{
			name: "некорректный email",
			in:   RegisterInput{Email: "bad", Password: "strong-password", Role: RoleStudent, Consents: []ConsentGrant{{Kind: ConsentPersonalData, Granted: true}}},
			want: ErrInvalidEmail,
		},
		{
			name: "короткий пароль",
			in:   RegisterInput{Email: "student@example.ru", Password: "short", Role: RoleStudent, Consents: []ConsentGrant{{Kind: ConsentPersonalData, Granted: true}}},
			want: ErrWeakPassword,
		},
		{
			name: "служебная роль",
			in:   RegisterInput{Email: "student@example.ru", Password: "strong-password", Role: RoleAdmin, Consents: []ConsentGrant{{Kind: ConsentPersonalData, Granted: true}}},
			want: ErrInvalidRole,
		},
		{
			name: "без согласия",
			in:   RegisterInput{Email: "student@example.ru", Password: "strong-password", Role: RoleStudent},
			want: ErrConsentRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv()
			_, err := env.service.Register(context.Background(), tt.in)
			if !errors.Is(err, tt.want) {
				t.Fatalf("ошибка = %v, ожидалась %v", err, tt.want)
			}
		})
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	env := newTestEnv()
	env.register(t)

	_, err := env.service.Register(context.Background(), RegisterInput{
		Email:    "STUDENT@example.ru",
		Password: "strong-password",
		Role:     RoleStudent,
		Consents: []ConsentGrant{{Kind: ConsentPersonalData, Granted: true}},
	})
	if !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("ошибка = %v", err)
	}
}

func TestVerifyEmail(t *testing.T) {
	env := newTestEnv()
	user := env.register(t)
	env.verify(t, user)

	if user.Status != StatusActive || user.EmailVerifiedAt == nil {
		t.Fatalf("статус = %s", user.Status)
	}

	t.Run("повторное использование", func(t *testing.T) {
		token := tokenFromLastMessage(t, env.enqueuer)
		err := env.service.VerifyEmail(context.Background(), token)
		if !errors.Is(err, ErrTokenUsed) {
			t.Fatalf("ошибка = %v", err)
		}
	})

	t.Run("истёкший токен", func(t *testing.T) {
		fresh := newTestEnv()
		fresh.register(t)
		fresh.clock.Advance(VerificationTTL + time.Minute)

		token := tokenFromLastMessage(t, fresh.enqueuer)
		err := fresh.service.VerifyEmail(context.Background(), token)
		if !errors.Is(err, ErrTokenExpired) {
			t.Fatalf("ошибка = %v", err)
		}
	})
}

func TestResendVerificationInvalidatesOldToken(t *testing.T) {
	env := newTestEnv()
	env.register(t)
	oldToken := tokenFromLastMessage(t, env.enqueuer)
	env.clock.Advance(EmailResendCooldown + time.Second)

	if err := env.service.ResendVerification(context.Background(), "student@example.ru"); err != nil {
		t.Fatalf("повторная отправка: %v", err)
	}
	newToken := tokenFromLastMessage(t, env.enqueuer)
	if oldToken == newToken {
		t.Fatal("токен не обновился")
	}

	if err := env.service.VerifyEmail(context.Background(), oldToken); !errors.Is(err, ErrTokenUsed) {
		t.Fatalf("старый токен: %v", err)
	}
	if err := env.service.VerifyEmail(context.Background(), newToken); err != nil {
		t.Fatalf("новый токен: %v", err)
	}
}

func TestResendVerificationCooldown(t *testing.T) {
	env := newTestEnv()
	env.register(t)
	sent := len(env.enqueuer.messages)

	if err := env.service.ResendVerification(context.Background(), "student@example.ru"); err != nil {
		t.Fatalf("повторная отправка: %v", err)
	}
	if len(env.enqueuer.messages) != sent {
		t.Fatal("письмо отправлено повторно в период охлаждения")
	}

	env.clock.Advance(EmailResendCooldown + time.Second)
	if err := env.service.ResendVerification(context.Background(), "student@example.ru"); err != nil {
		t.Fatalf("повторная отправка после охлаждения: %v", err)
	}
	if len(env.enqueuer.messages) != sent+1 {
		t.Fatal("письмо не отправлено после периода охлаждения")
	}
}

func TestResendVerificationIsNeutral(t *testing.T) {
	env := newTestEnv()

	if err := env.service.ResendVerification(context.Background(), "nobody@example.ru"); err != nil {
		t.Fatalf("ожидался нейтральный ответ: %v", err)
	}
	if len(env.enqueuer.messages) != 0 {
		t.Fatal("письмо отправлено для несуществующего email")
	}
}

func TestLogin(t *testing.T) {
	env := newTestEnv()
	user := env.register(t)

	t.Run("до подтверждения", func(t *testing.T) {
		_, _, err := env.service.Login(context.Background(), "student@example.ru", "strong-password", "127.0.0.1", "agent")
		if !errors.Is(err, ErrEmailNotVerified) {
			t.Fatalf("ошибка = %v", err)
		}
	})

	env.verify(t, user)

	t.Run("успешный вход", func(t *testing.T) {
		got, token, err := env.service.Login(context.Background(), "student@example.ru", "strong-password", "127.0.0.1", "agent")
		if err != nil {
			t.Fatalf("вход: %v", err)
		}
		if got.ID != user.ID || token == "" {
			t.Fatal("некорректный результат входа")
		}
		if len(env.repo.sessions) != 1 {
			t.Fatalf("сессий = %d", len(env.repo.sessions))
		}
	})

	t.Run("неверный пароль и неизвестный email", func(t *testing.T) {
		_, _, wrongPassword := env.service.Login(context.Background(), "student@example.ru", "wrong-password", "127.0.0.1", "agent")
		_, _, unknownEmail := env.service.Login(context.Background(), "nobody@example.ru", "strong-password", "127.0.0.1", "agent")
		if !errors.Is(wrongPassword, ErrInvalidCredentials) || !errors.Is(unknownEmail, ErrInvalidCredentials) {
			t.Fatalf("ошибки = %v / %v", wrongPassword, unknownEmail)
		}
	})
}

func TestLoginBlockedAccount(t *testing.T) {
	env := newTestEnv()
	user := env.register(t)
	env.verify(t, user)
	user.Status = StatusBlocked

	_, _, err := env.service.Login(context.Background(), "student@example.ru", "strong-password", "127.0.0.1", "agent")
	if !errors.Is(err, ErrAccountBlocked) {
		t.Fatalf("ошибка = %v", err)
	}
}

func TestLoginTooManyAttempts(t *testing.T) {
	env := newTestEnv()
	user := env.register(t)
	env.verify(t, user)

	for i := 0; i < MaxLoginAttempts; i++ {
		_, _, _ = env.service.Login(context.Background(), "student@example.ru", "wrong-password", "10.0.0.1", "agent")
	}

	_, _, err := env.service.Login(context.Background(), "student@example.ru", "strong-password", "10.0.0.1", "agent")
	if !errors.Is(err, ErrTooManyAttempts) {
		t.Fatalf("ошибка = %v", err)
	}
}

func TestLoginSuccessResetsAttempts(t *testing.T) {
	env := newTestEnv()
	user := env.register(t)
	env.verify(t, user)

	_, _, _ = env.service.Login(context.Background(), "student@example.ru", "wrong-password", "10.0.0.1", "agent")
	if _, _, err := env.service.Login(context.Background(), "student@example.ru", "strong-password", "10.0.0.1", "agent"); err != nil {
		t.Fatalf("успешный вход: %v", err)
	}

	count, err := env.repo.CountFailedLoginAttempts(context.Background(), nil, "student@example.ru", "10.0.0.1", env.clock.Now().Add(-LoginAttemptWindow))
	if err != nil {
		t.Fatalf("посчитать попытки: %v", err)
	}
	if count != 0 {
		t.Fatalf("счётчик не сброшен: %d", count)
	}
}

func TestAuthenticateAndLogout(t *testing.T) {
	env := newTestEnv()
	user := env.register(t)
	env.verify(t, user)

	_, token, err := env.service.Login(context.Background(), "student@example.ru", "strong-password", "127.0.0.1", "agent")
	if err != nil {
		t.Fatalf("вход: %v", err)
	}

	gotUser, session, err := env.service.Authenticate(context.Background(), token)
	if err != nil {
		t.Fatalf("аутентификация: %v", err)
	}
	if gotUser.ID != user.ID || session == nil {
		t.Fatal("некорректная аутентификация")
	}

	if err := env.service.Logout(context.Background(), session.ID); err != nil {
		t.Fatalf("выход: %v", err)
	}
	if _, _, err := env.service.Authenticate(context.Background(), token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("отозванная сессия принята: %v", err)
	}
}

func TestAuthenticateExpiredSession(t *testing.T) {
	env := newTestEnv()
	user := env.register(t)
	env.verify(t, user)

	_, token, err := env.service.Login(context.Background(), "student@example.ru", "strong-password", "127.0.0.1", "agent")
	if err != nil {
		t.Fatalf("вход: %v", err)
	}

	env.clock.Advance(SessionDuration + time.Minute)
	if _, _, err := env.service.Authenticate(context.Background(), token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("истёкшая сессия принята: %v", err)
	}
}

func TestPasswordResetFlow(t *testing.T) {
	env := newTestEnv()
	user := env.register(t)
	env.verify(t, user)

	_, token, err := env.service.Login(context.Background(), "student@example.ru", "strong-password", "127.0.0.1", "agent")
	if err != nil {
		t.Fatalf("вход: %v", err)
	}

	if err := env.service.RequestPasswordReset(context.Background(), "student@example.ru"); err != nil {
		t.Fatalf("запрос сброса: %v", err)
	}
	resetToken := tokenFromLastMessage(t, env.enqueuer)

	if err := env.service.ResetPassword(context.Background(), resetToken, "new-strong-password"); err != nil {
		t.Fatalf("сброс пароля: %v", err)
	}

	if _, _, err := env.service.Authenticate(context.Background(), token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("сессия не отозвана: %v", err)
	}
	if _, _, err := env.service.Login(context.Background(), "student@example.ru", "new-strong-password", "127.0.0.1", "agent"); err != nil {
		t.Fatalf("вход с новым паролем: %v", err)
	}
	if _, _, err := env.service.Login(context.Background(), "student@example.ru", "strong-password", "127.0.0.1", "agent"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("старый пароль принят: %v", err)
	}
}

func TestPasswordResetCooldown(t *testing.T) {
	env := newTestEnv()
	user := env.register(t)
	env.verify(t, user)

	if err := env.service.RequestPasswordReset(context.Background(), "student@example.ru"); err != nil {
		t.Fatalf("запрос сброса: %v", err)
	}
	sent := len(env.enqueuer.messages)

	if err := env.service.RequestPasswordReset(context.Background(), "student@example.ru"); err != nil {
		t.Fatalf("повторный запрос сброса: %v", err)
	}
	if len(env.enqueuer.messages) != sent {
		t.Fatal("письмо сброса отправлено повторно в период охлаждения")
	}

	env.clock.Advance(EmailResendCooldown + time.Second)
	if err := env.service.RequestPasswordReset(context.Background(), "student@example.ru"); err != nil {
		t.Fatalf("запрос сброса после охлаждения: %v", err)
	}
	if len(env.enqueuer.messages) != sent+1 {
		t.Fatal("письмо сброса не отправлено после периода охлаждения")
	}
}

func TestPasswordResetNeutral(t *testing.T) {
	env := newTestEnv()

	if err := env.service.RequestPasswordReset(context.Background(), "nobody@example.ru"); err != nil {
		t.Fatalf("ожидался нейтральный ответ: %v", err)
	}
	if len(env.enqueuer.messages) != 0 {
		t.Fatal("письмо отправлено для несуществующего email")
	}
}

func TestPasswordResetExpired(t *testing.T) {
	env := newTestEnv()
	user := env.register(t)
	env.verify(t, user)

	if err := env.service.RequestPasswordReset(context.Background(), "student@example.ru"); err != nil {
		t.Fatalf("запрос сброса: %v", err)
	}
	resetToken := tokenFromLastMessage(t, env.enqueuer)
	env.clock.Advance(PasswordResetTTL + time.Minute)

	err := env.service.ResetPassword(context.Background(), resetToken, "new-strong-password")
	if !errors.Is(err, ErrTokenExpired) {
		t.Fatalf("ошибка = %v", err)
	}
}

func TestChangePassword(t *testing.T) {
	env := newTestEnv()
	user := env.register(t)
	env.verify(t, user)

	_, firstToken, err := env.service.Login(context.Background(), "student@example.ru", "strong-password", "127.0.0.1", "first")
	if err != nil {
		t.Fatalf("первый вход: %v", err)
	}
	_, secondToken, err := env.service.Login(context.Background(), "student@example.ru", "strong-password", "127.0.0.2", "second")
	if err != nil {
		t.Fatalf("второй вход: %v", err)
	}
	_, secondSession, err := env.service.Authenticate(context.Background(), secondToken)
	if err != nil {
		t.Fatalf("аутентификация: %v", err)
	}

	t.Run("неверный текущий пароль", func(t *testing.T) {
		err := env.service.ChangePassword(context.Background(), user.ID, secondSession.ID, "wrong-password", "new-strong-password")
		if !errors.Is(err, ErrInvalidCurrentPassword) {
			t.Fatalf("ошибка = %v", err)
		}
	})

	t.Run("успешная смена", func(t *testing.T) {
		err := env.service.ChangePassword(context.Background(), user.ID, secondSession.ID, "strong-password", "new-strong-password")
		if err != nil {
			t.Fatalf("смена пароля: %v", err)
		}

		if _, _, err := env.service.Authenticate(context.Background(), firstToken); !errors.Is(err, ErrInvalidToken) {
			t.Fatalf("прочая сессия не отозвана: %v", err)
		}
		if _, _, err := env.service.Authenticate(context.Background(), secondToken); err != nil {
			t.Fatalf("текущая сессия отозвана: %v", err)
		}
	})
}

func TestAnonymizeAccount(t *testing.T) {
	env := newTestEnv()
	user := env.register(t)
	env.verify(t, user)

	_, token, err := env.service.Login(context.Background(), "student@example.ru", "strong-password", "127.0.0.1", "agent")
	if err != nil {
		t.Fatalf("вход: %v", err)
	}

	if err := env.service.AnonymizeAccount(context.Background(), user.ID); err != nil {
		t.Fatalf("обезличивание: %v", err)
	}
	if _, _, err := env.service.Authenticate(context.Background(), token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("сессия не отозвана: %v", err)
	}
	if _, _, err := env.service.Login(context.Background(), "student@example.ru", "strong-password", "127.0.0.1", "agent"); err == nil {
		t.Fatal("вход по прежним данным разрешён")
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestLoginAttemptWindowExpires(t *testing.T) {
	env := newTestEnv()
	user := env.register(t)
	env.verify(t, user)

	for i := 0; i < MaxLoginAttempts; i++ {
		_, _, _ = env.service.Login(context.Background(), "student@example.ru", "wrong-password", "10.0.0.9", "agent")
	}
	if _, _, err := env.service.Login(context.Background(), "student@example.ru", "strong-password", "10.0.0.9", "agent"); !errors.Is(err, ErrTooManyAttempts) {
		t.Fatalf("ожидалась блокировка, получено %v", err)
	}

	env.clock.Advance(LoginAttemptWindow + time.Minute)
	if _, _, err := env.service.Login(context.Background(), "student@example.ru", "strong-password", "10.0.0.9", "agent"); err != nil {
		t.Fatalf("после окна вход должен быть разрешён: %v", err)
	}
}
