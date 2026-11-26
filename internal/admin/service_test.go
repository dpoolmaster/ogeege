package admin

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"oge-ege/internal/auth"
	"oge-ege/internal/content"
	"oge-ege/internal/shared/db"
)

type fakeContent struct {
	ContentAdmin
	created []content.ExamInput
}

func (f *fakeContent) CreateExam(_ context.Context, in content.ExamInput) (*content.Exam, error) {
	f.created = append(f.created, in)
	return &content.Exam{ID: uuid.Must(uuid.NewV7()), Code: in.Code, Title: in.Title, Subject: in.Subject, Status: content.StatusDraft}, nil
}

type fakeUsers struct {
	UserAdmin
	users map[uuid.UUID]*auth.User
}

func newFakeUsers() *fakeUsers {
	return &fakeUsers{users: make(map[uuid.UUID]*auth.User)}
}

func (f *fakeUsers) GetUser(_ context.Context, id uuid.UUID) (*auth.User, error) {
	user, ok := f.users[id]
	if !ok {
		return nil, errors.New("пользователь не найден")
	}
	return user, nil
}

func (f *fakeUsers) SearchUsers(_ context.Context, _ string, _ uuid.UUID, _ int) ([]auth.User, error) {
	users := make([]auth.User, 0, len(f.users))
	for _, user := range f.users {
		users = append(users, *user)
	}
	return users, nil
}

func (f *fakeUsers) SetUserRole(_ context.Context, id uuid.UUID, role auth.Role) (*auth.User, error) {
	user, ok := f.users[id]
	if !ok {
		return nil, errors.New("пользователь не найден")
	}
	user.Role = role
	return user, nil
}

func (f *fakeUsers) SetUserStatus(_ context.Context, id uuid.UUID, status auth.Status) (*auth.User, error) {
	user, ok := f.users[id]
	if !ok {
		return nil, errors.New("пользователь не найден")
	}
	user.Status = status
	return user, nil
}

type fakeAuditRepo struct {
	Repository
	entries []*AuditEntry
	err     error
}

func (f *fakeAuditRepo) InsertAudit(_ context.Context, _ db.Querier, entry *AuditEntry) error {
	if f.err != nil {
		return f.err
	}
	f.entries = append(f.entries, entry)
	return nil
}

func (f *fakeAuditRepo) ListAudit(_ context.Context, _ db.Querier, _ AuditFilter) ([]AuditEntry, error) {
	entries := make([]AuditEntry, 0, len(f.entries))
	for _, entry := range f.entries {
		entries = append(entries, *entry)
	}
	return entries, nil
}

type fakeTx struct{}

func (fakeTx) WithTx(ctx context.Context, fn func(q db.Querier) error) error {
	return fn(nil)
}

type fixedClock struct{}

func (fixedClock) Now() time.Time {
	return time.Date(2026, time.November, 26, 12, 0, 0, 0, time.UTC)
}

func newAdminService(contentAdmin ContentAdmin, users UserAdmin, repo Repository) *Service {
	return NewService(contentAdmin, users, repo, nil, fakeTx{}, fixedClock{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestContentRoleMatrix(t *testing.T) {
	tests := []struct {
		name    string
		role    auth.Role
		wantErr bool
	}{
		{name: "методист", role: auth.RoleMethodist},
		{name: "администратор", role: auth.RoleAdmin},
		{name: "ученик", role: auth.RoleStudent, wantErr: true},
		{name: "родитель", role: auth.RoleParent, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := newAdminService(&fakeContent{}, newFakeUsers(), &fakeAuditRepo{})
			_, err := service.CreateExam(context.Background(), Actor{ID: uuid.Must(uuid.NewV7()), Role: tt.role}, content.ExamInput{Code: "x", Title: "x", Subject: "math"})
			if tt.wantErr && !errors.Is(err, ErrForbidden) {
				t.Fatalf("ожидалась ошибка доступа, получено %v", err)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("неожиданная ошибка: %v", err)
			}
		})
	}
}

func TestUserAndAuditRoleMatrix(t *testing.T) {
	userID := uuid.Must(uuid.NewV7())
	users := newFakeUsers()
	users.users[userID] = &auth.User{ID: userID, Email: "user@example.ru", Role: auth.RoleStudent, Status: auth.StatusActive}

	tests := []struct {
		name    string
		role    auth.Role
		wantErr bool
	}{
		{name: "администратор", role: auth.RoleAdmin},
		{name: "методист", role: auth.RoleMethodist, wantErr: true},
		{name: "ученик", role: auth.RoleStudent, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := newAdminService(&fakeContent{}, users, &fakeAuditRepo{})
			actor := Actor{ID: uuid.Must(uuid.NewV7()), Role: tt.role}

			_, err := service.SetUserRole(context.Background(), actor, userID, auth.RoleMethodist)
			if tt.wantErr && !errors.Is(err, ErrForbidden) {
				t.Fatalf("смена роли: ожидалась ошибка доступа, получено %v", err)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("смена роли: неожиданная ошибка: %v", err)
			}

			_, err = service.ListAudit(context.Background(), actor, AuditFilter{})
			if tt.wantErr && !errors.Is(err, ErrForbidden) {
				t.Fatalf("аудит: ожидалась ошибка доступа, получено %v", err)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("аудит: неожиданная ошибка: %v", err)
			}
		})
	}
}

func TestAuditRecordedAfterMutation(t *testing.T) {
	repo := &fakeAuditRepo{}
	service := newAdminService(&fakeContent{}, newFakeUsers(), repo)
	actor := Actor{ID: uuid.Must(uuid.NewV7()), Role: auth.RoleMethodist, IP: "127.0.0.1"}

	exam, err := service.CreateExam(context.Background(), actor, content.ExamInput{Code: "oge", Title: "ОГЭ", Subject: "math"})
	if err != nil {
		t.Fatalf("создать трек: %v", err)
	}
	if len(repo.entries) != 1 {
		t.Fatalf("записей аудита = %d", len(repo.entries))
	}

	entry := repo.entries[0]
	if entry.Action != "create" || entry.Entity != "exam" || entry.EntityID == nil || *entry.EntityID != exam.ID {
		t.Fatalf("запись аудита = %+v", entry)
	}
	if entry.ActorID != actor.ID || entry.ActorRole != string(auth.RoleMethodist) || entry.IP != actor.IP {
		t.Fatalf("актор аудита = %+v", entry)
	}
	if len(entry.Before) != 0 || len(entry.After) == 0 {
		t.Fatalf("снимки аудита: before=%s after=%s", entry.Before, entry.After)
	}
}

func TestAuditHasNoSecrets(t *testing.T) {
	userID := uuid.Must(uuid.NewV7())
	users := newFakeUsers()
	users.users[userID] = &auth.User{
		ID: userID, Email: "user@example.ru", PasswordHash: "secret-password-hash",
		Role: auth.RoleStudent, Status: auth.StatusActive,
	}
	repo := &fakeAuditRepo{}
	service := newAdminService(&fakeContent{}, users, repo)
	actor := Actor{ID: uuid.Must(uuid.NewV7()), Role: auth.RoleAdmin}

	if _, err := service.SetUserRole(context.Background(), actor, userID, auth.RoleMethodist); err != nil {
		t.Fatalf("сменить роль: %v", err)
	}
	if len(repo.entries) != 1 {
		t.Fatalf("записей аудита = %d", len(repo.entries))
	}

	payload, err := json.Marshal(repo.entries[0])
	if err != nil {
		t.Fatalf("сериализовать аудит: %v", err)
	}
	for _, secret := range []string{"secret-password-hash", "password_hash", "token"} {
		if strings.Contains(string(payload), secret) {
			t.Fatalf("в аудите найден секрет %q: %s", secret, payload)
		}
	}
}

func TestAuditFailureDoesNotRollback(t *testing.T) {
	repo := &fakeAuditRepo{err: errors.New("БД недоступна")}
	service := newAdminService(&fakeContent{}, newFakeUsers(), repo)
	actor := Actor{ID: uuid.Must(uuid.NewV7()), Role: auth.RoleMethodist}

	exam, err := service.CreateExam(context.Background(), actor, content.ExamInput{Code: "oge", Title: "ОГЭ", Subject: "math"})
	if err != nil {
		t.Fatalf("изменение должно остаться применённым: %v", err)
	}
	if exam == nil || len(repo.entries) != 0 {
		t.Fatalf("некорректное состояние: exam=%v entries=%d", exam, len(repo.entries))
	}
}
