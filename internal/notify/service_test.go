package notify

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"oge-ege/internal/shared/db"
	"oge-ege/internal/shared/logging"
	"oge-ege/internal/shared/mailer"
)

type fixedClock struct {
	now time.Time
}

func (f fixedClock) Now() time.Time { return f.now }

type fakeTx struct{}

func (fakeTx) WithTx(_ context.Context, fn func(q db.Querier) error) error {
	return fn(nil)
}

type fakeSender struct {
	messages []mailer.Message
	err      error
}

func (f *fakeSender) Send(_ context.Context, msg mailer.Message) error {
	f.messages = append(f.messages, msg)
	return f.err
}

type fakeRepo struct {
	templates map[string]*Template
	items     map[uuid.UUID]*OutboxItem
	due       []*OutboxItem
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		templates: map[string]*Template{
			TemplateCodeVerification: {
				Code:    TemplateCodeVerification,
				Subject: "Подтвердите email",
				BodyMD:  "Ссылка: {{.link}}",
				Channel: ChannelEmail,
				Version: 1,
				Status:  "active",
			},
		},
		items: make(map[uuid.UUID]*OutboxItem),
	}
}

func (f *fakeRepo) InsertOutbox(_ context.Context, _ db.Querier, item *OutboxItem) error {
	for _, existing := range f.items {
		if item.DedupKey != "" && item.DedupKey == existing.DedupKey {
			return nil
		}
	}
	copied := *item
	f.items[item.ID] = &copied
	return nil
}

func (f *fakeRepo) GetActiveTemplate(_ context.Context, _ db.Querier, code string) (*Template, error) {
	template, ok := f.templates[code]
	if !ok {
		return nil, errors.New("шаблон не найден")
	}
	return template, nil
}

func (f *fakeRepo) FetchDue(_ context.Context, _ db.Querier, _ time.Time, _ int) ([]OutboxItem, error) {
	items := make([]OutboxItem, 0, len(f.due))
	for _, item := range f.due {
		items = append(items, *item)
	}
	return items, nil
}

func (f *fakeRepo) MarkSent(_ context.Context, _ db.Querier, id uuid.UUID, now time.Time, attempts int) error {
	item, ok := f.items[id]
	if !ok {
		return errors.New("письмо не найдено")
	}
	item.Status = StatusSent
	item.Attempts = attempts
	item.SentAt = &now
	item.LastError = nil
	return nil
}

func (f *fakeRepo) Reschedule(_ context.Context, _ db.Querier, id uuid.UUID, attempts int, scheduledAt time.Time, lastError string) error {
	item, ok := f.items[id]
	if !ok {
		return errors.New("письмо не найдено")
	}
	item.Status = StatusPending
	item.Attempts = attempts
	item.ScheduledAt = scheduledAt
	item.LastError = &lastError
	return nil
}

func (f *fakeRepo) MarkFailed(_ context.Context, _ db.Querier, id uuid.UUID, attempts int, lastError string) error {
	item, ok := f.items[id]
	if !ok {
		return errors.New("письмо не найдено")
	}
	item.Status = StatusFailed
	item.Attempts = attempts
	item.LastError = &lastError
	return nil
}

func (f *fakeRepo) GetOutbox(_ context.Context, _ db.Querier, id uuid.UUID) (*OutboxItem, error) {
	item, ok := f.items[id]
	if !ok {
		return nil, errors.New("письмо не найдено")
	}
	return item, nil
}

func TestEnqueueDoesNotSendSynchronously(t *testing.T) {
	repo := newFakeRepo()
	sender := &fakeSender{}
	svc := NewService(repo, fakeTx{}, sender, fixedClock{now: time.Now()}, discardLogger())

	err := svc.Enqueue(context.Background(), nil, Message{
		Recipient: "student@example.ru",
		Template:  TemplateCodeVerification,
		Payload:   map[string]any{"link": "https://example.ru/verify"},
		DedupKey:  "verification:1",
	})
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}

	if len(sender.messages) != 0 {
		t.Fatal("письмо не должно отправляться синхронно")
	}
	if len(repo.items) != 1 {
		t.Fatalf("в outbox ожидалось 1 письмо, получено %d", len(repo.items))
	}
}

func TestEnqueueDeduplicates(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, fakeTx{}, &fakeSender{}, fixedClock{now: time.Now()}, discardLogger())

	msg := Message{
		Recipient: "student@example.ru",
		Template:  TemplateCodeVerification,
		Payload:   map[string]any{"link": "https://example.ru/verify"},
		DedupKey:  "verification:1",
	}
	if err := svc.Enqueue(context.Background(), nil, msg); err != nil {
		t.Fatalf("первая постановка: %v", err)
	}
	if err := svc.Enqueue(context.Background(), nil, msg); err != nil {
		t.Fatalf("повторная постановка: %v", err)
	}

	if len(repo.items) != 1 {
		t.Fatalf("ожидалось 1 письмо, получено %d", len(repo.items))
	}
}

func TestProcessDueSuccess(t *testing.T) {
	now := time.Date(2026, time.November, 26, 12, 0, 0, 0, time.UTC)
	repo := newFakeRepo()
	item := newDueItem(0)
	repo.items[item.ID] = item
	repo.due = []*OutboxItem{item}

	sender := &fakeSender{}
	svc := NewService(repo, fakeTx{}, sender, fixedClock{now: now}, discardLogger())

	processed, err := svc.ProcessDue(context.Background())
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if processed != 1 {
		t.Fatalf("обработано %d писем", processed)
	}
	if len(sender.messages) != 1 || sender.messages[0].To != item.Recipient {
		t.Fatalf("письмо не отправлено: %+v", sender.messages)
	}
	if item.Status != StatusSent || item.Attempts != 1 {
		t.Fatalf("состояние письма = %s/%d", item.Status, item.Attempts)
	}
}

func TestProcessDueRetryThenSuccess(t *testing.T) {
	now := time.Date(2026, time.November, 26, 12, 0, 0, 0, time.UTC)
	repo := newFakeRepo()
	item := newDueItem(0)
	repo.items[item.ID] = item
	repo.due = []*OutboxItem{item}

	sender := &fakeSender{err: errors.New("временная ошибка")}
	svc := NewService(repo, fakeTx{}, sender, fixedClock{now: now}, discardLogger())

	if _, err := svc.ProcessDue(context.Background()); err != nil {
		t.Fatalf("первая попытка: %v", err)
	}
	if item.Status != StatusPending || item.Attempts != 1 {
		t.Fatalf("после ошибки состояние = %s/%d", item.Status, item.Attempts)
	}
	if !item.ScheduledAt.After(now) {
		t.Fatalf("повтор не отложен: %s", item.ScheduledAt)
	}

	sender.err = nil
	if _, err := svc.ProcessDue(context.Background()); err != nil {
		t.Fatalf("вторая попытка: %v", err)
	}
	if item.Status != StatusSent || item.Attempts != 2 {
		t.Fatalf("после успеха состояние = %s/%d", item.Status, item.Attempts)
	}
}

func TestProcessDueExhaustsAttempts(t *testing.T) {
	now := time.Date(2026, time.November, 26, 12, 0, 0, 0, time.UTC)
	repo := newFakeRepo()
	item := newDueItem(MaxAttempts - 1)
	repo.items[item.ID] = item
	repo.due = []*OutboxItem{item}

	svc := NewService(repo, fakeTx{}, &fakeSender{err: errors.New("ошибка")}, fixedClock{now: now}, discardLogger())

	if _, err := svc.ProcessDue(context.Background()); err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if item.Status != StatusFailed || item.Attempts != MaxAttempts {
		t.Fatalf("состояние = %s/%d", item.Status, item.Attempts)
	}
	if item.LastError == nil || *item.LastError == "" {
		t.Fatal("причина ошибки не сохранена")
	}
}

func TestProcessDueSkipsAlreadySent(t *testing.T) {
	repo := newFakeRepo()
	item := newDueItem(0)
	item.Status = StatusSent
	repo.items[item.ID] = item
	repo.due = []*OutboxItem{item}

	sender := &fakeSender{}
	svc := NewService(repo, fakeTx{}, sender, fixedClock{now: time.Now()}, discardLogger())

	processed, err := svc.ProcessDue(context.Background())
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if processed != 0 || len(sender.messages) != 0 {
		t.Fatalf("отправленное письмо обработано повторно: %d/%d", processed, len(sender.messages))
	}
}

func TestProcessDueLogsWithoutPersonalData(t *testing.T) {
	var logs bytes.Buffer
	logger := logging.NewWithWriter(&logs, "debug")

	repo := newFakeRepo()
	item := newDueItem(0)
	item.Recipient = "student@example.ru"
	item.Payload = map[string]any{"link": "https://example.ru/verify?token=secret-token"}
	repo.items[item.ID] = item
	repo.due = []*OutboxItem{item}

	svc := NewService(repo, fakeTx{}, &fakeSender{err: errors.New("ошибка")}, fixedClock{now: time.Now()}, logger)

	if _, err := svc.ProcessDue(context.Background()); err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}

	output := logs.String()
	for _, secret := range []string{"student@example.ru", "secret-token", "Ссылка:"} {
		if strings.Contains(output, secret) {
			t.Fatalf("лог содержит персональные данные %q: %s", secret, output)
		}
	}
	if !strings.Contains(output, "attempts") {
		t.Fatalf("лог не содержит числа попыток: %s", output)
	}
}

func newDueItem(attempts int) *OutboxItem {
	return &OutboxItem{
		ID:          uuid.Must(uuid.NewV7()),
		Recipient:   "student@example.ru",
		Template:    TemplateCodeVerification,
		Channel:     ChannelEmail,
		Payload:     map[string]any{"link": "https://example.ru/verify"},
		Status:      StatusPending,
		Attempts:    attempts,
		ScheduledAt: time.Now(),
	}
}

func discardLogger() *slog.Logger {
	return logging.NewWithWriter(&bytes.Buffer{}, "error")
}
