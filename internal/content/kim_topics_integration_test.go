package content

import (
	"context"
	"math/rand"
	"os"
	"testing"

	"github.com/google/uuid"

	"oge-ege/internal/shared/clock"
	"oge-ege/internal/shared/db"
)

func TestListKimTopicsByExamYearIntegration(t *testing.T) {
	databaseURL, _ := os.LookupEnv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL не задан")
	}

	ctx := context.Background()
	pool, err := db.Open(ctx, databaseURL, 0, 2)
	if err != nil {
		t.Fatalf("открыть пул: %v", err)
	}
	defer pool.Close()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("начать транзакцию: %v", err)
	}
	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	service := NewService(NewPgRepository(), tx, txOnly{tx: tx}, clock.New(), rand.New(rand.NewSource(1)), discardLogger())

	exam := createExam(t, ctx, service, "it_links_"+uuid.Must(uuid.NewV7()).String())
	year := createYear(t, ctx, service, exam.ID)
	kim1 := createKim(t, ctx, service, year.ID, 1, 1, AnswerTypeShort, 1)
	kim2 := createKim(t, ctx, service, year.ID, 2, 1, AnswerTypeShort, 2)
	topic := createTopic(t, ctx, service, exam.ID, nil, "links", "Связи")

	for _, kim := range []*KimNumber{kim1, kim2} {
		if _, err := service.SetKimTopics(ctx, kim.ID, []KimTopicInput{{TopicID: topic.ID, Weight: 1}}); err != nil {
			t.Fatalf("связать тему: %v", err)
		}
	}

	links, err := service.ListKimTopicsByExamYear(ctx, year.ID)
	if err != nil {
		t.Fatalf("прочитать связи: %v", err)
	}
	if len(links) != 2 {
		t.Fatalf("связей = %d, ожидалось 2", len(links))
	}

	otherExam := createExam(t, ctx, service, "it_links_other_"+uuid.Must(uuid.NewV7()).String())
	otherYear := createYear(t, ctx, service, otherExam.ID)
	otherKim := createKim(t, ctx, service, otherYear.ID, 1, 1, AnswerTypeShort, 1)
	otherTopic := createTopic(t, ctx, service, otherExam.ID, nil, "other", "Другая")
	if _, err := service.SetKimTopics(ctx, otherKim.ID, []KimTopicInput{{TopicID: otherTopic.ID, Weight: 1}}); err != nil {
		t.Fatalf("связать чужую тему: %v", err)
	}

	links, err = service.ListKimTopicsByExamYear(ctx, year.ID)
	if err != nil {
		t.Fatalf("прочитать связи повторно: %v", err)
	}
	if len(links) != 2 {
		t.Fatalf("в выборку попали чужие годы: %d связей", len(links))
	}
}
