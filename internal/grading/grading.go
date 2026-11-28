// Package grading проверяет короткие ответы по стратегиям answer_spec.kind.
// Пакет не зависит от базы данных, HTTP и модуля content.
package grading

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Reason — причина результата проверки.
type Reason string

const (
	// ReasonMatched — ответ совпал с эталоном.
	ReasonMatched Reason = "matched"
	// ReasonMismatch — ответ не совпал с эталоном.
	ReasonMismatch Reason = "mismatch"
	// ReasonPartial — ответ совпал частично.
	ReasonPartial Reason = "partial"
	// ReasonNeedsReview — ответ требует ручной проверки.
	ReasonNeedsReview Reason = "needs_review"
	// ReasonUnsupported — тип проверки не поддерживается.
	ReasonUnsupported Reason = "unsupported"
)

// Result содержит результат проверки короткого ответа.
type Result struct {
	Correct          bool
	Reason           Reason
	NormalizedAnswer string
	Expected         []string
}

// ErrUnknownKind возвращается для неизвестного answer_spec.kind.
var ErrUnknownKind = errors.New("неизвестный тип проверки ответа")

// ErrInvalidSpec возвращается для некорректных правил проверки.
var ErrInvalidSpec = errors.New("некорректные правила проверки ответа")

type spec struct {
	Kind          string     `json:"kind"`
	Accepted      []string   `json:"accepted"`
	OptionIDs     []string   `json:"option_ids"`
	Tolerance     *tolerance `json:"tolerance"`
	Units         []string   `json:"units"`
	CaseSensitive bool       `json:"case_sensitive"`
	IgnoreYo      bool       `json:"ignore_yo"`
	IgnoreHyphen  bool       `json:"ignore_hyphen"`
	Length        int        `json:"length"`
	OrderMatters  *bool      `json:"order_matters"`
}

type tolerance struct {
	Type  string  `json:"type"`
	Value float64 `json:"value"`
}

type strategy struct {
	check     func(spec, string) Result
	validate  func(spec) error
	checkable bool
	public    bool
}

var registry = map[string]strategy{
	"number":   {check: checkNumber, validate: validateNumber, checkable: true, public: true},
	"text":     {check: checkText, validate: validateText, checkable: true, public: true},
	"digits":   {check: checkDigits, validate: validateDigits, checkable: true, public: true},
	"interval": {check: checkInterval, validate: validateInterval, checkable: true, public: true},
	"choice":   {check: checkChoice, validate: validateChoice, checkable: true, public: false},
	"extended": {check: nil, validate: validateExtended, checkable: false, public: false},
}

// Check проверяет ответ по правилам answer_spec.
func Check(raw json.RawMessage, answer string) (Result, error) {
	parsed, err := parse(raw)
	if err != nil {
		return Result{}, err
	}

	entry, ok := registry[parsed.Kind]
	if !ok {
		return Result{}, fmt.Errorf("%w: %s", ErrUnknownKind, parsed.Kind)
	}
	if !entry.checkable || entry.check == nil {
		return Result{Reason: ReasonNeedsReview, Expected: parsed.Accepted}, nil
	}

	return entry.check(parsed, answer), nil
}

// Validate проверяет форму answer_spec по схеме его типа.
func Validate(raw json.RawMessage) error {
	parsed, err := parse(raw)
	if err != nil {
		return err
	}

	entry, ok := registry[parsed.Kind]
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnknownKind, parsed.Kind)
	}

	return entry.validate(parsed)
}

// Kinds возвращает все типы, для которых есть валидация answer_spec.
func Kinds() []string {
	return collectKinds(func(entry strategy) bool { return true })
}

// CheckableKinds возвращает типы с автоматической проверкой короткого ответа.
func CheckableKinds() []string {
	return collectKinds(func(entry strategy) bool { return entry.checkable })
}

// PublicKinds возвращает типы, доступные публичному контуру.
func PublicKinds() []string {
	return collectKinds(func(entry strategy) bool { return entry.public })
}

// Supports сообщает, поддерживается ли тип автоматической проверкой.
func Supports(kind string) bool {
	entry, ok := registry[kind]

	return ok && entry.checkable
}

// SupportsPublic сообщает, доступен ли тип публичному контуру.
func SupportsPublic(kind string) bool {
	entry, ok := registry[kind]

	return ok && entry.public
}

func parse(raw json.RawMessage) (spec, error) {
	var parsed spec
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return spec{}, fmt.Errorf("разобрать answer_spec: %w", err)
	}
	if parsed.Kind == "" {
		return spec{}, fmt.Errorf("%w: пустой kind", ErrUnknownKind)
	}

	return parsed, nil
}

func collectKinds(include func(strategy) bool) []string {
	kinds := make([]string, 0, len(registry))
	for _, kind := range []string{"number", "text", "digits", "interval", "choice", "extended"} {
		if entry, ok := registry[kind]; ok && include(entry) {
			kinds = append(kinds, kind)
		}
	}

	return kinds
}

func matched(expected []string, normalized string) Result {
	return Result{
		Correct:          true,
		Reason:           ReasonMatched,
		NormalizedAnswer: normalized,
		Expected:         expected,
	}
}

func mismatch(expected []string, normalized string) Result {
	return Result{
		Correct:          false,
		Reason:           ReasonMismatch,
		NormalizedAnswer: normalized,
		Expected:         expected,
	}
}

func invalidSpec(message string) error {
	return fmt.Errorf("%w: %s", ErrInvalidSpec, message)
}

func validateNumber(parsed spec) error {
	if len(parsed.Accepted) == 0 {
		return invalidSpec("поле accepted обязательно")
	}
	for _, item := range parsed.Accepted {
		if _, ok := parseNumber(item, parsed.Units); !ok {
			return invalidSpec("эталонный ответ должен быть числом")
		}
	}
	if parsed.Tolerance != nil {
		switch parsed.Tolerance.Type {
		case "absolute", "relative":
		default:
			return invalidSpec("неизвестный тип допуска")
		}
		if parsed.Tolerance.Value < 0 {
			return invalidSpec("допуск не может быть отрицательным")
		}
	}

	return nil
}

func validateText(parsed spec) error {
	if len(parsed.Accepted) == 0 {
		return invalidSpec("поле accepted обязательно")
	}

	return nil
}

func validateDigits(parsed spec) error {
	if len(parsed.Accepted) == 0 {
		return invalidSpec("поле accepted обязательно")
	}
	if parsed.Length < 0 {
		return invalidSpec("длина не может быть отрицательной")
	}
	for _, item := range parsed.Accepted {
		if _, ok := onlyDigits(item); !ok {
			return invalidSpec("эталонный ответ должен состоять из цифр")
		}
	}

	return nil
}

func validateInterval(parsed spec) error {
	if len(parsed.Accepted) == 0 {
		return invalidSpec("поле accepted обязательно")
	}

	return nil
}

func validateChoice(parsed spec) error {
	if len(parsed.OptionIDs) == 0 {
		return invalidSpec("поле option_ids обязательно")
	}
	seen := make(map[string]struct{}, len(parsed.OptionIDs))
	for _, id := range parsed.OptionIDs {
		if id == "" {
			return invalidSpec("идентификатор варианта не может быть пустым")
		}
		if _, ok := seen[id]; ok {
			return invalidSpec("идентификаторы вариантов не должны повторяться")
		}
		seen[id] = struct{}{}
	}

	return nil
}

func validateExtended(spec) error {
	return nil
}
