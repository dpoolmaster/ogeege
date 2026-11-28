package learning

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"oge-ege/internal/content"
	"oge-ege/internal/grading"
)

func TestBuildTaskSnapshot(t *testing.T) {
	taskID := uuid.New()
	topicID := uuid.New()
	kim := content.KimNumber{ID: uuid.New(), MaxPrimaryScore: 2}
	task := content.Task{
		ID:            taskID,
		Version:       3,
		StatementMD:   "Условие",
		AnswerSpec:    json.RawMessage(`{"kind":"number","accepted":["4"]}`),
		AnswerDisplay: "4",
		Difficulty:    content.DifficultyMedium,
		KimNumberID:   kim.ID,
		TopicID:       &topicID,
	}
	attachments := []content.TaskAttachment{{Kind: "image", URL: "/media/a.png", Alt: "Рисунок"}}

	snapshot := buildTaskSnapshot(task, kim, attachments)
	if snapshot.TaskID != taskID || snapshot.Version != 3 || snapshot.StatementMD != "Условие" {
		t.Fatalf("снимок задания = %+v", snapshot)
	}
	if snapshot.MaxPoints != 2 || snapshot.KimNumberID != kim.ID || snapshot.TopicID == nil || *snapshot.TopicID != topicID {
		t.Fatalf("снимок задания = %+v", snapshot)
	}
	if len(snapshot.Attachments) != 1 || snapshot.Attachments[0].URL != "/media/a.png" {
		t.Fatalf("вложения снимка = %+v", snapshot.Attachments)
	}
}

func TestEvaluateAnswer(t *testing.T) {
	tests := []struct {
		name       string
		spec       string
		answer     string
		wantReason grading.Reason
		wantPoints float64
	}{
		{name: "верный", spec: `{"kind":"number","accepted":["4"]}`, answer: "4", wantReason: grading.ReasonMatched, wantPoints: 1},
		{name: "неверный", spec: `{"kind":"number","accepted":["4"]}`, answer: "5", wantReason: grading.ReasonMismatch, wantPoints: 0},
		{name: "неподдерживаемый", spec: `{"kind":"choice"}`, answer: "1", wantReason: grading.ReasonUnsupported, wantPoints: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := TaskSnapshot{AnswerSpec: json.RawMessage(tt.spec), MaxPoints: 1}
			result, points, err := evaluateAnswer(snapshot, tt.answer)
			if err != nil {
				t.Fatalf("неожиданная ошибка: %v", err)
			}
			if result.Reason != tt.wantReason {
				t.Fatalf("причина = %q, ожидалась %q", result.Reason, tt.wantReason)
			}
			if points != tt.wantPoints {
				t.Fatalf("баллы = %f, ожидалось %f", points, tt.wantPoints)
			}
		})
	}
}
