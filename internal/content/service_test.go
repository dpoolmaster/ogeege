package content

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"

	"oge-ege/internal/shared/db"
)

type validationRepo struct {
	Repository
	exam    *Exam
	year    *ExamYear
	kim     *KimNumber
	topic   *Topic
	kimLink *KimTopic
	topics  map[string]*Topic
	theory  *Theory
	updated *Theory
}

func (r *validationRepo) GetExam(_ context.Context, _ db.Querier, id uuid.UUID) (*Exam, error) {
	if r.exam == nil || r.exam.ID != id {
		return nil, ErrNotFound
	}
	return r.exam, nil
}

func (r *validationRepo) GetExamYear(_ context.Context, _ db.Querier, id uuid.UUID) (*ExamYear, error) {
	if r.year == nil || r.year.ID != id {
		return nil, ErrNotFound
	}
	return r.year, nil
}

func (r *validationRepo) GetKimNumber(_ context.Context, _ db.Querier, id uuid.UUID) (*KimNumber, error) {
	if r.kim == nil || r.kim.ID != id {
		return nil, ErrNotFound
	}
	return r.kim, nil
}

func (r *validationRepo) GetTopic(_ context.Context, _ db.Querier, id uuid.UUID) (*Topic, error) {
	if r.topic == nil || r.topic.ID != id {
		return nil, ErrNotFound
	}
	return r.topic, nil
}

func (r *validationRepo) GetKimTopic(_ context.Context, _ db.Querier, kimNumberID, topicID uuid.UUID) (*KimTopic, error) {
	if r.kimLink == nil || r.kimLink.KimNumberID != kimNumberID || r.kimLink.TopicID != topicID {
		return nil, ErrNotFound
	}
	return r.kimLink, nil
}

func (r *validationRepo) GetTheory(_ context.Context, _ db.Querier, id uuid.UUID) (*Theory, error) {
	if r.theory == nil || r.theory.ID != id {
		return nil, ErrNotFound
	}
	return r.theory, nil
}

func (r *validationRepo) UpdateTheory(_ context.Context, _ db.Querier, theory *Theory) error {
	r.updated = theory
	return nil
}

func TestNormalizeStatement(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "пробелы", in: "  a   b  ", want: "a b"},
		{name: "переносы", in: "a\r\nb\rc", want: "a b c"},
		{name: "формула", in: "$ x^2 $", want: "$ x^2 $"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeStatement(tt.in); got != tt.want {
				t.Fatalf("NormalizeStatement = %q, ожидалось %q", got, tt.want)
			}
		})
	}
}

func TestContentHash(t *testing.T) {
	first := ContentHash("Найдите  x + 1")
	second := ContentHash("  Найдите x + 1  ")
	if !bytes.Equal(first, second) {
		t.Fatal("эквивалентные условия должны иметь одинаковый хеш")
	}
	if bytes.Equal(first, ContentHash("Найдите x + 2")) {
		t.Fatal("разные условия не должны иметь одинаковый хеш")
	}
}

func TestValidateAnswerSpec(t *testing.T) {
	tests := []struct {
		name    string
		spec    string
		wantErr bool
	}{
		{name: "число", spec: `{"kind":"number","accepted":["2"]}`},
		{name: "интервал", spec: `{"kind":"interval","accepted":["[-2;3]"]}`},
		{name: "развёрнутый", spec: `{"kind":"extended"}`},
		{name: "пусто", spec: ``, wantErr: true},
		{name: "не JSON", spec: `{`, wantErr: true},
		{name: "неизвестный тип", spec: `{"kind":"unknown"}`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAnswerSpec([]byte(tt.spec))
			if tt.wantErr && err == nil {
				t.Fatal("ожидалась ошибка")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("неожиданная ошибка: %v", err)
			}
		})
	}
}

func TestValidateTask(t *testing.T) {
	examID := uuid.Must(uuid.NewV7())
	otherExamID := uuid.Must(uuid.NewV7())
	yearID := uuid.Must(uuid.NewV7())
	kimID := uuid.Must(uuid.NewV7())
	topicID := uuid.Must(uuid.NewV7())
	otherTopicID := uuid.Must(uuid.NewV7())

	base := validationRepo{
		exam:  &Exam{ID: examID, Code: "oge_math", Status: StatusPublished},
		year:  &ExamYear{ID: yearID, ExamID: examID, Year: 2026},
		kim:   &KimNumber{ID: kimID, ExamYearID: yearID, Number: 1, Part: 1, MaxPrimaryScore: 1},
		topic: &Topic{ID: topicID, ExamID: examID, Code: "numbers"},
	}
	valid := TaskInput{
		ExamYearID: yearID, KimNumberID: kimID, TopicID: &topicID,
		Difficulty: DifficultyEasy, StatementMD: "Вычислите $1+1$",
		AnswerSpec: json.RawMessage(`{"kind":"number","accepted":["2"]}`), Source: "seed",
	}

	t.Run("успех", func(t *testing.T) {
		repo := base
		repo.kimLink = &KimTopic{KimNumberID: kimID, TopicID: topicID}
		svc := NewService(&repo, nil, fakeTx{}, fixedClock{}, nil, discardLogger())
		if err := svc.validateTask(context.Background(), valid, true); err != nil {
			t.Fatalf("неожиданная ошибка: %v", err)
		}
	})

	tests := []struct {
		name  string
		repo  validationRepo
		input TaskInput
		field string
	}{
		{name: "нет года", repo: validationRepo{exam: base.exam, kim: base.kim, topic: base.topic}, input: valid, field: "exam_year_id"},
		{name: "нет номера", repo: validationRepo{exam: base.exam, year: base.year, topic: base.topic}, input: valid, field: "kim_number_id"},
		{name: "другой год", repo: validationRepo{exam: base.exam, year: base.year, kim: &KimNumber{ID: kimID, ExamYearID: uuid.Must(uuid.NewV7())}, topic: base.topic}, input: valid, field: "kim_number_id"},
		{name: "плохая сложность", repo: base, input: withField(valid, func(in *TaskInput) { in.Difficulty = "unknown" }), field: "difficulty"},
		{name: "пустое условие", repo: base, input: withField(valid, func(in *TaskInput) { in.StatementMD = " " }), field: "statement_md"},
		{name: "плохой answer_spec", repo: base, input: withField(valid, func(in *TaskInput) { in.AnswerSpec = []byte(`{"kind":"x"}`) }), field: "answer_spec"},
		{name: "плохая формула", repo: base, input: withField(valid, func(in *TaskInput) { in.StatementMD = "$x^{2$" }), field: "statement_md"},
		{name: "чужая тема", repo: validationRepo{exam: base.exam, year: base.year, kim: base.kim, topic: &Topic{ID: otherTopicID, ExamID: otherExamID}}, input: withField(valid, func(in *TaskInput) { in.TopicID = &otherTopicID }), field: "topic_id"},
		{name: "публикация без темы", repo: base, input: withField(valid, func(in *TaskInput) { in.TopicID = nil }), field: "topic_id"},
		{name: "тема не связана", repo: base, input: valid, field: "topic_id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.repo
			svc := NewService(&repo, nil, fakeTx{}, fixedClock{}, nil, discardLogger())
			forPublish := tt.name == "публикация без темы" || tt.name == "тема не связана"
			err := svc.validateTask(context.Background(), tt.input, forPublish)
			var validationErr *ValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("ожидалась ошибка валидации, получено %v", err)
			}
			if validationErr.Field != tt.field {
				t.Fatalf("поле = %q, ожидалось %q", validationErr.Field, tt.field)
			}
		})
	}
}

func TestServiceRoleIndependentValidation(t *testing.T) {
	service := NewService(&validationRepo{}, nil, fakeTx{}, fixedClock{}, nil, discardLogger())
	if _, err := service.CreateExam(context.Background(), ExamInput{}); err == nil {
		t.Fatal("ожидалась ошибка для пустого кода")
	}
}

func TestValidSubject(t *testing.T) {
	tests := []struct {
		name  string
		code  string
		valid bool
	}{
		{name: "математика", code: "math", valid: true},
		{name: "физика", code: "physics", valid: true},
		{name: "русский", code: "russian", valid: true},
		{name: "информатика", code: "informatics", valid: true},
		{name: "неизвестный", code: "chemistry", valid: false},
		{name: "пустой", code: "", valid: false},
		{name: "регистр", code: "Math", valid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidSubject(tt.code); got != tt.valid {
				t.Fatalf("ValidSubject(%q) = %v, ожидалось %v", tt.code, got, tt.valid)
			}
		})
	}
}

func TestCreateExamRejectsUnknownSubject(t *testing.T) {
	service := NewService(&validationRepo{}, nil, fakeTx{}, fixedClock{}, nil, discardLogger())
	_, err := service.CreateExam(context.Background(), ExamInput{Code: "oge_physics", Title: "ОГЭ физика", Subject: "chemistry"})
	if err == nil {
		t.Fatal("ожидалась ошибка для неизвестного предмета")
	}

	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("ожидалась ошибка валидации, получено %v", err)
	}
	if validationErr.Field != "subject" {
		t.Fatalf("поле = %q, ожидалось subject", validationErr.Field)
	}
}

func TestUpdateTheory(t *testing.T) {
	theoryID := uuid.Must(uuid.NewV7())
	base := func(status Status) *Theory {
		return &Theory{ID: theoryID, TopicID: uuid.Must(uuid.NewV7()), ContentMD: "Старый текст", Version: 1, Status: status}
	}

	tests := []struct {
		name    string
		theory  *Theory
		input   TheoryInput
		wantErr error
		field   string
	}{
		{name: "черновик", theory: base(StatusDraft), input: TheoryInput{ContentMD: "Новый текст $x^2$"}},
		{name: "опубликованная", theory: base(StatusPublished), input: TheoryInput{ContentMD: "Новый текст"}, wantErr: ErrConflict},
		{name: "нет версии", input: TheoryInput{ContentMD: "Новый текст"}, wantErr: ErrNotFound},
		{name: "пустой текст", theory: base(StatusDraft), input: TheoryInput{ContentMD: "  "}, field: "content_md"},
		{name: "плохая формула", theory: base(StatusDraft), input: TheoryInput{ContentMD: "$x^{2$"}, field: "statement_md"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &validationRepo{theory: tt.theory}
			service := NewService(repo, nil, fakeTx{}, fixedClock{}, nil, discardLogger())

			updated, err := service.UpdateTheory(context.Background(), theoryID, tt.input)
			switch {
			case tt.wantErr != nil:
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("ошибка = %v, ожидалось %v", err, tt.wantErr)
				}
				return
			case tt.field != "":
				var validationErr *ValidationError
				if !errors.As(err, &validationErr) {
					t.Fatalf("ожидалась ошибка валидации, получено %v", err)
				}
				if validationErr.Field != tt.field {
					t.Fatalf("поле = %q, ожидалось %q", validationErr.Field, tt.field)
				}
				return
			case err != nil:
				t.Fatalf("неожиданная ошибка: %v", err)
			}

			if repo.updated == nil {
				t.Fatal("версия не сохранена")
			}
			if updated.ContentMD != tt.input.ContentMD {
				t.Fatalf("текст = %q, ожидалось %q", updated.ContentMD, tt.input.ContentMD)
			}
		})
	}
}

func withField(input TaskInput, mutate func(*TaskInput)) TaskInput {
	mutate(&input)
	return input
}

type fakeTx struct{}

func (fakeTx) WithTx(ctx context.Context, fn func(q db.Querier) error) error {
	return fn(nil)
}

type fixedClock struct{}

func (fixedClock) Now() time.Time {
	return time.Date(2026, time.November, 26, 12, 0, 0, 0, time.UTC)
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func (r *validationRepo) CreateTopic(_ context.Context, _ db.Querier, topic *Topic) error {
	if r.topics == nil {
		r.topics = make(map[string]*Topic)
	}
	key := topic.ExamID.String() + "/" + topic.Code
	if _, ok := r.topics[key]; ok {
		return ErrConflict
	}
	r.topics[key] = topic
	return nil
}

func TestCreateTopicHierarchy(t *testing.T) {
	examID := uuid.Must(uuid.NewV7())
	otherExamID := uuid.Must(uuid.NewV7())
	parentID := uuid.Must(uuid.NewV7())

	tests := []struct {
		name    string
		parent  *uuid.UUID
		setup   func(*validationRepo)
		wantErr bool
		field   string
	}{
		{name: "корневая тема", setup: func(r *validationRepo) {}},
		{name: "вложенная тема", parent: &parentID, setup: func(r *validationRepo) {
			r.topic = &Topic{ID: parentID, ExamID: examID, Code: "parent"}
		}},
		{name: "чужой родитель", parent: &parentID, setup: func(r *validationRepo) {
			r.topic = &Topic{ID: parentID, ExamID: otherExamID, Code: "parent"}
		}, wantErr: true, field: "parent_id"},
		{name: "нет родителя", parent: &parentID, setup: func(r *validationRepo) {}, wantErr: true, field: "parent_id"},
		{name: "дубликат кода", setup: func(r *validationRepo) {
			r.topics = map[string]*Topic{examID.String() + "/numbers": {ID: uuid.Must(uuid.NewV7()), ExamID: examID, Code: "numbers"}}
		}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &validationRepo{exam: &Exam{ID: examID, Code: "oge", Status: StatusPublished}}
			tt.setup(repo)
			service := NewService(repo, nil, fakeTx{}, fixedClock{}, nil, discardLogger())

			_, err := service.CreateTopic(context.Background(), TopicInput{
				ExamID: examID, ParentID: tt.parent, Code: "numbers", Title: "Числа", OrderIndex: 1,
			})
			if tt.wantErr {
				if err == nil {
					t.Fatal("ожидалась ошибка")
				}
				if tt.field != "" {
					var validationErr *ValidationError
					if !errors.As(err, &validationErr) || validationErr.Field != tt.field {
						t.Fatalf("ошибка = %v, ожидалось поле %q", err, tt.field)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("неожиданная ошибка: %v", err)
			}
		})
	}
}

func TestValidateRubricSum(t *testing.T) {
	yearID := uuid.Must(uuid.NewV7())
	kimID := uuid.Must(uuid.NewV7())
	repo := &validationRepo{
		kim: &KimNumber{ID: kimID, ExamYearID: yearID, Number: 2, Part: 2, MaxPrimaryScore: 2},
	}
	service := NewService(repo, nil, fakeTx{}, fixedClock{}, nil, discardLogger())

	t.Run("сумма совпадает", func(t *testing.T) {
		if _, err := service.validateRubric(context.Background(), RubricInput{
			KimNumberID: kimID, ExamYearID: yearID,
			Criteria: RubricCriteria{{Code: "C1", Description: "Решение", MaxPoints: 2}},
		}); err != nil {
			t.Fatalf("неожиданная ошибка: %v", err)
		}
	})

	t.Run("сумма не совпадает", func(t *testing.T) {
		_, err := service.validateRubric(context.Background(), RubricInput{
			KimNumberID: kimID, ExamYearID: yearID,
			Criteria: RubricCriteria{{Code: "C1", Description: "Решение", MaxPoints: 3}},
		})
		var validationErr *ValidationError
		if !errors.As(err, &validationErr) || validationErr.Field != "criteria" {
			t.Fatalf("ошибка = %v", err)
		}
	})
}

func TestValidateTaskOptions(t *testing.T) {
	first := uuid.Must(uuid.NewV7())
	second := uuid.Must(uuid.NewV7())
	third := uuid.Must(uuid.NewV7())
	spec := json.RawMessage(`{"kind":"choice","option_ids":["` + first.String() + `","` + second.String() + `"]}`)

	valid := []TaskOptionInput{
		{ID: first, Position: 1, ContentMD: "Первый", IsCorrect: true},
		{ID: second, Position: 2, ContentMD: "Второй", IsCorrect: true},
		{ID: third, Position: 3, ContentMD: "Третий"},
	}

	tests := []struct {
		name   string
		spec   json.RawMessage
		inputs []TaskOptionInput
		want   bool
	}{
		{name: "корректно", spec: spec, inputs: valid},
		{name: "один вариант", spec: spec, inputs: valid[:1], want: true},
		{name: "нет правильного", spec: spec, inputs: []TaskOptionInput{{ID: first, Position: 1, ContentMD: "А"}, {ID: second, Position: 2, ContentMD: "Б"}}, want: true},
		{name: "дубликат позиции", spec: spec, inputs: []TaskOptionInput{{ID: first, Position: 1, ContentMD: "А", IsCorrect: true}, {ID: second, Position: 1, ContentMD: "Б", IsCorrect: true}}, want: true},
		{name: "пустое содержимое", spec: spec, inputs: []TaskOptionInput{{ID: first, Position: 1, ContentMD: " ", IsCorrect: true}, {ID: second, Position: 2, ContentMD: "Б", IsCorrect: true}}, want: true},
		{name: "лишний option_id", spec: json.RawMessage(`{"kind":"choice","option_ids":["` + third.String() + `"]}`), inputs: valid, want: true},
		{name: "не отмечен правильным", spec: spec, inputs: []TaskOptionInput{{ID: first, Position: 1, ContentMD: "А", IsCorrect: true}, {ID: second, Position: 2, ContentMD: "Б"}}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTaskOptions(tt.spec, tt.inputs)
			if tt.want && err == nil {
				t.Fatal("ожидалась ошибка")
			}
			if !tt.want && err != nil {
				t.Fatalf("неожиданная ошибка: %v", err)
			}
		})
	}
}

func TestValidateTaskOptionsRejectsOtherKinds(t *testing.T) {
	err := validateTaskOptions(json.RawMessage(`{"kind":"number","accepted":["1"]}`), []TaskOptionInput{
		{ID: uuid.Must(uuid.NewV7()), Position: 1, ContentMD: "А"},
	})
	if err == nil {
		t.Fatal("варианты для number должны отклоняться")
	}
}
