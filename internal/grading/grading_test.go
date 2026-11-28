package grading

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestCheckNumber(t *testing.T) {
	tests := []struct {
		name    string
		spec    string
		answer  string
		correct bool
	}{
		{name: "точка и запятая", spec: `{"kind":"number","accepted":["2.5"]}`, answer: "2,5", correct: true},
		{name: "пробелы и тысячи", spec: `{"kind":"number","accepted":["1234"]}`, answer: "1 234", correct: true},
		{name: "точка как тысяча", spec: `{"kind":"number","accepted":["1234"]}`, answer: "1.234", correct: false},
		{name: "дробь", spec: `{"kind":"number","accepted":["2.5","5/2"]}`, answer: "5/2", correct: true},
		{name: "отрицательный минус", spec: `{"kind":"number","accepted":["-3"]}`, answer: "−3", correct: true},
		{name: "единицы", spec: `{"kind":"number","accepted":["12"],"units":["см"]}`, answer: "12 см", correct: true},
		{name: "абсолютный допуск", spec: `{"kind":"number","accepted":["0.5"],"tolerance":{"type":"absolute","value":0.01}}`, answer: "0.505", correct: true},
		{name: "относительный допуск", spec: `{"kind":"number","accepted":["100"],"tolerance":{"type":"relative","value":0.01}}`, answer: "100.5", correct: true},
		{name: "неверный", spec: `{"kind":"number","accepted":["2"]}`, answer: "3", correct: false},
		{name: "не число", spec: `{"kind":"number","accepted":["2"]}`, answer: "два", correct: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Check(json.RawMessage(tt.spec), tt.answer)
			if err != nil {
				t.Fatalf("неожиданная ошибка: %v", err)
			}
			if result.Correct != tt.correct {
				t.Fatalf("Correct = %v, ожидалось %v (result: %+v)", result.Correct, tt.correct, result)
			}
			if result.Reason != ReasonMatched && result.Reason != ReasonMismatch {
				t.Fatalf("Reason = %q", result.Reason)
			}
		})
	}
}

func TestCheckText(t *testing.T) {
	tests := []struct {
		name    string
		spec    string
		answer  string
		correct bool
	}{
		{name: "регистр", spec: `{"kind":"text","accepted":["ЖИ-ШИ"]}`, answer: "жи-ши", correct: true},
		{name: "ё", spec: `{"kind":"text","accepted":["ежик"],"ignore_yo":true}`, answer: "ёжик", correct: true},
		{name: "регистр важен", spec: `{"kind":"text","accepted":["Да"],"case_sensitive":true}`, answer: "да", correct: false},
		{name: "дефис", spec: `{"kind":"text","accepted":["жи ши"],"ignore_hyphen":true}`, answer: "жи-ши", correct: true},
		{name: "лишние пробелы", spec: `{"kind":"text","accepted":["жи ши"]}`, answer: "  жи   ши ", correct: true},
		{name: "несколько формулировок", spec: `{"kind":"text","accepted":["два","2"]}`, answer: "2", correct: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Check(json.RawMessage(tt.spec), tt.answer)
			if err != nil {
				t.Fatalf("неожиданная ошибка: %v", err)
			}
			if result.Correct != tt.correct {
				t.Fatalf("Correct = %v, ожидалось %v", result.Correct, tt.correct)
			}
		})
	}
}

func TestCheckDigits(t *testing.T) {
	tests := []struct {
		name    string
		spec    string
		answer  string
		correct bool
	}{
		{name: "порядок важен", spec: `{"kind":"digits","accepted":["1324"]}`, answer: "1324", correct: true},
		{name: "порядок нарушен", spec: `{"kind":"digits","accepted":["1324"]}`, answer: "1234", correct: false},
		{name: "порядок не важен", spec: `{"kind":"digits","accepted":["1324"],"order_matters":false}`, answer: "4231", correct: true},
		{name: "длина", spec: `{"kind":"digits","accepted":["1324"],"length":4}`, answer: "132", correct: false},
		{name: "не цифры", spec: `{"kind":"digits","accepted":["1324"]}`, answer: "13a4", correct: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Check(json.RawMessage(tt.spec), tt.answer)
			if err != nil {
				t.Fatalf("неожиданная ошибка: %v", err)
			}
			if result.Correct != tt.correct {
				t.Fatalf("Correct = %v, ожидалось %v", result.Correct, tt.correct)
			}
		})
	}
}

func TestCheckInterval(t *testing.T) {
	tests := []struct {
		name    string
		spec    string
		answer  string
		correct bool
	}{
		{name: "совпадает", spec: `{"kind":"interval","accepted":["[-2;3]"]}`, answer: "[-2;3]", correct: true},
		{name: "обратный порядок", spec: `{"kind":"interval","accepted":["[-2;3]"]}`, answer: "3; -2", correct: true},
		{name: "другие скобки", spec: `{"kind":"interval","accepted":["[-2;3]"]}`, answer: "(-2;3)", correct: false},
		{name: "запятая", spec: `{"kind":"interval","accepted":["[-2;3]"]}`, answer: "[-2, 3]", correct: true},
		{name: "fallback", spec: `{"kind":"interval","accepted":["пусто"]}`, answer: "пусто", correct: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Check(json.RawMessage(tt.spec), tt.answer)
			if err != nil {
				t.Fatalf("неожиданная ошибка: %v", err)
			}
			if result.Correct != tt.correct {
				t.Fatalf("Correct = %v, ожидалось %v", result.Correct, tt.correct)
			}
		})
	}
}

func TestCheckErrorsAndUnsupported(t *testing.T) {
	tests := []struct {
		name       string
		spec       string
		wantErr    bool
		wantReason Reason
	}{
		{name: "некорректный JSON", spec: `{`, wantErr: true},
		{name: "неизвестный kind", spec: `{"kind":"magic"}`, wantErr: true},
		{name: "пустой kind", spec: `{"kind":""}`, wantErr: true},
		{name: "extended", spec: `{"kind":"extended"}`, wantReason: ReasonNeedsReview},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Check(json.RawMessage(tt.spec), "2")
			if tt.wantErr {
				if err == nil {
					t.Fatal("ожидалась ошибка")
				}
				if !errors.Is(err, ErrUnknownKind) && tt.name == "неизвестный kind" {
					t.Fatalf("ошибка не оборачивает ErrUnknownKind: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("неожиданная ошибка: %v", err)
			}
			if result.Correct {
				t.Fatal("неподдерживаемый тип не должен считаться верным")
			}
			if result.Reason != tt.wantReason {
				t.Fatalf("Reason = %q, ожидалось %q", result.Reason, tt.wantReason)
			}
		})
	}
}

func TestSupports(t *testing.T) {
	for _, kind := range []string{"number", "text", "digits", "interval", "choice"} {
		if !Supports(kind) {
			t.Fatalf("kind %q должен поддерживаться", kind)
		}
	}
	for _, kind := range []string{"extended", "magic"} {
		if Supports(kind) {
			t.Fatalf("kind %q не должен поддерживаться", kind)
		}
	}
	for _, kind := range []string{"number", "text", "digits", "interval"} {
		if !SupportsPublic(kind) {
			t.Fatalf("kind %q должен поддерживаться публично", kind)
		}
	}
	for _, kind := range []string{"choice", "extended", "magic"} {
		if SupportsPublic(kind) {
			t.Fatalf("kind %q не должен поддерживаться публично", kind)
		}
	}
}

func TestKinds(t *testing.T) {
	tests := []struct {
		name string
		got  []string
		want []string
	}{
		{name: "все", got: Kinds(), want: []string{"number", "text", "digits", "interval", "choice", "extended"}},
		{name: "проверяемые", got: CheckableKinds(), want: []string{"number", "text", "digits", "interval", "choice"}},
		{name: "публичные", got: PublicKinds(), want: []string{"number", "text", "digits", "interval"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.got) != len(tt.want) {
				t.Fatalf("Kinds = %v, ожидалось %v", tt.got, tt.want)
			}
			for index := range tt.want {
				if tt.got[index] != tt.want[index] {
					t.Fatalf("Kinds = %v, ожидалось %v", tt.got, tt.want)
				}
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		spec    string
		wantErr bool
	}{
		{name: "число", spec: `{"kind":"number","accepted":["2"]}`},
		{name: "текст", spec: `{"kind":"text","accepted":["да"]}`},
		{name: "цифры", spec: `{"kind":"digits","accepted":["1324"]}`},
		{name: "интервал", spec: `{"kind":"interval","accepted":["[-2;3]"]}`},
		{name: "выбор", spec: `{"kind":"choice","option_ids":["a","b"]}`},
		{name: "развёрнутый", spec: `{"kind":"extended"}`},
		{name: "неизвестный", spec: `{"kind":"magic"}`, wantErr: true},
		{name: "пустой", spec: `{"kind":""}`, wantErr: true},
		{name: "битый JSON", spec: `{`, wantErr: true},
		{name: "число без accepted", spec: `{"kind":"number"}`, wantErr: true},
		{name: "выбор без option_ids", spec: `{"kind":"choice"}`, wantErr: true},
		{name: "плохой допуск", spec: `{"kind":"number","accepted":["1"],"tolerance":{"type":"magic","value":1}}`, wantErr: true},
		{name: "отрицательный допуск", spec: `{"kind":"number","accepted":["1"],"tolerance":{"type":"absolute","value":-1}}`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(json.RawMessage(tt.spec))
			if tt.wantErr && err == nil {
				t.Fatal("ожидалась ошибка")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("неожиданная ошибка: %v", err)
			}
		})
	}
}

func TestCheckChoice(t *testing.T) {
	tests := []struct {
		name    string
		spec    string
		answer  string
		correct bool
	}{
		{name: "верный", spec: `{"kind":"choice","option_ids":["a"]}`, answer: "a", correct: true},
		{name: "неверный", spec: `{"kind":"choice","option_ids":["a"]}`, answer: "b", correct: false},
		{name: "несколько без порядка", spec: `{"kind":"choice","option_ids":["a","b"]}`, answer: "b,a", correct: true},
		{name: "json массив", spec: `{"kind":"choice","option_ids":["a","b"]}`, answer: `["a","b"]`, correct: true},
		{name: "лишний", spec: `{"kind":"choice","option_ids":["a","b"]}`, answer: "a,b,c", correct: false},
		{name: "не все", spec: `{"kind":"choice","option_ids":["a","b"]}`, answer: "a", correct: false},
		{name: "пустой", spec: `{"kind":"choice","option_ids":["a"]}`, answer: "", correct: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Check(json.RawMessage(tt.spec), tt.answer)
			if err != nil {
				t.Fatalf("неожиданная ошибка: %v", err)
			}
			if result.Correct != tt.correct {
				t.Fatalf("Correct = %v, ожидалось %v", result.Correct, tt.correct)
			}
			if !tt.correct && len(result.Expected) != 0 {
				t.Fatal("правильный набор не должен раскрываться")
			}
		})
	}
}

func TestResultContract(t *testing.T) {
	result, err := Check(json.RawMessage(`{"kind":"number","accepted":["2.5"]}`), "2,5")
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if !result.Correct || result.Reason != ReasonMatched {
		t.Fatalf("результат = %+v", result)
	}
	if result.NormalizedAnswer != "2.5" {
		t.Fatalf("нормализованный ответ = %q", result.NormalizedAnswer)
	}
	if len(result.Expected) != 1 || result.Expected[0] != "2.5" {
		t.Fatalf("эталон = %v", result.Expected)
	}

	miss, err := Check(json.RawMessage(`{"kind":"number","accepted":["2.5"]}`), "3")
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if miss.Correct || miss.Reason != ReasonMismatch || miss.NormalizedAnswer != "3" {
		t.Fatalf("результат = %+v", miss)
	}
}

func TestBoundaryCases(t *testing.T) {
	tests := []struct {
		name    string
		spec    string
		answer  string
		correct bool
	}{
		{name: "минус unicode с пробелом", spec: `{"kind":"number","accepted":["-1000"]}`, answer: "− 1 000", correct: true},
		{name: "апостроф тысячи", spec: `{"kind":"number","accepted":["1000"]}`, answer: "1'000", correct: true},
		{name: "дробь эквивалентна", spec: `{"kind":"number","accepted":["2.5"]}`, answer: "5/2", correct: true},
		{name: "нулевой относительный допуск", spec: `{"kind":"number","accepted":["0"],"tolerance":{"type":"relative","value":0.01}}`, answer: "0.005", correct: true},
		{name: "отрицательный допуск игнорируется", spec: `{"kind":"number","accepted":["1"],"tolerance":{"type":"absolute","value":-1}}`, answer: "1.5", correct: false},
		{name: "единицы в начале", spec: `{"kind":"number","accepted":["12"],"units":["см"]}`, answer: "см 12", correct: true},
		{name: "несколько эталонов", spec: `{"kind":"number","accepted":["1","2","3"]}`, answer: "2", correct: true},
		{name: "интервал без скобок", spec: `{"kind":"interval","accepted":["[-2;3]"]}`, answer: "-2;3", correct: true},
		{name: "множество без порядка", spec: `{"kind":"interval","accepted":["{1;2;3}"]}`, answer: "{3, 1, 2}", correct: true},
		{name: "множество другой набор", spec: `{"kind":"interval","accepted":["{1;2;3}"]}`, answer: "{1;2}", correct: false},
		{name: "некорректный интервал", spec: `{"kind":"interval","accepted":["[-2;3]"]}`, answer: "abc", correct: false},
		{name: "цифры порядок не важен", spec: `{"kind":"digits","accepted":["123"],"order_matters":false}`, answer: "321", correct: true},
		{name: "цифры длина", spec: `{"kind":"digits","accepted":["123"],"length":3}`, answer: "12", correct: false},
		{name: "текст ё", spec: `{"kind":"text","accepted":["ёж"],"ignore_yo":true}`, answer: "еж", correct: true},
		{name: "текст дефис", spec: `{"kind":"text","accepted":["жи-ши"],"ignore_hyphen":true}`, answer: "жи ши", correct: true},
		{name: "текст повторные пробелы", spec: `{"kind":"text","accepted":["жи ши"]}`, answer: "жи     ши", correct: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Check(json.RawMessage(tt.spec), tt.answer)
			if err != nil {
				t.Fatalf("неожиданная ошибка: %v", err)
			}
			if result.Correct != tt.correct {
				t.Fatalf("Correct = %v, ожидалось %v (result: %+v)", result.Correct, tt.correct, result)
			}
		})
	}
}
