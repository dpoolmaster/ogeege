package learning

import (
	"encoding/json"
	"strings"
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

	snapshot := buildTaskSnapshot(task, kim, attachments, nil)
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
		{name: "неподдерживаемый", spec: `{"kind":"extended"}`, answer: "1", wantReason: grading.ReasonNeedsReview, wantPoints: 0},
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

func TestBuildTaskSnapshotOptionsWithoutCorrectness(t *testing.T) {
	kim := content.KimNumber{ID: uuid.New(), MaxPrimaryScore: 1}
	task := content.Task{
		ID: uuid.New(), KimNumberID: kim.ID, StatementMD: "Выберите",
		AnswerSpec: json.RawMessage(`{"kind":"choice","option_ids":["a"]}`),
	}
	options := []content.TaskOption{
		{ID: uuid.New(), Position: 1, ContentMD: "Верный", IsCorrect: true},
		{ID: uuid.New(), Position: 2, ContentMD: "Неверный"},
	}

	snapshot := buildTaskSnapshot(task, kim, nil, options)
	if len(snapshot.Options) != 2 || snapshot.Options[0].ContentMD != "Верный" {
		t.Fatalf("варианты снимка = %+v", snapshot.Options)
	}

	raw, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("сериализовать снимок: %v", err)
	}
	if strings.Contains(string(raw), "is_correct") {
		t.Fatal("снимок не должен раскрывать правильность вариантов")
	}
}

func TestOldSnapshotWithoutOptionsReads(t *testing.T) {
	raw := json.RawMessage(`{"task_id":"01930000-0000-7000-8000-000000000001","statement_md":"Условие","answer_spec":{"kind":"number","accepted":["1"]}}`)
	var snapshot TaskSnapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		t.Fatalf("старый снимок без options должен читаться: %v", err)
	}
	if len(snapshot.Options) != 0 {
		t.Fatalf("options = %+v", snapshot.Options)
	}
}

func TestSessionResponseHidesCorrectness(t *testing.T) {
	optionID := uuid.New()
	view := SessionView{
		Session: Session{ID: uuid.New(), Kind: SessionKindPractice, Status: SessionInProgress},
		Items: []SessionItemView{{
			Item: SessionItem{ID: uuid.New(), Position: 1, Status: SessionItemPending},
			Snapshot: TaskSnapshot{
				StatementMD: "Выберите",
				AnswerSpec:  json.RawMessage(`{"kind":"choice","option_ids":["` + optionID.String() + `"]}`),
				Options:     []OptionSnapshot{{ID: optionID, Position: 1, ContentMD: "Верный"}},
			},
		}},
	}

	response := sessionResponse(view)
	if len(response.Items[0].Options) != 1 || response.Items[0].Options[0].ID != optionID {
		t.Fatalf("варианты JSON = %+v", response.Items[0].Options)
	}
	raw, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("сериализовать ответ: %v", err)
	}
	if strings.Contains(string(raw), "is_correct") {
		t.Fatal("JSON сессии не должен раскрывать правильность вариантов")
	}
}
