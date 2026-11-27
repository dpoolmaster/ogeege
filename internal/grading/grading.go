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
	// ReasonUnsupported — тип проверки не поддерживается публичным контуром.
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

type spec struct {
	Kind          string     `json:"kind"`
	Accepted      []string   `json:"accepted"`
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

// Check проверяет ответ по правилам answer_spec.
func Check(raw json.RawMessage, answer string) (Result, error) {
	var parsed spec
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return Result{}, fmt.Errorf("разобрать answer_spec: %w", err)
	}

	switch parsed.Kind {
	case "number":
		return checkNumber(parsed, answer), nil
	case "text":
		return checkText(parsed, answer), nil
	case "digits":
		return checkDigits(parsed, answer), nil
	case "interval":
		return checkInterval(parsed, answer), nil
	case "choice", "extended":
		return Result{Reason: ReasonUnsupported, Expected: parsed.Accepted}, nil
	case "":
		return Result{}, fmt.Errorf("%w: пустой kind", ErrUnknownKind)
	default:
		return Result{}, fmt.Errorf("%w: %s", ErrUnknownKind, parsed.Kind)
	}
}

// Supports сообщает, поддерживается ли тип ответа публичной проверкой.
func Supports(kind string) bool {
	switch kind {
	case "number", "text", "digits", "interval":
		return true
	default:
		return false
	}
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
