package content

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"oge-ege/internal/shared/ids"
)

func buildTaskOptions(taskID uuid.UUID, inputs []TaskOptionInput) ([]TaskOption, error) {
	options := make([]TaskOption, 0, len(inputs))
	for _, input := range inputs {
		id := input.ID
		if id == uuid.Nil {
			generated, err := ids.New()
			if err != nil {
				return nil, fmt.Errorf("сгенерировать идентификатор варианта: %w", err)
			}
			id = generated
		}
		options = append(options, TaskOption{
			ID:        id,
			TaskID:    taskID,
			Position:  input.Position,
			ContentMD: strings.TrimSpace(input.ContentMD),
			IsCorrect: input.IsCorrect,
		})
	}

	return options, nil
}

func validateTaskOptions(answerSpec json.RawMessage, inputs []TaskOptionInput) error {
	if len(inputs) == 0 {
		return nil
	}

	kind, err := answerSpecKind(answerSpec)
	if err != nil {
		return invalid("answer_spec", "правила проверки должны быть корректным JSON")
	}
	if kind != "choice" {
		return invalid("options", "варианты допустимы только для типа choice")
	}

	seenIDs := make(map[uuid.UUID]struct{}, len(inputs))
	seenPositions := make(map[int]struct{}, len(inputs))
	correct := make(map[string]struct{})
	for _, input := range inputs {
		if input.Position <= 0 {
			return invalid("options", "позиция варианта должна быть положительной")
		}
		if _, ok := seenPositions[input.Position]; ok {
			return invalid("options", "позиции вариантов не должны повторяться")
		}
		seenPositions[input.Position] = struct{}{}
		if input.ID != uuid.Nil {
			if _, ok := seenIDs[input.ID]; ok {
				return invalid("options", "идентификаторы вариантов не должны повторяться")
			}
			seenIDs[input.ID] = struct{}{}
		}
		if strings.TrimSpace(input.ContentMD) == "" {
			return invalid("options", "содержимое варианта обязательно")
		}
		if input.IsCorrect {
			correct[input.ID.String()] = struct{}{}
		}
	}
	if len(inputs) < 2 {
		return invalid("options", "для choice нужно не менее двух вариантов")
	}
	if len(correct) == 0 {
		return invalid("options", "для choice нужен хотя бы один правильный вариант")
	}

	optionIDs, err := answerSpecOptionIDs(answerSpec)
	if err != nil {
		return invalid("answer_spec", "правила проверки должны быть корректным JSON")
	}
	declared := make(map[string]struct{}, len(optionIDs))
	for _, id := range optionIDs {
		parsed, err := uuid.Parse(id)
		if err != nil {
			return invalid("answer_spec", "идентификатор правильного варианта некорректен")
		}
		if _, ok := seenIDs[parsed]; !ok {
			return invalid("answer_spec", "правильный вариант отсутствует среди вариантов задания")
		}
		declared[id] = struct{}{}
	}
	for id := range correct {
		if _, ok := declared[id]; !ok {
			return invalid("options", "правильный вариант не указан в answer_spec.option_ids")
		}
	}
	for id := range declared {
		if _, ok := correct[id]; !ok {
			return invalid("options", "вариант из answer_spec.option_ids должен быть отмечен правильным")
		}
	}

	return nil
}

func answerSpecKind(spec json.RawMessage) (string, error) {
	var payload struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(spec, &payload); err != nil {
		return "", err
	}

	return payload.Kind, nil
}

func answerSpecOptionIDs(spec json.RawMessage) ([]string, error) {
	var payload struct {
		OptionIDs []string `json:"option_ids"`
	}
	if err := json.Unmarshal(spec, &payload); err != nil {
		return nil, err
	}

	return payload.OptionIDs, nil
}

func taskOptionsInput(options []TaskOption) []TaskOptionInput {
	inputs := make([]TaskOptionInput, 0, len(options))
	for _, option := range options {
		inputs = append(inputs, TaskOptionInput{
			ID:        option.ID,
			Position:  option.Position,
			ContentMD: option.ContentMD,
			IsCorrect: option.IsCorrect,
		})
	}

	return inputs
}
