package notify

import (
	"context"
	"os"
	"strings"
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
	prefix := "test-" + uuid.Must(uuid.NewV7()).String()
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM notify.outbox WHERE dedup_key LIKE $1`, prefix+"%")
	})

	template, err := repo.GetActiveTemplate(ctx, pool, TemplateCodeVerification)
	if err != nil {
		t.Fatalf("активный шаблон: %v", err)
	}
	if !strings.Contains(template.BodyMD, "{{.link}}") {
		t.Fatalf("шаблон не содержит ссылку: %s", template.BodyMD)
	}

	item := &OutboxItem{
		ID:          uuid.Must(uuid.NewV7()),
		Channel:     ChannelEmail,
		Recipient:   "student@example.ru",
		Template:    TemplateCodeVerification,
		Payload:     map[string]any{"link": "https://example.ru/verify"},
		Status:      StatusPending,
		ScheduledAt: time.Now().Add(-time.Second),
		DedupKey:    prefix + "-1",
	}
	if err := repo.InsertOutbox(ctx, pool, item); err != nil {
		t.Fatalf("вставить письмо: %v", err)
	}

	duplicate := *item
	duplicate.ID = uuid.Must(uuid.NewV7())
	if err := repo.InsertOutbox(ctx, pool, &duplicate); err != nil {
		t.Fatalf("вставить дубликат: %v", err)
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM notify.outbox WHERE dedup_key = $1`, item.DedupKey).Scan(&count); err != nil {
		t.Fatalf("посчитать письма: %v", err)
	}
	if count != 1 {
		t.Fatalf("ожидалось 1 письмо после дедупликации, получено %d", count)
	}

	due, err := repo.FetchDue(ctx, pool, time.Now(), 10000)
	if err != nil {
		t.Fatalf("выбрать письма: %v", err)
	}
	found := false
	for _, dueItem := range due {
		if dueItem.ID == item.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("письмо не попало в выборку outbox")
	}

	if err := repo.MarkSent(ctx, pool, item.ID, time.Now(), 1); err != nil {
		t.Fatalf("пометить отправленным: %v", err)
	}
	stored, err := repo.GetOutbox(ctx, pool, item.ID)
	if err != nil {
		t.Fatalf("прочитать письмо: %v", err)
	}
	if stored.Status != StatusSent || stored.Attempts != 1 {
		t.Fatalf("состояние = %s/%d", stored.Status, stored.Attempts)
	}

	retryItem := &OutboxItem{
		ID:          uuid.Must(uuid.NewV7()),
		Channel:     ChannelEmail,
		Recipient:   "student@example.ru",
		Template:    TemplateCodeVerification,
		Payload:     map[string]any{"link": "https://example.ru/verify"},
		Status:      StatusPending,
		ScheduledAt: time.Now().Add(-time.Second),
		DedupKey:    prefix + "-2",
	}
	if err := repo.InsertOutbox(ctx, pool, retryItem); err != nil {
		t.Fatalf("вставить письмо для повтора: %v", err)
	}
	future := time.Now().Add(time.Minute)
	if err := repo.Reschedule(ctx, pool, retryItem.ID, 1, future, "временная ошибка"); err != nil {
		t.Fatalf("перенести отправку: %v", err)
	}
	stored, err = repo.GetOutbox(ctx, pool, retryItem.ID)
	if err != nil {
		t.Fatalf("прочитать письмо: %v", err)
	}
	if stored.Status != StatusPending || stored.Attempts != 1 {
		t.Fatalf("состояние после переноса = %s/%d", stored.Status, stored.Attempts)
	}

	failedItem := *retryItem
	failedItem.ID = uuid.Must(uuid.NewV7())
	failedItem.DedupKey = prefix + "-3"
	if err := repo.InsertOutbox(ctx, pool, &failedItem); err != nil {
		t.Fatalf("вставить письмо для отказа: %v", err)
	}
	if err := repo.MarkFailed(ctx, pool, failedItem.ID, MaxAttempts, "окончательная ошибка"); err != nil {
		t.Fatalf("пометить неуспешным: %v", err)
	}
	stored, err = repo.GetOutbox(ctx, pool, failedItem.ID)
	if err != nil {
		t.Fatalf("прочитать письмо: %v", err)
	}
	if stored.Status != StatusFailed || stored.Attempts != MaxAttempts {
		t.Fatalf("состояние после отказа = %s/%d", stored.Status, stored.Attempts)
	}
}
