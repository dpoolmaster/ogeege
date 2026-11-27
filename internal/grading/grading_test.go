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
		name        string
		spec        string
		wantErr     bool
		wantReason  Reason
		unsupported bool
	}{
		{name: "некорректный JSON", spec: `{`, wantErr: true},
		{name: "неизвестный kind", spec: `{"kind":"magic"}`, wantErr: true},
		{name: "пустой kind", spec: `{"kind":""}`, wantErr: true},
		{name: "choice", spec: `{"kind":"choice","accepted":["2"]}`, wantReason: ReasonUnsupported, unsupported: true},
		{name: "extended", spec: `{"kind":"extended"}`, wantReason: ReasonUnsupported, unsupported: true},
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
			if tt.unsupported && result.Correct {
				t.Fatal("неподдерживаемый тип не должен считаться верным")
			}
			if result.Reason != tt.wantReason {
				t.Fatalf("Reason = %q, ожидалось %q", result.Reason, tt.wantReason)
			}
		})
	}
}

func TestSupports(t *testing.T) {
	for _, kind := range []string{"number", "text", "digits", "interval"} {
		if !Supports(kind) {
			t.Fatalf("kind %q должен поддерживаться", kind)
		}
	}
	for _, kind := range []string{"choice", "extended", "magic"} {
		if Supports(kind) {
			t.Fatalf("kind %q не должен поддерживаться", kind)
		}
	}
}
