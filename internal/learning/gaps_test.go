package learning

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"oge-ege/internal/content"
	"oge-ege/internal/shared/render/ui"
)

func TestComputeTopicWeights(t *testing.T) {
	topic1 := content.Topic{ID: uuid.New(), OrderIndex: 1}
	topic2 := content.Topic{ID: uuid.New(), OrderIndex: 2}
	kim1 := content.KimNumber{ID: uuid.New(), Number: 1, MaxPrimaryScore: 1}
	kim2 := content.KimNumber{ID: uuid.New(), Number: 2, MaxPrimaryScore: 3}
	links := []content.KimTopic{
		{KimNumberID: kim1.ID, TopicID: topic1.ID, Weight: 1},
		{KimNumberID: kim2.ID, TopicID: topic1.ID, Weight: 0.5},
		{KimNumberID: kim2.ID, TopicID: topic2.ID, Weight: 1},
	}

	weights := ComputeTopicWeights([]content.Topic{topic1, topic2}, []content.KimNumber{kim1, kim2}, links)
	if diff := weights[topic1.ID] - 0.625; diff > 0.0001 || diff < -0.0001 {
		t.Fatalf("вес темы 1 = %f, ожидалось 0.625", weights[topic1.ID])
	}
	if diff := weights[topic2.ID] - 1; diff > 0.0001 || diff < -0.0001 {
		t.Fatalf("вес темы 2 = %f, ожидалось 1", weights[topic2.ID])
	}
}

func TestComputeTopicWeightsWithoutLinks(t *testing.T) {
	topic := content.Topic{ID: uuid.New()}
	weights := ComputeTopicWeights([]content.Topic{topic}, nil, nil)
	if weights[topic.ID] != 0 {
		t.Fatalf("тема без связей должна иметь вес 0, получено %f", weights[topic.ID])
	}
}

func TestBuildPrioritiesOrdering(t *testing.T) {
	now := time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)
	noAttempts := content.Topic{ID: uuid.New(), Title: "Без попыток", OrderIndex: 1}
	weak := content.Topic{ID: uuid.New(), Title: "Слабый", OrderIndex: 2}
	strong := content.Topic{ID: uuid.New(), Title: "Сильный", OrderIndex: 3}
	mastery := map[uuid.UUID]TopicMastery{
		weak.ID:   {TopicID: weak.ID, Mastery: 0.1, Attempts: 3, LastPracticedAt: &now},
		strong.ID: {TopicID: strong.ID, Mastery: 0.9, Attempts: 5, LastPracticedAt: &now},
	}
	weights := map[uuid.UUID]float64{noAttempts.ID: 1, weak.ID: 1, strong.ID: 1}

	priorities := BuildPriorities([]content.Topic{strong, noAttempts, weak}, weights, mastery, now)
	if len(priorities) != 3 {
		t.Fatalf("приоритетов = %d", len(priorities))
	}
	if priorities[0].TopicID != noAttempts.ID {
		t.Fatalf("первой должна быть тема без попыток, получено %s", priorities[0].Title)
	}
	if priorities[1].TopicID != weak.ID {
		t.Fatalf("второй должна быть слабая тема, получено %s", priorities[1].Title)
	}
	if priorities[2].TopicID != strong.ID {
		t.Fatalf("третьей должна быть сильная тема, получено %s", priorities[2].Title)
	}
}

func TestBuildPrioritiesRecency(t *testing.T) {
	now := time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)
	old := now.AddDate(0, 0, -30)
	fresh := content.Topic{ID: uuid.New(), Title: "Свежая", OrderIndex: 1}
	stale := content.Topic{ID: uuid.New(), Title: "Давняя", OrderIndex: 2}
	mastery := map[uuid.UUID]TopicMastery{
		fresh.ID: {TopicID: fresh.ID, Mastery: 0.5, Attempts: 3, LastPracticedAt: &now},
		stale.ID: {TopicID: stale.ID, Mastery: 0.5, Attempts: 3, LastPracticedAt: &old},
	}
	weights := map[uuid.UUID]float64{fresh.ID: 1, stale.ID: 1}

	priorities := BuildPriorities([]content.Topic{fresh, stale}, weights, mastery, now)
	if priorities[0].TopicID != stale.ID {
		t.Fatal("давняя практика должна повышать приоритет")
	}
}

func TestBuildGapMapNeutralCell(t *testing.T) {
	topic1 := content.Topic{ID: uuid.New(), Title: "Тема 1", OrderIndex: 1}
	topic2 := content.Topic{ID: uuid.New(), Title: "Тема 2", OrderIndex: 2}
	kim1 := content.KimNumber{ID: uuid.New(), Number: 1}
	kim2 := content.KimNumber{ID: uuid.New(), Number: 2}
	links := []content.KimTopic{{KimNumberID: kim1.ID, TopicID: topic1.ID, Weight: 1}}
	topicMastery := map[uuid.UUID]TopicMastery{topic1.ID: {TopicID: topic1.ID, Mastery: 0.9, Attempts: 1}}
	kimMastery := map[uuid.UUID]KimMastery{kim1.ID: {KimNumberID: kim1.ID, Mastery: 0.9, Attempts: 1}}

	view := BuildGapMap([]content.Topic{topic1, topic2}, []content.KimNumber{kim1, kim2}, links, topicMastery, kimMastery, time.Now())
	if !view.HasData {
		t.Fatal("ожидались данные карты")
	}
	if view.Cells[0][0].State != ui.GapCorrect {
		t.Fatalf("ячейка со связью = %q", view.Cells[0][0].State)
	}
	if view.Cells[0][1].Label != "нет связи" {
		t.Fatalf("ячейка без связи = %q", view.Cells[0][1].Label)
	}
}

func TestBuildGapMapEmpty(t *testing.T) {
	topic := content.Topic{ID: uuid.New(), Title: "Тема"}
	kim := content.KimNumber{ID: uuid.New(), Number: 1}
	view := BuildGapMap([]content.Topic{topic}, []content.KimNumber{kim}, nil, nil, nil, time.Now())
	if view.HasData {
		t.Fatal("без освоения карта должна быть пустой")
	}
}
